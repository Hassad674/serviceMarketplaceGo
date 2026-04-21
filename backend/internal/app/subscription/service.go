// Package subscription is the application layer for the Premium plan
// feature. It orchestrates the Stripe Subscriptions API, persists local
// rows, and exposes a SubscriptionReader to the billing feature so fees
// are waived while the subscription is active.
//
// Removable-by-design: the whole feature can be deleted from main.go
// wiring. When the payment service receives a nil SubscriptionReader it
// falls back to the pleins-tarifs path.
package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/billing"
	domain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Default grace window when invoice.payment_failed lands. Chosen to match
// Stripe's default Smart Retries schedule so we don't revoke Premium
// before Stripe itself stops retrying.
const defaultGraceWindow = 72 * time.Hour

// Service orchestrates the subscription lifecycle.
type Service struct {
	subs       repository.SubscriptionRepository
	users      repository.UserRepository
	amounts    repository.ProviderMilestoneAmountsReader
	stripe     service.StripeSubscriptionService
	lookupKeys PlanLookupKeys
	urls       URLs
}

// PlanLookupKeys maps the four (plan, cycle) combinations to the Stripe
// price lookup_keys seeded by cmd/seed-stripe. Keeps the logical names in
// code and the real price IDs in Stripe — same code runs in test and
// prod, only the Stripe account differs.
type PlanLookupKeys struct {
	FreelanceMonthly string
	FreelanceAnnual  string
	AgencyMonthly    string
	AgencyAnnual     string
}

// DefaultLookupKeys are the canonical lookup_keys the seed script creates.
// Exposed as a helper so main.go can wire without copy-pasting strings.
func DefaultLookupKeys() PlanLookupKeys {
	return PlanLookupKeys{
		FreelanceMonthly: "premium_freelance_monthly",
		FreelanceAnnual:  "premium_freelance_annual",
		AgencyMonthly:    "premium_agency_monthly",
		AgencyAnnual:     "premium_agency_annual",
	}
}

// URLs groups the return URLs for Stripe Checkout and Customer Portal.
// Configured at startup from env; kept separate from the service's core
// dependencies so tests can inject harmless defaults.
type URLs struct {
	CheckoutSuccess string // appended with ?session_id=... by Stripe
	CheckoutCancel  string
	PortalReturn    string
}

// ServiceDeps bundles every constructor parameter. The app service never
// imports concrete types — only interfaces from port/.
type ServiceDeps struct {
	Subscriptions repository.SubscriptionRepository
	Users         repository.UserRepository
	Amounts       repository.ProviderMilestoneAmountsReader
	Stripe        service.StripeSubscriptionService
	LookupKeys    PlanLookupKeys
	URLs          URLs
}

// NewService wires the subscription application service. Every dependency
// is required — the feature cannot run with any of them nil.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		subs:       deps.Subscriptions,
		users:      deps.Users,
		amounts:    deps.Amounts,
		stripe:     deps.Stripe,
		lookupKeys: deps.LookupKeys,
		urls:       deps.URLs,
	}
}

// SubscribeInput is the payload of the POST /api/v1/subscriptions endpoint.
type SubscribeInput struct {
	UserID       uuid.UUID
	Plan         domain.Plan
	BillingCycle domain.BillingCycle
	// AutoRenew flips the default: when true, the created subscription
	// renews automatically at period end. When false (the product
	// default), cancel_at_period_end is set and the user gets exactly
	// one charge for the chosen period.
	AutoRenew bool
}

// SubscribeOutput is what the handler returns to the client. The caller
// redirects the user to CheckoutURL; a webhook back to us then flips the
// persisted row to active.
type SubscribeOutput struct {
	CheckoutURL string
}

