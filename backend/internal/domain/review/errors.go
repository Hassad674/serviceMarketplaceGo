package review

import "errors"

var (
	ErrMissingProposal = errors.New("proposal ID is required")
	ErrMissingReviewer = errors.New("reviewer ID is required")
	ErrMissingReviewed = errors.New("reviewed user ID is required")
	ErrSelfReview      = errors.New("cannot review yourself")
	ErrInvalidRating   = errors.New("rating must be between 1 and 5")
	ErrCommentTooLong  = errors.New("comment exceeds 2000 characters")
	ErrAlreadyReviewed = errors.New("you have already reviewed this proposal")
	ErrNotParticipant  = errors.New("you are not a participant of this proposal")
	ErrNotCompleted    = errors.New("proposal must be completed before reviewing")
	ErrNotFound        = errors.New("review not found")
)
