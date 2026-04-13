// Package skill owns the hybrid catalog of marketplace skills and the
// per-organization skill attachments exposed on public profiles.
//
// Unlike the frozen expertise catalog (see domain/expertise), the skill
// catalog is open-ended: admin-seeded "curated" entries drive the
// browse-by-expertise panels, and user-created entries populate the
// long tail via autocomplete. Both kinds coexist in a single table
// and are distinguished by CatalogEntry.IsCurated.
//
// Design note — zero cross-domain imports: every reference to an
// expertise key is a plain string, and every reference to an
// organization type is a string alias (see limits.go). Keeping this
// package free of other domain imports preserves the hexagonal
// invariant that domain modules are independent and removable: the
// only shared dependency for the skills feature is the organizations
// TABLE, referenced by the migration — never the Go package.
package skill

import (
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// CatalogEntry represents one row of the skills catalog.
//
// SkillText is the canonical primary key used by every query and FK —
// always lowercase, trimmed, and with internal whitespace collapsed.
// DisplayText preserves the casing originally submitted by the user so
// the UI can render "Next.js" rather than "next.js".
//
// ExpertiseKeys references the frozen expertise catalog. Validation
// against the expertise domain is intentionally left to the caller:
// importing the expertise package here would break the feature
// independence rule. In practice the app/skill service performs that
// check via expertise.IsValidKey before calling NewCatalogEntry.
//
// IsCurated distinguishes admin seed entries (true — surfaced by the
// browse panels) from user contributions (false — only visible through
// autocomplete until, possibly, promoted manually).
//
// UsageCount is a denormalized cache of how many profiles currently
// attach this skill. The repository is responsible for keeping it in
// sync via IncrementUsageCount / DecrementUsageCount.
type CatalogEntry struct {
	SkillText     string
	DisplayText   string
	ExpertiseKeys []string
	IsCurated     bool
	UsageCount    int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProfileSkill represents one skill attached to an organization's
// public profile, with display order preserved in Position. Positions
// are 0-indexed and expected to be contiguous from zero; the app
// service reassigns them on every ReplaceForOrg call.
//
// DisplayText is a read-path convenience: the canonical display
// casing lives in skills_catalog, but the postgres adapter joins
// the two tables on List so downstream DTOs can render the
// preserved casing ("React", "Next.js") without a second lookup.
// Write paths (ReplaceForOrg) leave it empty — the join happens on
// read only, and callers must never rely on DisplayText when
// constructing a ProfileSkill for persistence.
type ProfileSkill struct {
	OrganizationID uuid.UUID
	SkillText      string
	DisplayText    string
	Position       int
	CreatedAt      time.Time
}

// NormalizeSkillText transforms a raw user input into the canonical
// form used as primary key in skills_catalog. It applies, in order:
//
//  1. Strip leading / trailing whitespace (Unicode-aware).
//  2. Lowercase (Unicode-aware).
//  3. Collapse any run of internal whitespace to a single ASCII space.
//
// The function is deliberately total: it never returns an error. An
// empty or whitespace-only input normalizes to the empty string, and
// the caller (typically NewCatalogEntry) rejects it explicitly with
// ErrInvalidSkillText. Keeping normalization side-effect-free means it
// can also be used at the read path (autocomplete query normalization)
// without worrying about validation semantics.
//
// Examples:
//
//	"React"       -> "react"
//	" React JS "  -> "react js"
//	"Next.js"     -> "next.js"
//	"  REACT  "   -> "react"
//	"A  B"        -> "a b"
//	""            -> ""
//	"\t \n"       -> ""
func NormalizeSkillText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	lowered := strings.ToLower(trimmed)
	return collapseWhitespace(lowered)
}

// collapseWhitespace rewrites any run of Unicode whitespace characters
// inside s to a single ASCII space. Leading and trailing whitespace is
// assumed to already be trimmed by the caller (NormalizeSkillText).
func collapseWhitespace(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inSpace {
				builder.WriteByte(' ')
				inSpace = true
			}
			continue
		}
		builder.WriteRune(r)
		inSpace = false
	}
	return builder.String()
}

// NewCatalogEntry constructs a validated catalog entry from raw inputs.
//
// The function:
//
//   - Normalizes rawText via NormalizeSkillText. An empty result yields
//     ErrInvalidSkillText.
//   - Trims displayText and rejects empty / whitespace-only values with
//     ErrInvalidDisplayText.
//   - Deduplicates expertiseKeys preserving the first occurrence order,
//     so callers can pass a list straight from a form submission without
//     worrying about accidental duplicates from double-clicks.
//   - Leaves expertise key validation to the caller. Importing the
//     expertise package here would violate the feature-independence
//     rule; the app/skill service calls expertise.IsValidKey on each
//     key before invoking this constructor.
//
// The returned entry has zero timestamps — the repository layer fills
// CreatedAt / UpdatedAt on insert using the DB defaults.
func NewCatalogEntry(rawText, displayText string, expertiseKeys []string, curated bool) (*CatalogEntry, error) {
	normalized := NormalizeSkillText(rawText)
	if normalized == "" {
		return nil, ErrInvalidSkillText
	}
	display := strings.TrimSpace(displayText)
	if display == "" {
		return nil, ErrInvalidDisplayText
	}
	return &CatalogEntry{
		SkillText:     normalized,
		DisplayText:   display,
		ExpertiseKeys: dedupePreserveOrder(expertiseKeys),
		IsCurated:     curated,
		UsageCount:    0,
	}, nil
}

// dedupePreserveOrder returns a copy of in with duplicates removed,
// keeping the first occurrence of each value. A nil or empty input
// yields an empty (non-nil) slice so the DB adapter can marshal it
// directly to an empty TEXT[] without a nil check.
func dedupePreserveOrder(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// IsInExpertise reports whether this entry is tagged with the given
// expertise key. The comparison is an exact string match — callers are
// expected to pass a canonical key from the expertise catalog.
func (e *CatalogEntry) IsInExpertise(key string) bool {
	for _, k := range e.ExpertiseKeys {
		if k == key {
			return true
		}
	}
	return false
}
