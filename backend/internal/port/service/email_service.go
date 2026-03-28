package service

import "context"

type EmailService interface {
	SendPasswordReset(ctx context.Context, to string, resetURL string) error
	SendNotification(ctx context.Context, to, subject, html string) error
}
