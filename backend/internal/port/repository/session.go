package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/session"
)

// UserSessionRepository persists the audit trail of authentication
// sessions (B.4). Implementations live in adapter/postgres.
//
// The repository is small by design (ISP). Querying for active
// sessions is the only listing surface exposed; admin / forensic
// queries are deliberately NOT in this interface — they will live
// behind a separate AdminSessionRepository when the Sécurité page
// ships, so the regular auth path cannot accidentally read across
// users.
type UserSessionRepository interface {
	// Create inserts a new session row. Returns an error if the JTI
	// is already taken (unique constraint).
	Create(ctx context.Context, s *session.Session) error

	// FindByJTI returns the session attached to jti, or
	// session.ErrNotFound when no row matches. Used on every refresh
	// so the rotation chain can be linked.
	FindByJTI(ctx context.Context, jti string) (*session.Session, error)

	// Touch bumps last_used_at to now() for the row matching jti.
	// Idempotent — calling it on a revoked or already-touched row is
	// a no-op against the lifecycle.
	Touch(ctx context.Context, jti string) error

	// Revoke marks the session as revoked by setting revoked_at to
	// now(). Idempotent: revoking an already-revoked row keeps the
	// original timestamp.
	Revoke(ctx context.Context, jti string) error

	// RevokeAllForUser revokes every still-active session attached to
	// userID. Used on token-reuse detection (assume the user is
	// compromised) and on global-logout flows.
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error

	// ListActiveByUser returns every session for userID that is not
	// revoked and not yet expired, newest expiry first. Used by the
	// future Sécurité page; capped server-side at 50 rows.
	ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]*session.Session, error)
}
