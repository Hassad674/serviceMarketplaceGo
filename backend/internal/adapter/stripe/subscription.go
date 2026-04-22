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
	"github.com/stripe/stripe-go/v82/invoice"
	"github.com/stripe/stripe-go/v82/price"
	stripesub "github.com/stripe/stripe-go/v82/subscription"
	subscriptionschedule "github.com/stripe/stripe-go/v82/subscriptionschedule"

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

// ChangeCycleImmediate swaps the subscription to a new price right now
// with proration_behavior=always_invoice. ONLY for upgrades (monthly →
// annual): the delta is charged immediately on the saved payment method.
//
// Using this on a downgrade would make Stripe recalculate the period
// based on the new (shorter) interval from the existing billing anchor —
// the user would lose the prepaid annual access. Downgrades MUST go
// through ScheduleCycleChange.
func (s *SubscriptionService) ChangeCycleImmediate(ctx context.Context, stripeSubID, newPriceID string) (portservice.SubscriptionSnapshot, error) {
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

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(itemID),
				Price: stripe.String(newPriceID),
			},
		},
		ProrationBehavior: stripe.String("always_invoice"),
	}
	params.Context = ctx

	updated, err := stripesub.Update(stripeSubID, params)
	if err != nil {
		return portservice.SubscriptionSnapshot{}, fmt.Errorf("change cycle update: %w", err)
	}
	return toSnapshot(updated), nil
}

// ScheduleCycleChange defers a cycle switch to the end of the current
// period via Stripe Subscription Schedules. The flow:
//
//  1. Create a schedule from_subscription — Stripe mirrors the current
//     state as phase 1, preserving the existing billing anchor and
//     period_end.
//  2. Append phase 2 starting at phase 1's end, billing at the new
//     price with proration_behavior=none so no invoice is generated
//     at the hand-off.
//
// The subscription keeps running phase 1 normally. When Stripe reaches
// the phase boundary, it transitions the underlying subscription to the
// new price and fires customer.subscription.updated — the local row is
// updated by the webhook handler at that moment.
//
// Returns the schedule id, the phase-2 effective date (same as phase-1
// end), and a fresh snapshot of the subscription so the app layer can
// store the pending state without a follow-up round trip.
func (s *SubscriptionService) ScheduleCycleChange(ctx context.Context, stripeSubID, newPriceID string) (portservice.ScheduledCycleChange, error) {
	existing, err := stripesub.Get(stripeSubID, &stripe.SubscriptionParams{
		Params: stripe.Params{Context: ctx},
	})
	if err != nil {
		return portservice.ScheduledCycleChange{}, fmt.Errorf("fetch subscription for schedule: %w", err)
	}
	if existing.Items == nil || len(existing.Items.Data) == 0 {
		return portservice.ScheduledCycleChange{}, errors.New("subscription has no items")
	}
	currentItem := existing.Items.Data[0]
	if currentItem.Price == nil {
		return portservice.ScheduledCycleChange{}, errors.New("subscription item has no price")
	}
	currentPriceID := currentItem.Price.ID
	phaseBoundary := currentItem.CurrentPeriodEnd

	// Step 1: create the schedule from the existing subscription. If a
	// schedule already exists (user re-scheduling a re-scheduling), reuse
	// it rather than creating a duplicate.
	var scheduleID string
	if existing.Schedule != nil && existing.Schedule.ID != "" {
		scheduleID = existing.Schedule.ID
	} else {
		createParams := &stripe.SubscriptionScheduleParams{
			FromSubscription: stripe.String(stripeSubID),
		}
		createParams.Context = ctx
		created, cErr := subscriptionschedule.New(createParams)
		if cErr != nil {
			return portservice.ScheduledCycleChange{}, fmt.Errorf("create subscription schedule: %w", cErr)
		}
		scheduleID = created.ID
	}

	// Step 2: update schedule with 2 phases. Phase 1 is the CURRENT
	// period (same price + same end date). Phase 2 starts at phase-1
	// end with the new price, proration_behavior=none so the phase
	// hand-off produces no invoice.
	phase1 := &stripe.SubscriptionSchedulePhaseParams{
		Items: []*stripe.SubscriptionSchedulePhaseItemParams{
			{
				Price:    stripe.String(currentPriceID),
				Quantity: stripe.Int64(1),
			},
		},
		EndDate: stripe.Int64(phaseBoundary),
	}
	phase2 := &stripe.SubscriptionSchedulePhaseParams{
		Items: []*stripe.SubscriptionSchedulePhaseItemParams{
			{
				Price:    stripe.String(newPriceID),
				Quantity: stripe.Int64(1),
			},
		},
		// omit EndDate so Stripe renews phase 2 indefinitely at the
		// new interval.
		ProrationBehavior: stripe.String("none"),
	}
	updateParams := &stripe.SubscriptionScheduleParams{
		Phases: []*stripe.SubscriptionSchedulePhaseParams{phase1, phase2},
		// EndBehavior=release reverts to a plain subscription at the
		// end of phase 2, but since phase 2 has no end date it renews
		// at the new price indefinitely — same practical result as a
		// normal subscription on the new cycle.
		EndBehavior: stripe.String("release"),
	}
	updateParams.Context = ctx

	updated, err := subscriptionschedule.Update(scheduleID, updateParams)
	if err != nil {
		return portservice.ScheduledCycleChange{}, fmt.Errorf("update subscription schedule: %w", err)
	}

	// Fetch the subscription again so the snapshot reflects any state
	// changes (schedule pointer, etc.).
	refreshed, err := stripesub.Get(stripeSubID, &stripe.SubscriptionParams{
		Params: stripe.Params{Context: ctx},
	})
	if err != nil {
		return portservice.ScheduledCycleChange{}, fmt.Errorf("refresh subscription after schedule: %w", err)
	}

	return portservice.ScheduledCycleChange{
		ScheduleID:  updated.ID,
		EffectiveAt: time.Unix(phaseBoundary, 0).UTC(),
		Snapshot:    toSnapshot(refreshed),
	}, nil
}

