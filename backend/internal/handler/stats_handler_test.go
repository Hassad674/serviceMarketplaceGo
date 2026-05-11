package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"

	domainstats "marketplace-backend/internal/domain/stats"
)

// fakeStatsService is the local mock satisfying handler.StatsService.
type fakeStatsService struct {
	GetVisibilityFn func(ctx context.Context, orgID uuid.UUID, days int) (*domainstats.Visibility, error)
	GetKeywordsFn   func(ctx context.Context, orgID uuid.UUID, days, limit int) ([]domainstats.KeywordRow, error)
	GetAppsFn       func(ctx context.Context, orgID uuid.UUID, days int) (*domainstats.ApplicationsTimeSeries, error)
}

func (f *fakeStatsService) GetVisibility(ctx context.Context, orgID uuid.UUID, days int) (*domainstats.Visibility, error) {
	if f.GetVisibilityFn != nil {
		return f.GetVisibilityFn(ctx, orgID, days)
	}
	return nil, nil
}

func (f *fakeStatsService) GetKeywords(ctx context.Context, orgID uuid.UUID, days, limit int) ([]domainstats.KeywordRow, error) {
	if f.GetKeywordsFn != nil {
		return f.GetKeywordsFn(ctx, orgID, days, limit)
	}
	return nil, nil
}

func (f *fakeStatsService) GetEnterpriseApplications(ctx context.Context, orgID uuid.UUID, days int) (*domainstats.ApplicationsTimeSeries, error) {
	if f.GetAppsFn != nil {
		return f.GetAppsFn(ctx, orgID, days)
	}
	return nil, nil
}

// withOrg attaches an org id to the request context so the handler's
// middleware.GetOrganizationID lookup succeeds without going through
// the auth middleware.
func withOrg(r *http.Request, orgID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ContextKeyOrganizationID, orgID)
	return r.WithContext(ctx)
}

