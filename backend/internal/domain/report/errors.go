package report

import "errors"

var (
	ErrNotFound                = errors.New("report not found")
	ErrMissingReporter         = errors.New("reporter ID is required")
	ErrMissingTarget           = errors.New("target ID is required")
	ErrInvalidTargetType       = errors.New("target type must be 'message' or 'user'")
	ErrInvalidReason           = errors.New("invalid report reason")
	ErrReasonNotAllowedForType = errors.New("this reason is not allowed for this target type")
	ErrSelfReport              = errors.New("cannot report yourself")
	ErrAlreadyReported         = errors.New("you have already reported this")
	ErrDescriptionTooLong      = errors.New("description exceeds maximum length")
)
