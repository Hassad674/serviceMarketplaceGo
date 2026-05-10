package repository

import (
	"context"

	"marketplace-backend/internal/domain/stats"
)

// SearchQueryStatsRepository is the read-only aggregator over the
// existing search_queries table (migrations 111-113). It is kept
// strictly separate from the SearchAnalytics repository (which owns
// the writer) so the stats feature can be removed by deleting its
// folder + this single file without touching the search engine.
type SearchQueryStatsRepository interface {
	// TopKeywordsForOrg returns the top N keywords visitors typed
	// before clicking through to the requested organization's
	// public profile. The aggregation joins clicked_result_id =
	// organization_id and groups by lowercased query.
	//
	// Ordering: by descending click count. AvgPosition is the mean
	// of clicked_position across the rows in each group.
	TopKeywordsForOrg(ctx context.Context, filter KeywordFilter) ([]stats.KeywordRow, error)
}

// EnterpriseApplicationsStatsRepository returns the per-day count of
// job applications received on jobs owned by the requested
// organization. Read-only.
type EnterpriseApplicationsStatsRepository interface {
	// AggregateApplications returns the totals + daily series for
	// the requested window. The returned ApplicationsTimeSeries is
	// fully populated (non-nil Series, possibly empty).
	AggregateApplications(ctx context.Context, filter VisibilityFilter) (*stats.ApplicationsTimeSeries, error)
}
