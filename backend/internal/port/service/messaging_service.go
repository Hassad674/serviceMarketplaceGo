package service

import (
	"context"

	"github.com/google/uuid"
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