// Subscribe starts a Stripe Checkout session and persists an incomplete
// subscription row. The row is finalised (status=active) by the
// invoice.payment_succeeded webhook — the Checkout redirect itself does
// not grant Premium.
func (s *Service) Subscribe(ctx context.Context, in SubscribeInput) (*SubscribeOutput, error) {
	if !in.Plan.IsValid() {
		return nil, domain.ErrInvalidPlan
	}
	if !in.BillingCycle.IsValid() {
		return nil, domain.ErrInvalidCycle
	}

	// Reject if the user already has an open subscription. The DB unique
	// index is the last line of defence; checking here returns a clean
	// domain error instead of a SQL constraint violation.
	if existing, err := s.subs.FindOpenByUser(ctx, in.UserID); err == nil && existing != nil {
		return nil, domain.ErrAlreadySubscribed
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("subscribe: probe existing subscription: %w", err)
	}

	user, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, fmt.Errorf("subscribe: fetch user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("subscribe: user not found")
	}

	// Map (plan, cycle) to the Stripe lookup key, resolve it to a live
	// Stripe price id. ResolvePriceID is expected to be fast (cached /
	// constant-time by the adapter) because we call it on every subscribe.
	lookupKey, err := s.lookupKeyFor(in.Plan, in.BillingCycle)
	if err != nil {
		return nil, err
	}
	priceID, err := s.stripe.ResolvePriceID(ctx, lookupKey)
	if err != nil {
		return nil, fmt.Errorf("subscribe: resolve price: %w", err)
	}

	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.FirstName + " " + user.LastName
	}
	customerID, err := s.stripe.EnsureCustomer(ctx, in.UserID.String(), user.Email, displayName)
	if err != nil {
		return nil, fmt.Errorf("subscribe: ensure customer: %w", err)
	}

	url, err := s.stripe.CreateCheckoutSession(ctx, service.CreateCheckoutSessionInput{
		UserID:            in.UserID.String(),
		CustomerID:        customerID,
		PriceID:           priceID,
		CancelAtPeriodEnd: !in.AutoRenew,
		SuccessURL:        s.urls.CheckoutSuccess,
		CancelURL:         s.urls.CheckoutCancel,
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe: create checkout session: %w", err)
	}

	return &SubscribeOutput{CheckoutURL: url}, nil
}

// GetStatus returns the user's current open subscription, or
// subscription.ErrNotFound when the user is on the free tier. The status
// modal in the UI uses this to decide whether to render the upgrade or
// manage panel.
func (s *Service) GetStatus(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subs.FindOpenByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// ToggleAutoRenew flips cancel_at_period_end on both Stripe and the local
// row. The adapter returns a snapshot; we reflect the authoritative
// fields (not just the flag — Stripe may have updated the period too).
func (s *Service) ToggleAutoRenew(ctx context.Context, userID uuid.UUID, on bool) (*domain.Subscription, error) {
	sub, err := s.subs.FindOpenByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	snap, err := s.stripe.UpdateCancelAtPeriodEnd(ctx, sub.StripeSubscriptionID, !on)
	if err != nil {
		return nil, fmt.Errorf("toggle auto renew: stripe update: %w", err)
	}

	sub.SetAutoRenew(on)
	sub.CancelAtPeriodEnd = snap.CancelAtPeriodEnd
	if !snap.CurrentPeriodStart.IsZero() && !snap.CurrentPeriodEnd.IsZero() {
		_ = sub.UpdatePeriod(snap.CurrentPeriodStart, snap.CurrentPeriodEnd)
	}
	if err := s.subs.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("toggle auto renew: persist: %w", err)
	}
	return sub, nil
}

