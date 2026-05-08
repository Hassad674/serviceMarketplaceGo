package main

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/handler/middleware"
)

// userStateAdapter bridges the concrete postgres.UserRepository to
// the narrow middleware.UserStateChecker contract so the auth
// middleware can read the LIVE (is_admin, status) pair on every
// authenticated request — without importing the full user repository
// surface.
//
// Why this matters: the snapshot of is_admin baked into the session
// cookie / JWT at login time never refreshes. A direct
// `UPDATE users SET is_admin = true` issued from operator tooling
// would not propagate to in-flight sessions until each user logs
// out and back in. The auth middleware now consults this adapter
// (fronted by a Redis cache, see redis.NewCachedUserStateChecker)
// and overrides the snapshot with the authoritative DB value.
//
// The adapter does NOT add caching — the cache lives in the redis
// adapter that decorates this one. Keeping responsibilities split
// makes it easy to swap the cache out (or disable it for tests)
// without touching the postgres path.
type userStateAdapter struct {
	repo *postgres.UserRepository
}

// GetUserState satisfies middleware.UserStateChecker.
func (a userStateAdapter) GetUserState(
	ctx context.Context,
	userID uuid.UUID,
) (middleware.UserState, error) {
	isAdmin, status, err := a.repo.GetUserAuthState(ctx, userID)
	if err != nil {
		return middleware.UserState{}, err
	}
	return middleware.UserState{
		IsAdmin: isAdmin,
		Status:  status,
	}, nil
}
