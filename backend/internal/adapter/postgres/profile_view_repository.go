package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/port/repository"
)

// profile_view_repository.go implements repository.ProfileViewRepository
// and repository.EnterpriseApplicationsStatsRepository on top of the
// `profile_view_events` table (migration 140) and the existing
// `job_applications` + `jobs` tables (migrations 011/028/061).
//
// The Record path uses a 5s timeout because INET casts and FK
// validations occasionally hit the planner cold; the aggregation
// queries use 5s as well — they are bounded by the
// (organization_id, created_at) index range scan and only ever scan
// the last 90 days of rows.

// ProfileViewRepository is the Postgres adapter for view event
// persistence + aggregations.
type ProfileViewRepository struct {
	db *sql.DB
}

// NewProfileViewRepository wires the adapter to a sql.DB handle. The
// caller (cmd/api wire layer) owns the lifecycle of the underlying
// pool.
func NewProfileViewRepository(db *sql.DB) *ProfileViewRepository {
	return &ProfileViewRepository{db: db}
}

// Record inserts a new profile_view_events row. Uses the entry's ID
// and CreatedAt verbatim — the domain layer owns identity +
// timestamp so the persisted row matches what the service returned to
// the handler.
func (r *ProfileViewRepository) Record(ctx context.Context, event *domainstats.ViewEvent) error {
	if event == nil {
		return fmt.Errorf("profile_view_events: nil event")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `
		INSERT INTO profile_view_events (
			id, organization_id, persona, viewer_user_id,
			viewer_ip_anonymized, viewer_ua_hash, came_from,
			search_query, search_position, referrer_url, created_at
		) VALUES ($1, $2, $3, $4, $5::inet, $6, $7, $8, $9, $10, $11)
	`

	var viewerUserID interface{}
	if event.ViewerUserID != nil {
		viewerUserID = *event.ViewerUserID
	}
	var (
		searchQuery interface{}
		searchPos   interface{}
		referrer    interface{}
	)
	if event.SearchQuery != nil {
		searchQuery = *event.SearchQuery
	}
	if event.SearchPosition != nil {
		searchPos = *event.SearchPosition
	}
	if event.ReferrerURL != nil {
		referrer = *event.ReferrerURL
	}

	if _, err := r.db.ExecContext(ctx, stmt,
		event.ID,
		event.OrganizationID,
		string(event.Persona),
		viewerUserID,
		event.ViewerIPAnonymized,
		event.ViewerUAHash,
		string(event.CameFrom),
		searchQuery,
		searchPos,
		referrer,
		event.CreatedAt,
	); err != nil {
		return fmt.Errorf("profile_view_events: insert: %w", err)
	}
	return nil
}

// AggregateVisibility returns totals + the per-day series for the
// requested window. The query runs in two roundtrips (totals + series)
// so the planner can use the (org, created_at) covering index for
// each — a single mega-CTE benchmarks worse on modest dataset sizes.
func (r *ProfileViewRepository) AggregateVisibility(ctx context.Context, filter repository.VisibilityFilter) (*domainstats.Visibility, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	totals, err := r.queryVisibilityTotals(ctx, filter)
	if err != nil {
		return nil, err
	}
	series, err := r.queryDailyViews(ctx, filter)
	if err != nil {
		return nil, err
	}
	totals.Series = series
	totals.OrganizationID = filter.OrganizationID.String()
	totals.PeriodDays = filter.PeriodDays
	return totals, nil
}

// queryVisibilityTotals returns the aggregate counters for the window.
// The avg_position column ignores NULL search_position rows.
func (r *ProfileViewRepository) queryVisibilityTotals(ctx context.Context, filter repository.VisibilityFilter) (*domainstats.Visibility, error) {
	const stmt = `
		SELECT
			COUNT(*)::int                                              AS total_views,
			COUNT(DISTINCT (viewer_ip_anonymized, viewer_ua_hash))::int AS unique_viewers,
			COUNT(*) FILTER (WHERE came_from = 'search')::int          AS search_appearances,
			COALESCE(AVG(search_position) FILTER (WHERE search_position IS NOT NULL), 0)::float8 AS avg_position
		FROM profile_view_events
		WHERE organization_id = $1
		  AND created_at >= NOW() - ($2::int * INTERVAL '1 day')
	`
	row := r.db.QueryRowContext(ctx, stmt, filter.OrganizationID, int(filter.PeriodDays))
	var v domainstats.Visibility
	if err := row.Scan(&v.TotalViews, &v.UniqueViewers, &v.SearchAppearances, &v.AvgSearchPosition); err != nil {
		return nil, fmt.Errorf("profile_view_events: aggregate totals: %w", err)
	}
	return &v, nil
}

// queryDailyViews returns the per-day total view count for the
// window. Days with zero views are NOT padded — the frontend pads
// the series itself so the SQL stays cheap.
func (r *ProfileViewRepository) queryDailyViews(ctx context.Context, filter repository.VisibilityFilter) ([]domainstats.DailyBucket, error) {
	const stmt = `
		SELECT
			date_trunc('day', created_at) AS day,
			COUNT(*)::int                  AS views
		FROM profile_view_events
		WHERE organization_id = $1
		  AND created_at >= NOW() - ($2::int * INTERVAL '1 day')
		GROUP BY day
		ORDER BY day ASC
	`
	rows, err := r.db.QueryContext(ctx, stmt, filter.OrganizationID, int(filter.PeriodDays))
	if err != nil {
		return nil, fmt.Errorf("profile_view_events: daily series: %w", err)
	}
	defer rows.Close()

	out := make([]domainstats.DailyBucket, 0, int(filter.PeriodDays))
	for rows.Next() {
		var b domainstats.DailyBucket
		if err := rows.Scan(&b.Date, &b.Count); err != nil {
			return nil, fmt.Errorf("profile_view_events: daily series scan: %w", err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("profile_view_events: daily series rows: %w", err)
	}
	return out, nil
}

// AggregateApplications returns the per-day count of job_applications
// for jobs owned by the requested org. Joins job_applications →
// jobs.organization_id so we never count rows belonging to another
// org. NB: `jobs.organization_id` is nullable (legacy rows) — a NULL
// value naturally drops the row from the result set, which matches
// the desired "applications received by THIS org" semantics.
func (r *ProfileViewRepository) AggregateApplications(ctx context.Context, filter repository.VisibilityFilter) (*domainstats.ApplicationsTimeSeries, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const totalsStmt = `
		SELECT COUNT(*)::int
		FROM job_applications a
		JOIN jobs j ON j.id = a.job_id
		WHERE j.organization_id = $1
		  AND a.created_at >= NOW() - ($2::int * INTERVAL '1 day')
	`
	var total int
	if err := r.db.QueryRowContext(ctx, totalsStmt, filter.OrganizationID, int(filter.PeriodDays)).Scan(&total); err != nil {
		return nil, fmt.Errorf("job_applications: total: %w", err)
	}

	const seriesStmt = `
		SELECT
			date_trunc('day', a.created_at) AS day,
			COUNT(*)::int                    AS applications
		FROM job_applications a
		JOIN jobs j ON j.id = a.job_id
		WHERE j.organization_id = $1
		  AND a.created_at >= NOW() - ($2::int * INTERVAL '1 day')
		GROUP BY day
		ORDER BY day ASC
	`
	rows, err := r.db.QueryContext(ctx, seriesStmt, filter.OrganizationID, int(filter.PeriodDays))
	if err != nil {
		return nil, fmt.Errorf("job_applications: series: %w", err)
	}
	defer rows.Close()

	series := make([]domainstats.DailyBucket, 0, int(filter.PeriodDays))
	for rows.Next() {
		var b domainstats.DailyBucket
		if err := rows.Scan(&b.Date, &b.Count); err != nil {
			return nil, fmt.Errorf("job_applications: series scan: %w", err)
		}
		series = append(series, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_applications: series rows: %w", err)
	}

	return &domainstats.ApplicationsTimeSeries{
		OrganizationID: filter.OrganizationID.String(),
		PeriodDays:     filter.PeriodDays,
		TotalCount:     total,
		Series:         series,
	}, nil
}

