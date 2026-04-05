package embedded

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	notifdomain "marketplace-backend/internal/domain/notification"
	portservice "marketplace-backend/internal/port/service"
)

// notificationSenderAdapter wraps the platform's NotificationSender to
// satisfy the local NotificationSink interface. Keeps the Notifier
// decoupled from the notification module shape.
type notificationSenderAdapter struct {
	sender portservice.NotificationSender
}

// NewNotificationSenderAdapter wires the global NotificationSender into a
// NotificationSink usable by the Notifier.
func NewNotificationSenderAdapter(sender portservice.NotificationSender) NotificationSink {
	return &notificationSenderAdapter{sender: sender}
}

func (a *notificationSenderAdapter) Send(
	ctx context.Context,
	userID uuid.UUID,
	t notifdomain.NotificationType,
	title, body string,
	metadata map[string]any,
) error {
	var raw json.RawMessage
	if len(metadata) > 0 {
		if b, err := json.Marshal(metadata); err == nil {
			raw = b
		}
	}
	return a.sender.Send(ctx, portservice.NotificationInput{
		UserID: userID,
		Type:   string(t),
		Title:  title,
		Body:   body,
		Data:   raw,
	})
}
