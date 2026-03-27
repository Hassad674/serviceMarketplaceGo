package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// MessageSender allows features to send messages into conversations
// without importing the messaging app package directly.
type MessageSender interface {
	SendSystemMessage(ctx context.Context, input SystemMessageInput) error
}

// SystemMessageInput contains the data needed to inject a system message
// into an existing conversation.
type SystemMessageInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        string
	Type           string
	Metadata       json.RawMessage
}
