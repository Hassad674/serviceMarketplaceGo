package expertise

import "errors"

// Sentinel errors surfaced by the expertise domain. These are the
// only error values the app layer ever compares against with
// errors.Is — everything else is either a transient infrastructure
// error (wrapped with fmt.Errorf by the caller) or an internal bug.
//
// Keeping them as package-level sentinels (not typed error structs)
// matches the convention already in place in domain/user, domain/profile,
// and domain/organization: the handler layer maps each sentinel to an
// HTTP status + code without ever inspecting error strings.
var (
	// ErrUnknownKey is returned when one of the submitted domain
	// keys is not part of the frozen catalog. The handler layer
	// maps this to HTTP 400 validation_error.
	ErrUnknownKey = errors.New("unknown expertise domain key")

	// ErrDuplicate is returned when the submitted list contains the
	// same key more than once. Duplicates are rejected rather than
	// silently deduplicated so the client notices the bug and can
	// fix its UI state.
	ErrDuplicate = errors.New("duplicate expertise domain key")

	// ErrOverMax is returned when the submitted list exceeds the
	// per-org-type maximum (agency=8, provider_personal=5).
	ErrOverMax = errors.New("too many expertise domains for this organization type")

	// ErrForbiddenOrgType is returned when the authenticated user's
	// organization type does not support the expertise feature at
	// all (currently: enterprise). The handler layer maps this to
	// HTTP 403 forbidden so the frontend can hide the section for
	// these org types.
	ErrForbiddenOrgType = errors.New("expertise feature is not available for this organization type")
)
