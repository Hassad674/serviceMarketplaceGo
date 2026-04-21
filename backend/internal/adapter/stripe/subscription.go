package stripe

import (
	"context"
	"errors"
	"fmt"
	"time"

	stripe "github.com/stripe/stripe-go/v82"
	billingportalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/price"
	stripesub "github.com/stripe/stripe-go/v82/subscription"

	portservice "marketplace-backend/internal/port/service"
)

// SubscriptionService implements port/service.StripeSubscriptionService
// using the Stripe Subscriptions API. Kept in its own struct (not methods
// on the main stripe.Service) so the subscription feature stays
// removable: delete this file + the wiring in main.go and the Stripe
// adapter compiles without Premium support.
type SubscriptionService struct {
	// apiKey is passed to every Stripe API call via the stripe.Key
	// package-level var, which NewService already set. Kept here only
	// in case a future refactor swaps the global for a scoped client.
	apiKey string
}

// NewSubscriptionService returns a new subscription adapter. secretKey
// MUST be the Stripe secret key already loaded into stripe.Key by
// NewService; the SDK does not tolerate the global being empty.
func NewSubscriptionService(secretKey string) *SubscriptionService {
	return &SubscriptionService{apiKey: secretKey}
}

// EnsureCustomer looks up (by metadata.user_id) a Stripe customer for
// the given user and creates one if absent. Idempotent across concurrent
// callers thanks to the lookup-first strategy; two racing Subscribe
// calls can result in a duplicate customer but Stripe tolerates that and
// the subscription is attached to the one chosen by the CreateSession
// call — acceptable cost for keeping the code simple.
func (s *SubscriptionService) EnsureCustomer(ctx context.Context, userID, email, displayName string) (string, error) {
	// Search by metadata['user_id'] to find an existing customer. This
	// avoids duplicates across re-subscriptions.
	searchParams := &stripe.CustomerSearchParams{
		SearchParams: stripe.SearchParams{
			Query:   fmt.Sprintf("metadata['user_id']:'%s'", userID),
			Context: ctx,
			Limit:   stripe.Int64(1),
		},
	}
	iter := customer.Search(searchParams)
	if iter.Next() {
		return iter.Customer().ID, nil
	}
	if err := iter.Err(); err != nil && !isStripeNoMatchErr(err) {
		return "", fmt.Errorf("search customer: %w", err)
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(displayName),
	}
	params.AddMetadata("user_id", userID)
	params.Context = ctx
	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("create customer: %w", err)
	}
	return c.ID, nil
}

// ResolvePriceID maps a stable lookup_key (e.g. "premium_freelance_monthly")
// to the real Stripe price id. The seed script creates prices with
// lookup_keys; the application code never hardcodes price IDs.
func (s *SubscriptionService) ResolvePriceID(ctx context.Context, lookupKey string) (string, error) {
	params := &stripe.PriceListParams{
		LookupKeys: stripe.StringSlice([]string{lookupKey}),
	}
	params.Context = ctx
	params.Active = stripe.Bool(true)
	iter := price.List(params)
	for iter.Next() {
		p := iter.Price()
		if p.LookupKey == lookupKey {
			return p.ID, nil
		}
	}
	if err := iter.Err(); err != nil {
		return "", fmt.Errorf("list prices: %w", err)
	}
	return "", fmt.Errorf("no active price with lookup_key %q (did you run `make seed-stripe`?)", lookupKey)
}

// CreateCheckoutSession builds a Stripe Checkout URL the user is
// redirected to. CancelAtPeriodEnd is forwarded to subscription_data so
// the very first charge creates a subscription with auto-renew OFF —
// matching the product default.
func (s *SubscriptionService) CreateCheckoutSession(ctx context.Context, in portservice.CreateCheckoutSessionInput) (string, error) {
	if in.SuccessURL == "" || in.CancelURL == "" {
		return "", errors.New("checkout session: success and cancel URLs are required")
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   stripe.String(in.CustomerID),
		SuccessURL: stripe.String(in.SuccessURL),
		CancelURL:  stripe.String(in.CancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(in.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": in.UserID,
			},
		},
	}
	if in.CancelAtPeriodEnd {
		// Stripe's SubscriptionData does not expose cancel_at_period_end
		// at creation time for Checkout sessions — we apply it via a
		// post-creation subscription.update in the webhook handler once
		// the subscription id is known. The flag MUST live on the
		// SubscriptionData metadata (not the session metadata) so it
		// propagates onto the Stripe Subscription object itself,
		// where the webhook handler can read it.
		params.SubscriptionData.Metadata["cancel_at_period_end"] = "true"
	}
	params.Context = ctx

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return sess.URL, nil
}