// ChangeCycle switches monthly <-> annual. Proration is immediate; Stripe
// sends a new invoice for the delta or credits the difference.
func (s *Service) ChangeCycle(ctx context.Context, userID uuid.UUID, newCycle domain.BillingCycle) (*domain.Subscription, error) {
	if !newCycle.IsValid() {
		return nil, domain.ErrInvalidCycle
	}

	sub, err := s.subs.FindOpenByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub.BillingCycle == newCycle {
		return nil, domain.ErrSameCycle
	}

	lookupKey, err := s.lookupKeyFor(sub.Plan, newCycle)
	if err != nil {
		return nil, err
	}
	newPriceID, err := s.stripe.ResolvePriceID(ctx, lookupKey)
	if err != nil {
		return nil, fmt.Errorf("change cycle: resolve price: %w", err)
	}

	snap, err := s.stripe.ChangeCycle(ctx, sub.StripeSubscriptionID, newPriceID)
	if err != nil {
		return nil, fmt.Errorf("change cycle: stripe update: %w", err)
	}

	if err := sub.ChangeCycle(newCycle, snap.PriceID, snap.CurrentPeriodStart, snap.CurrentPeriodEnd); err != nil {
		return nil, err
	}
	if err := s.subs.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("change cycle: persist: %w", err)
	}
	return sub, nil
}

// StatsOutput is the data the management modal renders under
// "Tu as économisé X € depuis ton abonnement".
type StatsOutput struct {
	SavedFeeCents int64 // sum across all milestones released while Premium
	SavedCount    int   // number of milestones counted
	Since         time.Time
}

// GetStats computes the cumulative platform fee the user would have paid
// if they had not been subscribed, between started_at and now. Uses the
// billing schedule directly — the source of truth is the same function
// the fee-preview endpoint calls.
func (s *Service) GetStats(ctx context.Context, userID uuid.UUID) (*StatsOutput, error) {
	sub, err := s.subs.FindOpenByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	user, uErr := s.users.GetByID(ctx, userID)
	if uErr != nil {
		return nil, fmt.Errorf("stats: fetch user: %w", uErr)
	}
	if user == nil {
		return nil, fmt.Errorf("stats: user not found")
	}

	amounts, aErr := s.amounts.ListProviderMilestoneAmountsSince(ctx, userID, sub.StartedAt)
	if aErr != nil {
		return nil, fmt.Errorf("stats: list amounts: %w", aErr)
	}

	role := billing.RoleFromUser(string(user.Role))
	var saved int64
	for _, a := range amounts {
		saved += billing.Calculate(role, a).FeeCents
	}

	return &StatsOutput{
		SavedFeeCents: saved,
		SavedCount:    len(amounts),
		Since:         sub.StartedAt,
	}, nil
}

// GetPortalURL returns a Stripe Customer Portal link so the user can
// update their payment method and view invoices. Fails with
// subscription.ErrNoActiveSub when there is no open subscription to
// manage — the UI only exposes this action to subscribers.
func (s *Service) GetPortalURL(ctx context.Context, userID uuid.UUID) (string, error) {
	sub, err := s.subs.FindOpenByUser(ctx, userID)
	if err != nil {
		return "", err
	}
	url, pErr := s.stripe.CreatePortalSession(ctx, sub.StripeCustomerID, s.urls.PortalReturn)
	if pErr != nil {
		return "", fmt.Errorf("portal: create session: %w", pErr)
	}
	return url, nil
}

// IsActive implements service.SubscriptionReader: the answer the billing
// layer needs on every fee calculation. Fails-open conservatively: on any
// error the caller sees false (no waiver) so a downed cache never
// accidentally grants free milestones.
func (s *Service) IsActive(ctx context.Context, userID uuid.UUID) (bool, error) {
	sub, err := s.subs.FindOpenByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return sub.IsPremium(time.Now()), nil
}

