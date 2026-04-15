package referral

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Limits used by validation. Public so the handler layer can echo them in
// validation error messages if needed.
const (
	MinRatePct        = 0.0
	MaxRatePct        = 50.0
	MinDurationMonths = 1
	MaxDurationMonths = 24
	DefaultDurationMonths = 6
	MaxIntroMessageLen    = 2000
)

// Referral is the root aggregate of the apport d'affaires (business referral)
// feature. One row tracks one introduction between a provider and a client
// made by a referrer (apporteur), through its full lifecycle from pending
// negotiation to commission distribution.
//
// Currency-relevant amounts (rate %) live on the Referral; cent amounts live
// on Commission rows so multi-milestone payouts each have their own audit row.
type Referral struct {
	ID                   uuid.UUID
	ReferrerID           uuid.UUID
	ProviderID           uuid.UUID
	ClientID             uuid.UUID
	RatePct              float64
	DurationMonths       int16
	IntroSnapshot        IntroSnapshot
	IntroSnapshotVersion int
	IntroMessageProvider string
	IntroMessageClient   string
	Status               Status
	Version              int
	ActivatedAt          *time.Time
	ExpiresAt            *time.Time
	LastActionAt         time.Time
	RejectionReason      string
	RejectedBy           *uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// NewReferralInput is the validated input bag for NewReferral. Grouping into
// a struct keeps the constructor signature stable as new optional fields are
// added (the project's 4-param rule).
type NewReferralInput struct {
	ReferrerID           uuid.UUID
	ProviderID           uuid.UUID
	ClientID             uuid.UUID
	RatePct              float64
	DurationMonths       int16
	IntroSnapshot        IntroSnapshot
	IntroMessageProvider string
	IntroMessageClient   string
}

// NewReferral constructs a Referral in StatusPendingProvider. It enforces the
// full set of domain invariants (self-deal forbidden, rate within bounds,
// snapshot well-formed) and returns a sentinel error from errors.go on the
// first failure so the app layer can map it to an HTTP status with errors.Is.
func NewReferral(input NewReferralInput) (*Referral, error) {
	if input.ReferrerID == uuid.Nil || input.ProviderID == uuid.Nil || input.ClientID == uuid.Nil {
		return nil, ErrNotAuthorized
	}
	if input.ReferrerID == input.ProviderID || input.ReferrerID == input.ClientID || input.ProviderID == input.ClientID {
		return nil, ErrSelfReferral
	}
	if input.RatePct < MinRatePct || input.RatePct > MaxRatePct {
		return nil, ErrRateOutOfRange
	}
	duration := input.DurationMonths
	if duration == 0 {
		duration = DefaultDurationMonths
	}
	if duration < MinDurationMonths || duration > MaxDurationMonths {
		return nil, ErrDurationOutOfRange
	}
	if err := validateIntroMessage(input.IntroMessageProvider); err != nil {
		return nil, err
	}
	if err := validateIntroMessage(input.IntroMessageClient); err != nil {
		return nil, err
	}
	if err := input.IntroSnapshot.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Referral{
		ID:                   uuid.New(),
		ReferrerID:           input.ReferrerID,
		ProviderID:           input.ProviderID,
		ClientID:             input.ClientID,
		RatePct:              input.RatePct,
		DurationMonths:       duration,
		IntroSnapshot:        input.IntroSnapshot,
		IntroSnapshotVersion: SnapshotVersion,
		IntroMessageProvider: input.IntroMessageProvider,
		IntroMessageClient:   input.IntroMessageClient,
		Status:               StatusPendingProvider,
		Version:              1,
		LastActionAt:         now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

func validateIntroMessage(msg string) error {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return ErrEmptyMessage
	}
	if len([]rune(trimmed)) > MaxIntroMessageLen {
		return ErrMessageTooLong
	}
	return nil
}

// ─── State machine ─────────────────────────────────────────────────────────
//
// All transition methods follow the same shape:
//   1. Refuse if the referral is in a terminal state (defence in depth).
//   2. Refuse if the current status doesn't match the expected one.
//   3. Refuse if the actor is not the authorised party for this transition.
//   4. Validate any input (new rate, reason length).
//   5. Mutate state, bump LastActionAt + UpdatedAt.
//
// Negotiation rounds are bilateral apporteur ↔ provider: the client never
// negotiates the rate (Modèle A — the provider absorbs the commission and the
// client pays its normal price, oblivious to the rate).

// AcceptByProvider transitions pending_provider → pending_client.
// Called when the provider agrees to the current rate without changes.
func (r *Referral) AcceptByProvider(actorID uuid.UUID) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingProvider {
		return ErrInvalidTransition
	}
	if actorID != r.ProviderID {
		return ErrNotAuthorized
	}
	r.transitionTo(StatusPendingClient)
	return nil
}

// RejectByProvider transitions pending_provider → rejected.
func (r *Referral) RejectByProvider(actorID uuid.UUID, reason string) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingProvider {
		return ErrInvalidTransition
	}
	if actorID != r.ProviderID {
		return ErrNotAuthorized
	}
	r.RejectionReason = strings.TrimSpace(reason)
	r.RejectedBy = &actorID
	r.transitionTo(StatusRejected)
	return nil
}