// ReleaseSchedule detaches the given schedule from its subscription,
// letting the subscription revert to a plain one on the current price.
// Used when the user cancels a pending downgrade or when the app layer
// needs to clear the schedule before an immediate upgrade.
func (s *SubscriptionService) ReleaseSchedule(ctx context.Context, scheduleID string) error {
	params := &stripe.SubscriptionScheduleReleaseParams{}
	params.Context = ctx
	_, err := subscriptionschedule.Release(scheduleID, params)
	if err != nil {
		return fmt.Errorf("release subscription schedule: %w", err)
	}
	return nil
}

// PreviewCycleChange asks Stripe what the next invoice would look like
// if the subscription switched to newPriceID with the given proration
// behaviour. Nothing is persisted server-side; the call is side-effect
// free. The app layer fronts this via GET /subscriptions/me/cycle-preview.
//
// Stripe's invoices.upcoming is the canonical way to compute a delta
// ahead of time; we forward amount_due + currency + period dates so the
// UI can render a confirm step like "Tu seras facturé 419,00 € aujourd'hui".
func (s *SubscriptionService) PreviewCycleChange(ctx context.Context, stripeSubID, newPriceID string, prorateImmediately bool) (portservice.InvoicePreview, error) {
	existing, err := stripesub.Get(stripeSubID, &stripe.SubscriptionParams{
		Params: stripe.Params{Context: ctx},
	})
	if err != nil {
		return portservice.InvoicePreview{}, fmt.Errorf("fetch subscription for preview: %w", err)
	}
	if existing.Items == nil || len(existing.Items.Data) == 0 {
		return portservice.InvoicePreview{}, errors.New("subscription has no items")
	}
	itemID := existing.Items.Data[0].ID

	prorationBehavior := "none"
	if prorateImmediately {
		prorationBehavior = "always_invoice"
	}

	params := &stripe.InvoiceCreatePreviewParams{
		Subscription: stripe.String(stripeSubID),
		SubscriptionDetails: &stripe.InvoiceCreatePreviewSubscriptionDetailsParams{
			Items: []*stripe.InvoiceCreatePreviewSubscriptionDetailsItemParams{
				{
					ID:    stripe.String(itemID),
					Price: stripe.String(newPriceID),
				},
			},
			ProrationBehavior: stripe.String(prorationBehavior),
		},
	}
	params.Context = ctx

	preview, err := invoice.CreatePreview(params)
	if err != nil {
		return portservice.InvoicePreview{}, fmt.Errorf("invoice preview: %w", err)
	}

	return portservice.InvoicePreview{
		AmountDueCents: preview.AmountDue,
		Currency:       string(preview.Currency),
		PeriodStart:    time.Unix(preview.PeriodStart, 0).UTC(),
		PeriodEnd:      time.Unix(preview.PeriodEnd, 0).UTC(),
	}, nil
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
	if sub.Customer != nil {
		snap.CustomerID = sub.Customer.ID
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
