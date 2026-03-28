package notification

import "errors"

var (
	ErrNotFound    = errors.New("notification not found")
	ErrMissingUser = errors.New("user ID is required")
	ErrInvalidType = errors.New("invalid notification type")
	ErrEmptyTitle  = errors.New("title is required")
	ErrNotOwner    = errors.New("you do not own this notification")
)
