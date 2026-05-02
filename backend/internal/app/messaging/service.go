package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	appmoderation "marketplace-backend/internal/app/moderation"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/sanitize"
)

type Service struct {
	// messages stays on the wide MessageRepository — the messaging
	// service straddles all three segregated children (Reader for the
	// many lookup paths, Writer for create / update / mark-read,
	// BroadcasterStore for the WS fan-out helpers). Composing locally
	// would reproduce the wide port verbatim.
	messages repository.MessageRepository
	users    repository.UserRepository
	// orgs is narrowed to OrganizationReader — messaging only ever
	// resolves the recipient org by id (FindByID).
	orgs                   repository.OrganizationReader
	orgMembers             repository.OrganizationMemberRepository
	presence               service.PresenceService
	broadcaster            service.MessageBroadcaster
	storage                service.StorageService
	rateLimiter            service.MessagingRateLimiter
	mediaRecorder          service.MediaRecorder
	moderationOrchestrator *appmoderation.Service
}

type ServiceDeps struct {
	Messages      repository.MessageRepository
	Users         repository.UserRepository
	Organizations repository.OrganizationReader
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

// SetModerationOrchestrator wires the central moderation pipeline.
// Optional: when nil, automated text moderation is disabled (the
// messaging service still works, just without flagging/hiding).
//
// This single setter replaces the legacy SetTextModeration +
// SetAdminNotifier + SetAuditRepo trio — the orchestrator now owns
// each of those collaborators internally so messaging only needs to
// know about ONE moderation entry point.
func (s *Service) SetModerationOrchestrator(svc *appmoderation.Service) {
	s.moderationOrchestrator = svc
}

type StartConversationInput struct {
	SenderID       uuid.UUID
	SenderOrgID    uuid.UUID
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

	// Resolve the sender's organization if the handler did not pass
	// one explicitly. The fan-out below needs to exclude the sender's
	// entire team, not just the sender themselves.
	senderOrgID := input.SenderOrgID
	if senderOrgID == uuid.Nil {
		senderOrgID, err = s.resolveUserOrgID(ctx, input.SenderID)
		if err != nil {
			return nil, uuid.UUID{}, fmt.Errorf("resolve sender org: %w", err)
		}
	}

	allowed, err := s.rateLimiter.Allow(ctx, input.SenderID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("check rate limit: %w", err)
	}
	if !allowed {
		return nil, uuid.UUID{}, message.ErrRateLimitExceeded
	}

	convID, _, err := s.messages.FindOrCreateConversation(ctx, input.SenderID, recipientUserID, senderOrgID, input.SenderID)
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

	if err := s.messages.CreateMessage(ctx, msg, senderOrgID, input.SenderID); err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("create message: %w", err)
	}

	if err := s.messages.IncrementUnreadForRecipients(ctx, convID, input.SenderID, senderOrgID); err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("increment unread: %w", err)
	}

	s.broadcastNewMessage(ctx, convID, input.SenderID, msg)
	s.recordMediaIfNeeded(msg)
	s.moderateTextIfNeeded(msg)

	return msg, convID, nil
}

type SendMessageInput struct {
	SenderID       uuid.UUID
	SenderOrgID    uuid.UUID
	ConversationID uuid.UUID
	Content        string
	Type           message.MessageType
	Metadata       json.RawMessage
	ReplyToID      *uuid.UUID
}

func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (*message.Message, error) {
	// Resolve the sender's organization if the caller did not pass one
	// explicitly (WS / legacy paths). The fan-out below needs the org
	// id to know which team to exclude from the +1 unread bump.
	senderOrgID := input.SenderOrgID
	if senderOrgID == uuid.Nil {
		resolved, err := s.resolveUserOrgID(ctx, input.SenderID)
		if err != nil {
			return nil, fmt.Errorf("resolve sender org: %w", err)
		}
		senderOrgID = resolved
	}

	ok, err := s.messages.IsOrgAuthorizedForConversation(ctx, input.ConversationID, senderOrgID)
	if err != nil {
		return nil, fmt.Errorf("check org authorized: %w", err)
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

	if err := s.messages.CreateMessage(ctx, msg, senderOrgID, input.SenderID); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Best-effort: bump the sender's last_active_at so the
	// Typesense indexer can rank recently-active profiles higher.
	// Failure must never block message delivery.
	if err := s.users.TouchLastActive(ctx, input.SenderID); err != nil {
		slog.Warn("messaging: touch last_active_at on message send failed",
			"user_id", input.SenderID, "error", err)
	}

	// Populate reply preview for the broadcast and HTTP response.
	if msg.ReplyToID != nil {
		s.populateReplyPreview(ctx, msg)
	}

	if err := s.messages.IncrementUnreadForRecipients(ctx, input.ConversationID, input.SenderID, senderOrgID); err != nil {
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

// ListMessages reads the message history of a conversation. Authorization
// is org-scoped since phase R11: any operator of an organization that has
// a participant in the conversation can read it, not only the original
// two endpoints.
func (s *Service) ListMessages(ctx context.Context, orgID, userID, conversationID uuid.UUID, cursorStr string, limit int) ([]*message.Message, string, error) {
	if err := s.requireOrgAuthorized(ctx, conversationID, orgID, userID); err != nil {
		return nil, "", err
	}

	params := repository.ListMessagesParams{
		ConversationID: conversationID,
		Cursor:         cursorStr,
		Limit:          limit,
		// BUG-NEW-04 path 8/8: thread the caller's tenant context to
		// the repo so the SELECT runs inside RunInTxWithTenant under
		// prod NOSUPERUSER NOBYPASSRLS.
		CallerOrgID:  orgID,
		CallerUserID: userID,
	}

	return s.messages.ListMessages(ctx, params)
}

// GetMessagesSinceSeq returns the messages created after a given seq,
// used by the WebSocket adapter for re-sync on reconnect. Since phase
// R11 the authorization check is org-scoped — operators who joined
// the team after the conversation was opened can re-sync too.
func (s *Service) GetMessagesSinceSeq(ctx context.Context, userID, conversationID uuid.UUID, sinceSeq int) ([]*message.Message, error) {
	orgID, err := s.resolveUserOrgID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user org: %w", err)
	}
	if err := s.requireOrgAuthorized(ctx, conversationID, orgID, userID); err != nil {
		return nil, err
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
