// Package milestone is the pure domain layer for proposal milestones.
//
// A Milestone represents a single funding/delivery step within a Proposal.
// The milestone owns the payment lifecycle — a proposal is an agreement
// between two parties, while its milestones are the concrete escrow-and-release
// units that actually move money.
//
// Every Proposal has at least one Milestone. A fixed-price "one-time" mission
// is modelled internally as a single-milestone proposal — there is no second
// code path for legacy single-amount proposals.
//
// This package has zero external dependencies beyond the Go stdlib and
// github.com/google/uuid. Any orchestration (payment, notification, auth,
// persistence) is the responsibility of the app or adapter layers.
package milestone

import (
	"time"

	"github.com/google/uuid"
)

// MaxMilestonesPerProposal caps how many milestones a single proposal can
// have. This prevents abusive structures (e.g. 200 x 5 EUR milestones) while
// leaving ample room for legitimate multi-phase projects.
const MaxMilestonesPerProposal = 20

// Milestone is a single escrow-and-release step within a proposal.
// Amount is stored in centimes (1 EUR = 100 centimes).
type Milestone struct {
	ID          uuid.UUID
	ProposalID  uuid.UUID
	Sequence    int
	Title       string
	Description string
	Amount      int64
	Deadline    *time.Time
	Status      MilestoneStatus

	// Version is the optimistic concurrency counter. Every successful Update
	// in the adapter layer increments this. Callers reading for update must
	// pass back the version they observed; a mismatch yields ErrConcurrentUpdate.
	Version int

	// Lifecycle timestamps — each one is set exactly once when its matching
	// transition occurs, except SubmittedAt which is cleared by Reject so
	// that a resubmission restarts the auto-approval timer from scratch.
	FundedAt    *time.Time
	SubmittedAt *time.Time
	ApprovedAt  *time.Time
	ReleasedAt  *time.Time
	DisputedAt  *time.Time
	CancelledAt *time.Time

	// ActiveDisputeID points to the dispute currently frozen on this
	// milestone. Cleared by RestoreFromDispute when the dispute resolves.
	ActiveDisputeID *uuid.UUID

	// LastDisputeID is the most recent dispute that ever touched this
	// milestone. Set on OpenDispute and NEVER cleared, so the UI can keep
	// displaying the historical resolution after restoration.
	LastDisputeID *uuid.UUID

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewMilestoneInput is the validated factory input for a single milestone.
type NewMilestoneInput struct {
	ProposalID  uuid.UUID
	Sequence    int
	Title       string
	Description string
	Amount      int64
	Deadline    *time.Time
}

// NewMilestone builds a validated, pending_funding Milestone.
//
// It is deliberately the only constructor exposed — all new milestones go
// through the same validation funnel (empty title/description, non-positive
// amount, sequence < 1).
func NewMilestone(input NewMilestoneInput) (*Milestone, error) {
	if input.Title == "" {
		return nil, ErrEmptyTitle
	}
	if input.Description == "" {
		return nil, ErrEmptyDescription
	}
	if input.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if input.Sequence < 1 {
		return nil, ErrInvalidSequence
	}
	now := time.Now()
	return &Milestone{
		ID:          uuid.New(),
		ProposalID:  input.ProposalID,
		Sequence:    input.Sequence,
		Title:       input.Title,
		Description: input.Description,
		Amount:      input.Amount,
		Deadline:    input.Deadline,
		Status:      StatusPendingFunding,
		Version:     0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// NewMilestoneBatch builds a validated, atomic set of milestones for a
// single proposal. Sequences must be consecutive starting at 1, count must
// be between 1 and MaxMilestonesPerProposal.
//
// Returning an error here guarantees the caller never sees a half-valid
// batch. On success, all milestones are in StatusPendingFunding.
func NewMilestoneBatch(proposalID uuid.UUID, inputs []NewMilestoneInput) ([]*Milestone, error) {
	if len(inputs) == 0 {
		return nil, ErrEmptyBatch
	}
	if len(inputs) > MaxMilestonesPerProposal {
		return nil, ErrTooManyMilestones
	}

	// Enforce consecutive sequences 1..N so the ordering is unambiguous in
	// both DB and UI. Any gap or duplicate aborts the batch.
	seen := make(map[int]struct{}, len(inputs))
	for _, in := range inputs {
		if in.Sequence < 1 || in.Sequence > len(inputs) {
			return nil, ErrNonConsecutiveSequence
		}
		if _, dup := seen[in.Sequence]; dup {
			return nil, ErrNonConsecutiveSequence
		}
		seen[in.Sequence] = struct{}{}
	}

	out := make([]*Milestone, 0, len(inputs))
	for _, in := range inputs {
		in.ProposalID = proposalID
		m, err := NewMilestone(in)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// Fund transitions pending_funding -> funded. Called after the Stripe
// PaymentIntent has been captured and the escrow is safely held.
func (m *Milestone) Fund() error {
	if m.Status != StatusPendingFunding {
		return ErrInvalidStatus
	}
	now := time.Now()
	m.Status = StatusFunded
	m.FundedAt = &now
	m.UpdatedAt = now
	return nil
}

// Submit transitions funded -> submitted. Called by the provider to mark
// the milestone as ready for client review. Starts the auto-approval window.
func (m *Milestone) Submit() error {
	if m.Status != StatusFunded {
		return ErrInvalidStatus
	}
	now := time.Now()
	m.Status = StatusSubmitted
	m.SubmittedAt = &now
	m.UpdatedAt = now
	return nil
}

// Approve transitions submitted -> approved. Called when the client
// explicitly approves the submitted work, OR when the scheduler auto-
// approves after the review window expires.
//
// The domain does not distinguish between explicit and auto approvals —
// the app layer is responsible for recording the actor in the audit trail.
func (m *Milestone) Approve() error {
	if m.Status != StatusSubmitted {
		return ErrInvalidStatus
	}
	now := time.Now()
	m.Status = StatusApproved
	m.ApprovedAt = &now
	m.UpdatedAt = now
	return nil
}

// Reject transitions submitted -> funded. Called by the client to refuse
// the submitted work and ask for revisions. The provider can then Submit()
// again after applying corrections.
//
// SubmittedAt is intentionally cleared so that the next Submit resets the
// auto-approval timer from zero.
func (m *Milestone) Reject() error {
	if m.Status != StatusSubmitted {
		return ErrInvalidStatus
	}
	m.Status = StatusFunded
	m.SubmittedAt = nil
	m.UpdatedAt = time.Now()
	return nil
}

// Release transitions approved -> released. This is a terminal state: the
// escrow has been transferred to the provider's Stripe account.
func (m *Milestone) Release() error {
	if m.Status != StatusApproved {
		return ErrInvalidStatus
	}
	now := time.Now()
	m.Status = StatusReleased
	m.ReleasedAt = &now
	m.UpdatedAt = now
	return nil
}

// OpenDispute transitions funded|submitted -> disputed, recording the
// dispute ID in both ActiveDisputeID and LastDisputeID (same pattern as
// the proposal-level dispute tracking).
//
// A milestone in pending_funding or any terminal state cannot be disputed.
func (m *Milestone) OpenDispute(disputeID uuid.UUID) error {
	if m.Status != StatusFunded && m.Status != StatusSubmitted {
		return ErrInvalidStatus
	}
	now := time.Now()
	m.ActiveDisputeID = &disputeID
	m.LastDisputeID = &disputeID
	m.Status = StatusDisputed
	m.DisputedAt = &now
	m.UpdatedAt = now
	return nil
}

// RestoreFromDispute returns a disputed milestone to one of three valid
// outcomes: back to funded (work continues), released (escrow awarded to
// provider despite dispute), or refunded (escrow returned to client).
//
// ActiveDisputeID is cleared. LastDisputeID is kept for historical display.
// If target is StatusReleased, ReleasedAt is set now so the timeline reflects
// the resolution moment.
func (m *Milestone) RestoreFromDispute(target MilestoneStatus) error {
	if m.Status != StatusDisputed {
		return ErrInvalidStatus
	}
	switch target {
	case StatusFunded, StatusReleased, StatusRefunded:
	default:
		return ErrInvalidRestoreTarget
	}
	now := time.Now()
	m.ActiveDisputeID = nil
	m.Status = target
	if target == StatusReleased && m.ReleasedAt == nil {
		m.ReleasedAt = &now
	}
	m.UpdatedAt = now
	return nil
}

// Cancel transitions pending_funding -> cancelled. Called when the proposal
// is cancelled at a milestone boundary or auto-closes after the client
// fails to fund the next milestone in time.
//
// Cancel is not legal on any other status: once funded, the escrow exists
// and a dispute/release path must be followed instead.
func (m *Milestone) Cancel() error {
	if m.Status != StatusPendingFunding {
		return ErrInvalidStatus
	}
	now := time.Now()
	m.Status = StatusCancelled
	m.CancelledAt = &now
	m.UpdatedAt = now
	return nil
}

// IsTerminal is a convenience wrapper over the status helper.
func (m *Milestone) IsTerminal() bool { return m.Status.IsTerminal() }

// IsActive is a convenience wrapper over the status helper.
func (m *Milestone) IsActive() bool { return m.Status.IsActive() }

// SumAmount returns the total of the given milestones' amounts in centimes.
// Used by the app layer to keep the cached proposal.amount in sync.
func SumAmount(milestones []*Milestone) int64 {
	var total int64
	for _, m := range milestones {
		total += m.Amount
	}
	return total
}

// FindCurrentActive returns the first non-terminal milestone by ascending
// sequence, or nil if all milestones are terminal. This is the milestone
// "in play" in the sequential flow — the one whose CTA appears in the UI.
func FindCurrentActive(milestones []*Milestone) *Milestone {
	var current *Milestone
	for _, m := range milestones {
		if m.IsTerminal() {
			continue
		}
		if current == nil || m.Sequence < current.Sequence {
			current = m
		}
	}
	return current
}

// AllReleased reports whether every milestone in the slice has reached the
// terminal released state. Used to decide if the proposal is fully completed.
func AllReleased(milestones []*Milestone) bool {
	if len(milestones) == 0 {
		return false
	}
	for _, m := range milestones {
		if m.Status != StatusReleased {
			return false
		}
	}
	return true
}

// AnyFunded reports whether at least one milestone has been funded (i.e. the
// client has engaged real money on the contract). Used to distinguish a
// proposal that has started versus one that is merely accepted.
func AnyFunded(milestones []*Milestone) bool {
	for _, m := range milestones {
		switch m.Status {
		case StatusFunded, StatusSubmitted, StatusApproved, StatusReleased, StatusDisputed, StatusRefunded:
			return true
		}
	}
	return false
}
