package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// NotificationSender is the cross-feature interface for dispatching notifications.
// Features inject this interface without knowing the notification module.
type NotificationSender interface {
	Send(ctx context.Context, input NotificationInput) error
}

// NotificationInput contains everything needed to create and dispatch a notification.
type NotificationInput struct {
	UserID uuid.UUID
	Type   string
	Title  string
	Body   string
	Data   json.RawMessage
}
