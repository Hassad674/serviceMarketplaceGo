package dispute

import "errors"

var (
	ErrDisputeNotFound             = errors.New("dispute not found")
	ErrInvalidStatus               = errors.New("invalid dispute status for this operation")
	ErrInvalidReason               = errors.New("invalid dispute reason for this role")
	ErrEmptyDescription            = errors.New("dispute description cannot be empty")
	ErrDescriptionTooLong          = errors.New("dispute description cannot exceed 5000 characters")
	ErrInvalidAmount               = errors.New("requested amount must be positive and not exceed proposal amount")
	ErrNotParticipant              = errors.New("user is not a participant of this dispute")
	ErrNotAuthorized               = errors.New("not authorized to perform this action on this dispute")
	ErrAlreadyDisputed             = errors.New("proposal already has an active dispute")
	ErrProposalNotDisputable       = errors.New("only active or completion_requested proposals can be disputed")
	ErrAmountMismatch              = errors.New("resolution amounts must sum to the proposal amount")
	ErrCancellationRequiresConsent = errors.New("respondent has replied; cancellation requires their consent")
	ErrCancellationAlreadyRequested = errors.New("a cancellation request is already pending")
	ErrNoCancellationPending       = errors.New("no cancellation request pending")
	ErrCounterProposalNotFound     = errors.New("counter-proposal not found")
	ErrCounterProposalNotPending   = errors.New("counter-proposal is not pending")
	ErrCannotRespondToOwnProposal  = errors.New("cannot respond to your own counter-proposal")
)
