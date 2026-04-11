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
type ConversationSummary struct {
	ConversationID uuid.UUID
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
	IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
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
	IncrementUnread(ctx context.Context, conversationID, senderID uuid.UUID) error
	MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID, seq int) error
	GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error)
	GetTotalUnreadBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error)
	GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error
	MarkMessagesAsRead(ctx context.Context, conversationID, readerID uuid.UUID, upToSeq int) error
	GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	SaveMessageHistory(ctx context.Context, messageID, performedBy uuid.UUID, content, action string) error
	UpdateMessageModeration(ctx context.Context, messageID uuid.UUID, status string, score float64, labelsJSON []byte) error
}
