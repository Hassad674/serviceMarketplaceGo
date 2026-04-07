package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AdminConversation represents a conversation for admin moderation view.
type AdminConversation struct {
	ID                 uuid.UUID
	Participants       []ConversationParticipant
	MessageCount       int
	LastMessage        *string
	LastMessageAt      *time.Time
	CreatedAt          time.Time
	PendingReportCount int
	ReportedMessage    *string
}

// ConversationParticipant is a lightweight user representation for conversation listing.
type ConversationParticipant struct {
	ID          uuid.UUID
	DisplayName string
	Email       string
	Role        string
}

// AdminMessage represents a message for admin moderation view.
type AdminMessage struct {
	ID               uuid.UUID
	ConversationID   uuid.UUID
	SenderID         uuid.UUID
	SenderName       string
	SenderRole       string
	Content          string
	Type             string
	Metadata         json.RawMessage
	ReplyToID        *uuid.UUID
	ModerationStatus string
	ModerationScore  float64
	ModerationLabels json.RawMessage
	CreatedAt        time.Time
}

// AdminConversationFilters groups query parameters for admin conversation listing.
type AdminConversationFilters struct {
	Cursor string
	Limit  int
	Page   int
	Sort   string
	Filter string
}

// AdminConversationRepository defines persistence operations for admin conversation queries.
type AdminConversationRepository interface {
	// List returns conversations with pagination, sorting, and filtering.
	List(ctx context.Context, filters AdminConversationFilters) ([]AdminConversation, string, int, error)

	// GetByID returns a single conversation with participants and stats.
	GetByID(ctx context.Context, conversationID uuid.UUID) (*AdminConversation, error)

	// ListMessages returns messages for a conversation with cursor pagination.
	ListMessages(ctx context.Context, conversationID uuid.UUID, cursor string, limit int) ([]AdminMessage, string, error)

	// UpdateMessageModeration updates the moderation status/score/labels on a message.
	UpdateMessageModeration(ctx context.Context, messageID uuid.UUID, status string, score float64, labelsJSON []byte) error

	// HideMessage sets message status to 'hidden' so it is no longer visible to users.
	HideMessage(ctx context.Context, messageID uuid.UUID) error
}
