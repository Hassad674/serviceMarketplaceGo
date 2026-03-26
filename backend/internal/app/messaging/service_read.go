package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
)

type MarkAsReadInput struct {
	UserID         uuid.UUID
	ConversationID uuid.UUID
	Seq            int
}

func (s *Service) MarkAsRead(ctx context.Context, input MarkAsReadInput) error {
	ok, err := s.messages.IsParticipant(ctx, input.ConversationID, input.UserID)
	if err != nil {
		return fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return message.ErrNotParticipant
	}

	if err := s.messages.MarkAsRead(ctx, input.ConversationID, input.UserID, input.Seq); err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}

	// Update message statuses to "read" and broadcast to sender
	if err := s.messages.MarkMessagesAsRead(ctx, input.ConversationID, input.UserID, input.Seq); err != nil {
		slog.Warn("failed to mark messages as read",
			"error", err,
			"conversation_id", input.ConversationID,
			"user_id", input.UserID,
		)
	}

	s.broadcastReadReceipt(ctx, input.ConversationID, input.UserID, input.Seq)

	return nil
}

func (s *Service) broadcastReadReceipt(ctx context.Context, convID, readerID uuid.UUID, upToSeq int) {
	participantIDs, err := s.messages.GetParticipantIDs(ctx, convID)
	if err != nil {
		return
	}

	// Notify the other participants (senders) that their messages were read
	var recipientIDs []uuid.UUID
	for _, id := range participantIDs {
		if id != readerID {
			recipientIDs = append(recipientIDs, id)
		}
	}

	if len(recipientIDs) == 0 {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"conversation_id": convID.String(),
		"reader_id":       readerID.String(),
		"up_to_seq":       upToSeq,
		"status":          "read",
	})
	if err != nil {
		slog.Error("failed to marshal read receipt payload", "error", err)
		return
	}

	if err := s.broadcaster.BroadcastStatusUpdate(ctx, recipientIDs, payload); err != nil {
		slog.Error("broadcast read receipt failed",
			"error", err,
			"conversation_id", convID,
			"reader_id", readerID,
		)
	}
}

func (s *Service) GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.messages.GetTotalUnread(ctx, userID)
}

func (s *Service) DeliverMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	msg, err := s.messages.GetMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}

	ok, err := s.messages.IsParticipant(ctx, msg.ConversationID, userID)
	if err != nil {
		return fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return message.ErrNotParticipant
	}

	if msg.Status == message.MessageStatusSent {
		return s.messages.UpdateMessageStatus(ctx, messageID, message.MessageStatusDelivered)
	}

	return nil
}
