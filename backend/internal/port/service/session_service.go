package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateSessionInput groups the fields required to create a new session.
// Mirror of AccessTokenInput so web (cookie session) and mobile (JWT)
// carry exactly the same context.
type CreateSessionInput struct {
	UserID  uuid.UUID
	Role    string
	IsAdmin bool

	// Organization context — nil / empty for Providers.
	OrganizationID *uuid.UUID
	OrgRole        string
}

// Session is the decoded content of a persisted session record.
type Session struct {
	ID        string
	UserID    uuid.UUID
	Role      string
	IsAdmin   bool
	CreatedAt time.Time

	// Organization context — nil / empty for solo users.
	OrganizationID *uuid.UUID
	OrgRole        string
}

type SessionService interface {
	Create(ctx context.Context, input CreateSessionInput) (*Session, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	CreateWSToken(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateWSToken(ctx context.Context, token string) (uuid.UUID, error)
}
