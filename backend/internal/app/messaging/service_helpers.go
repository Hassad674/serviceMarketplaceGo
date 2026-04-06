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
	s.doBroadcast(ctx, convID, senderID, msg, true)
}

// broadcastSystemMessage sends a WS event to ALL participants including the
// sender. System messages (proposal status changes, call events) are created
// by the backend on behalf of a user -- the sender has no local copy, so they
// must receive the WS event to see the message appear in real time.
func (s *Service) broadcastSystemMessage(ctx context.Context, convID, senderID uuid.UUID, msg *message.Message) {
	s.doBroadcast(ctx, convID, senderID, msg, false)
}

func (s *Service) doBroadcast(ctx context.Context, convID, senderID uuid.UUID, msg *message.Message, excludeSender bool) {
	participantIDs, err := s.messages.GetParticipantIDs(ctx, convID)
	if err != nil {
		return
	}

	// For regular messages the sender already has a local copy (optimistic
	// update), so we skip them. For system messages every participant needs
	// the event because no one created the message client-side.
	var broadcastIDs []uuid.UUID
	var unreadIDs []uuid.UUID
	for _, id := range participantIDs {
		if excludeSender && id == senderID {
			continue
		}
		broadcastIDs = append(broadcastIDs, id)
		// Unread counts are only bumped for users other than the sender.
		if id != senderID {
			unreadIDs = append(unreadIDs, id)
		}
	}

	if len(broadcastIDs) == 0 {
		return
	}

	payload, err := json.Marshal(marshalMessageForWS(msg))
	if err != nil {
		slog.Error("failed to marshal message for broadcast", "error", err)
		return
	}

	if err := s.broadcaster.BroadcastNewMessage(ctx, broadcastIDs, payload); err != nil {
		slog.Error("broadcast new message failed",
			"error", err,
			"conversation_id", convID,
		)
	}

	if len(unreadIDs) == 0 {
		return
	}

	// Send unread count updates (batch query to avoid N+1)
	unreadCounts, err := s.messages.GetTotalUnreadBatch(ctx, unreadIDs)
	if err != nil {
		slog.Error("get total unread batch failed", "error", err)
		return
	}
	for _, recipientID := range unreadIDs {
		count := unreadCounts[recipientID]
		if err := s.broadcaster.BroadcastUnreadCount(ctx, recipientID, count); err != nil {
			slog.Error("broadcast unread count failed",
				"error", err,
				"recipient_id", recipientID,
			)
		}
	}
}

// broadcastMessageEdited sends a WS event with the updated message to all participants.
func (s *Service) broadcastMessageEdited(ctx context.Context, msg *message.Message) {
	participantIDs, err := s.messages.GetParticipantIDs(ctx, msg.ConversationID)
	if err != nil {
		slog.Error("get participants for edit broadcast", "error", err)
		return
	}

	payload, err := json.Marshal(marshalMessageForWS(msg))
	if err != nil {
		slog.Error("marshal edited message", "error", err)
		return
	}

	_ = s.broadcaster.BroadcastMessageEdited(ctx, participantIDs, payload)
}

// broadcastMessageDeleted sends a WS event with the message_id and conversation_id to all participants.
func (s *Service) broadcastMessageDeleted(ctx context.Context, msg *message.Message) {
	participantIDs, err := s.messages.GetParticipantIDs(ctx, msg.ConversationID)
	if err != nil {
		slog.Error("get participants for delete broadcast", "error", err)
		return
	}

	payload, err := json.Marshal(map[string]string{
		"message_id":      msg.ID.String(),
		"conversation_id": msg.ConversationID.String(),
	})
	if err != nil {
		return
	}

	_ = s.broadcaster.BroadcastMessageDeleted(ctx, participantIDs, payload)
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
		"reply_to":        nil,
		"seq":             msg.Seq,
		"status":          string(msg.Status),
		"edited_at":       nil,
		"deleted_at":      nil,
		"created_at":      msg.CreatedAt.Format(time.RFC3339),
	}

	if msg.ReplyPreview != nil {
		result["reply_to"] = map[string]any{
			"id":        msg.ReplyPreview.ID.String(),
			"sender_id": msg.ReplyPreview.SenderID.String(),
			"content":   msg.ReplyPreview.Content,
			"type":      string(msg.ReplyPreview.Type),
		}
	}

	if msg.EditedAt != nil {
		result["edited_at"] = msg.EditedAt.Format(time.RFC3339)
	}
	if msg.DeletedAt != nil {
		result["deleted_at"] = msg.DeletedAt.Format(time.RFC3339)
	}

	return result
}

// recordMediaIfNeeded fires a background media record for file and voice messages
// so that attachments sent in conversations appear in the admin media table.
func (s *Service) recordMediaIfNeeded(msg *message.Message) {
	if s.mediaRecorder == nil {
		return
	}

	switch msg.Type {
	case message.MessageTypeFile:
		var meta message.FileMetadata
		if err := json.Unmarshal(msg.Metadata, &meta); err != nil {
			slog.Error("parse file metadata for media record", "error", err, "msg_id", msg.ID)
			return
		}
		go s.mediaRecorder.RecordUploadRaw(
			msg.SenderID, meta.URL, meta.Filename, meta.MimeType, meta.Size, "message_attachment",
		)
	case message.MessageTypeVoice:
		var meta message.VoiceMetadata
		if err := json.Unmarshal(msg.Metadata, &meta); err != nil {
			slog.Error("parse voice metadata for media record", "error", err, "msg_id", msg.ID)
			return
		}
		go s.mediaRecorder.RecordUploadRaw(
			msg.SenderID, meta.URL, "voice_message.ogg", meta.MimeType, meta.Size, "message_attachment",
		)
	}
}

// moderateTextIfNeeded fires a background text moderation check for text messages.
// Results are stored in the database asynchronously (never blocks the send flow).
func (s *Service) moderateTextIfNeeded(msg *message.Message) {
	if s.textModeration == nil {
		return
	}
	if msg.Type != message.MessageTypeText || msg.Content == "" {
		return
	}

	go s.runTextModeration(msg.ID, msg.Content)
}

// runTextModeration calls the text moderation service and updates the message.
func (s *Service) runTextModeration(msgID uuid.UUID, content string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := s.textModeration.AnalyzeText(ctx, content)
	if err != nil {
		slog.Error("text moderation failed", "error", err, "msg_id", msgID)
		return
	}

	if result.IsSafe {
		return
	}

	status := "flagged"
	if result.MaxScore >= 0.9 {
		status = "hidden"
	}

	labelsJSON, err := json.Marshal(result.Labels)
	if err != nil {
		slog.Error("marshal moderation labels", "error", err, "msg_id", msgID)
		return
	}

	if err := s.messages.UpdateMessageModeration(ctx, msgID, status, result.MaxScore, labelsJSON); err != nil {
		slog.Error("update message moderation", "error", err, "msg_id", msgID)
	}
}

// populateReplyPreview fetches the replied-to message and attaches a preview.
func (s *Service) populateReplyPreview(ctx context.Context, msg *message.Message) {
	if msg.ReplyToID == nil {
		return
	}
	replied, err := s.messages.GetMessage(ctx, *msg.ReplyToID)
	if err != nil {
		slog.Warn("failed to fetch reply-to message", "reply_to_id", msg.ReplyToID, "error", err)
		return
	}
	msg.ReplyPreview = &message.ReplyPreview{
		ID:       replied.ID,
		SenderID: replied.SenderID,
		Content:  message.TruncateContent(replied.Content, 100),
		Type:     replied.Type,
	}
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
