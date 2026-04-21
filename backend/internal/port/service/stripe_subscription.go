package service

import (
	"context"
	"time"
)

// StripeSubscriptionService is the port the app layer talks to when it
// needs Stripe-side effects. The adapter implementation lives in
// internal/adapter/stripe/subscription.go. Every method takes / returns
// primitives and plain structs so the app layer never imports the Stripe
// SDK types.
type StripeSubscriptionService interface {
	// EnsureCustomer returns the Stripe customer id for the given user,
	// creating one on first call. Idempotent: the adapter is free to
	// cache the mapping, but the method must still succeed when the
	// customer already exists.
	EnsureCustomer(ctx context.Context, userID string, email string, displayName string) (customerID string, err error)

	// CreateCheckoutSession kicks off a Stripe-hosted checkout flow for
	// the given price. Returns the URL the web/mobile client redirects
	// to. cancelAtPeriodEnd is forwarded to the created subscription so
	// the auto-renew-off default is honoured from the very first charge.
	CreateCheckoutSession(ctx context.Context, in CreateCheckoutSessionInput) (url string, err error)

	// ResolvePriceID maps a logical lookup key (e.g. "premium_freelance_monthly")
	// to the Stripe price id. The seed-stripe script creates prices with
	// stable lookup_keys so the app code stays env-agnostic.
	ResolvePriceID(ctx context.Context, lookupKey string) (priceID string, err error)

	// UpdateCancelAtPeriodEnd flips Stripe's cancel_at_period_end on the
	// subscription. The server-side state is the source of truth; the
	// caller must refresh the local row from the returned snapshot.
	UpdateCancelAtPeriodEnd(ctx context.Context, stripeSubscriptionID string, cancelAtEnd bool) (SubscriptionSnapshot, error)

	// ChangeCycle switches the subscription to a new price (monthly <->
	// annual).
	//
	// When `prorateImmediately` is true (upgrade monthly→annual), the
	// adapter sets proration_behavior="always_invoice" so the user is
	// charged the delta immediately. When false (downgrade annual→
	// monthly), the adapter sets proration_behavior="none" — no refund,
	// the new price takes effect only at the next renewal, matching
	// the product rule "annual is prepaid, keep the benefit".
	ChangeCycle(ctx context.Context, stripeSubscriptionID string, newPriceID string, prorateImmediately bool) (SubscriptionSnapshot, error)

	// CreatePortalSession returns a Customer Portal URL so the user can
	// update their payment method and view invoices without us
	// reimplementing those screens.
	CreatePortalSession(ctx context.Context, customerID string, returnURL string) (url string, err error)
}

// CreateCheckoutSessionInput groups the many parameters Stripe Checkout
// needs. Each field is required unless noted.
type CreateCheckoutSessionInput struct {
	UserID            string // internal id, echoed back via metadata so the webhook can correlate
	CustomerID        string // Stripe customer id (from EnsureCustomer)
	PriceID           string // Stripe price id (from ResolvePriceID)
	CancelAtPeriodEnd bool   // default-off renewal flag
	SuccessURL        string // where to return after successful payment
	CancelURL         string // where to return if user bails
}

// SubscriptionSnapshot is a minimal projection of the Stripe subscription
// object — only the fields the app layer reflects into its own row. Kept
// here (not in the adapter) so the port is fully self-describing.
type SubscriptionSnapshot struct {
	ID                 string
	CustomerID         string // Stripe customer id (e.g. cus_XXX)
	Status             string
	PriceID            string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
}