// NegotiateByProvider counter-offers a new rate. The state moves to
// pending_referrer (waiting for the apporteur to validate or counter-counter).
// Increments version because a new distinct rate is now on the table.
func (r *Referral) NegotiateByProvider(actorID uuid.UUID, newRate float64) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingProvider {
		return ErrInvalidTransition
	}
	if actorID != r.ProviderID {
		return ErrNotAuthorized
	}
	if newRate < MinRatePct || newRate > MaxRatePct {
		return ErrRateOutOfRange
	}
	r.RatePct = newRate
	r.Version++
	r.transitionTo(StatusPendingReferrer)
	return nil
}

// AcceptByReferrer transitions pending_referrer → pending_client.
// Called when the apporteur accepts the provider's counter-offer as-is.
func (r *Referral) AcceptByReferrer(actorID uuid.UUID) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingReferrer {
		return ErrInvalidTransition
	}
	if actorID != r.ReferrerID {
		return ErrNotAuthorized
	}
	r.transitionTo(StatusPendingClient)
	return nil
}

// RejectByReferrer transitions pending_referrer → rejected.
// The apporteur ends the negotiation rather than counter-counter.
func (r *Referral) RejectByReferrer(actorID uuid.UUID, reason string) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingReferrer {
		return ErrInvalidTransition
	}
	if actorID != r.ReferrerID {
		return ErrNotAuthorized
	}
	r.RejectionReason = strings.TrimSpace(reason)
	r.RejectedBy = &actorID
	r.transitionTo(StatusRejected)
	return nil
}

// NegotiateByReferrer counter-counter-offers a new rate. Status moves back to
// pending_provider for the provider to react. Version increments.
func (r *Referral) NegotiateByReferrer(actorID uuid.UUID, newRate float64) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingReferrer {
		return ErrInvalidTransition
	}
	if actorID != r.ReferrerID {
		return ErrNotAuthorized
	}
	if newRate < MinRatePct || newRate > MaxRatePct {
		return ErrRateOutOfRange
	}
	r.RatePct = newRate
	r.Version++
	r.transitionTo(StatusPendingProvider)
	return nil
}

// AcceptByClient transitions pending_client → active.
// This is the activation: ActivatedAt and ExpiresAt are stamped now, and the
// caller is expected to spin up the provider↔client conversation system message.
// Note: the client does NOT see the rate (Modèle A) — only Accept/Reject is offered.
func (r *Referral) AcceptByClient(actorID uuid.UUID) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingClient {
		return ErrInvalidTransition
	}
	if actorID != r.ClientID {
		return ErrNotAuthorized
	}
	now := time.Now().UTC()
	expires := now.AddDate(0, int(r.DurationMonths), 0)
	r.ActivatedAt = &now
	r.ExpiresAt = &expires
	r.Status = StatusActive
	r.LastActionAt = now
	r.UpdatedAt = now
	return nil
}

// RejectByClient transitions pending_client → rejected.
func (r *Referral) RejectByClient(actorID uuid.UUID, reason string) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusPendingClient {
		return ErrInvalidTransition
	}
	if actorID != r.ClientID {
		return ErrNotAuthorized
	}
	r.RejectionReason = strings.TrimSpace(reason)
	r.RejectedBy = &actorID
	r.transitionTo(StatusRejected)
	return nil
}

// Cancel transitions any pre-active pending state → cancelled.
// Only the referrer can cancel. After activation, use Terminate instead.
func (r *Referral) Cancel(actorID uuid.UUID) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if !r.Status.IsPending() {
		return ErrInvalidTransition
	}
	if actorID != r.ReferrerID {
		return ErrNotAuthorized
	}
	r.transitionTo(StatusCancelled)
	return nil
}

// Terminate transitions active → terminated.
// Only the referrer can voluntarily end an active referral.
// Existing attributions are NOT touched — they stay valid until paid out.
func (r *Referral) Terminate(actorID uuid.UUID) error {
	if err := r.guardActive(); err != nil {
		return err
	}
	if r.Status != StatusActive {
		return ErrInvalidTransition
	}
	if actorID != r.ReferrerID {
		return ErrNotAuthorized
	}
	r.transitionTo(StatusTerminated)
	return nil
}

// Expire is the cron entry point. It accepts EITHER a stale pending state
// (no action for 14 days, see expiry rule in the cron worker) OR an active
// referral whose ExpiresAt has passed.
func (r *Referral) Expire() error {
	if r.Status.IsTerminal() {
		return ErrAlreadyTerminal
	}
	if !r.Status.IsPending() && r.Status != StatusActive {
		return ErrInvalidTransition
	}
	r.transitionTo(StatusExpired)
	return nil
}

// guardActive returns ErrAlreadyTerminal if the referral is in a terminal state.
// All non-cron transitions go through this guard.
func (r *Referral) guardActive() error {
	if r.Status.IsTerminal() {
		return ErrAlreadyTerminal
	}
	return nil
}

// transitionTo updates the status and the bookkeeping timestamps in one place,
// so every action goes through the same write path. LastActionAt is critical:
// the cron expirer queries it to know which intros have been silent for 14 days.
func (r *Referral) transitionTo(s Status) {
	now := time.Now().UTC()
	r.Status = s
	r.LastActionAt = now
	r.UpdatedAt = now
}

// IsExclusivityActive reports whether this referral is currently inside its
// exclusivity window — the only state in which the attributor will create new
// attributions for incoming proposals.
func (r *Referral) IsExclusivityActive(now time.Time) bool {
	if r.Status != StatusActive {
		return false
	}
	if r.ExpiresAt == nil {
		return false
	}
	return now.Before(*r.ExpiresAt)
}
