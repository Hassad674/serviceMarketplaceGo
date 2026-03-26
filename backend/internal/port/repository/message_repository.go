package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
)

type ListConversationsParams struct {
	UserID uuid.UUID
	Cursor string
	Limit  int
}

type ListMessagesParams struct {
	ConversationID uuid.UUID
	Cursor         string
	Limit          int
}

type ConversationSummary struct {
	ConversationID uuid.UUID
	OtherUserID    uuid.UUID
	OtherUserName  string
	OtherUserRole  string
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
	UpdateMessage(ctx context.Context, msg *message.Message) error
	IncrementUnread(ctx context.Context, conversationID, senderID uuid.UUID) error
	MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID, seq int) error
	GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error)
	GetTotalUnreadBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error)
	GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error
	MarkMessagesAsRead(ctx context.Context, conversationID, readerID uuid.UUID, upToSeq int) error
	GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