func TestStatsHandler_GetVisibility_Unauthorized(t *testing.T) {
	t.Parallel()
	h := handler.NewStatsHandler(&fakeStatsService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/visibility?days=30", nil)
	rec := httptest.NewRecorder()

	h.GetVisibility(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestStatsHandler_GetVisibility_HappyPath(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	called := 0
	svc := &fakeStatsService{GetVisibilityFn: func(_ context.Context, gotOrg uuid.UUID, days int) (*domainstats.Visibility, error) {
		called++
		assert.Equal(t, orgID, gotOrg)
		assert.Equal(t, 30, days)
		return &domainstats.Visibility{
			OrganizationID:    orgID.String(),
			PeriodDays:        domainstats.Period30Days,
			TotalViews:        100,
			UniqueViewers:     42,
			SearchAppearances: 7,
			AvgSearchPosition: 3.5,
			Series: []domainstats.DailyBucket{
				{Date: time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC), Count: 5},
			},
		}, nil
	}}
	h := handler.NewStatsHandler(svc)
	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/visibility?days=30", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetVisibility(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, called)

	var body struct {
		Data struct {
			OrganizationID    string  `json:"organization_id"`
			PeriodDays        int     `json:"period_days"`
			TotalViews        int     `json:"total_views"`
			UniqueViewers     int     `json:"unique_viewers"`
			SearchAppearances int     `json:"search_appearances"`
			AvgSearchPosition float64 `json:"avg_search_position"`
			Series            []struct {
				Date  string `json:"date"`
				Count int    `json:"count"`
			} `json:"series"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, orgID.String(), body.Data.OrganizationID)
	assert.Equal(t, 30, body.Data.PeriodDays)
	assert.Equal(t, 100, body.Data.TotalViews)
	assert.Equal(t, 42, body.Data.UniqueViewers)
	assert.Equal(t, 7, body.Data.SearchAppearances)
	assert.Equal(t, 3.5, body.Data.AvgSearchPosition)
	assert.Len(t, body.Data.Series, 1)
}

func TestStatsHandler_GetVisibility_DefaultsTo30Days(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	captured := 0
	svc := &fakeStatsService{GetVisibilityFn: func(_ context.Context, _ uuid.UUID, days int) (*domainstats.Visibility, error) {
		captured = days
		return &domainstats.Visibility{}, nil
	}}
	h := handler.NewStatsHandler(svc)

	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/visibility", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetVisibility(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 30, captured)

	// Garbage query param falls back to default.
	req2 := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/visibility?days=hello", nil), orgID)
	rec2 := httptest.NewRecorder()
	h.GetVisibility(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, 30, captured)
}

func TestStatsHandler_GetVisibility_Period365(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	captured := 0
	svc := &fakeStatsService{GetVisibilityFn: func(_ context.Context, _ uuid.UUID, days int) (*domainstats.Visibility, error) {
		captured = days
		return &domainstats.Visibility{
			OrganizationID: orgID.String(),
			PeriodDays:     domainstats.Period365Days,
			TotalViews:     1000,
			UniqueViewers:  410,
			Series: []domainstats.DailyBucket{
				{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Count: 3, Unique: 2},
			},
		}, nil
	}}
	h := handler.NewStatsHandler(svc)
	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/visibility?days=365", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetVisibility(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 365, captured)
	// Series exposes both `count` (total) and `unique` (distinct fingerprints) so
	// the frontend can render a two-line chart without an extra request.
	var body struct {
		Data struct {
			Series []struct {
				Date   string `json:"date"`
				Count  int    `json:"count"`
				Unique int    `json:"unique"`
			} `json:"series"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Len(t, body.Data.Series, 1)
	assert.Equal(t, 3, body.Data.Series[0].Count)
	assert.Equal(t, 2, body.Data.Series[0].Unique)
}

func TestStatsHandler_GetEnterpriseApplications_UniqueFallsBackToCount(t *testing.T) {
	t.Parallel()
	// Applications repository never populates Unique on its DailyBuckets
	// because applications cannot be deduplicated by viewer fingerprint.
	// The handler must surface Count as the unique value so the JSON
	// contract stays consistent (every series carries non-zero unique
	// when count > 0).
	orgID := uuid.New()
	svc := &fakeStatsService{GetAppsFn: func(_ context.Context, _ uuid.UUID, _ int) (*domainstats.ApplicationsTimeSeries, error) {
		return &domainstats.ApplicationsTimeSeries{
			OrganizationID: orgID.String(),
			PeriodDays:     domainstats.Period30Days,
			TotalCount:     5,
			Series: []domainstats.DailyBucket{
				{Date: time.Now().UTC().Truncate(24 * time.Hour), Count: 5, Unique: 0},
			},
		}, nil
	}}
	h := handler.NewStatsHandler(svc)
	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/enterprise-applications?days=30", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetEnterpriseApplications(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data struct {
			Series []struct {
				Count  int `json:"count"`
				Unique int `json:"unique"`
			} `json:"series"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Len(t, body.Data.Series, 1)
	assert.Equal(t, 5, body.Data.Series[0].Count)
	assert.Equal(t, 5, body.Data.Series[0].Unique, "applications: unique falls back to count")
}

func TestStatsHandler_GetVisibility_InvalidPeriod(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	svc := &fakeStatsService{GetVisibilityFn: func(context.Context, uuid.UUID, int) (*domainstats.Visibility, error) {
		return nil, domainstats.ErrPeriodInvalid
	}}
	h := handler.NewStatsHandler(svc)
	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/visibility?days=42", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetVisibility(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStatsHandler_GetKeywords_HappyPath(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	captured := 0
	svc := &fakeStatsService{GetKeywordsFn: func(_ context.Context, _ uuid.UUID, days, limit int) ([]domainstats.KeywordRow, error) {
		captured = limit
		assert.Equal(t, 7, days)
		return []domainstats.KeywordRow{
			{Keyword: "go developer", Count: 5, AvgPosition: 2.5},
		}, nil
	}}
	h := handler.NewStatsHandler(svc)

	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/keywords?days=7&limit=20", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetKeywords(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 20, captured)

	var body struct {
		Data []struct {
			Keyword     string  `json:"keyword"`
			Count       int     `json:"count"`
			AvgPosition float64 `json:"avg_position"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "go developer", body.Data[0].Keyword)
}

func TestStatsHandler_GetKeywords_EmptyResults(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	svc := &fakeStatsService{GetKeywordsFn: func(context.Context, uuid.UUID, int, int) ([]domainstats.KeywordRow, error) {
		return []domainstats.KeywordRow{}, nil
	}}
	h := handler.NewStatsHandler(svc)
	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/keywords", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetKeywords(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"data":[]`)
}

func TestStatsHandler_GetEnterpriseApplications_HappyPath(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	svc := &fakeStatsService{GetAppsFn: func(_ context.Context, _ uuid.UUID, days int) (*domainstats.ApplicationsTimeSeries, error) {
		assert.Equal(t, 30, days)
		return &domainstats.ApplicationsTimeSeries{
			OrganizationID: orgID.String(),
			PeriodDays:     domainstats.Period30Days,
			TotalCount:     12,
			Series: []domainstats.DailyBucket{
				{Date: time.Now().UTC().Truncate(24 * time.Hour), Count: 3},
			},
		}, nil
	}}
	h := handler.NewStatsHandler(svc)
	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/enterprise-applications?days=30", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetEnterpriseApplications(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			TotalCount int `json:"total_count"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, 12, body.Data.TotalCount)
}

func TestStatsHandler_GetEnterpriseApplications_NoOrg(t *testing.T) {
	t.Parallel()
	h := handler.NewStatsHandler(&fakeStatsService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/enterprise-applications?days=30", nil)
	rec := httptest.NewRecorder()
	h.GetEnterpriseApplications(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestStatsHandler_GetVisibility_InternalError(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	svc := &fakeStatsService{GetVisibilityFn: func(context.Context, uuid.UUID, int) (*domainstats.Visibility, error) {
		return nil, errors.New("db down")
	}}
	h := handler.NewStatsHandler(svc)
	req := withOrg(httptest.NewRequest(http.MethodGet, "/api/v1/me/stats/visibility?days=30", nil), orgID)
	rec := httptest.NewRecorder()
	h.GetVisibility(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
