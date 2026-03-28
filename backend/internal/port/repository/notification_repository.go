package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/notification"
)

// NotificationRepository defines persistence operations for notifications.
type NotificationRepository interface {
	Create(ctx context.Context, n *notification.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*notification.Notification, error)
	List(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*notification.Notification, string, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	GetPreferences(ctx context.Context, userID uuid.UUID) ([]*notification.Preferences, error)
	UpsertPreference(ctx context.Context, pref *notification.Preferences) error
	CreateDeviceToken(ctx context.Context, dt *notification.DeviceToken) error
	ListDeviceTokens(ctx context.Context, userID uuid.UUID) ([]*notification.DeviceToken, error)
	DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error
}
