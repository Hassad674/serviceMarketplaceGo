package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PasswordReset struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

type PasswordResetRepository interface {
	Create(ctx context.Context, pr *PasswordReset) error
	GetByToken(ctx context.Context, token string) (*PasswordReset, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
