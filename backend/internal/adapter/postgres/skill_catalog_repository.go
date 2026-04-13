package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	domainskill "marketplace-backend/internal/domain/skill"
)

// SkillCatalogRepository is the PostgreSQL-backed implementation of
// repository.SkillCatalogRepository. It owns exactly one table,
// skills_catalog (migration 081), and never reads from or writes to
// profile_skills — that relation belongs to ProfileSkillRepository.
//
// The catalog table is append-and-update only: rows are never deleted
// from the catalog once inserted. The repository therefore exposes no
// Delete method. Cleanup of unused user-created rows would be a future
// job, not a user-facing operation.
type SkillCatalogRepository struct {
	db *sql.DB
}

// NewSkillCatalogRepository returns a repository ready to talk to the
// given *sql.DB. Like every other postgres adapter in this package,
// the constructor is intentionally tiny — any tuning of the DB handle
// (SetMaxOpenConns, SetMaxIdleConns, …) is the caller's responsibility.
func NewSkillCatalogRepository(db *sql.DB) *SkillCatalogRepository {
	return &SkillCatalogRepository{db: db}
}

// maxCuratedListLimit caps ListCuratedByExpertise so a misbehaving
// caller cannot ask for an unbounded list. 500 is deliberately far
// above the typical UI need (panels show ~50 at a time) to leave
// room for admin tooling without being infinite.
const maxCuratedListLimit = 500

// Upsert inserts a new catalog entry or updates an existing one by
// skill_text (the primary key). display_text, expertise_keys and
// is_curated are refreshed; usage_count is intentionally preserved
// — the contract in port/repository forbids clobbering the counter,
// which is maintained exclusively by IncrementUsageCount and
// DecrementUsageCount.
func (r *SkillCatalogRepository) Upsert(ctx context.Context, entry *domainskill.CatalogEntry) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO skills_catalog (skill_text, display_text, expertise_keys, is_curated, usage_count, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 0, now(), now())
		 ON CONFLICT (skill_text) DO UPDATE
		   SET display_text   = EXCLUDED.display_text,
		       expertise_keys = EXCLUDED.expertise_keys,
		       is_curated     = EXCLUDED.is_curated,
		       updated_at     = now()`,
		entry.SkillText,
		entry.DisplayText,
		pq.Array(entry.ExpertiseKeys),
		entry.IsCurated,
	)
	if err != nil {
		return fmt.Errorf("upsert skill catalog entry: %w", err)
	}
	return nil
}

// FindByText returns the catalog entry for a normalized skill text,
// or (nil, nil) when no row matches. The contract in port/repository
// explicitly forbids returning a sentinel "not found" error here —
// callers must nil-check the result.
func (r *SkillCatalogRepository) FindByText(ctx context.Context, skillText string) (*domainskill.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT skill_text, display_text, expertise_keys, is_curated, usage_count, created_at, updated_at
		   FROM skills_catalog
		  WHERE skill_text = $1`,
		skillText,
	)

	entry, err := scanCatalogEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find skill by text: %w", err)
	}
	return entry, nil
}

// ListCuratedByExpertise returns curated entries tagged with the
// given expertise key, ordered by usage_count DESC. The limit is
// clamped to [1, maxCuratedListLimit] so a zero or negative caller
// value still yields a small first page and a huge caller value
// cannot exhaust the server.
func (r *SkillCatalogRepository) ListCuratedByExpertise(ctx context.Context, expertiseKey string, limit int) ([]*domainskill.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit = clampListLimit(limit)

	rows, err := r.db.QueryContext(ctx,
		`SELECT skill_text, display_text, expertise_keys, is_curated, usage_count, created_at, updated_at
		   FROM skills_catalog
		  WHERE is_curated = true
		    AND expertise_keys @> ARRAY[$1]::text[]
		  ORDER BY usage_count DESC, skill_text ASC
		  LIMIT $2`,
		expertiseKey, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list curated by expertise: %w", err)
	}
	defer rows.Close()

	return collectCatalogRows(rows)
}

