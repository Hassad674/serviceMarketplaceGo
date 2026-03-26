package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type Service struct {
	messages    repository.MessageRepository
	users       repository.UserRepository
	presence    service.PresenceService
	broadcaster service.MessageBroadcaster
	storage     service.StorageService
	rateLimiter service.MessagingRateLimiter
}

type ServiceDeps struct {
	Messages    repository.MessageRepository
	Users       repository.UserRepository
	Presence    service.PresenceService
	Broadcaster service.MessageBroadcaster
	Storage     service.StorageService
	RateLimiter service.MessagingRateLimiter
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		messages:    deps.Messages,
		users:       deps.Users,
		presence:    deps.Presence,
		broadcaster: deps.Broadcaster,
		storage:     deps.Storage,
		rateLimiter: deps.RateLimiter,
	}
}

type StartConversationInput struct {
	SenderID    uuid.UUID
	RecipientID uuid.UUID
	Content     string
	Type        message.MessageType
	Metadata    json.RawMessage
}

func (s *Service) StartConversation(ctx context.Context, input StartConversationInput) (*message.Message, uuid.UUID, error) {
	if input.SenderID == input.RecipientID {
		return nil, uuid.UUID{}, message.ErrSelfConversation
	}

	if _, err := s.users.GetByID(ctx, input.RecipientID); err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("get recipient: %w", err)
	}

	allowed, err := s.rateLimiter.Allow(ctx, input.SenderID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("check rate limit: %w", err)
	}
	if !allowed {
		return nil, uuid.UUID{}, message.ErrRateLimitExceeded
	}

	convID, _, err := s.messages.FindOrCreateConversation(ctx, input.SenderID, input.RecipientID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("find or create conversation: %w", err)
	}

	msg, err := message.NewMessage(convID, input.SenderID, input.Content, input.Type, input.Metadata, 0)
	if err != nil {
		return nil, uuid.UUID{}, err
	}

	if err := s.messages.CreateMessage(ctx, msg); err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("create message: %w", err)
	}

	if err := s.messages.IncrementUnread(ctx, convID, input.SenderID); err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("increment unread: %w", err)
	}

	s.broadcastNewMessage(ctx, convID, input.SenderID, msg)

	return msg, convID, nil
}

type SendMessageInput struct {
	SenderID       uuid.UUID
	ConversationID uuid.UUID
	Content        string
	Type           message.MessageType
	Metadata       json.RawMessage
}

func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (*message.Message, error) {
	ok, err := s.messages.IsParticipant(ctx, input.ConversationID, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return nil, message.ErrNotParticipant
	}

	allowed, err := s.rateLimiter.Allow(ctx, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("check rate limit: %w", err)
	}
	if !allowed {
		return nil, message.ErrRateLimitExceeded
	}

	msg, err := message.NewMessage(input.ConversationID, input.SenderID, input.Content, input.Type, input.Metadata, 0)
	if err != nil {
		return nil, err
	}

	if err := s.messages.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	if err := s.messages.IncrementUnread(ctx, input.ConversationID, input.SenderID); err != nil {
		return nil, fmt.Errorf("increment unread: %w", err)
	}

	s.broadcastNewMessage(ctx, input.ConversationID, input.SenderID, msg)

	return msg, nil
}

func (s *Service) ListConversations(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]repository.ConversationSummary, string, error) {
	params := repository.ListConversationsParams{
		UserID: userID,
		Cursor: cursorStr,
		Limit:  limit,
	}

	summaries, nextCursor, err := s.messages.ListConversations(ctx, params)
	if err != nil {
		return nil, "", fmt.Errorf("list conversations: %w", err)
	}

	s.enrichWithPresence(ctx, summaries)

	return summaries, nextCursor, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID uuid.UUID, cursorStr string, limit int) ([]*message.Message, string, error) {
	ok, err := s.messages.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, "", fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return nil, "", message.ErrNotParticipant
	}

	params := repository.ListMessagesParams{
		ConversationID: conversationID,
		Cursor:         cursorStr,
		Limit:          limit,
	}

	return s.messages.ListMessages(ctx, params)
}

func (s *Service) GetMessagesSinceSeq(ctx context.Context, userID, conversationID uuid.UUID, sinceSeq int) ([]*message.Message, error) {
	ok, err := s.messages.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return nil, message.ErrNotParticipant
	}

	return s.messages.GetMessagesSinceSeq(ctx, conversationID, sinceSeq, 50)
}

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
		// Best-effort: log but don't fail the request
		_ = err
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

	payload, _ := json.Marshal(map[string]any{
		"conversation_id": convID.String(),
		"reader_id":       readerID.String(),
		"up_to_seq":       upToSeq,
		"status":          "read",
	})

	_ = s.broadcaster.BroadcastStatusUpdate(ctx, recipientIDs, payload)
}

type EditMessageInput struct {
	UserID    uuid.UUID
	MessageID uuid.UUID
	Content   string
}

func (s *Service) EditMessage(ctx context.Context, input EditMessageInput) (*message.Message, error) {
	msg, err := s.messages.GetMessage(ctx, input.MessageID)
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	if msg.SenderID != input.UserID {
		return nil, message.ErrCannotEditOther
	}
	if msg.DeletedAt != nil {
		return nil, message.ErrMessageDeleted
	}

	if input.Content == "" {
		return nil, message.ErrEmptyContent
	}
	if len(input.Content) > message.MaxContentLength {
		return nil, message.ErrContentTooLong
	}

	msg.Edit(input.Content)

	if err := s.messages.UpdateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("update message: %w", err)
	}

	return msg, nil
}

type DeleteMessageInput struct {
	UserID    uuid.UUID
	MessageID uuid.UUID
}

func (s *Service) DeleteMessage(ctx context.Context, input DeleteMessageInput) error {
	msg, err := s.messages.GetMessage(ctx, input.MessageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}

	if msg.SenderID != input.UserID {
		return message.ErrCannotDeleteOther
	}

	msg.SoftDelete()

	if err := s.messages.UpdateMessage(ctx, msg); err != nil {
		return fmt.Errorf("update message: %w", err)
	}

	return nil
}

func (s *Service) GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.messages.GetTotalUnread(ctx, userID)
}

type GetPresignedURLInput struct {
	UserID      uuid.UUID
	Filename    string
	ContentType string
}

func (s *Service) GetPresignedUploadURL(ctx context.Context, input GetPresignedURLInput) (string, string, error) {
	key := fmt.Sprintf("messaging/%s/%d_%s", input.UserID.String(), time.Now().UnixMilli(), input.Filename)

	uploadURL, err := s.storage.GetPresignedUploadURL(ctx, key, input.ContentType, 15*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("get presigned url: %w", err)
	}

	publicURL := s.storage.GetPublicURL(key)

	return uploadURL, publicURL, nil
}

func (s *Service) GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	return s.messages.GetParticipantIDs(ctx, conversationID)
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

	payload, _ := json.Marshal(marshalMessageForWS(msg))

	_ = s.broadcaster.BroadcastNewMessage(ctx, recipientIDs, payload)

	// Send unread count updates
	for _, recipientID := range recipientIDs {
		count, err := s.messages.GetTotalUnread(ctx, recipientID)
		if err == nil {
			_ = s.broadcaster.BroadcastUnreadCount(ctx, recipientID, count)
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
