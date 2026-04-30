package repository

// Segregated reader / writer / broadcast-store interfaces for the
// messaging feature. Carved out of MessageRepository (21 methods).
//
// Three families:
//   - MessageReader  — conversation + message read paths (UI rendering,
//     authorization probes, history slicing).
//   - MessageWriter  — direct message persistence and read-state updates.
//   - MessageBroadcasterStore — the fan-out helpers used by the WS hub
//     and unread counters: list every operator on either side of a
//     conversation so a broadcast lands on every member of the team.
//
// The single postgres adapter implements ALL three because of structural
// typing. Wiring in cmd/api/main.go continues to pass the concrete
// adapter to consumers; only the *declared dependency type* on the
// consumer side narrows.

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
)

// MessageReader exposes read paths over conversations and messages.
type MessageReader interface {
	GetConversation(ctx context.Context, id uuid.UUID) (*message.Conversation, error)
	ListConversations(ctx context.Context, params ListConversationsParams) ([]ConversationSummary, string, error)
	IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
	IsOrgAuthorizedForConversation(ctx context.Context, conversationID, orgID uuid.UUID) (bool, error)
	GetMessage(ctx context.Context, id uuid.UUID) (*message.Message, error)
	ListMessages(ctx context.Context, params ListMessagesParams) ([]*message.Message, string, error)
	GetMessagesSinceSeq(ctx context.Context, conversationID uuid.UUID, sinceSeq int, limit int) ([]*message.Message, error)
	ListMessagesSinceTime(ctx context.Context, conversationID uuid.UUID, since time.Time, limit int) ([]*message.Message, error)
	GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error)
	GetTotalUnreadBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error)
	GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// MessageWriter exposes mutation paths: conversation creation, message
// CRUD, read-marker updates, and the moderation/history audit trail.
type MessageWriter interface {
	FindOrCreateConversation(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, bool, error)
	CreateMessage(ctx context.Context, msg *message.Message) error
	UpdateMessage(ctx context.Context, msg *message.Message) error
	MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID, seq int) error
	UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error
	MarkMessagesAsRead(ctx context.Context, conversationID, readerID uuid.UUID, upToSeq int) error
	SaveMessageHistory(ctx context.Context, messageID, performedBy uuid.UUID, content, action string) error
}

// MessageBroadcasterStore covers the org-fan-out helpers used by the
// WebSocket broadcaster, unread counters, and any other path that needs
// the full set of recipients to push a live event to.
type MessageBroadcasterStore interface {
	IncrementUnreadForRecipients(ctx context.Context, conversationID, senderUserID, senderOrgID uuid.UUID) error
	GetOrgMemberRecipients(ctx context.Context, conversationID, excludeUserID uuid.UUID) ([]uuid.UUID, error)
}

// Compile-time guarantee that the wide MessageRepository contract is
// always equivalent to the union of its segregated children.
var _ MessageRepository = (interface {
	MessageReader
	MessageWriter
	MessageBroadcasterStore
})(nil)
