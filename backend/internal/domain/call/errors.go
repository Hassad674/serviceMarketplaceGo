package call

import "errors"

var (
	ErrInvalidCallType   = errors.New("invalid call type")
	ErrInvalidTransition = errors.New("invalid call status transition")
	ErrSelfCall          = errors.New("cannot call yourself")
	ErrCallNotFound      = errors.New("call not found")
	ErrUserBusy          = errors.New("user is already in a call")
	ErrRecipientOffline  = errors.New("recipient is offline")
	ErrNotParticipant    = errors.New("user is not a participant of this call")
	ErrNotConfigured     = errors.New("call service is not configured")
)
