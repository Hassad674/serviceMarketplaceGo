package subscription

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/billing"
	domain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/port/service"
)

// CyclePreviewResult is the app-layer preview — Stripe's raw invoice
// numbers PLUS a direction flag the handler needs to pick the right
// copy. We don't infer direction from the amount because Stripe's
// invoices.upcoming returns the NEXT invoice even for downgrades (e.g.
// 49 € for the first monthly period after the annual ends), which would
// read as "charged today" if the UI only looked at amount > 0.
type CyclePreviewResult struct {
	AmountDueCents int64
	Currency       string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	// ProrateImmediately is true for upgrades only: the charge is
	// effective today. Downgrades set this to false — no charge today,
	// the quoted amount is the FIRST monthly invoice that Stripe will
	// issue when phase 2 of the schedule fires at PeriodStart.
	ProrateImmediately bool
}

// PreviewCycleChange computes what Stripe would bill if the user
// switched to `newCycle`. No state is mutated. Used by the manage
// modal's two-step confirmation so the UI always surfaces an exact
// number before the user clicks "Confirmer".
func (s *Service) PreviewCycleChange(ctx context.Context, organizationID uuid.UUID, newCycle domain.BillingCycle) (*CyclePreviewResult, error) {
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
	isUpgrade := sub.BillingCycle == domain.CycleMonthly && newCycle == domain.CycleAnnual
	if !isUpgrade && sub.CancelAtPeriodEnd {
		return nil, domain.ErrAutoRenewOffBlocksDowngrade
	}
	lookupKey, err := s.lookupKeyFor(sub.Plan, newCycle)
	if err != nil {
		return nil, err
	}
	newPriceID, err := s.stripe.ResolvePriceID(ctx, lookupKey)
	if err != nil {
		return nil, fmt.Errorf("preview cycle: resolve price: %w", err)
	}
	preview, err := s.stripe.PreviewCycleChange(ctx, sub.StripeSubscriptionID, newPriceID, isUpgrade)
	if err != nil {
		return nil, fmt.Errorf("preview cycle: stripe: %w", err)
	}
	return &CyclePreviewResult{
		AmountDueCents:     preview.AmountDueCents,
		Currency:           preview.Currency,
		PeriodStart:        preview.PeriodStart,
		PeriodEnd:          preview.PeriodEnd,
		ProrateImmediately: isUpgrade,
	}, nil
}

// StatsOutput is the data the management modal renders under
// "Tu as économisé X € depuis ton abonnement".
type StatsOutput struct {
	SavedFeeCents int64 // sum across all milestones released while Premium
	SavedCount    int   // number of milestones counted
	Since         time.Time
}

// GetStats computes the cumulative platform fee the org's member would
// have paid if the org had not been subscribed, between started_at and
// now. Uses the billing schedule directly — the source of truth is the
// same function the fee-preview endpoint calls.
//
// The org owns the subscription (organizationID) but milestone amounts
// are still tracked per individual provider (actorUserID). Until the
// amounts feature is itself migrated to org-scope, we read amounts for
// the requesting actor — that keeps the "what YOU saved" display honest
// while Premium is org-wide.
func (s *Service) GetStats(ctx context.Context, organizationID, actorUserID uuid.UUID) (*StatsOutput, error) {
	sub, err := s.subs.FindOpenByOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	actor, uErr := s.users.GetByID(ctx, actorUserID)
	if uErr != nil {
		return nil, fmt.Errorf("stats: fetch actor: %w", uErr)
	}
	if actor == nil {
		return nil, fmt.Errorf("stats: actor not found")
	}

	amounts, aErr := s.amounts.ListProviderMilestoneAmountsSince(ctx, actorUserID, sub.StartedAt)
	if aErr != nil {
		return nil, fmt.Errorf("stats: list amounts: %w", aErr)
	}

	role := billing.RoleFromUser(string(actor.Role))
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
func (s *Service) GetPortalURL(ctx context.Context, organizationID uuid.UUID) (string, error) {
	sub, err := s.subs.FindOpenByOrganization(ctx, organizationID)
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
	// The port's semantic stays user-oriented because billing is a
	// per-provider concern (milestone payments are to individuals). We
	// resolve the user's organization internally so the subscription
	// table can be queried by its new FK. A user without an organization
	// is not subscribed — return false, not an error, so billing keeps
	// charging the standard fee without spamming the logs.
	orgID, err := s.ResolveActorOrganization(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidOrganization) {
			return false, nil
		}
		return false, err
	}
	sub, err := s.subs.FindOpenByOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return sub.IsPremium(time.Now()), nil
}

