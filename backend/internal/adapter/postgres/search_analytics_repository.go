package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"marketplace-backend/internal/app/searchanalytics"
)

// search_analytics_repository.go implements searchanalytics.Repository
// on top of the `search_queries` table created by migration 111.
//
// Schema reminder:
//
//	id                 UUID PRIMARY KEY DEFAULT gen_random_uuid()
//	user_id            UUID NULL REFERENCES users(id) ON DELETE SET NULL
//	session_id         TEXT NULL
//	query              TEXT NOT NULL
//	filters            JSONB NOT NULL DEFAULT '{}'
//	persona            TEXT NOT NULL   (freelance|agency|referrer|all)
//	results_count      INTEGER NOT NULL
//	latency_ms         INTEGER NOT NULL
//	clicked_result_id  UUID NULL
//	clicked_position   INTEGER NULL
//	created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
//
// Phase 3 extends the row with two columns that were missing on the
// original 111 migration: `search_id` (deterministic per-query hash
// from the app layer, used to dedupe pages + attach clicks) and
// `clicked_at`. Migration 112 adds both.

// SearchAnalyticsRepository is the Postgres implementation.
type SearchAnalyticsRepository struct {
	db *sql.DB
}

// NewSearchAnalyticsRepository wires a repository against an
// existing *sql.DB. Connection-pool sizing is the caller's
// responsibility (db is configured in cmd/api/main.go).
func NewSearchAnalyticsRepository(db *sql.DB) *SearchAnalyticsRepository {
	return &SearchAnalyticsRepository{db: db}
}

