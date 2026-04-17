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

// Compile-time assertion that the repository implements the port.
var _ searchanalytics.Repository = (*SearchAnalyticsRepository)(nil)
