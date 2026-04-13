package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	domainskill "marketplace-backend/internal/domain/skill"
)

// SkillCatalogRepository is the PostgreSQL-backed implementation of
// repository.SkillCatalogRepository. It owns the skills_catalog table
// (migration 081) and performs the usage_count updates, the
// ListCuratedByExpertise panel query, and the trigram autocomplete.
//
// The repository is stateless — a single instance is shared across
// all handlers via main.go wiring.
type SkillCatalogRepository struct {
	db *sql.DB
}

// NewSkillCatalogRepository returns a catalog repository bound to the
// given *sql.DB.
func NewSkillCatalogRepository(db *sql.DB) *SkillCatalogRepository {
	return &SkillCatalogRepository{db: db}
}

// Upsert inserts a new catalog entry or updates an existing one. On
// conflict the display_text, expertise_keys, and is_curated columns
// are refreshed; usage_count is preserved to avoid racing with the
// concurrent increment path.
func (r *SkillCatalogRepository) Upsert(
	ctx context.Context,
	entry *domainskill.CatalogEntry,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO skills_catalog (skill_text, display_text, expertise_keys, is_curated)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (skill_text) DO UPDATE SET
		   display_text   = EXCLUDED.display_text,
		   expertise_keys = EXCLUDED.expertise_keys,
		   is_curated     = EXCLUDED.is_curated,
		   updated_at     = now()`,
		entry.SkillText,
		entry.DisplayText,
		pq.StringArray(entry.ExpertiseKeys),
		entry.IsCurated,
	)
	if err != nil {
		return fmt.Errorf("upsert skill catalog entry: %w", err)
	}
	return nil
}

// FindByText returns the catalog entry for the given normalized text,
// or (nil, nil) if no row matches — matching the convention used by
// every other repository in this package.
func (r *SkillCatalogRepository) FindByText(
	ctx context.Context,
	skillText string,
) (*domainskill.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT skill_text, display_text, expertise_keys, is_curated, usage_count, created_at, updated_at
		   FROM skills_catalog
		  WHERE skill_text = $1`,
		skillText,
	)
	entry, err := scanCatalogRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find skill by text: %w", err)
	}
	return entry, nil
}

// ListCuratedByExpertise returns curated entries whose expertise_keys
// array contains the given key, sorted by usage_count desc then by
// display_text asc for deterministic ties.
func (r *SkillCatalogRepository) ListCuratedByExpertise(
	ctx context.Context,
	expertiseKey string,
	limit int,
) ([]*domainskill.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT skill_text, display_text, expertise_keys, is_curated, usage_count, created_at, updated_at
		   FROM skills_catalog
		  WHERE is_curated = true AND $1 = ANY(expertise_keys)
		  ORDER BY usage_count DESC, display_text ASC
		  LIMIT $2`,
		expertiseKey,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list curated skills by expertise: %w", err)
	}
	defer rows.Close()
	return collectCatalogRows(rows)
}

// CountCuratedByExpertise returns the total number of curated entries
// tagged with the given expertise key.
func (r *SkillCatalogRepository) CountCuratedByExpertise(
	ctx context.Context,
	expertiseKey string,
) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM skills_catalog
		  WHERE is_curated = true AND $1 = ANY(expertise_keys)`,
		expertiseKey,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count curated skills by expertise: %w", err)
	}
	return n, nil
}

// SearchAutocomplete returns catalog entries matching the query string
// via ILIKE prefix first, then trigram similarity as a fallback. The
// result set is capped at limit and biased towards curated entries.
func (r *SkillCatalogRepository) SearchAutocomplete(
	ctx context.Context,
	q string,
	limit int,
) ([]*domainskill.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 {
		limit = 20
	}
	// Prefix match via ILIKE, with curated rows ranked above user
	// contributions. Trigram similarity is implicit in the ORDER BY
	// via pg_trgm's `<->` distance operator — closer strings sort first.
	rows, err := r.db.QueryContext(ctx,
		`SELECT skill_text, display_text, expertise_keys, is_curated, usage_count, created_at, updated_at
		   FROM skills_catalog
		  WHERE skill_text ILIKE $1 || '%' OR skill_text % $1
		  ORDER BY is_curated DESC, usage_count DESC, skill_text <-> $1
		  LIMIT $2`,
		q,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search skills: %w", err)
	}
	defer rows.Close()
	return collectCatalogRows(rows)
}

// IncrementUsageCount bumps the counter for a single skill. Missing
// rows are silently ignored to stay idempotent in the face of deletes.
func (r *SkillCatalogRepository) IncrementUsageCount(
	ctx context.Context,
	skillText string,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE skills_catalog SET usage_count = usage_count + 1 WHERE skill_text = $1`,
		skillText,
	)
	if err != nil {
		return fmt.Errorf("increment skill usage: %w", err)
	}
	return nil
}

// DecrementUsageCount decrements the counter clamped at zero.
func (r *SkillCatalogRepository) DecrementUsageCount(
	ctx context.Context,
	skillText string,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE skills_catalog
		    SET usage_count = GREATEST(usage_count - 1, 0)
		  WHERE skill_text = $1`,
		skillText,
	)
	if err != nil {
		return fmt.Errorf("decrement skill usage: %w", err)
	}
	return nil
}

// scanCatalogRow is used by FindByText (single-row scan). It takes a
// scan closure so both *sql.Row and *sql.Rows can share the code path.
func scanCatalogRow(scan func(dest ...any) error) (*domainskill.CatalogEntry, error) {
	var e domainskill.CatalogEntry
	var keys pq.StringArray
	if err := scan(
		&e.SkillText,
		&e.DisplayText,
		&keys,
		&e.IsCurated,
		&e.UsageCount,
		&e.CreatedAt,
		&e.UpdatedAt,
	); err != nil {
		return nil, err
	}
	e.ExpertiseKeys = []string(keys)
	if e.ExpertiseKeys == nil {
		e.ExpertiseKeys = []string{}
	}
	return &e, nil
}

// collectCatalogRows drains rows into a slice of entries, handling
// the usual errors (scan, iterate) uniformly.
func collectCatalogRows(rows *sql.Rows) ([]*domainskill.CatalogEntry, error) {
	out := make([]*domainskill.CatalogEntry, 0, 8)
	for rows.Next() {
		entry, err := scanCatalogRow(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan skill row: %w", err)
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill rows: %w", err)
	}
	return out, nil
}
