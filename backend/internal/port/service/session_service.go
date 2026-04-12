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

	// Permissions is the list of effective permission keys for the
	// user's org membership, with the org's role overrides already
	// applied. Stored in the session so the RequirePermission
	// middleware can honor customized permissions without an extra
	// database round-trip on every request. Empty when the user has
	// no org.
	Permissions []string

	// SessionVersion mirrors the one in AccessTokenInput. Copied from
	// users.session_version at login so the cookie session stays in
	// sync with the JWT issued for mobile clients.
	SessionVersion int
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

	// Permissions mirrors CreateSessionInput.Permissions. The auth
	// middleware writes this into request context under
	// ContextKeyPermissions so the RequirePermission middleware can
	// consult the customized set first.
	Permissions []string

	// SessionVersion at the time the session was created. The auth
	// middleware compares this against the current value in the DB
	// and rejects stale sessions the same way it handles stale JWTs.
	SessionVersion int
}

type SessionService interface {
	Create(ctx context.Context, input CreateSessionInput) (*Session, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	CreateWSToken(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateWSToken(ctx context.Context, token string) (uuid.UUID, error)
}
