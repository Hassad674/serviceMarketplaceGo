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
	"log/slog"
	"time"

	"github.com/google/uuid"

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
	subs repository.SubscriptionRepository
	// users is narrowed to UserReader — the subscription service only
	// resolves the actor by id (GetByID); membership / auth flows go
	// through the auth service.
	users      repository.UserReader
	amounts    repository.ProviderMilestoneAmountsReader
	stripe     service.StripeSubscriptionService
	lookupKeys PlanLookupKeys
	urls       URLs

	// billingProfile is the OPTIONAL reader used to pre-enrich the
	// Stripe Customer with the org's address + name before creating an
	// Embedded Checkout session. Wired by main.go via SetBillingProfileReader
	// after the invoicing service is built (invoicing is wired AFTER
	// subscription so it can't be passed to ServiceDeps directly).
	// Nil = no enrichment, Subscribe still works.
	billingProfile service.BillingProfileSnapshotReader

	// feeWaiver is the OPTIONAL hook that retroactively zeroes the
	// platform fee on every still-in-flight payment_record of the org
	// when its subscription activates. Wired post-construction in
	// main.go because the payment service is built before subscription.
	// Nil = no waiver applied; activation just records the subscription.
	feeWaiver service.ActiveRecordsFeeWaiver
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

// URLs groups the return URLs for Stripe Embedded Checkout and Customer
// Portal. Configured at startup from env; kept separate from the
// service's core dependencies so tests can inject harmless defaults.
type URLs struct {
	// CheckoutReturn is the single embedded-mode return URL. Stripe
	// substitutes "{CHECKOUT_SESSION_ID}" with the real id so the
	// return page can correlate. Required.
	CheckoutReturn string
	PortalReturn   string
}

