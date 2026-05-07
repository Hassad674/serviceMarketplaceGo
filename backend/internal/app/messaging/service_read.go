package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/system"
)

type MarkAsReadInput struct {
	UserID         uuid.UUID
	OrgID          uuid.UUID
	ConversationID uuid.UUID
	Seq            int
}

func (s *Service) MarkAsRead(ctx context.Context, input MarkAsReadInput) error {
	if err := s.requireOrgAuthorized(ctx, input.ConversationID, input.OrgID, input.UserID); err != nil {
		return err
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
	// Phase R11 — notify every operator on both sides except the
	// reader themselves. Senders get the read receipt, the reader's
	// own teammates also see the conversation flip to "all read".
	recipientIDs, err := s.messages.GetOrgMemberRecipients(ctx, convID, readerID)
	if err != nil {
		return
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
	orgID, err := s.resolveUserOrgID(ctx, userID)
	if err != nil {
		return fmt.Errorf("resolve user org: %w", err)
	}

	// Resolve the message under the recipient's tenant context so
	// RLS admits the row before we even reach the org-authorisation
	// check. GetMessageForCaller installs orgID + userID as the
	// tenant context.
	msg, err := s.messages.GetMessageForCaller(ctx, messageID, orgID, userID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}

	if err := s.requireOrgAuthorized(ctx, msg.ConversationID, orgID, userID); err != nil {
		return err
	}

	if msg.Status == message.MessageStatusSent {
		// UpdateMessageStatus is a delivery-marker write — the
		// recipient is updating their own row. Tag system since the
		// authorisation has already passed.
		return s.messages.UpdateMessageStatus(system.WithSystemActor(ctx), messageID, message.MessageStatusDelivered)
	}

	return nil
}
