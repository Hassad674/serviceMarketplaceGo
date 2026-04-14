package freelanceprofile

import "errors"

// Sentinel errors surfaced by the freelance_profile domain. The
// handler layer compares against these with errors.Is to map them
// to a stable HTTP status + error code. Never inspect the error
// string — extend this list when a new failure mode appears.
//
// Availability-related errors are NOT defined here: they live on
// the legacy profile package (ErrInvalidAvailabilityStatus) because
// the enum itself is shared and we want a single sentinel for both
// personas, so the frontend can handle the error by enum value once.
var (
	// ErrProfileNotFound is returned by the repository layer when a
	// GetByOrgID lookup misses. HTTP 404.
	ErrProfileNotFound = errors.New("freelance profile not found")
)