// UpdateCancelAtPeriodEnd flips the flag on the Stripe subscription and
// returns a snapshot of the resulting state. The caller persists the
// snapshot so our DB stays in sync.
func (s *SubscriptionService) UpdateCancelAtPeriodEnd(ctx context.Context, stripeSubID string, cancelAtEnd bool) (portservice.SubscriptionSnapshot, error) {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(cancelAtEnd),
	}
	params.Context = ctx
	sub, err := stripesub.Update(stripeSubID, params)
	if err != nil {
		return portservice.SubscriptionSnapshot{}, fmt.Errorf("update cancel_at_period_end: %w", err)
	}
	return toSnapshot(sub), nil
}

// ChangeCycle swaps the subscription to a new price (monthly <-> annual).
//
// Direction governs proration:
//
//   - Upgrade (monthly → annual, `prorateImmediately=true`): Stripe
//     invoices the delta immediately via proration_behavior=always_invoice.
//   - Downgrade (annual → monthly, `prorateImmediately=false`): Stripe
//     keeps the current period intact via proration_behavior=none. No
//     credit, no refund. The new price applies only on the next renewal
//     — matching the product rule "annual is prepaid, the user keeps the
//     benefit until the period ends".
//
// The caller reflects the returned snapshot (new price, period dates) into
// its row. On downgrade, current_period_end does not change.
func (s *SubscriptionService) ChangeCycle(ctx context.Context, stripeSubID, newPriceID string, prorateImmediately bool) (portservice.SubscriptionSnapshot, error) {
	// Fetch current subscription to get the active item id — we UPDATE
	// the price, not create a new line item.
	existing, err := stripesub.Get(stripeSubID, &stripe.SubscriptionParams{
		Params: stripe.Params{Context: ctx},
	})
	if err != nil {
		return portservice.SubscriptionSnapshot{}, fmt.Errorf("fetch subscription for cycle change: %w", err)
	}
	if existing.Items == nil || len(existing.Items.Data) == 0 {
		return portservice.SubscriptionSnapshot{}, errors.New("subscription has no items")
	}
	itemID := existing.Items.Data[0].ID

	prorationBehavior := "none"
	if prorateImmediately {
		prorationBehavior = "always_invoice"
	}

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(itemID),
				Price: stripe.String(newPriceID),
			},
		},
		ProrationBehavior: stripe.String(prorationBehavior),
	}
	params.Context = ctx

	updated, err := stripesub.Update(stripeSubID, params)
	if err != nil {
		return portservice.SubscriptionSnapshot{}, fmt.Errorf("change cycle update: %w", err)
	}
	return toSnapshot(updated), nil
}

// CreatePortalSession returns a Customer Portal URL. The Portal handles
// payment method updates, invoices, and lets the user cancel — we do
// NOT reimplement those screens.
func (s *SubscriptionService) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}
	params.Context = ctx
	sess, err := billingportalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}
	return sess.URL, nil
}

// toSnapshot projects the verbose stripe.Subscription into our
// SubscriptionSnapshot so the app layer stays free of SDK types.
func toSnapshot(sub *stripe.Subscription) portservice.SubscriptionSnapshot {
	snap := portservice.SubscriptionSnapshot{
		ID:                sub.ID,
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.Price != nil {
			snap.PriceID = item.Price.ID
		}
		if item.CurrentPeriodStart > 0 {
			snap.CurrentPeriodStart = time.Unix(item.CurrentPeriodStart, 0).UTC()
		}
		if item.CurrentPeriodEnd > 0 {
			snap.CurrentPeriodEnd = time.Unix(item.CurrentPeriodEnd, 0).UTC()
		}
	}
	return snap
}

// isStripeNoMatchErr returns true when the Stripe Search API returns an
// empty result set via an error rather than an empty iterator. Defensive
// — the SDK behaviour differs between versions.
func isStripeNoMatchErr(err error) bool {
	var stripeErr *stripe.Error
	if errors.As(err, &stripeErr) {
		return stripeErr.Code == "resource_missing"
	}
	return false
}
