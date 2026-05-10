package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/port/repository"
)

// SearchQueryStatsRepository is the read-only Postgres adapter for
// repository.SearchQueryStatsRepository. It aggregates over the
// existing `search_queries` table (migration 111) via the
// `clicked_result_id` column populated by the search engine when the
// user clicks a result.
//
// Strict separation from postgres.SearchAnalyticsRepository (which
// owns INSERTs into search_queries): the stats feature must be
// removable by deleting this single file + its consumers without
// touching the search engine writer.
type SearchQueryStatsRepository struct {
	db *sql.DB
}

// NewSearchQueryStatsRepository wires the adapter to a sql.DB handle.
func NewSearchQueryStatsRepository(db *sql.DB) *SearchQueryStatsRepository {
	return &SearchQueryStatsRepository{db: db}
}

// TopKeywordsForOrg returns the top N keywords visitors typed before
// clicking through to the requested organization's public profile in
// the last `period` days. Keywords are lowercased + trimmed at the
// SQL layer so casing variants ("Go developer" / "go developer")
// collapse into a single row.
//
// The query relies on the `idx_search_queries_created_at` index for
// the time-window filter and benefits from a partial index scan when
// `clicked_result_id` is non-null on a small fraction of rows. With
// a few million search rows this still resolves in single-digit ms.
func (r *SearchQueryStatsRepository) TopKeywordsForOrg(ctx context.Context, filter repository.KeywordFilter) ([]domainstats.KeywordRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `
		SELECT
			LOWER(TRIM(query))                                                AS keyword,
			COUNT(*)::int                                                     AS click_count,
			COALESCE(AVG(clicked_position) FILTER (WHERE clicked_position IS NOT NULL), 0)::float8 AS avg_position
		FROM search_queries
		WHERE clicked_result_id = $1
		  AND created_at >= NOW() - ($2::int * INTERVAL '1 day')
		  AND TRIM(query) <> ''
		GROUP BY LOWER(TRIM(query))
		ORDER BY click_count DESC, keyword ASC
		LIMIT $3
	`
	rows, err := r.db.QueryContext(ctx, stmt,
		filter.OrganizationID,
		int(filter.PeriodDays),
		filter.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search_queries: top keywords: %w", err)
	}
	defer rows.Close()

	out := make([]domainstats.KeywordRow, 0, filter.Limit)
	for rows.Next() {
		var k domainstats.KeywordRow
		if err := rows.Scan(&k.Keyword, &k.Count, &k.AvgPosition); err != nil {
			return nil, fmt.Errorf("search_queries: top keywords scan: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search_queries: top keywords rows: %w", err)
	}
	return out, nil
}
