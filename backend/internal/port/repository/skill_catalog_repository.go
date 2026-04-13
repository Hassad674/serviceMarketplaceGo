package repository

import (
	"context"

	domainskill "marketplace-backend/internal/domain/skill"
)

// SkillCatalogRepository is the persistence contract for the hybrid
// skills catalog (curated admin seed entries + user-created entries).
// Implementations live in adapter/ and must not leak SQL or driver-
// specific types across this interface boundary.
//
// Normalization is the caller's responsibility: every SkillText passed
// to these methods is expected to already be canonical (lowercase,
// trimmed, collapsed spaces) as produced by domainskill.NormalizeSkillText.
type SkillCatalogRepository interface {
	// Upsert inserts a new catalog entry or updates an existing one
	// by skill_text. The seed loader uses this to ingest the curated
	// list; the user-initiated skill creation path uses it when a
	// freelancer types a skill not yet in the catalog.
	//
	// Implementations MUST preserve usage_count on update — Upsert
	// never resets the counter to the one on the incoming entry.
	Upsert(ctx context.Context, entry *domainskill.CatalogEntry) error

	// FindByText returns the catalog entry for a normalized skill
	// text. Returns (nil, nil) when no row matches — callers must
	// nil-check rather than relying on a sentinel error, matching the
	// convention established by the other repository interfaces in
	// this package.
	FindByText(ctx context.Context, skillText string) (*domainskill.CatalogEntry, error)

	// ListCuratedByExpertise returns curated entries whose
	// expertise_keys contain the given key, sorted by usage_count
	// DESC. This powers the "browse by expertise" panel in the
	// skills picker. The limit is chosen by the caller (typical: 50)
	// to cap the response size.
	ListCuratedByExpertise(ctx context.Context, expertiseKey string, limit int) ([]*domainskill.CatalogEntry, error)

	// CountCuratedByExpertise returns how many curated entries are
	// tagged with the given expertise key. Used by the panel header
	// to show a counter ("142 skills in Development") without having
	// to over-fetch the whole list.
	CountCuratedByExpertise(ctx context.Context, expertiseKey string) (int, error)

	// SearchAutocomplete returns catalog entries matching the query
	// prefix or trigram-similarity (curated first, then by usage
	// count). The incoming q string is expected to already be
	// normalized by the app layer via domainskill.NormalizeSkillText.
	// Implementations MUST respect limit to avoid unbounded result
	// sets — typical callers pass 20.
	SearchAutocomplete(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error)

	// IncrementUsageCount bumps the counter for a single skill by 1.
	// Called by the profile skill service immediately after a skill
	// is attached to an organization. Must be idempotent on a row-
	// not-found basis: if the referenced skill has just been deleted
	// (race) the implementation should return without error, since
	// FK constraints would have failed earlier in the caller's flow.
	IncrementUsageCount(ctx context.Context, skillText string) error

	// DecrementUsageCount decrements the counter for a single skill
	// by 1, clamped at zero. Called when a skill is removed from an
	// organization's profile. Must never produce a negative count,
	// even if the caller races and decrements more times than
	// IncrementUsageCount was called.
	DecrementUsageCount(ctx context.Context, skillText string) error
}
