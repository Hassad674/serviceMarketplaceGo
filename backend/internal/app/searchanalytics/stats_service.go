// stats_service.go exposes aggregate analytics over the
// `search_queries` table. Separate service from CaptureSearch /
// RecordClick so the two halves of the analytics subsystem have
// different dependencies: capture is fire-and-forget writes, stats
// is a single read served to admin dashboards.
//
// The stats service is deliberately read-only — no method on this
// type mutates the database. Callers pass a time range, the service
// returns a typed Stats struct. Zero-result queries and top queries
// are bounded by a limit (default 20, max 100) so a malicious admin
// cannot pull the full history in one call.
//
// Design invariants:
//
//   - The service owns its own narrow port (StatsRepository) so the
//     capture-side repo does not need to know about aggregates, and
//     tests for either half can ship independent fakes.
//   - p95 is computed over the time range as-is, without re-bucketing,
//     so operators get a single honest number per window rather than
//     a rolling one that drifts.
//   - All time parameters are treated as UTC. The handler layer is
//     responsible for parsing RFC3339 input.
//
// Removing the admin stats feature = delete this file + stats_handler.go
// + wiring in main.go. Capture/click still work.
package searchanalytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// DefaultStatsLimit caps the number of rows returned for both
// TopQueries and ZeroResultQueries when the caller does not pin a
// limit explicitly.
const DefaultStatsLimit = 20

// MaxStatsLimit caps the requested limit. 100 is plenty for a UI
// panel and cheap enough to serve uncached. Requests above this
// saturate at MaxStatsLimit without returning an error.
const MaxStatsLimit = 100

// DefaultStatsWindow is the time range when the caller omits
// from/to. One week matches the current dashboard rollup cadence.
const DefaultStatsWindow = 7 * 24 * time.Hour

// ErrInvalidRange is returned when the stats window is invalid
// (e.g. from > to). Surfaces as a clean 400 from the handler.
var ErrInvalidRange = errors.New("search analytics: from must be before to")

// StatsRepository is the read-side port. The Postgres adapter
// implements it with a mix of single aggregations (counts, rates)
// and bounded top-N queries for the ranked lists.
type StatsRepository interface {
	// Totals returns the single-row aggregation: total searches,
	// zero-result count, average + p95 latency over the window.
	Totals(ctx context.Context, filter StatsFilter) (Totals, error)
	// TopQueries returns the most-searched queries over the window,
	// each with its average result count and click-through ratio.
	// Limit is already normalized (>=1, <=MaxStatsLimit).
	TopQueries(ctx context.Context, filter StatsFilter, limit int) ([]TopQuery, error)
	// ZeroResultQueries returns the most-frequent queries that
	// returned zero results. Limit is already normalized.
	ZeroResultQueries(ctx context.Context, filter StatsFilter, limit int) ([]ZeroResultQuery, error)
}

// StatsFilter is the common WHERE clause every stats query applies.
// Persona is optional — empty string means "all personas".
type StatsFilter struct {
	From    time.Time
	To      time.Time
	Persona string
}

// Totals groups the scalar aggregates. Fields are primitives so the
// JSON encoder can ship them without extra adaptation.
type Totals struct {
	TotalSearches  int     `json:"total_searches"`
	ZeroResults    int     `json:"zero_result_searches"`
	ZeroResultRate float64 `json:"zero_result_rate"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
}

// TopQuery is one row in the most-searched list. Count is the number
// of distinct search buckets (rows share a search_id when the user
// paginates), AvgResults is the mean of results_count across those
// rows, and CTR is clicked_count / total_count.
type TopQuery struct {
	Query      string  `json:"query"`
	Count      int     `json:"count"`
	AvgResults float64 `json:"avg_results"`
	CTR        float64 `json:"ctr"`
}

// ZeroResultQuery is one row in the zero-result list. A single query
// text may appear many times over the window; Count is how often.
type ZeroResultQuery struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

// Stats is the full payload returned by StatsService.Compute.
type Stats struct {
	TotalSearches     int               `json:"total_searches"`
	ZeroResults       int               `json:"zero_result_searches"`
	ZeroResultRate    float64           `json:"zero_result_rate"`
	AvgLatencyMs      float64           `json:"avg_latency_ms"`
	P95LatencyMs      float64           `json:"p95_latency_ms"`
	TopQueries        []TopQuery        `json:"top_queries"`
	ZeroResultQueries []ZeroResultQuery `json:"zero_result_queries"`
	From              time.Time         `json:"from"`
	To                time.Time         `json:"to"`
	Persona           string            `json:"persona,omitempty"`
}

// StatsQuery is the input to StatsService.Compute. All fields are
// optional — the service applies sensible defaults.
type StatsQuery struct {
	From    time.Time
	To      time.Time
	Persona string
	Limit   int
}

// StatsService orchestrates the stats reads. Stateless; safe for
// concurrent use.
type StatsService struct {
	repo   StatsRepository
	logger *slog.Logger
	clock  func() time.Time
}

// StatsServiceConfig groups the constructor inputs.
type StatsServiceConfig struct {
	Repository StatsRepository
	Logger     *slog.Logger
	// Clock is injectable so unit tests can pin the default window.
	// Defaults to time.Now when nil.
	Clock func() time.Time
}

// NewStatsService builds the service. Repository is required; logger
// and clock default sensibly.
func NewStatsService(cfg StatsServiceConfig) (*StatsService, error) {
	if cfg.Repository == nil {
		return nil, fmt.Errorf("searchanalytics: stats repository is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	return &StatsService{repo: cfg.Repository, logger: logger, clock: clock}, nil
}

// Compute assembles the Stats payload by fanning out to the
// repository. Uses a single context per read so a slow query times
// out cleanly.
func (s *StatsService) Compute(ctx context.Context, q StatsQuery) (*Stats, error) {
	filter, limit, err := s.normalize(q)
	if err != nil {
		return nil, err
	}

	totals, err := s.repo.Totals(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("searchanalytics: totals: %w", err)
	}
	top, err := s.repo.TopQueries(ctx, filter, limit)
	if err != nil {
		return nil, fmt.Errorf("searchanalytics: top queries: %w", err)
	}
	zero, err := s.repo.ZeroResultQueries(ctx, filter, limit)
	if err != nil {
		return nil, fmt.Errorf("searchanalytics: zero-result queries: %w", err)
	}

	return &Stats{
		TotalSearches:     totals.TotalSearches,
		ZeroResults:       totals.ZeroResults,
		ZeroResultRate:    totals.ZeroResultRate,
		AvgLatencyMs:      totals.AvgLatencyMs,
		P95LatencyMs:      totals.P95LatencyMs,
		TopQueries:        top,
		ZeroResultQueries: zero,
		From:              filter.From,
		To:                filter.To,
		Persona:           filter.Persona,
	}, nil
}

// normalize fills defaults and validates the window. Extracted so
// Compute stays flat and every branch has an independent unit test.
func (s *StatsService) normalize(q StatsQuery) (StatsFilter, int, error) {
	now := s.clock().UTC()
	to := q.To
	if to.IsZero() {
		to = now
	}
	from := q.From
	if from.IsZero() {
		from = to.Add(-DefaultStatsWindow)
	}
	if !from.Before(to) {
		return StatsFilter{}, 0, ErrInvalidRange
	}
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultStatsLimit
	}
	if limit > MaxStatsLimit {
		limit = MaxStatsLimit
	}
	return StatsFilter{
		From:    from.UTC(),
		To:      to.UTC(),
		Persona: q.Persona,
	}, limit, nil
}
