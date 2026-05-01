package milestone

import "errors"

var (
	// ErrMilestoneNotFound is returned when a lookup by ID yields no result.
	ErrMilestoneNotFound = errors.New("milestone not found")

	// ErrInvalidStatus is returned when a transition is attempted from an
	// incompatible current status.
	ErrInvalidStatus = errors.New("invalid milestone status for this operation")

	// ErrEmptyTitle is returned when a milestone is created with an empty title.
	ErrEmptyTitle = errors.New("milestone title cannot be empty")

	// ErrEmptyDescription is returned when a milestone is created with an empty description.
	ErrEmptyDescription = errors.New("milestone description cannot be empty")

	// ErrInvalidAmount is returned when a milestone amount is zero or negative.
	// There is intentionally no minimum amount (the legacy 30 EUR floor was a
	// credit-bonus fraud rule, not a domain constraint).
	ErrInvalidAmount = errors.New("milestone amount must be greater than zero")

	// ErrInvalidSequence is returned when a milestone sequence number is < 1.
	ErrInvalidSequence = errors.New("milestone sequence must be at least 1")

	// ErrInvalidRestoreTarget is returned when RestoreFromDispute is called
	// with a target status outside {funded, released, refunded}.
	ErrInvalidRestoreTarget = errors.New("invalid target status for dispute restoration")

	// ErrTooManyMilestones is returned when a batch would exceed MaxMilestonesPerProposal.
	ErrTooManyMilestones = errors.New("too many milestones for a single proposal")

	// ErrEmptyBatch is returned when a proposal is created with zero milestones.
	ErrEmptyBatch = errors.New("proposal must have at least one milestone")

	// ErrNonConsecutiveSequence is returned when a batch of milestones has
	// non-consecutive sequence numbers (must be 1, 2, 3, ... without gaps).
	ErrNonConsecutiveSequence = errors.New("milestone sequences must be consecutive starting at 1")

	// ErrMilestonesNotSequential is returned when a batch of milestones
	// carries non-strictly-increasing deadlines: each milestone N+1 must
	// have a deadline strictly AFTER milestone N. Same-day deadlines are
	// rejected (cleanest contract — see ValidateMilestoneDeadlineOrder).
	// Milestones without a deadline are allowed and skipped during the
	// ordering check (deadline is optional).
	ErrMilestonesNotSequential = errors.New("milestone deadlines must be strictly increasing")

	// ErrMilestoneDeadlineAfterProject is returned when a milestone's
	// deadline falls after the proposal-level overall deadline. The
	// project deadline is the outer bound; no milestone can be due past it.
	ErrMilestoneDeadlineAfterProject = errors.New("milestone deadline cannot be after the project deadline")

	// ErrConcurrentUpdate is returned when an optimistic-locked update finds
	// a stale version, indicating another transaction has modified the row.
	ErrConcurrentUpdate = errors.New("milestone was modified by another transaction")

	// ErrDeliverableNotFound is returned when a deliverable lookup yields no result.
	ErrDeliverableNotFound = errors.New("milestone deliverable not found")

	// ErrDeliverableLocked is returned when attempting to delete a deliverable
	// on a milestone whose status no longer allows modifications
	// (submitted, approved, released, disputed, etc.).
	ErrDeliverableLocked = errors.New("milestone deliverables are locked in this status")
)