// CountCuratedByExpertise returns how many curated entries are
// tagged with the given expertise key. Used by panel headers to
// display counters without over-fetching the list.
func (r *SkillCatalogRepository) CountCuratedByExpertise(ctx context.Context, expertiseKey string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		   FROM skills_catalog
		  WHERE is_curated = true
		    AND expertise_keys @> ARRAY[$1]::text[]`,
		expertiseKey,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count curated by expertise: %w", err)
	}
	return count, nil
}

// SearchAutocomplete returns catalog entries matching the query by
// either prefix or trigram similarity. The ordering rules are:
//
//  1. Exact matches come first (skill_text = q).
//  2. Curated entries rank above user-created ones.
//  3. Prefix matches rank above pure fuzzy (similarity) matches.
//  4. Inside each tier, sort by usage_count DESC.
//  5. Final tie-break alphabetically so results are deterministic.
//
// The trigram comparison uses pg_trgm's `%` operator (default
// similarity threshold 0.3). The index created by migration 081
// (idx_skills_catalog_text_trgm) backs this operator for fuzzy
// matches; the prefix branch uses a simple LIKE.
func (r *SkillCatalogRepository) SearchAutocomplete(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit = clampListLimit(limit)

	rows, err := r.db.QueryContext(ctx,
		`SELECT skill_text, display_text, expertise_keys, is_curated, usage_count, created_at, updated_at
		   FROM skills_catalog
		  WHERE skill_text LIKE $1 || '%' OR skill_text % $1
		  ORDER BY
		    (skill_text = $1) DESC,
		    is_curated DESC,
		    CASE WHEN skill_text LIKE $1 || '%' THEN 0 ELSE 1 END,
		    usage_count DESC,
		    skill_text ASC
		  LIMIT $2`,
		q, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search skill autocomplete: %w", err)
	}
	defer rows.Close()

	return collectCatalogRows(rows)
}

// IncrementUsageCount bumps the counter for a single skill by 1.
// If the target row no longer exists (race with a concurrent
// delete) the UPDATE simply matches zero rows — the method is
// idempotent on that edge case per the interface contract.
func (r *SkillCatalogRepository) IncrementUsageCount(ctx context.Context, skillText string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE skills_catalog
		    SET usage_count = usage_count + 1,
		        updated_at  = now()
		  WHERE skill_text = $1`,
		skillText,
	)
	if err != nil {
		return fmt.Errorf("increment skill usage count: %w", err)
	}
	return nil
}

// DecrementUsageCount decrements the counter for a single skill by
// 1, clamped at zero. The clamp is enforced in SQL with GREATEST so
// a racing caller can never produce a negative count.
func (r *SkillCatalogRepository) DecrementUsageCount(ctx context.Context, skillText string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE skills_catalog
		    SET usage_count = GREATEST(0, usage_count - 1),
		        updated_at  = now()
		  WHERE skill_text = $1`,
		skillText,
	)
	if err != nil {
		return fmt.Errorf("decrement skill usage count: %w", err)
	}
	return nil
}

// clampListLimit snaps the caller's requested limit into the
// [1, maxCuratedListLimit] window. A zero or negative value
// becomes 1 (never return an empty page for a bad limit) and a
// value above the cap is lowered to the cap.
func clampListLimit(limit int) int {
	if limit < 1 {
		return 1
	}
	if limit > maxCuratedListLimit {
		return maxCuratedListLimit
	}
	return limit
}

// scanCatalogEntry pulls a single row into a CatalogEntry. Used by
// both the single-row (QueryRowContext) and multi-row (Rows) paths.
func scanCatalogEntry(scanner interface{ Scan(dest ...any) error }) (*domainskill.CatalogEntry, error) {
	var entry domainskill.CatalogEntry
	var expertiseKeys pq.StringArray
	if err := scanner.Scan(
		&entry.SkillText,
		&entry.DisplayText,
		&expertiseKeys,
		&entry.IsCurated,
		&entry.UsageCount,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return nil, err
	}
	entry.ExpertiseKeys = []string(expertiseKeys)
	if entry.ExpertiseKeys == nil {
		entry.ExpertiseKeys = []string{}
	}
	return &entry, nil
}

// collectCatalogRows drains a *sql.Rows into a slice of catalog
// entries. Always returns a non-nil slice so the handler layer can
// marshal an empty result directly to `[]` without a nil check.
func collectCatalogRows(rows *sql.Rows) ([]*domainskill.CatalogEntry, error) {
	entries := make([]*domainskill.CatalogEntry, 0, 16)
	for rows.Next() {
		entry, err := scanCatalogEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill catalog row: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill catalog rows: %w", err)
	}
	return entries, nil
}