// InsertSearch appends a row for a captured search. Uses the
// canonical ON CONFLICT DO NOTHING idiom keyed on `search_id` so
// multi-page loads of the same query share a single row.
func (r *SearchAnalyticsRepository) InsertSearch(ctx context.Context, row *searchanalytics.SearchRow) error {
	if row == nil {
		return fmt.Errorf("search analytics insert: nil row")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// filters is a JSONB column; we feed it a {"filter_by": "<expr>"}
	// object so the analytics dashboards can parse either the raw
	// Typesense filter expression OR expand the object later without
	// a schema migration.
	filtersJSON, err := json.Marshal(map[string]string{
		"filter_by": row.FilterBy,
		"sort_by":   row.SortBy,
	})
	if err != nil {
		return fmt.Errorf("search analytics: marshal filters: %w", err)
	}

	const query = `
INSERT INTO search_queries (
    search_id, user_id, session_id, query, filters, persona,
    results_count, latency_ms, created_at
) VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, ''), $4, $5::jsonb, $6, $7, $8, $9)
ON CONFLICT (search_id) DO NOTHING`

	_, err = r.db.ExecContext(ctx, query,
		row.SearchID,
		row.UserID,
		row.SessionID,
		row.Query,
		string(filtersJSON),
		row.Persona,
		row.ResultsCount,
		row.LatencyMs,
		row.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("search analytics: insert: %w", err)
	}
	return nil
}

// RecordClick updates the most-recent search row with the click
// payload. Returns searchanalytics.ErrNotFound when the row no
// longer exists so the handler can respond with a clean 404.
func (r *SearchAnalyticsRepository) RecordClick(ctx context.Context, searchID, docID string, position int, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	const query = `
UPDATE search_queries
   SET clicked_result_id = NULLIF($2, '')::uuid,
       clicked_position  = $3,
       clicked_at        = $4
 WHERE search_id = $1`

	res, err := r.db.ExecContext(ctx, query, searchID, docID, position, at)
	if err != nil {
		return fmt.Errorf("search analytics: click update: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("search analytics: click rows affected: %w", err)
	}
	if rows == 0 {
		return searchanalytics.ErrNotFound
	}
	return nil
}

// Totals returns the scalar aggregates for the stats dashboard: total
// searches, zero-result count, zero-result rate, average latency, and
// p95 latency. Single SQL round-trip using PERCENTILE_CONT + conditional
// counts so we avoid N+1 trips for the trivial fields.
//
// NULLs: an empty result window returns zero in every field (divide-by-
// zero is guarded on the Go side so the JSON response always ships sane
// numbers).
func (r *SearchAnalyticsRepository) Totals(ctx context.Context, filter searchanalytics.StatsFilter) (searchanalytics.Totals, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
SELECT
    COUNT(*)                                                       AS total,
    COALESCE(SUM(CASE WHEN results_count = 0 THEN 1 ELSE 0 END),0) AS zero_results,
    COALESCE(AVG(latency_ms)::float8, 0)                           AS avg_latency,
    COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms)::float8, 0) AS p95_latency
FROM search_queries
WHERE created_at >= $1 AND created_at < $2
  AND ($3 = '' OR persona = $3)`

	var t searchanalytics.Totals
	err := r.db.QueryRowContext(ctx, query, filter.From, filter.To, filter.Persona).
		Scan(&t.TotalSearches, &t.ZeroResults, &t.AvgLatencyMs, &t.P95LatencyMs)
	if err != nil {
		return searchanalytics.Totals{}, fmt.Errorf("search analytics: totals: %w", err)
	}
	if t.TotalSearches > 0 {
		t.ZeroResultRate = float64(t.ZeroResults) / float64(t.TotalSearches)
	}
	return t, nil
}

// TopQueries returns the `limit` most-searched queries over the
// window, ordered by count DESC. Each row also carries the mean
// result count and the CTR (clicks / count).
//
// Case-insensitive grouping via lower(query) so "React" and "react"
// collapse. We surface the lower-cased text — slightly lossy but
// vastly more useful for analytics.
func (r *SearchAnalyticsRepository) TopQueries(ctx context.Context, filter searchanalytics.StatsFilter, limit int) ([]searchanalytics.TopQuery, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
SELECT
    lower(query)                      AS q,
    COUNT(*)                          AS count,
    COALESCE(AVG(results_count)::float8, 0) AS avg_results,
    CASE WHEN COUNT(*) = 0 THEN 0
         ELSE COUNT(clicked_result_id)::float8 / COUNT(*)::float8
    END                               AS ctr
FROM search_queries
WHERE created_at >= $1 AND created_at < $2
  AND ($3 = '' OR persona = $3)
  AND query <> ''
GROUP BY lower(query)
ORDER BY count DESC, q ASC
LIMIT $4`

	rows, err := r.db.QueryContext(ctx, query, filter.From, filter.To, filter.Persona, limit)
	if err != nil {
		return nil, fmt.Errorf("search analytics: top queries: %w", err)
	}
	defer rows.Close()

	out := make([]searchanalytics.TopQuery, 0, limit)
	for rows.Next() {
		var row searchanalytics.TopQuery
		if err := rows.Scan(&row.Query, &row.Count, &row.AvgResults, &row.CTR); err != nil {
			return nil, fmt.Errorf("search analytics: top queries scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search analytics: top queries rows: %w", err)
	}
	return out, nil
}

// ZeroResultQueries returns the `limit` most-frequent queries that
// produced zero results, ordered by count DESC. Same case-folding as
// TopQueries.
//
// Uses the existing partial index `idx_search_queries_zero_results`
// for the WHERE results_count = 0 predicate — the planner picks it
// up automatically when the filter is inlined.
func (r *SearchAnalyticsRepository) ZeroResultQueries(ctx context.Context, filter searchanalytics.StatsFilter, limit int) ([]searchanalytics.ZeroResultQuery, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
SELECT
    lower(query) AS q,
    COUNT(*)     AS count
FROM search_queries
WHERE results_count = 0
  AND created_at >= $1 AND created_at < $2
  AND ($3 = '' OR persona = $3)
  AND query <> ''
GROUP BY lower(query)
ORDER BY count DESC, q ASC
LIMIT $4`

	rows, err := r.db.QueryContext(ctx, query, filter.From, filter.To, filter.Persona, limit)
	if err != nil {
		return nil, fmt.Errorf("search analytics: zero-result queries: %w", err)
	}
	defer rows.Close()

	out := make([]searchanalytics.ZeroResultQuery, 0, limit)
	for rows.Next() {
		var row searchanalytics.ZeroResultQuery
		if err := rows.Scan(&row.Query, &row.Count); err != nil {
			return nil, fmt.Errorf("search analytics: zero-result queries scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search analytics: zero-result queries rows: %w", err)
	}
	return out, nil
}

// Compile-time assertion that the repository implements both ports.
var (
	_ searchanalytics.Repository      = (*SearchAnalyticsRepository)(nil)
	_ searchanalytics.StatsRepository = (*SearchAnalyticsRepository)(nil)
)
