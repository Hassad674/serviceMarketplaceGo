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

	// CreateCheckoutSession kicks off a Stripe Embedded Checkout flow
	// for the given price. Returns the session client_secret that the
	// web/mobile client mounts via @stripe/react-stripe-js — there is no
	// hosted Stripe URL anymore. cancelAtPeriodEnd is forwarded to the
	// created subscription so the auto-renew-off default is honoured
	// from the very first charge. BillingAddressCollection is kept on
	// "auto" (Stripe default) and tax_id collection is disabled — the
	// caller is expected to have already filled the Stripe Customer's
	// address via EnrichCustomerWithBillingProfile so Stripe's form has
	// nothing to re-collect from the user.
	CreateCheckoutSession(ctx context.Context, in CreateCheckoutSessionInput) (clientSecret string, err error)

	// EnrichCustomerWithBillingProfile pushes the org's billing profile
	// onto the Stripe Customer record (address, name, email, VAT in
	// metadata) BEFORE the session is created. Idempotent — every
	// Subscribe call runs it so the Customer reflects the freshest copy
	// of our DB. Best-effort: errors are surfaced but the caller is
	// expected to log + continue, since Embedded Checkout will still
	// work without the enrichment (it just shows a less-prefilled UI).
	EnrichCustomerWithBillingProfile(ctx context.Context, customerID string, snap BillingProfileStripeSnapshot) error

	// ResolvePriceID maps a logical lookup key (e.g. "premium_freelance_monthly")
	// to the Stripe price id. The seed-stripe script creates prices with
	// stable lookup_keys so the app code stays env-agnostic.
	ResolvePriceID(ctx context.Context, lookupKey string) (priceID string, err error)

	// UpdateCancelAtPeriodEnd flips Stripe's cancel_at_period_end on the
	// subscription. The server-side state is the source of truth; the
	// caller must refresh the local row from the returned snapshot.
	UpdateCancelAtPeriodEnd(ctx context.Context, stripeSubscriptionID string, cancelAtEnd bool) (SubscriptionSnapshot, error)

	// ChangeCycleImmediate swaps the subscription to a new price with
	// immediate effect and proration (always_invoice). Used only for
	// UPGRADES (monthly → annual): the user is charged the delta now.
	// DO NOT use for downgrades — Stripe will recompute the period end
	// and the user loses the access they already paid for; use
	// ScheduleCycleChange instead.
	ChangeCycleImmediate(ctx context.Context, stripeSubscriptionID string, newPriceID string) (SubscriptionSnapshot, error)

	// ScheduleCycleChange defers a cycle change to the end of the
	// current period via a Stripe Subscription Schedule. Used for
	// DOWNGRADES (annual → monthly): the user keeps their annual
	// access until the period ends, then Stripe transitions to the new
	// price automatically. Returns the schedule id and effective date
	// so the app layer can store the pending state on the domain row.
	ScheduleCycleChange(ctx context.Context, stripeSubscriptionID string, newPriceID string) (ScheduledCycleChange, error)

	// ReleaseSchedule detaches the subscription from its schedule
	// without altering the current billing cycle. Used when the user
	// cancels a pending downgrade (re-upgrades before the phase fires)
	// or when the orchestration layer needs to revert to a plain
	// subscription to then run an immediate upgrade on top.
	ReleaseSchedule(ctx context.Context, stripeScheduleID string) error

	// PreviewCycleChange computes the amount that would be charged /
	// credited if the subscription switched to the given price with the
	// given proration behaviour. Backed by Stripe's invoices.upcoming
	// API — no state is mutated. The UI surfaces this number in the
	// confirm step so the user always sees what they will pay BEFORE
	// clicking "Confirmer".
	PreviewCycleChange(ctx context.Context, stripeSubscriptionID string, newPriceID string, prorateImmediately bool) (InvoicePreview, error)

	// CreatePortalSession returns a Customer Portal URL so the user can
	// update their payment method and view invoices without us
	// reimplementing those screens.
	CreatePortalSession(ctx context.Context, customerID string, returnURL string) (url string, err error)
}

// CreateCheckoutSessionInput groups the parameters Stripe Embedded
// Checkout needs. Each field is required unless noted.
type CreateCheckoutSessionInput struct {
	OrganizationID    string // internal org id, echoed back via metadata so the webhook can correlate
	CustomerID        string // Stripe customer id (from EnsureCustomer)
	PriceID           string // Stripe price id (from ResolvePriceID)
	CancelAtPeriodEnd bool   // default-off renewal flag
	// ReturnURL is where Stripe redirects the embedded form after a
	// successful or cancelled payment. Embedded mode uses a SINGLE
	// return URL (not the success/cancel pair of hosted mode); the
	// caller renders a polling page that hits /subscriptions/me until
	// the webhook flips the row to active. Must contain the literal
	// string "{CHECKOUT_SESSION_ID}" — Stripe substitutes it with the
	// real session id so the return page can correlate.
	ReturnURL string
}

// ScheduledCycleChange is what the adapter returns after wiring up a
// subscription schedule for a deferred cycle change.
type ScheduledCycleChange struct {
	ScheduleID  string    // stripe schedule id (sub_sched_...)
	EffectiveAt time.Time // when phase 2 (new price) starts — usually current period_end
	Snapshot    SubscriptionSnapshot
}

// InvoicePreview captures what Stripe would bill for a hypothetical
// cycle change. All amounts are in cents (net of Stripe fees).
//   - AmountDueCents > 0: the customer owes that amount now (upgrade).
//   - AmountDueCents == 0: nothing is charged now (downgrade, scheduled).
//   - AmountDueCents < 0: a credit will be applied on the next invoice
//     (rare; we don't refund, so we carry the credit forward).
type InvoicePreview struct {
	AmountDueCents int64
	Currency       string
	PeriodStart    time.Time
	PeriodEnd      time.Time
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