// EnforceCancelAtPeriodEnd sets cancel_at_period_end on the given Stripe
// subscription id WITHOUT requiring the local DB row to exist. The
// `customer.subscription.created` webhook calls this to apply the user's
// "auto-renew off" choice captured in subscription metadata — Stripe
// Checkout doesn't expose the flag at creation time, so we apply it
// post-hoc before persisting the local row.
//
// Thin wrapper over the Stripe adapter: the webhook handler depends on
// the subscription app service (not the Stripe adapter directly) so
// removing the feature still cleanly disables this call path.
func (s *Service) EnforceCancelAtPeriodEnd(ctx context.Context, stripeSubID string, cancelAtEnd bool) error {
	_, err := s.stripe.UpdateCancelAtPeriodEnd(ctx, stripeSubID, cancelAtEnd)
	return err
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

	// If a Stripe Subscription Schedule had staged a cycle change and
	// its phase boundary just fired, the incoming snapshot carries the
	// NEW price id. Detect that by (pending set) + (price id changed),
	// then promote the pending cycle into the current cycle via the
	// domain's ApplyScheduledCycle — this flips BillingCycle + clears
	// the pending tuple atomically so the row never lands in a
	// half-updated state.
	if sub.HasPendingCycleChange() && snap.PriceID != "" && snap.PriceID != sub.StripePriceID {
		if err := sub.ApplyScheduledCycle(snap.PriceID, snap.CurrentPeriodStart, snap.CurrentPeriodEnd); err != nil {
			return fmt.Errorf("handle snapshot: apply scheduled cycle: %w", err)
		}
		sub.CancelAtPeriodEnd = snap.CancelAtPeriodEnd
		return s.subs.Update(ctx, sub)
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
// customer.subscription.created — that event carries the internal
// organization id (via metadata) AND the final Stripe subscription object.
func (s *Service) RegisterFromCheckout(
	ctx context.Context,
	organizationID uuid.UUID,
	plan domain.Plan,
	cycle domain.BillingCycle,
	stripeCustomerID string,
	snap service.SubscriptionSnapshot,
) error {
	sub, err := domain.NewSubscription(domain.NewSubscriptionInput{
		OrganizationID:       organizationID,
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
	activeNow := snap.Status == "active" || snap.Status == "trialing"
	if activeNow {
		if aErr := sub.Activate(); aErr != nil {
			return fmt.Errorf("register: activate: %w", aErr)
		}
	}
	if cErr := s.subs.Create(ctx, sub); cErr != nil {
		return cErr
	}
	// Retroactive fee waiver: as soon as the subscription is active,
	// zero the platform fee on every still-in-flight payment_record
	// of the org. Hook is best-effort — a failure here is logged but
	// does not roll back the subscription registration.
	if activeNow && s.feeWaiver != nil {
		if wErr := s.feeWaiver.WaivePlatformFeeOnActiveRecords(ctx, organizationID); wErr != nil {
			slog.Warn("subscription: fee waiver hook failed",
				"organization_id", organizationID, "error", wErr)
		}
	}
	return nil
}

// ResolveActorOrganization looks up the organization the given user
// belongs to. The handler layer calls this on every request to bridge
// from the JWT user_id claim to the org_id that owns the subscription.
// Returns subscription.ErrInvalidOrganization when the user has no
// organization (fresh account pre-onboarding).
func (s *Service) ResolveActorOrganization(ctx context.Context, actorUserID uuid.UUID) (uuid.UUID, error) {
	u, err := s.users.GetByID(ctx, actorUserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve org: fetch user: %w", err)
	}
	if u == nil || u.OrganizationID == nil {
		return uuid.Nil, domain.ErrInvalidOrganization
	}
	return *u.OrganizationID, nil
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
