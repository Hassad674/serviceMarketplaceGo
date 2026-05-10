package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/stats"
)

// VisibilityFilter narrows aggregation queries to one organization +
// a fixed rolling window. The PeriodDays value is enforced upstream;
// repositories trust it.
type VisibilityFilter struct {
	OrganizationID uuid.UUID
	PeriodDays     stats.PeriodDays
}

// KeywordFilter extends VisibilityFilter with a Limit on the row
// count returned. Limit is clamped upstream — the repository trusts
// it (1..100).
type KeywordFilter struct {
	OrganizationID uuid.UUID
	PeriodDays     stats.PeriodDays
	Limit          int
}

// ProfileViewRepository persists profile_view_events rows and answers
// the per-org aggregation queries the stats dashboard renders.
//
// Append-only by design — the table is the primary source of truth
// for view stats so updates and deletes are not part of the contract.
//
// Each method imposes its own context timeout (5s) at the adapter
// level — callers do not need to wrap.
type ProfileViewRepository interface {
	// Record inserts a single ViewEvent row. Returns nil on success.
	// Domain errors (validation) are caught upstream by the service;
	// adapter errors (DB unreachable, FK violation) are wrapped and
	// returned verbatim.
	Record(ctx context.Context, event *stats.ViewEvent) error

	// AggregateVisibility returns the totals + daily series for the
	// requested window. The returned Visibility is fully populated
	// (non-nil Series, possibly empty) — callers may emit it as
	// JSON without further normalization.
	AggregateVisibility(ctx context.Context, filter VisibilityFilter) (*stats.Visibility, error)
}
