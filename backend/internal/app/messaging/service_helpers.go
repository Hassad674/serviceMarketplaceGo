package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
)

func (s *Service) GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	return s.messages.GetParticipantIDs(ctx, conversationID)
}

// GetContactIDs returns distinct user IDs sharing conversations with the given user.
func (s *Service) GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return s.messages.GetContactIDs(ctx, userID)
}

func (s *Service) broadcastNewMessage(ctx context.Context, convID, senderID uuid.UUID, msg *message.Message) {
	participantIDs, err := s.messages.GetParticipantIDs(ctx, convID)
	if err != nil {
		return
	}

	// Filter out sender
	var recipientIDs []uuid.UUID
	for _, id := range participantIDs {
		if id != senderID {
			recipientIDs = append(recipientIDs, id)
		}
	}

	if len(recipientIDs) == 0 {
		return
	}

	payload, err := json.Marshal(marshalMessageForWS(msg))
	if err != nil {
		slog.Error("failed to marshal message for broadcast", "error", err)
		return
	}

	if err := s.broadcaster.BroadcastNewMessage(ctx, recipientIDs, payload); err != nil {
		slog.Error("broadcast new message failed",
			"error", err,
			"conversation_id", convID,
		)
	}

	// Send unread count updates (batch query to avoid N+1)
	unreadCounts, err := s.messages.GetTotalUnreadBatch(ctx, recipientIDs)
	if err != nil {
		slog.Error("get total unread batch failed", "error", err)
		return
	}
	for _, recipientID := range recipientIDs {
		count := unreadCounts[recipientID]
		if err := s.broadcaster.BroadcastUnreadCount(ctx, recipientID, count); err != nil {
			slog.Error("broadcast unread count failed",
				"error", err,
				"recipient_id", recipientID,
			)
		}
	}
}

// marshalMessageForWS converts a domain Message into a JSON-friendly map
// matching the client-side Message type (snake_case keys).
func marshalMessageForWS(msg *message.Message) map[string]any {
	metadata := json.RawMessage("null")
	if len(msg.Metadata) > 0 {
		metadata = msg.Metadata
	}

	result := map[string]any{
		"id":              msg.ID.String(),
		"conversation_id": msg.ConversationID.String(),
		"sender_id":       msg.SenderID.String(),
		"content":         msg.Content,
		"type":            string(msg.Type),
		"metadata":        metadata,
		"seq":             msg.Seq,
		"status":          string(msg.Status),
		"edited_at":       nil,
		"deleted_at":      nil,
		"created_at":      msg.CreatedAt.Format(time.RFC3339),
	}

	if msg.EditedAt != nil {
		result["edited_at"] = msg.EditedAt.Format(time.RFC3339)
	}
	if msg.DeletedAt != nil {
		result["deleted_at"] = msg.DeletedAt.Format(time.RFC3339)
	}

	return result
}

func (s *Service) enrichWithPresence(ctx context.Context, summaries []repository.ConversationSummary) {
	if len(summaries) == 0 {
		return
	}

	userIDs := make([]uuid.UUID, len(summaries))
	for i, sm := range summaries {
		userIDs[i] = sm.OtherUserID
	}

	// Best-effort: don't fail the whole request if presence is unavailable
	onlineMap, err := s.presence.BulkIsOnline(ctx, userIDs)
	if err != nil {
		return
	}

	for i := range summaries {
		summaries[i].Online = onlineMap[summaries[i].OtherUserID]
	}
}
