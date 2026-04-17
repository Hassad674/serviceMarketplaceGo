// Package searchanalytics captures every successful Typesense search
// into the `search_queries` table plus records click-throughs on
// individual result cards.
//
// Design invariants:
//
//   - Analytics is FIRE-AND-FORGET for captures. A database hiccup
//     must never surface as a failed search — the user-facing query
//     path is authoritative, analytics is advisory.
//   - Click updates are synchronous (the frontend awaits the 204) so
//     we can return a structured error if the click target cannot be
//     matched. The click handler itself is fast — single UPDATE.
//   - The service package owns exactly two ports (Repository and
//     Clock); no leaking concrete Postgres types upward.
//
// Removing the feature = delete this folder + the three wiring
// blocks in cmd/api/main.go. Search still works, just without the
// analytics backfill.
package searchanalytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Repository is the narrow persistence port. Implemented by the
// Postgres adapter in internal/adapter/postgres.
type Repository interface {
	// InsertSearch records a new row. Must be idempotent by SearchID
	// so multi-page loads of the same query bucket together without
	// duplicating rows. Concrete impls use ON CONFLICT DO NOTHING on
	// the (search_id) column.
	InsertSearch(ctx context.Context, row *SearchRow) error

	// RecordClick updates the row identified by SearchID with the
	// click payload. Returns ErrNotFound when the row does not exist
	// (click arrived on a rotated search bucket) so the handler can
	// respond with a 404 rather than a 500.
	RecordClick(ctx context.Context, searchID, clickedDocID string, position int, at time.Time) error
}

// ErrNotFound is returned by RecordClick when no matching row exists.
var ErrNotFound = errors.New("search analytics: row not found")

// SearchRow mirrors the `search_queries` table. Fields map 1:1 to
// SQL columns — the analytics adapter is a thin encoder.
type SearchRow struct {
	SearchID     string
	UserID       string
	SessionID    string
	Query        string
	FilterBy     string
	SortBy       string
	Persona      string
	ResultsCount int
	LatencyMs    int
	CreatedAt    time.Time
}

// Service is the application-layer orchestrator. Stateless beyond
// its dependencies and safe to share across goroutines.
type Service struct {
	repo   Repository
	logger *slog.Logger
	clock  func() time.Time
}

// Config groups the constructor inputs.
type Config struct {
	Repository Repository
	Logger     *slog.Logger
	// Clock is injectable so unit tests can pin createdAt values.
	// Defaults to time.Now when nil.
	Clock func() time.Time
}

// NewService builds the service. The repository is required; logger
// and clock default sensibly.
func NewService(cfg Config) (*Service, error) {
	if cfg.Repository == nil {
		return nil, fmt.Errorf("searchanalytics: repository is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	return &Service{repo: cfg.Repository, logger: logger, clock: clock}, nil
}

// CaptureEvent is the typed payload produced by the search app
// layer. Matches AnalyticsEvent one-for-one so cmd/api can adapt
// without pulling app/search into this package.
type CaptureEvent struct {
	SearchID     string
	UserID       string
	SessionID    string
	Query        string
	FilterBy     string
	SortBy       string
	Persona      string
	ResultsCount int
	LatencyMs    int
}

// CaptureSearch writes the event to the search_queries table. Never
// returns an error because the caller is fire-and-forget; failures
// land in the logger at WARN.
//
// The spec requires this to be non-blocking: we spawn a detached
// goroutine so the search request returns immediately while the
// INSERT runs. A 3-second deadline bounds the detached lifetime so
// a stuck DB connection cannot leak goroutines.
func (s *Service) CaptureSearch(ctx context.Context, evt CaptureEvent) {
	if s == nil || s.repo == nil {
		return
	}
	if evt.SearchID == "" {
		// Without an ID we cannot deduplicate pages of the same
		// query — skip the capture and log a warning so the bug
		// surfaces in dev.
		s.logger.Warn("searchanalytics: skipping capture with empty search_id")
		return
	}
	row := s.buildRow(evt)
	go s.persist(row)
}

// persist runs the INSERT on a background context with a bounded
// deadline. Extracted so CaptureSearch stays minimal and so the
// goroutine body can be tested in isolation via a sync helper.
func (s *Service) persist(row *SearchRow) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.repo.InsertSearch(ctx, row); err != nil {
		s.logger.Warn("searchanalytics: insert failed",
			"error", err,
			"search_id", row.SearchID,
			"persona", row.Persona)
	}
}

// buildRow normalises the event into a SearchRow with the server
// clock applied.
func (s *Service) buildRow(evt CaptureEvent) *SearchRow {
	persona := evt.Persona
	if persona == "" {
		persona = "all"
	}
	return &SearchRow{
		SearchID:     evt.SearchID,
		UserID:       evt.UserID,
		SessionID:    evt.SessionID,
		Query:        evt.Query,
		FilterBy:     evt.FilterBy,
		SortBy:       evt.SortBy,
		Persona:      persona,
		ResultsCount: evt.ResultsCount,
		LatencyMs:    evt.LatencyMs,
		CreatedAt:    s.clock(),
	}
}

// RecordClick persists the click-through against a previously-
// captured search. Synchronous so the caller can surface a 404 when
// the search bucket has rotated (search IDs are minute-bucketed).
func (s *Service) RecordClick(ctx context.Context, searchID, clickedDocID string, position int) error {
	if searchID == "" || clickedDocID == "" {
		return fmt.Errorf("searchanalytics: search_id and doc_id are required")
	}
	if position < 0 {
		return fmt.Errorf("searchanalytics: position must be non-negative")
	}
	err := s.repo.RecordClick(ctx, searchID, clickedDocID, position, s.clock())
	if err != nil {
		return err
	}
	return nil
}
