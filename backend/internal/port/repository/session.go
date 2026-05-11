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
	// Sécurité page; capped server-side at 50 rows.
	ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]*session.Session, error)

	// UpdateGeoCity patches the city / country_code columns on the row
	// matching jti. Used by the fire-and-forget GeoIP goroutine that
	// runs AFTER session creation — the auth flow itself never blocks
	// on a third-party lookup (SEC-SESSIONS / migration 150).
	//
	// Idempotent and best-effort: a missing row is not an error from
	// the caller's POV. Empty strings overwrite to '' to keep the
	// schema invariant ("'' means unknown").
	UpdateGeoCity(ctx context.Context, jti string, city string, countryCode string) error

	// FindByID returns the session whose primary key equals id, or
	// session.ErrNotFound when no row matches. Used by the DELETE
	// /me/sessions/{id} endpoint to verify ownership before revoking.
	FindByID(ctx context.Context, id uuid.UUID) (*session.Session, error)

	// RevokeByID marks the row whose primary key equals id as revoked
	// by setting revoked_at to NOW(). Used by the Sécurité page's
	// "Révoquer" button — the caller has already enforced
	// session.UserID == requestingUserID before invoking this.
	RevokeByID(ctx context.Context, id uuid.UUID) error

	// RevokeAllForUserExceptJTI revokes every still-active session for
	// userID EXCEPT the one matching exceptJTI. Used by the Sécurité
	// page's "Tout révoquer sauf cette session" button. When exceptJTI
	// is empty the behaviour collapses to RevokeAllForUser.
	RevokeAllForUserExceptJTI(ctx context.Context, userID uuid.UUID, exceptJTI string) error
}
