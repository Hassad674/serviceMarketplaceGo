package service

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
)

type PresenceService interface {
	SetOnline(ctx context.Context, userID uuid.UUID) error
	SetOffline(ctx context.Context, userID uuid.UUID) error
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
	BulkIsOnline(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type MessageBroadcaster interface {
	BroadcastNewMessage(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
	BroadcastTyping(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
	BroadcastStatusUpdate(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
	BroadcastUnreadCount(ctx context.Context, userID uuid.UUID, count int) error
	BroadcastPresence(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
}

type MessagingRateLimiter interface {
	Allow(ctx context.Context, userID uuid.UUID) (bool, error)
}

// MessagingQuerier defines the messaging operations needed by the WebSocket adapter.
// This allows the WS adapter to depend on an interface instead of importing app/messaging directly.
type MessagingQuerier interface {
	GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	GetMessagesSinceSeq(ctx context.Context, userID, conversationID uuid.UUID, sinceSeq int) ([]*message.Message, error)
	DeliverMessage(ctx context.Context, messageID, userID uuid.UUID) error
	GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
