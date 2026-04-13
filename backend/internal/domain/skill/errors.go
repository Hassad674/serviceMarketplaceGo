package skill

import "errors"

// Sentinel errors surfaced by the skill domain. These are the only
// error values the app layer ever compares against with errors.Is —
// everything else is either a transient infrastructure error (wrapped
// with fmt.Errorf by the caller) or an internal bug.
//
// Keeping them as package-level sentinels (not typed error structs)
// matches the convention already in place in domain/expertise,
// domain/user, and domain/profile: the handler layer maps each
// sentinel to an HTTP status + code without ever inspecting error
// strings.
var (
	// ErrInvalidSkillText is returned when the raw skill text, after
	// normalization, is empty. Indicates the caller submitted an empty
	// or whitespace-only value and must be mapped to HTTP 400.
	ErrInvalidSkillText = errors.New("invalid skill text")

	// ErrInvalidDisplayText is returned when the user-visible display
	// text, after trimming, is empty. Also HTTP 400.
	ErrInvalidDisplayText = errors.New("invalid skill display text")

	// ErrEmptySkill is returned when an operation requires at least
	// one skill but received an empty slice (for example when the
	// service rejects a zero-skill payload on a non-destructive
	// endpoint). HTTP 400.
	ErrEmptySkill = errors.New("empty skill payload")

	// ErrSkillNotFound is returned by the repository layer when a
	// lookup by skill_text finds no row. HTTP 404.
	ErrSkillNotFound = errors.New("skill not found")

	// ErrDuplicateSkill is returned when the submitted list contains
	// the same skill more than once. Duplicates are rejected rather
	// than silently deduplicated so the client notices the bug and
	// can fix its UI state. HTTP 400.
	ErrDuplicateSkill = errors.New("duplicate skill")

	// ErrInvalidExpertiseKey is returned when one of the expertise
	// keys attached to a catalog entry is not part of the frozen
	// expertise catalog. HTTP 400.
	//
	// The domain/skill package itself does NOT perform this check
	// (importing the expertise package would violate feature
	// independence). The app/skill service validates each key against
	// expertise.IsValidKey before insert and surfaces this sentinel.
	ErrInvalidExpertiseKey = errors.New("invalid expertise key for skill")

	// ErrTooManySkills is returned when the submitted list exceeds the
	// per-org-type maximum (agency=40, provider_personal=25). HTTP 400.
	ErrTooManySkills = errors.New("too many skills for this organization type")

	// ErrSkillsDisabledForOrgType is returned when the authenticated
	// user's organization type does not support the skills feature at
	// all (currently: enterprise). The handler layer maps this to
	// HTTP 403 forbidden so the frontend can hide the section for
	// these org types.
	ErrSkillsDisabledForOrgType = errors.New("skills feature is not available for this organization type")
)