// HandleSubscriptionSnapshot reflects a Stripe subscription snapshot
// (from customer.subscription.updated / .created / .deleted webhooks)
// into our row. Idempotent: safe to replay.
//
// stripeSubID is the source of truth lookup key; internalStatus is the
// mapped Status the caller already resolved from snap.Status.
func (s *Service) HandleSubscriptionSnapshot(
	ctx context.Context,
	snap service.SubscriptionSnapshot,
	deleted bool,
) error {
	sub, err := s.subs.FindByStripeID(ctx, snap.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Webhook landed for a row we never created — ignore. This
			// happens when someone tests Stripe CLI `stripe trigger` on
			// a subscription id unrelated to our DB.
			return nil
		}
		return fmt.Errorf("handle snapshot: find: %w", err)
	}

	if deleted {
		if err := sub.MarkCanceled(); err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
			return fmt.Errorf("handle snapshot: cancel: %w", err)
		}
		return s.subs.Update(ctx, sub)
	}

	switch snap.Status {
	case "active", "trialing":
		if err := sub.Activate(); err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
			return fmt.Errorf("handle snapshot: activate: %w", err)
		}
	case "past_due":
		grace := time.Now().Add(defaultGraceWindow)
		if err := sub.MarkPastDue(grace); err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
			return fmt.Errorf("handle snapshot: past_due: %w", err)
		}
	case "unpaid":
		if err := sub.MarkUnpaid(); err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
			return fmt.Errorf("handle snapshot: unpaid: %w", err)
		}
	case "canceled":
		if err := sub.MarkCanceled(); err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
			return fmt.Errorf("handle snapshot: canceled: %w", err)
		}
		// incomplete / incomplete_expired: nothing to do, row already reflects it.
	}

	if snap.PriceID != "" {
		sub.StripePriceID = snap.PriceID
	}
	if !snap.CurrentPeriodStart.IsZero() && !snap.CurrentPeriodEnd.IsZero() {
		_ = sub.UpdatePeriod(snap.CurrentPeriodStart, snap.CurrentPeriodEnd)
	}
	sub.CancelAtPeriodEnd = snap.CancelAtPeriodEnd

	return s.subs.Update(ctx, sub)
}

// RegisterFromCheckout persists the subscription row just after the
// Checkout session converts. Called by the webhook handler on
// customer.subscription.created — that event carries the internal user id
// (via metadata) AND the final Stripe subscription object.
func (s *Service) RegisterFromCheckout(
	ctx context.Context,
	userID uuid.UUID,
	plan domain.Plan,
	cycle domain.BillingCycle,
	stripeCustomerID string,
	snap service.SubscriptionSnapshot,
) error {
	sub, err := domain.NewSubscription(domain.NewSubscriptionInput{
		UserID:               userID,
		Plan:                 plan,
		BillingCycle:         cycle,
		StripeCustomerID:     stripeCustomerID,
		StripeSubscriptionID: snap.ID,
		StripePriceID:        snap.PriceID,
		CurrentPeriodStart:   snap.CurrentPeriodStart,
		CurrentPeriodEnd:     snap.CurrentPeriodEnd,
		CancelAtPeriodEnd:    snap.CancelAtPeriodEnd,
	})
	if err != nil {
		return fmt.Errorf("register: build domain: %w", err)
	}
	if snap.Status == "active" || snap.Status == "trialing" {
		if aErr := sub.Activate(); aErr != nil {
			return fmt.Errorf("register: activate: %w", aErr)
		}
	}
	return s.subs.Create(ctx, sub)
}

// lookupKeyFor maps a (plan, cycle) pair to the Stripe lookup key. Defined
// as a method so sub-agents testing the service don't have to reimplement
// the mapping.
func (s *Service) lookupKeyFor(plan domain.Plan, cycle domain.BillingCycle) (string, error) {
	switch {
	case plan == domain.PlanFreelance && cycle == domain.CycleMonthly:
		return s.lookupKeys.FreelanceMonthly, nil
	case plan == domain.PlanFreelance && cycle == domain.CycleAnnual:
		return s.lookupKeys.FreelanceAnnual, nil
	case plan == domain.PlanAgency && cycle == domain.CycleMonthly:
		return s.lookupKeys.AgencyMonthly, nil
	case plan == domain.PlanAgency && cycle == domain.CycleAnnual:
		return s.lookupKeys.AgencyAnnual, nil
	}
	return "", domain.ErrInvalidPlan
}
