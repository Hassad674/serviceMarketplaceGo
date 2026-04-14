package referrerprofile

import "errors"

// Sentinel errors surfaced by the referrer_profile domain. The
// handler layer compares against these with errors.Is to map them
// to a stable HTTP status + error code.
//
// Availability-related errors live on the legacy profile package
// (profile.ErrInvalidAvailabilityStatus) because the enum itself is
// shared across both personas — frontends should handle the error
// value once rather than having per-persona sentinels that mean the
// same thing.
var (
	// ErrProfileNotFound is returned by the repository layer when a
	// GetByOrgID lookup misses AND the caller is using the strict
	// (non-lazy) path. HTTP 404. The app service's GetOrCreate path
	// catches this and creates a fresh row instead of surfacing it.
	ErrProfileNotFound = errors.New("referrer profile not found")
)
