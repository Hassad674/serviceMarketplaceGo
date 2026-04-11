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
	"marketplace-backend/pkg/sanitize"
)

type Service struct {
	messages       repository.MessageRepository
	users          repository.UserRepository
	orgs           repository.OrganizationRepository
	orgMembers     repository.OrganizationMemberRepository
	presence       service.PresenceService
	broadcaster    service.MessageBroadcaster
	storage        service.StorageService
	rateLimiter    service.MessagingRateLimiter
	mediaRecorder  service.MediaRecorder
	textModeration service.TextModerationService
	adminNotifier  service.AdminNotifierService
}

type ServiceDeps struct {
	Messages      repository.MessageRepository
	Users         repository.UserRepository
	Organizations repository.OrganizationRepository
	OrgMembers    repository.OrganizationMemberRepository
	Presence      service.PresenceService
	Broadcaster   service.MessageBroadcaster
	Storage       service.StorageService
	RateLimiter   service.MessagingRateLimiter
	MediaRecorder service.MediaRecorder
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		messages:      deps.Messages,
		users:         deps.Users,
		orgs:          deps.Organizations,
		orgMembers:    deps.OrgMembers,
		presence:      deps.Presence,
		broadcaster:   deps.Broadcaster,
		storage:       deps.Storage,
		rateLimiter:   deps.RateLimiter,
		mediaRecorder: deps.MediaRecorder,
	}
}

// SetMediaRecorder sets the media recorder after construction.
// This breaks the circular init dependency: messaging is created before media.
func (s *Service) SetMediaRecorder(recorder service.MediaRecorder) {
	s.mediaRecorder = recorder
}

// SetTextModeration sets the text moderation service after construction.
func (s *Service) SetTextModeration(svc service.TextModerationService) {
	s.textModeration = svc
}

// SetAdminNotifier sets the admin notifier after construction.
func (s *Service) SetAdminNotifier(n service.AdminNotifierService) {
	s.adminNotifier = n
}

type StartConversationInput struct {
	SenderID       uuid.UUID
	RecipientOrgID uuid.UUID
	Content        string
	Type           message.MessageType
	Metadata       json.RawMessage
}

// StartConversation opens (or reuses) a conversation between the sender
// and the Owner of the target organization. Under the Stripe Dashboard
// model all operators of the target org share the same inbox, so we
// always anchor the conversation to the Owner's user id — whichever
// operator is on call will answer from the shared thread.
func (s *Service) StartConversation(ctx context.Context, input StartConversationInput) (*message.Message, uuid.UUID, error) {
	org, err := s.orgs.FindByID(ctx, input.RecipientOrgID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("get recipient org: %w", err)
	}
	recipientUserID := org.OwnerUserID

	if input.SenderID == recipientUserID {
		return nil, uuid.UUID{}, message.ErrSelfConversation
	}

	input.Content = sanitize.StripHTML(input.Content)

	if _, err := s.users.GetByID(ctx, recipientUserID); err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("get recipient: %w", err)
	}

	allowed, err := s.rateLimiter.Allow(ctx, input.SenderID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("check rate limit: %w", err)
	}
	if !allowed {
		return nil, uuid.UUID{}, message.ErrRateLimitExceeded
	}

	convID, _, err := s.messages.FindOrCreateConversation(ctx, input.SenderID, recipientUserID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("find or create conversation: %w", err)
	}

	msg, err := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       input.SenderID,
		Content:        input.Content,
		Type:           input.Type,
		Metadata:       input.Metadata,
	})
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
	s.recordMediaIfNeeded(msg)
	s.moderateTextIfNeeded(msg)

	return msg, convID, nil
}

type SendMessageInput struct {
	SenderID       uuid.UUID
	ConversationID uuid.UUID
	Content        string
	Type           message.MessageType
	Metadata       json.RawMessage
	ReplyToID      *uuid.UUID
}

func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (*message.Message, error) {
	ok, err := s.messages.IsParticipant(ctx, input.ConversationID, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return nil, message.ErrNotParticipant
	}

	input.Content = sanitize.StripHTML(input.Content)

	allowed, err := s.rateLimiter.Allow(ctx, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("check rate limit: %w", err)
	}
	if !allowed {
		return nil, message.ErrRateLimitExceeded
	}

	msg, err := message.NewMessage(message.NewMessageInput{
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		Content:        input.Content,
		Type:           input.Type,
		Metadata:       input.Metadata,
		ReplyToID:      input.ReplyToID,
	})
	if err != nil {
		return nil, err
	}

	if err := s.messages.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Populate reply preview for the broadcast and HTTP response.
	if msg.ReplyToID != nil {
		s.populateReplyPreview(ctx, msg)
	}

	if err := s.messages.IncrementUnread(ctx, input.ConversationID, input.SenderID); err != nil {
		return nil, fmt.Errorf("increment unread: %w", err)
	}

	s.broadcastNewMessage(ctx, input.ConversationID, input.SenderID, msg)
	s.recordMediaIfNeeded(msg)
	s.moderateTextIfNeeded(msg)

	return msg, nil
}

// ListConversations returns conversations visible to the caller's
// organization. Every operator of the same org sees the same list
// (Stripe Dashboard shared workspace), but each carries their own
// personal unread counter from their participants row.
func (s *Service) ListConversations(ctx context.Context, orgID, userID uuid.UUID, cursorStr string, limit int) ([]repository.ConversationSummary, string, error) {
	params := repository.ListConversationsParams{
		OrganizationID: orgID,
		UserID:         userID,
		Cursor:         cursorStr,
		Limit:          limit,
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
	if time.Since(msg.CreatedAt) > time.Hour {
		return nil, message.ErrEditWindowExpired
	}

	if input.Content == "" {
		return nil, message.ErrEmptyContent
	}
	if len(input.Content) > message.MaxContentLength {
		return nil, message.ErrContentTooLong
	}

	// Save old content before editing
	_ = s.messages.SaveMessageHistory(ctx, msg.ID, input.UserID, msg.Content, "edited")

	msg.Edit(input.Content)

	if err := s.messages.UpdateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("update message: %w", err)
	}

	// Broadcast edit to other participants
	s.broadcastMessageEdited(ctx, msg)

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

	// 1-hour deletion window
	if time.Since(msg.CreatedAt) > time.Hour {
		return message.ErrDeleteWindowExpired
	}

	// Save content before deleting
	_ = s.messages.SaveMessageHistory(ctx, msg.ID, input.UserID, msg.Content, "deleted")

	msg.SoftDelete()

	if err := s.messages.UpdateMessage(ctx, msg); err != nil {
		return fmt.Errorf("update message: %w", err)
	}

	// Broadcast deletion to all participants
	s.broadcastMessageDeleted(ctx, msg)

	return nil
}
