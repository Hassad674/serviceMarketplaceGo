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

	// AddFamilyMember records a JTI as a member of the family
	// identified by familyRootJTI. The auth service appends every
	// rotated token's JTI to the family set so a future replay
	// detection can iterate the set and blacklist the entire chain
	// in one shot. Empty familyRootJTI or empty jti are no-ops.
	//
	// The TTL is the family's absolute lifetime cap (MaxFamilyAge in
	// the auth service); each Add refreshes the set's expiry so the
	// most-recently-rotated member's lifetime drives garbage
	// collection. Once the cap is hit, the family is forced to
	// re-login anyway so we don't need it in memory.
	AddFamilyMember(ctx context.Context, familyRootJTI string, jti string, ttl time.Duration) error

	// FamilyMembers returns every JTI currently recorded for the
	// family. Used during reuse-detection to iterate descendants and
	// blacklist them all. Empty family or a missing key returns
	// (nil, nil). A Redis failure returns (nil, err) so the caller
	// can decide whether to proceed (we still bump session_version
	// as a fallback even when this fails).
	FamilyMembers(ctx context.Context, familyRootJTI string) ([]string, error)

	// DeleteFamily removes the family record entirely. Called after
	// reuse-detection has read out every member and added them to
	// the per-jti blacklist — the set itself is no longer useful.
	// A missing key is a no-op.
	DeleteFamily(ctx context.Context, familyRootJTI string) error
}
