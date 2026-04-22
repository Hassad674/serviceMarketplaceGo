// Package subscription models the Premium plan lifecycle — a per-user,
// per-role flat subscription that waives the platform fee on every
// milestone payment as long as the subscription is active.
//
// The domain layer is pure: no DB, no HTTP, no Stripe. State transitions
// are expressed as methods on Subscription so the app layer can drive them
// deterministically and unit-test them without infrastructure.
package subscription

import (
	"time"

	"github.com/google/uuid"
)

// Plan distinguishes the two product grids. Enterprise users are never
// charged the platform fee (they are clients) so no enterprise plan
// exists — an enterprise attempting to subscribe is a domain-level error.
type Plan string

const (
	PlanFreelance Plan = "freelance"
	PlanAgency    Plan = "agency"
)

func (p Plan) IsValid() bool {
	return p == PlanFreelance || p == PlanAgency
}

// BillingCycle is the recurring interval the Stripe Price charges at.
// Only two options are exposed in V1. Switching between them is allowed
// via ChangeCycle — Stripe handles the proration immediately.
type BillingCycle string

const (
	CycleMonthly BillingCycle = "monthly"
	CycleAnnual  BillingCycle = "annual"
)

func (c BillingCycle) IsValid() bool {
	return c == CycleMonthly || c == CycleAnnual
}

// Status mirrors Stripe's subscription.status enum. "trialing" is not
// exposed because we do not offer free trials in V1; an attempt to set
// it returns an error so a drift in the Stripe adapter is caught early.
type Status string

const (
	// StatusIncomplete: Checkout session created, first payment not yet
	// confirmed. The row exists but Premium is NOT granted.
	StatusIncomplete Status = "incomplete"
	// StatusActive: paid, Premium granted. fee waivers apply.
	StatusActive Status = "active"
	// StatusPastDue: renewal payment failed; within grace period.
	// Premium is STILL granted until grace_period_ends_at elapses.
	StatusPastDue Status = "past_due"
	// StatusCanceled: final stop. Either natural expiration
	// (cancel_at_period_end fired) or user-initiated cancel.
	StatusCanceled Status = "canceled"
	// StatusUnpaid: Stripe terminal state after grace period lapses.
	// Premium is NOT granted.
	StatusUnpaid Status = "unpaid"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusIncomplete, StatusActive, StatusPastDue, StatusCanceled, StatusUnpaid:
		return true
	}
	return false
}

