package profile

import "errors"

// Sentinel errors surfaced by the profile domain. The handler layer
// compares against these with errors.Is to map them to a stable HTTP
// status + error code. Never inspect the error string — extend this
// list when a new failure mode appears.
var (
	// ErrProfileNotFound is returned by the repository layer when a
	// GetByOrganizationID lookup misses. HTTP 404.
	ErrProfileNotFound = errors.New("profile not found")

	// ErrInvalidCountryCode signals a country_code that is not a
	// two-letter uppercase ISO 3166-1 alpha-2 value (empty is
	// accepted — see ValidateCountryCode). HTTP 400.
	ErrInvalidCountryCode = errors.New("invalid country code")

	// ErrInvalidAvailabilityStatus signals an availability enum
	// outside the closed set {available_now, available_soon,
	// not_available}. HTTP 400.
	ErrInvalidAvailabilityStatus = errors.New("invalid availability status")
)