// ServiceDeps bundles every constructor parameter. The app service never
// imports concrete types — only interfaces from port/.
type ServiceDeps struct {
	Subscriptions repository.SubscriptionRepository
	Users         repository.UserReader
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

// SetBillingProfileReader wires the optional reader used to pre-enrich
// the Stripe Customer before creating an Embedded Checkout session.
// Called by main.go after the invoicing service is built (since
// invoicing is wired AFTER subscription so it can't be passed via
// ServiceDeps). Safe to call with nil — the service silently skips
// enrichment in that case.
func (s *Service) SetBillingProfileReader(r service.BillingProfileSnapshotReader) {
	s.billingProfile = r
}

// SetFeeWaiver wires the optional hook that retroactively zeroes the
// platform fee on every still-in-flight payment_record of the org
// when its subscription activates. Called by main.go after the
// payment service is built. Safe to call with nil — activation will
// then just record the subscription without altering existing
// records.
func (s *Service) SetFeeWaiver(w service.ActiveRecordsFeeWaiver) {
	s.feeWaiver = w
}

// SubscribeInput is the payload of the POST /api/v1/subscriptions endpoint.
// OrganizationID is the owner of the subscription — Premium is granted to
// the org, not to the individual who clicked subscribe. ActorUserID is the
// person triggering the flow; their email + display name seed the Stripe
// Customer record so Stripe emails go to a real human mailbox.
type SubscribeInput struct {
	OrganizationID uuid.UUID
	ActorUserID    uuid.UUID
	Plan           domain.Plan
	BillingCycle   domain.BillingCycle
	// AutoRenew flips the default: when true, the created subscription
	// renews automatically at period end. When false (the product
	// default), cancel_at_period_end is set and the user gets exactly
	// one charge for the chosen period.
	AutoRenew bool
}

// SubscribeOutput is what the handler returns to the client. The caller
// mounts ClientSecret in @stripe/react-stripe-js (web) or in a WebView
// pointed at our /subscribe/embed page (mobile); the webhook back to us
// flips the persisted row to active once the payment lands.
type SubscribeOutput struct {
	ClientSecret string
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

	// Reject if the org already has an open subscription. The DB unique
	// index is the last line of defence; checking here returns a clean
	// domain error instead of a SQL constraint violation.
	if existing, err := s.subs.FindOpenByOrganization(ctx, in.OrganizationID); err == nil && existing != nil {
		return nil, domain.ErrAlreadySubscribed
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("subscribe: probe existing subscription: %w", err)
	}

	actor, err := s.users.GetByID(ctx, in.ActorUserID)
	if err != nil {
		return nil, fmt.Errorf("subscribe: fetch actor: %w", err)
	}
	if actor == nil {
		return nil, fmt.Errorf("subscribe: actor not found")
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

	displayName := actor.DisplayName
	if displayName == "" {
		displayName = actor.FirstName + " " + actor.LastName
	}
	// The Stripe Customer is keyed by organization id — that is the
	// entity being billed. The actor's email + name seed the default
	// billing contact on the Customer.
	customerID, err := s.stripe.EnsureCustomer(ctx, in.OrganizationID.String(), actor.Email, displayName)
	if err != nil {
		return nil, fmt.Errorf("subscribe: ensure customer: %w", err)
	}

	// Pre-enrich the Stripe Customer with the org's billing profile so
	// Stripe's embedded form doesn't have to re-collect address/name.
	// Best-effort: a missing reader (invoicing module disabled) or an
	// API failure must NOT block the subscribe — we log and continue.
	// Stripe will simply show whatever it already has on the customer.
	if s.billingProfile != nil {
		snap, sErr := s.billingProfile.GetBillingProfileSnapshotForStripe(ctx, in.OrganizationID)
		if sErr != nil {
			slog.Warn("subscribe: billing profile snapshot read failed, skipping customer enrichment",
				"org_id", in.OrganizationID, "error", sErr)
		} else if !snap.IsEmpty() {
			if eErr := s.stripe.EnrichCustomerWithBillingProfile(ctx, customerID, snap); eErr != nil {
				slog.Warn("subscribe: stripe customer enrichment failed, continuing without",
					"org_id", in.OrganizationID, "customer_id", customerID, "error", eErr)
			}
		}
	}

	clientSecret, err := s.stripe.CreateCheckoutSession(ctx, service.CreateCheckoutSessionInput{
		OrganizationID:    in.OrganizationID.String(),
		CustomerID:        customerID,
		PriceID:           priceID,
		CancelAtPeriodEnd: !in.AutoRenew,
		ReturnURL:         s.urls.CheckoutReturn,
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe: create checkout session: %w", err)
	}

	return &SubscribeOutput{ClientSecret: clientSecret}, nil
}

// GetStatus returns the org's current open subscription, or
// subscription.ErrNotFound when the org is on the free tier. The status
// modal in the UI uses this to decide whether to render the upgrade or
// manage panel.
func (s *Service) GetStatus(ctx context.Context, organizationID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subs.FindOpenByOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// ToggleAutoRenew flips cancel_at_period_end on both Stripe and the local
// row. The adapter returns a snapshot; we reflect the authoritative
// fields (not just the flag — Stripe may have updated the period too).
//
// Turning auto-renew OFF while a cycle downgrade is scheduled is a
// contradiction: cancel_at_period_end=TRUE says "end at period_end",
// but the schedule says "transition to the new cycle and keep charging
// at period_end". Stripe resolves this in favour of the schedule, which
// silently billed users who thought they had opted out. We release the
// schedule FIRST so cancel_at_period_end is authoritative on the sub.
// If the user later wants the downgrade back, they re-enable auto-renew
// and re-schedule — cheap and explicit.
func (s *Service) ToggleAutoRenew(ctx context.Context, organizationID uuid.UUID, on bool) (*domain.Subscription, error) {
	sub, err := s.subs.FindOpenByOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	if !on && sub.StripeScheduleID != nil && *sub.StripeScheduleID != "" {
		if rErr := s.stripe.ReleaseSchedule(ctx, *sub.StripeScheduleID); rErr != nil {
			return nil, fmt.Errorf("toggle auto renew: release pending schedule: %w", rErr)
		}
		sub.ClearScheduledCycle()
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

// ChangeCycle switches monthly <-> annual following product rules:
//
//   - Upgrade (monthly → annual): immediate. Stripe charges the delta
//     via proration_behavior=always_invoice; the annual period starts
//     now. If the user previously scheduled a downgrade and changes
//     their mind, the schedule is released first so the direct upgrade
//     can run cleanly.
//
//   - Downgrade (annual → monthly): deferred via a Stripe Subscription
//     Schedule. The annual period keeps running until current_period_end;
//     Stripe fires customer.subscription.updated at the phase boundary
//     and the webhook handler promotes the pending cycle into the
//     current one (ApplyScheduledCycle). No refund, no credit, no
//     change to the current period_end.
//
// The domain row reflects CURRENT billing_cycle in both cases; on a
// downgrade, PendingBillingCycle + PendingCycleEffectiveAt + StripeScheduleID
// are populated so the UI can render "Annuel jusqu'au JJ/MM/YYYY → Mensuel
// ensuite". The row flips cycle only when the phase transition actually
// happens.
func (s *Service) ChangeCycle(ctx context.Context, organizationID uuid.UUID, newCycle domain.BillingCycle) (*domain.Subscription, error) {
	if !newCycle.IsValid() {
		return nil, domain.ErrInvalidCycle
	}

	sub, err := s.subs.FindOpenByOrganization(ctx, organizationID)
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

	isUpgrade := sub.BillingCycle == domain.CycleMonthly && newCycle == domain.CycleAnnual

	// A downgrade schedules a future transition to the new cycle and
	// keeps charging the user on the new cadence. If auto-renew is off,
	// the subscription is meant to END at period_end — scheduling a
	// transition on top contradicts that intent and Stripe resolves the
	// contradiction by renewing anyway. Reject loudly so the UI can
	// prompt the user to re-enable auto-renew first.
	if !isUpgrade && sub.CancelAtPeriodEnd {
		return nil, domain.ErrAutoRenewOffBlocksDowngrade
	}

	if isUpgrade {
		// If the user re-upgrades after previously scheduling a downgrade,
		// release the schedule first so Stripe accepts a direct subscription
		// update (Stripe rejects subscription.update when a schedule is
		// managing the subscription).
		if sub.StripeScheduleID != nil && *sub.StripeScheduleID != "" {
			if rErr := s.stripe.ReleaseSchedule(ctx, *sub.StripeScheduleID); rErr != nil {
				return nil, fmt.Errorf("change cycle: release stale schedule: %w", rErr)
			}
		}

		snap, upErr := s.stripe.ChangeCycleImmediate(ctx, sub.StripeSubscriptionID, newPriceID)
		if upErr != nil {
			return nil, fmt.Errorf("change cycle: stripe upgrade: %w", upErr)
		}
		if err := sub.ChangeCycle(newCycle, snap.PriceID, snap.CurrentPeriodStart, snap.CurrentPeriodEnd); err != nil {
			return nil, err
		}
		if err := s.subs.Update(ctx, sub); err != nil {
			return nil, fmt.Errorf("change cycle: persist upgrade: %w", err)
		}
		return sub, nil
	}

	// Downgrade path — schedule the transition at current period end.
	scheduled, err := s.stripe.ScheduleCycleChange(ctx, sub.StripeSubscriptionID, newPriceID)
	if err != nil {
		return nil, fmt.Errorf("change cycle: stripe schedule: %w", err)
	}
	if err := sub.SchedulePendingCycle(newCycle, scheduled.EffectiveAt, scheduled.ScheduleID); err != nil {
		return nil, err
	}
	if err := s.subs.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("change cycle: persist downgrade: %w", err)
	}
	return sub, nil
}
