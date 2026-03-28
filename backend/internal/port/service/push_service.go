package service

import "context"

// PushService sends push notifications to mobile devices via FCM or APNs.
type PushService interface {
	SendPush(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}
