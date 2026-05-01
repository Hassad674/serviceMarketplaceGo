package service

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// PublicProfileReader is the narrow read contract the public agency
// profile endpoint depends on. Defined here so the Redis cache
// decorator can satisfy the same interface as the app service —
// callers (the HTTP handler) never know whether they are talking to
// the cache or the underlying DB-backed service.
//
// The single GetProfile method returns the full profile.Profile
// aggregate as the agency endpoint already exposes today; the cache
// JSON-encodes the value so the wire format and the Redis blob
// stay in lockstep.
//
// Implementations MAY return profile.ErrProfileNotFound (or wrapped)
// when no row matches; the cache is responsible for absorbing the
// not-found signal via a short negative-TTL entry so 404 spam never
// degenerates into DB pressure.
type PublicProfileReader interface {
	GetProfile(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error)
}
