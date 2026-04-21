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

	// ErrForbiddenOrgType is returned when an org-type-gated write is
	// attempted by an org of the wrong type (e.g. a provider_personal
	// org trying to edit its client profile — v1 exposes the client
	// profile only to agency and enterprise orgs). HTTP 403.
	ErrForbiddenOrgType = errors.New("feature is not available for this organization type")

	// ErrClientDescriptionTooLong signals a client_description payload
	// that exceeds the domain's max length. HTTP 400.
	ErrClientDescriptionTooLong = errors.New("client description exceeds maximum length")
)

// MaxClientDescriptionLength is the cap enforced on the
// client_description field. Mirrors the informal bio/about limits
// used elsewhere in the profile feature — the number is deliberately
// generous so the UI never has to truncate typical copy, but bounded
// so a payload abuse cannot inflate the row indefinitely.
const MaxClientDescriptionLength = 2000
