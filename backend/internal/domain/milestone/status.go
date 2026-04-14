package milestone

// MilestoneStatus represents the lifecycle state of a single milestone within a proposal.
//
// State machine (happy path):
//
//	pending_funding -> funded -> submitted -> approved -> released
//
// Branching transitions:
//
//	submitted -> funded              (client rejects the submission, provider resubmits)
//	funded    -> disputed            (either party opens a dispute on funded or submitted work)
//	submitted -> disputed
//	disputed  -> funded | released   (dispute resolution restores the milestone)
//	disputed  -> refunded            (dispute resolution fully refunds the client)
//	pending_funding -> cancelled     (proposal cancelled at a milestone boundary or auto-closed)
//
// Terminal states: released, cancelled, refunded.
type MilestoneStatus string

const (
	// StatusPendingFunding means the milestone has been agreed but not yet funded.
	// The next sequential milestone (or the first, right after proposal acceptance) sits here.
	StatusPendingFunding MilestoneStatus = "pending_funding"

	// StatusFunded means the client has paid the escrow for this milestone and
	// the provider can begin work.
	StatusFunded MilestoneStatus = "funded"

	// StatusSubmitted means the provider has marked the milestone as ready for review.
	// An auto-approval timer starts running from SubmittedAt.
	StatusSubmitted MilestoneStatus = "submitted"

	// StatusApproved means the client has explicitly approved the submission,
	// OR the auto-approval timer expired without a client response.
	// This is a brief state before StatusReleased once the Stripe transfer is dispatched.
	StatusApproved MilestoneStatus = "approved"

	// StatusReleased is a terminal state: the escrow has been transferred to the provider.
	StatusReleased MilestoneStatus = "released"

	// StatusDisputed means a dispute has been opened and the escrow is frozen
	// pending resolution.
	StatusDisputed MilestoneStatus = "disputed"

	// StatusCancelled is a terminal state reached only from pending_funding.
	// A cancelled milestone was never funded, so no refund is required.
	StatusCancelled MilestoneStatus = "cancelled"

	// StatusRefunded is a terminal state reached from disputed when a dispute
	// resolution refunds the full milestone amount to the client.
	StatusRefunded MilestoneStatus = "refunded"
)

// IsValid reports whether the status is one of the recognised values.
func (s MilestoneStatus) IsValid() bool {
	switch s {
	case StatusPendingFunding, StatusFunded, StatusSubmitted, StatusApproved,
		StatusReleased, StatusDisputed, StatusCancelled, StatusRefunded:
		return true
	}
	return false
}

// IsTerminal reports whether the status is terminal (no further transitions allowed).
func (s MilestoneStatus) IsTerminal() bool {
	switch s {
	case StatusReleased, StatusCancelled, StatusRefunded:
		return true
	}
	return false
}

// IsActive reports whether the milestone currently holds or is owed escrow funds.
// Active milestones block the proposal from being considered complete.
func (s MilestoneStatus) IsActive() bool {
	switch s {
	case StatusFunded, StatusSubmitted, StatusApproved, StatusDisputed:
		return true
	}
	return false
}

// CanTransitionTo reports whether a direct transition from s to next is legal.
// The domain methods on Milestone enforce the same rules, this helper is
// exposed for callers that need to check legality without mutating state
// (e.g. UI enablement decisions, test assertions).
func (s MilestoneStatus) CanTransitionTo(next MilestoneStatus) bool {
	switch s {
	case StatusPendingFunding:
		return next == StatusFunded || next == StatusCancelled
	case StatusFunded:
		return next == StatusSubmitted || next == StatusDisputed
	case StatusSubmitted:
		return next == StatusApproved || next == StatusFunded || next == StatusDisputed
	case StatusApproved:
		return next == StatusReleased
	case StatusDisputed:
		return next == StatusFunded || next == StatusReleased || next == StatusRefunded
	}
	return false
}
