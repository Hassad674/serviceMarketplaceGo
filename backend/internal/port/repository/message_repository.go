package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
)

type ListConversationsParams struct {
	// OrganizationID scopes the list to conversations where the caller's
	// org is a participant on at least one side (Stripe Dashboard shared
	// workspace). Required since phase R4.
	OrganizationID uuid.UUID
	// UserID is still needed to surface the calling operator's personal
	// unread counter. Each operator tracks their own "unread since I
	// last looked" state even inside a shared conversation.
	UserID uuid.UUID
	Cursor string
	Limit  int
}

type ListMessagesParams struct {
	ConversationID uuid.UUID
	Cursor         string
	Limit          int
}

// ConversationSummary is the enriched row returned by ListConversations.
// Since phase R4 it describes the *other organization* on the other side
// of the conversation (the Stripe Dashboard model: teams chat with teams).
// Online state still reflects any individual user from that org being
// currently connected.
//
// OtherUserID is kept alongside the org fields because proposals and
// calls still anchor on user ids in their own subsystems — the
// conversation's other participant is a stable user handle that lets
// those flows target the right row without a second round-trip.
type ConversationSummary struct {
	ConversationID uuid.UUID
	OtherUserID    uuid.UUID
	OtherOrgID     uuid.UUID
	OtherOrgName   string
	OtherOrgType   string
	OtherPhotoURL  string
	LastMessage    *string
	LastMessageAt  *time.Time
	LastMessageSeq int
	UnreadCount    int
	Online         bool
}

type MessageRepository interface {
	FindOrCreateConversation(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, bool, error)
	GetConversation(ctx context.Context, id uuid.UUID) (*message.Conversation, error)
	ListConversations(ctx context.Context, params ListConversationsParams) ([]ConversationSummary, string, error)
	// IsParticipant checks the direct-participant table — used by narrow
	// call sites (e.g. the conversation-org backfill) that need to know
	// whether a user is one of the two original endpoints. The messaging
	// authorization path has moved to IsOrgAuthorizedForConversation
	// since phase R11 (Stripe Dashboard shared-inbox model).
	IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
	// IsOrgAuthorizedForConversation returns true when the caller's
	// organization has at least one direct participant in the given
	// conversation. This is the primary authorization guard for all
	// messaging operations since phase R11 — it allows any operator
	// of the team to read, write, mark-read, etc. in a conversation
	// that was originally opened by a colleague.
	IsOrgAuthorizedForConversation(ctx context.Context, conversationID, orgID uuid.UUID) (bool, error)
	CreateMessage(ctx context.Context, msg *message.Message) error
	GetMessage(ctx context.Context, id uuid.UUID) (*message.Message, error)
	ListMessages(ctx context.Context, params ListMessagesParams) ([]*message.Message, string, error)
	GetMessagesSinceSeq(ctx context.Context, conversationID uuid.UUID, sinceSeq int, limit int) ([]*message.Message, error)
	// ListMessagesSinceTime returns messages of a conversation created at or
	// after the given timestamp, in chronological order. Used by features
	// that need a time-bounded slice of the history (e.g. dispute AI summary
	// limited to messages exchanged after the mission started).
	ListMessagesSinceTime(ctx context.Context, conversationID uuid.UUID, since time.Time, limit int) ([]*message.Message, error)
	UpdateMessage(ctx context.Context, msg *message.Message) error
	// IncrementUnreadForRecipients fans out a +1 unread bump to every
	// user that belongs to any organization participating in the
	// conversation EXCEPT the sender's own org. It upserts into
	// conversation_read_state so an operator that joined the team
	// after the conversation was opened still receives the bump on
	// their next poll, without needing a seed row anywhere.
	IncrementUnreadForRecipients(ctx context.Context, conversationID, senderUserID, senderOrgID uuid.UUID) error
	MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID, seq int) error
	GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error)
	GetTotalUnreadBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error)
	// GetParticipantIDs returns the user ids of the DIRECT participants
	// (the rows in conversation_participants). Kept for narrow callers
	// such as the conversation-org backfill and admin tooling. The
	// fan-out used by broadcasts and unread bumps has moved to
	// GetOrgMemberRecipients since R11.
	GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	// GetOrgMemberRecipients returns the user ids of every member of
	// every organization that has a direct participant in the given
	// conversation, excluding the given user (typically the sender).
	// Used by broadcasters (WS, push) and unread-count batches so
	// that all operators on both sides of a conversation get the
	// live events, not just the two original endpoints.
	GetOrgMemberRecipients(ctx context.Context, conversationID, excludeUserID uuid.UUID) ([]uuid.UUID, error)
	UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error
	MarkMessagesAsRead(ctx context.Context, conversationID, readerID uuid.UUID, upToSeq int) error
	GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	SaveMessageHistory(ctx context.Context, messageID, performedBy uuid.UUID, content, action string) error
	// UpdateMessageModeration removed in Phase 7 — moderation now lives
	// in moderation_results, accessed via ModerationResultsRepository.
}
