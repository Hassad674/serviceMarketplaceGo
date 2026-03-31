package messaging

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/service"
)

// FindOrCreateConversation finds or creates a conversation between two users
// and sends an initial system message. Returns the conversation ID.
func (s *Service) FindOrCreateConversation(ctx context.Context, input service.FindOrCreateConversationInput) (uuid.UUID, error) {
	convID, _, err := s.messages.FindOrCreateConversation(ctx, input.UserA, input.UserB)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find or create conversation: %w", err)
	}

	if input.Content != "" {
		sysInput := service.SystemMessageInput{
			ConversationID: convID,
			SenderID:       input.UserA,
			Content:        input.Content,
			Type:           input.Type,
		}
		if err := s.SendSystemMessage(ctx, sysInput); err != nil {
			return uuid.Nil, fmt.Errorf("send system message: %w", err)
		}
	}

	return convID, nil
}

// SendSystemMessage injects a system-level message into a conversation.
// This is used by other features (e.g. proposals) to send event messages
// without rate limiting or participant verification.
func (s *Service) SendSystemMessage(ctx context.Context, input service.SystemMessageInput) error {
	msgType := message.MessageType(input.Type)

	msg, err := message.NewMessage(message.NewMessageInput{
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		Content:        input.Content,
		Type:           msgType,
		Metadata:       input.Metadata,
	})
	if err != nil {
		return fmt.Errorf("create system message: %w", err)
	}

	if err := s.messages.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("persist system message: %w", err)
	}

	if err := s.messages.IncrementUnread(ctx, input.ConversationID, input.SenderID); err != nil {
		return fmt.Errorf("increment unread: %w", err)
	}

	s.broadcastSystemMessage(ctx, input.ConversationID, input.SenderID, msg)

	return nil
}
