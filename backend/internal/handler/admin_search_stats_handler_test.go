package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/handler/middleware"
)

// fakeStatsComputer is the handler-side mock. Records the last input
// so tests can assert parsing worked before the service was called.
type fakeStatsComputer struct {
	stats *searchanalytics.Stats
	err   error
	last  searchanalytics.StatsQuery
	calls int
}

func (f *fakeStatsComputer) Compute(_ context.Context, q searchanalytics.StatsQuery) (*searchanalytics.Stats, error) {
	f.calls++
	f.last = q
	return f.stats, f.err
}

// adminRequest returns an httptest.Request with the is_admin flag
// flipped in the context. Mirrors what middleware.RequireAdmin does
// in production so the handler-side defensive check passes.
func adminRequest(target string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyIsAdmin, true)
	return req.WithContext(ctx)
}

func TestAdminSearchStatsHandler_GetStats_Defaults(t *testing.T) {
	fake := &fakeStatsComputer{
		stats: &searchanalytics.Stats{
			TotalSearches:  42,
			ZeroResults:    3,
			ZeroResultRate: 3.0 / 42.0,
			AvgLatencyMs:   55,
			P95LatencyMs:   120,
			TopQueries:     []searchanalytics.TopQuery{{Query: "react", Count: 2}},
		},
	}
	h := NewAdminSearchStatsHandler(fake)

	req := adminRequest("/api/v1/admin/search/stats")
	rec := httptest.NewRecorder()
	h.GetStats(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, fake.calls)

	// No from/to/persona/limit supplied — service gets a zero-value
	// StatsQuery so it can apply its own defaults.
	assert.True(t, fake.last.From.IsZero())
	assert.True(t, fake.last.To.IsZero())
	assert.Empty(t, fake.last.Persona)
	assert.Zero(t, fake.last.Limit)

	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.EqualValues(t, 42, body["total_searches"])
}

func TestAdminSearchStatsHandler_GetStats_ParsesQueryParams(t *testing.T) {
	fake := &fakeStatsComputer{stats: &searchanalytics.Stats{}}
	h := NewAdminSearchStatsHandler(fake)

	req := adminRequest("/api/v1/admin/search/stats?from=2026-04-01T00:00:00Z&to=2026-04-17T12:00:00Z&persona=freelance&limit=25")
	rec := httptest.NewRecorder()
	h.GetStats(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "freelance", fake.last.Persona)
	assert.Equal(t, 25, fake.last.Limit)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), fake.last.From)
	assert.Equal(t, time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC), fake.last.To)
}

func TestAdminSearchStatsHandler_GetStats_RejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   int
	}{
		{"bad_from", "/stats?from=yesterday", http.StatusBadRequest},
		{"bad_to", "/stats?to=soon", http.StatusBadRequest},
		{"bad_persona", "/stats?persona=hacker", http.StatusBadRequest},
		{"bad_limit_zero", "/stats?limit=0", http.StatusBadRequest},
		{"bad_limit_too_big", "/stats?limit=5000", http.StatusBadRequest},
		{"bad_limit_text", "/stats?limit=abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeStatsComputer{stats: &searchanalytics.Stats{}}
			h := NewAdminSearchStatsHandler(fake)
			req := adminRequest(tt.target)
			rec := httptest.NewRecorder()
			h.GetStats(rec, req)
			assert.Equal(t, tt.want, rec.Code)
			assert.Zero(t, fake.calls, "service must not be called on bad input")
		})
	}
}

func TestAdminSearchStatsHandler_GetStats_Forbidden(t *testing.T) {
	fake := &fakeStatsComputer{stats: &searchanalytics.Stats{}}
	h := NewAdminSearchStatsHandler(fake)
	// NO admin flag in context.
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()
	h.GetStats(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Zero(t, fake.calls)
}

func TestAdminSearchStatsHandler_GetStats_ServicePropagatesInvalidRange(t *testing.T) {
	fake := &fakeStatsComputer{err: searchanalytics.ErrInvalidRange}
	h := NewAdminSearchStatsHandler(fake)
	req := adminRequest("/stats?from=2026-04-17T12:00:00Z&to=2026-04-01T00:00:00Z")
	rec := httptest.NewRecorder()
	h.GetStats(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminSearchStatsHandler_GetStats_ServiceFailure(t *testing.T) {
	fake := &fakeStatsComputer{err: errors.New("db down")}
	h := NewAdminSearchStatsHandler(fake)
	req := adminRequest("/stats")
	rec := httptest.NewRecorder()
	h.GetStats(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestNewAdminSearchStatsHandler_PanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when StatsComputer is nil")
		}
	}()
	NewAdminSearchStatsHandler(nil)
}
