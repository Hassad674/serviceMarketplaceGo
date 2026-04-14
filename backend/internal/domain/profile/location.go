package profile

// Location-related value validators and normalizers. Lives in a
// dedicated file (rather than entity.go) to keep entity.go purely
// descriptive of the Profile struct while the validation logic —
// which grows as more enumerations are added — stays isolated and
// easy to test.

// Work-mode enum values. These are the only accepted entries in
// Profile.WorkMode, which is itself a slice because an organization
// may declare any combination (remote + hybrid, or the three of them,
// etc). The domain keeps the catalog frozen: adding a new mode is a
// code change that must update this file plus the frontend i18n.
const (
	WorkModeRemote = "remote"
	WorkModeOnSite = "on_site"
	WorkModeHybrid = "hybrid"
)

// allowedWorkModes is the O(1) lookup set backing IsValidWorkMode.
// Built as a var to avoid leaking a mutable slice — callers never
// see this map.
var allowedWorkModes = map[string]struct{}{
	WorkModeRemote: {},
	WorkModeOnSite: {},
	WorkModeHybrid: {},
}

// IsValidWorkMode reports whether mode is one of the three frozen
// catalog values. Empty strings and unknown values return false.
func IsValidWorkMode(mode string) bool {
	_, ok := allowedWorkModes[mode]
	return ok
}

// NormalizeWorkModes deduplicates and filters invalid entries from
// in, preserving the order of first occurrence. A nil or empty input
// yields an empty (non-nil) slice so the DB adapter can marshal it
// straight to an empty TEXT[] without a nil check.
//
// The function is deliberately total: it never returns an error.
// Invalid entries are silently dropped rather than surfaced because
// the frontend is the source of the catalog labels — receiving an
// unknown key here is a bug in the caller, not a user error.
func NormalizeWorkModes(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, m := range in {
		if !IsValidWorkMode(m) {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

// ValidateCountryCode returns nil for the empty string (unset) or a
// two-letter uppercase ASCII code (ISO 3166-1 alpha-2). Any other
// shape yields ErrInvalidCountryCode so the handler can surface a
// 400 to the client.
//
// The service layer is expected to uppercase and trim the input
// before calling this — we check the canonical form here rather than
// normalizing in place because the signature-free "bool" alternative
// would force every caller to remember two steps.
func ValidateCountryCode(code string) error {
	if code == "" {
		return nil
	}
	if len(code) != 2 {
		return ErrInvalidCountryCode
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return ErrInvalidCountryCode
		}
	}
	return nil
}

// IsValidLanguageCode reports whether code is a plausible ISO 639-1
// two-letter lowercase identifier ("fr", "en", "es", ...). The check
// is intentionally lenient: we do not maintain a canonical whitelist
// of every living language because the frontend already curates the
// selectable set via its i18n registry. The domain only rejects
// obviously-malformed values (wrong length, non-alphabetic characters,
// uppercase letters) so a user cannot slip "fr-FR" or "English" or
// "FR" into the database.
func IsValidLanguageCode(code string) bool {
	if len(code) != 2 {
		return false
	}
	for _, r := range code {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// NormalizeLanguageCodes filters invalid entries from in and
// deduplicates, preserving first-occurrence order. Unlike the country
// code, we do NOT accept mixed case — the frontend is expected to
// emit lowercase codes, and anything else is dropped defensively.
// An empty input returns an empty (non-nil) slice.
func NormalizeLanguageCodes(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, c := range in {
		if !IsValidLanguageCode(c) {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}
