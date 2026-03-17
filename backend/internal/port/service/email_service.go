package service

import "context"

type EmailService interface {
	SendPasswordReset(ctx context.Context, to string, resetURL string) error
}
