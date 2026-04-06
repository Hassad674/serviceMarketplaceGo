package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        string
	UserID    uuid.UUID
	Role      string
	IsAdmin   bool
	CreatedAt time.Time
}

type SessionService interface {
	Create(ctx context.Context, userID uuid.UUID, role string, isAdmin bool) (*Session, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	CreateWSToken(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateWSToken(ctx context.Context, token string) (uuid.UUID, error)
}
