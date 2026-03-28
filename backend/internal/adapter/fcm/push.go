package fcm

import (
	"context"
	"fmt"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// PushService sends push notifications via Firebase Cloud Messaging.
type PushService struct {
	client *messaging.Client
}

// NewPushService creates a new FCM push service from the credentials file path.
func NewPushService(credentialsPath string) (*PushService, error) {
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("init firebase messaging: %w", err)
	}

	return &PushService{client: client}, nil
}

// SendPush sends a push notification to the given device tokens.
func (s *PushService) SendPush(
	ctx context.Context,
	tokens []string,
	title, body string,
	data map[string]string,
) error {
	if len(tokens) == 0 {
		return nil
	}

	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				ChannelID:   "marketplace_notifications",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound:            "default",
					MutableContent:   true,
					ContentAvailable: true,
				},
			},
		},
	}

	response, err := s.client.SendEachForMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("send multicast: %w", err)
	}

	if response.FailureCount > 0 {
		for i, resp := range response.Responses {
			if resp.Error != nil {
				slog.Warn("fcm send failed for token",
					"token_index", i,
					"error", resp.Error,
				)
			}
		}
	}

	slog.Debug("fcm push sent",
		"success", response.SuccessCount,
		"failure", response.FailureCount,
	)

	return nil
}
