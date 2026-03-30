package message

import "errors"

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrNotParticipant       = errors.New("user is not a participant")
	ErrEmptyContent         = errors.New("message content cannot be empty")
	ErrContentTooLong       = errors.New("message content exceeds maximum length")
	ErrInvalidMessageType   = errors.New("invalid message type")
	ErrCannotEditOther      = errors.New("cannot edit another user's message")
	ErrCannotDeleteOther    = errors.New("cannot delete another user's message")
	ErrDeleteWindowExpired  = errors.New("messages can only be deleted within 1 hour of sending")
	ErrEditWindowExpired    = errors.New("messages can only be edited within 1 hour of sending")
	ErrMessageDeleted       = errors.New("message has been deleted")
	ErrSelfConversation     = errors.New("cannot create conversation with yourself")
	ErrRateLimitExceeded    = errors.New("message rate limit exceeded")
	ErrInvalidFileType      = errors.New("file type not allowed")
)
