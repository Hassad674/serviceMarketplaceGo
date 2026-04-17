package main

import (
	"context"

	appsearch "marketplace-backend/internal/app/search"
	"marketplace-backend/internal/app/searchanalytics"
)

// search_analytics_adapter.go is the tiny glue that forwards events
// from the query service's AnalyticsRecorder port into the
// searchanalytics service. Implemented in cmd/api (not in either
// feature package) because it is specifically a wiring concern —
// both packages stay independent and neither imports the other.

// searchAnalyticsRecorder bridges app/search.AnalyticsRecorder to
// searchanalytics.Service. A nil receiver is a no-op so the caller
// can wire it unconditionally.
type searchAnalyticsRecorder struct {
	svc *searchanalytics.Service
}

// newSearchAnalyticsRecorder returns nil when the service is nil so
// ServiceDeps.Analytics stays nil and the query hot path skips the
// capture step entirely.
func newSearchAnalyticsRecorder(svc *searchanalytics.Service) appsearch.AnalyticsRecorder {
	if svc == nil {
		return nil
	}
	return &searchAnalyticsRecorder{svc: svc}
}

// CaptureSearch forwards the event. We intentionally translate the
// types rather than aliasing because the two packages live on
// opposite sides of the hexagonal boundary.
func (r *searchAnalyticsRecorder) CaptureSearch(ctx context.Context, evt appsearch.AnalyticsEvent) {
	if r == nil || r.svc == nil {
		return
	}
	r.svc.CaptureSearch(ctx, searchanalytics.CaptureEvent{
		SearchID:     evt.SearchID,
		UserID:       evt.UserID,
		SessionID:    evt.SessionID,
		Query:        evt.Query,
		FilterBy:     evt.FilterBy,
		SortBy:       evt.SortBy,
		Persona:      evt.Persona,
		ResultsCount: evt.ResultsCount,
		LatencyMs:    evt.LatencyMs,
	})
}