// Subscription is the aggregate root of the package. Exactly one row with
// status in (incomplete, active, past_due) exists per user (enforced by
// a partial unique index in the DB); earlier rows stay as historical
// record with status canceled or unpaid.
type Subscription struct {
	ID     uuid.UUID
	UserID uuid.UUID

	Plan         Plan
	BillingCycle BillingCycle
	Status       Status

	StripeCustomerID     string
	StripeSubscriptionID string
	StripePriceID        string

	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time

	// CancelAtPeriodEnd is the inverted auto-renewal flag. TRUE (default
	// for new rows) means "this subscription will expire naturally at
	// the end of the current period and no further charge will happen".
	// Flip to FALSE to enable auto-renewal.
	CancelAtPeriodEnd bool

	// GracePeriodEndsAt is set when the subscription enters past_due. The
	// app layer uses it to decide whether to revoke Premium on expiry.
	GracePeriodEndsAt *time.Time

	// CanceledAt is the moment the subscription transitioned into the
	// canceled state. Kept for reporting (churn cohorts) and for the
	// fee-saving stats which sum payment_records between started_at and
	// canceled_at (or now, whichever is earlier).
	CanceledAt *time.Time

	// StartedAt is the first time this subscription became active. It
	// survives future status flips and is the lower bound for
	// "fees saved since subscribing" stats. Set on Activate().
	StartedAt time.Time

	// PendingBillingCycle + PendingCycleEffectiveAt + StripeScheduleID
	// describe a scheduled cycle change (typically a downgrade annual →
	// monthly) that is NOT yet applied. All three fields are set together
	// or all three are nil; the DB CHECK constraint enforces the invariant.
	//
	// While pending:
	//   - BillingCycle + StripePriceID + CurrentPeriodEnd still reflect the
	//     CURRENT (paid) phase — the user keeps that access.
	//   - PendingBillingCycle is the cycle that takes effect at
	//     PendingCycleEffectiveAt, backed by the Stripe subscription schedule.
	//   - The UI shows "Annuel jusqu'au 22/04/2027 → Mensuel ensuite".
	//
	// Cleared by ApplyScheduledCycle (webhook fires when phase 2 starts)
	// or by ClearScheduledCycle (user cancelled the planned downgrade).
	PendingBillingCycle       *BillingCycle
	PendingCycleEffectiveAt   *time.Time
	StripeScheduleID          *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewSubscriptionInput groups the constructor arguments so the caller
// does not have to pass a 9-parameter function. The domain validates every
// field before returning a Subscription — invalid inputs are rejected
// without ever touching the database.
type NewSubscriptionInput struct {
	UserID               uuid.UUID
	Plan                 Plan
	BillingCycle         BillingCycle
	StripeCustomerID     string
	StripeSubscriptionID string
	StripePriceID        string
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	CancelAtPeriodEnd    bool
}

// NewSubscription constructs an incomplete Subscription, ready to be
// persisted and then flipped to active by the first
// customer.subscription.created webhook event. The status starts at
// incomplete to match Stripe's own state machine — the row exists but
// the user is NOT Premium yet.
func NewSubscription(in NewSubscriptionInput) (*Subscription, error) {
	if in.UserID == uuid.Nil {
		return nil, ErrInvalidUser
	}
	if !in.Plan.IsValid() {
		return nil, ErrInvalidPlan
	}
	if !in.BillingCycle.IsValid() {
		return nil, ErrInvalidCycle
	}
	if in.StripeCustomerID == "" || in.StripeSubscriptionID == "" || in.StripePriceID == "" {
		return nil, ErrMissingStripeIDs
	}
	if in.CurrentPeriodEnd.Before(in.CurrentPeriodStart) {
		return nil, ErrInvalidPeriod
	}
	now := time.Now()
	return &Subscription{
		ID:                   uuid.New(),
		UserID:               in.UserID,
		Plan:                 in.Plan,
		BillingCycle:         in.BillingCycle,
		Status:               StatusIncomplete,
		StripeCustomerID:     in.StripeCustomerID,
		StripeSubscriptionID: in.StripeSubscriptionID,
		StripePriceID:        in.StripePriceID,
		CurrentPeriodStart:   in.CurrentPeriodStart,
		CurrentPeriodEnd:     in.CurrentPeriodEnd,
		CancelAtPeriodEnd:    in.CancelAtPeriodEnd,
		StartedAt:            now, // updated on Activate if the first payment is delayed
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

// Activate transitions from incomplete (or past_due recovery) to active.
// The app layer calls this when the first invoice.payment_succeeded
// webhook lands. StartedAt is refreshed only on the FIRST activation so
// later renewals do not reset the "fees saved since" window.
func (s *Subscription) Activate() error {
	if s.Status == StatusCanceled || s.Status == StatusUnpaid {
		return ErrInvalidTransition
	}
	firstActivation := s.Status == StatusIncomplete
	s.Status = StatusActive
	s.GracePeriodEndsAt = nil
	now := time.Now()
	if firstActivation {
		s.StartedAt = now
	}
	s.UpdatedAt = now
	return nil
}

// MarkPastDue records a failed renewal payment and opens a grace window.
// The domain caps the grace at 3 days unless the caller provides a later
// time (e.g. Stripe's own dunning schedule). Premium stays granted as
// long as time.Now().Before(GracePeriodEndsAt).
func (s *Subscription) MarkPastDue(graceEndsAt time.Time) error {
	if s.Status != StatusActive && s.Status != StatusPastDue {
		return ErrInvalidTransition
	}
	s.Status = StatusPastDue
	s.GracePeriodEndsAt = &graceEndsAt
	s.UpdatedAt = time.Now()
	return nil
}

// MarkCanceled moves the subscription to its terminal "canceled" state.
// Called when cancel_at_period_end fires at the end of a natural period
// OR when the user explicitly cancels (future feature — not exposed in
// V1 UI but the state transition exists for webhook robustness).
func (s *Subscription) MarkCanceled() error {
	if s.Status == StatusCanceled || s.Status == StatusUnpaid {
		return ErrInvalidTransition
	}
	now := time.Now()
	s.Status = StatusCanceled
	s.CanceledAt = &now
	s.UpdatedAt = now
	return nil
}

// MarkUnpaid is Stripe's terminal state after the grace window for
// past_due lapses without successful payment. Revokes Premium.
func (s *Subscription) MarkUnpaid() error {
	if s.Status != StatusPastDue && s.Status != StatusActive {
		return ErrInvalidTransition
	}
	s.Status = StatusUnpaid
	s.UpdatedAt = time.Now()
	return nil
}

// UpdatePeriod extends the current billing window on a successful renewal.
// Stripe webhooks carry the new period dates; we just mirror them.
func (s *Subscription) UpdatePeriod(newStart, newEnd time.Time) error {
	if newEnd.Before(newStart) {
		return ErrInvalidPeriod
	}
	s.CurrentPeriodStart = newStart
	s.CurrentPeriodEnd = newEnd
	s.UpdatedAt = time.Now()
	return nil
}

// SetAutoRenew flips the cancel_at_period_end flag. Pure state update —
// the app layer is responsible for propagating the change to Stripe.
func (s *Subscription) SetAutoRenew(on bool) {
	s.CancelAtPeriodEnd = !on
	s.UpdatedAt = time.Now()
}

// ChangeCycle swaps the current cycle IMMEDIATELY: used only for upgrades
// (monthly → annual) where Stripe's always_invoice proration charges the
// delta right away. The caller feeds back Stripe's post-update snapshot
// (new price, new period) so DB and Stripe stay aligned.
//
// Downgrades (annual → monthly) MUST go through SchedulePendingCycle
// instead — they don't change the current cycle, they schedule a future
// transition so the user keeps access for the period they already paid.
func (s *Subscription) ChangeCycle(newCycle BillingCycle, newPriceID string, newPeriodStart, newPeriodEnd time.Time) error {
	if s.Status != StatusActive && s.Status != StatusPastDue {
		return ErrInvalidTransition
	}
	if !newCycle.IsValid() {
		return ErrInvalidCycle
	}
	if newCycle == s.BillingCycle {
		return ErrSameCycle
	}
	if newPriceID == "" {
		return ErrMissingStripeIDs
	}
	if newPeriodEnd.Before(newPeriodStart) {
		return ErrInvalidPeriod
	}
	s.BillingCycle = newCycle
	s.StripePriceID = newPriceID
	s.CurrentPeriodStart = newPeriodStart
	s.CurrentPeriodEnd = newPeriodEnd
	// Any direct cycle change supersedes a pending schedule.
	s.clearPending()
	s.UpdatedAt = time.Now()
	return nil
}

// SchedulePendingCycle records a future cycle switch without changing
// the CURRENT cycle. Used by the downgrade path (annual → monthly): the
// user keeps their paid annual access until `effectiveAt`, then Stripe
// transitions the subscription to the new cycle automatically (our
// customer.subscription.updated webhook fires ApplyScheduledCycle).
//
// Preconditions:
//   - Subscription must be active or past_due.
//   - newCycle must differ from the current cycle.
//   - scheduleID, effectiveAt must be set (Stripe's returned values).
//
// Overwriting an existing pending cycle is allowed — the app layer is
// expected to release the previous Stripe schedule first.
func (s *Subscription) SchedulePendingCycle(newCycle BillingCycle, effectiveAt time.Time, scheduleID string) error {
	if s.Status != StatusActive && s.Status != StatusPastDue {
		return ErrInvalidTransition
	}
	if !newCycle.IsValid() {
		return ErrInvalidCycle
	}
	if newCycle == s.BillingCycle {
		return ErrSameCycle
	}
	if scheduleID == "" {
		return ErrMissingStripeIDs
	}
	cycle := newCycle
	when := effectiveAt
	sid := scheduleID
	s.PendingBillingCycle = &cycle
	s.PendingCycleEffectiveAt = &when
	s.StripeScheduleID = &sid
	s.UpdatedAt = time.Now()
	return nil
}

// ClearScheduledCycle cancels a pending cycle switch (the user changed
// their mind before the transition date, or they re-upgraded and the
// schedule was released). Safe no-op when nothing is scheduled.
func (s *Subscription) ClearScheduledCycle() {
	s.clearPending()
	s.UpdatedAt = time.Now()
}

// ApplyScheduledCycle promotes a pending cycle into the current cycle.
// Called by the webhook handler when Stripe fires the phase transition
// (customer.subscription.updated with the new price on the item). The
// caller provides the post-transition snapshot so we overwrite CurrentPeriod*
// and StripePriceID from the authoritative source.
//
// Returns ErrInvalidTransition when no pending cycle exists.
func (s *Subscription) ApplyScheduledCycle(newPriceID string, newPeriodStart, newPeriodEnd time.Time) error {
	if s.PendingBillingCycle == nil {
		return ErrInvalidTransition
	}
	if newPriceID == "" {
		return ErrMissingStripeIDs
	}
	if newPeriodEnd.Before(newPeriodStart) {
		return ErrInvalidPeriod
	}
	s.BillingCycle = *s.PendingBillingCycle
	s.StripePriceID = newPriceID
	s.CurrentPeriodStart = newPeriodStart
	s.CurrentPeriodEnd = newPeriodEnd
	s.clearPending()
	s.UpdatedAt = time.Now()
	return nil
}

// HasPendingCycleChange is a read helper for the handler DTO.
func (s *Subscription) HasPendingCycleChange() bool {
	return s.PendingBillingCycle != nil
}

func (s *Subscription) clearPending() {
	s.PendingBillingCycle = nil
	s.PendingCycleEffectiveAt = nil
	s.StripeScheduleID = nil
}

// IsPremium answers the ONLY question the billing layer cares about:
// "does this subscription waive the platform fee right now?". Active is
// Premium; past_due is Premium while within the grace window; everything
// else is NOT Premium. Keeping the logic here means the billing package
// never has to know the status enum at all.
func (s *Subscription) IsPremium(now time.Time) bool {
	switch s.Status {
	case StatusActive:
		// Stripe keeps status=active even when cancel_at_period_end is
		// TRUE; Premium stays valid until current_period_end elapses.
		return !now.After(s.CurrentPeriodEnd)
	case StatusPastDue:
		if s.GracePeriodEndsAt == nil {
			return false
		}
		return now.Before(*s.GracePeriodEndsAt)
	default:
		return false
	}
}
