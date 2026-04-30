package service

import (
	"context"
	"time"
)

// RefreshBlacklistService records refresh-token JTIs that must no longer
// be honored. Used by the auth service to enforce single-use rotation
// (SEC-06): every successful /auth/refresh call blacklists the JTI of
// the token that was just exchanged, and Logout blacklists the current
// refresh token immediately.
//
// The interface is intentionally minimal — Add + Has — so the
// implementation can be backed by Redis SET NX with a per-entry TTL,
// keeping memory bounded automatically as old entries expire.
//
// Both methods MUST treat an empty jti as a no-op + (false, nil) so
// callers do not need to defensively check before delegating; tokens
// without a JTI are extremely old and cannot be tracked anyway.
type RefreshBlacklistService interface {
	// Add stores the jti in the blacklist with the given TTL. The TTL
	// should match the original token's remaining time-to-expire so
	// the blacklist entry is no longer needed once the token would
	// have naturally expired. A negative or zero ttl is a no-op +
	// returns nil (the caller has nothing to gain from blacklisting a
	// token that is already expired).
	Add(ctx context.Context, jti string, ttl time.Duration) error

	// Has reports whether the jti has been blacklisted. Returns
	// (false, nil) for empty jti or a fresh token. Returns (true, nil)
	// when the jti was added before its TTL expired. A Redis failure
	// returns (false, err) — fail open is safer than fail closed for
	// this read path because the SessionVersion / token expiry checks
	// are still in place to catch a compromise.
	Has(ctx context.Context, jti string) (bool, error)
}
