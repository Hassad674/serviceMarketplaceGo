package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// StatsService is the narrow contract the StatsHandler needs from the
// app layer. Defined locally so tests can inject a fake without
// importing the concrete *appstats.Service type.
type StatsService interface {
	GetVisibility(ctx context.Context, orgID uuid.UUID, periodDays int) (*domainstats.Visibility, error)
	GetKeywords(ctx context.Context, orgID uuid.UUID, periodDays, limit int) ([]domainstats.KeywordRow, error)
	GetEnterpriseApplications(ctx context.Context, orgID uuid.UUID, periodDays int) (*domainstats.ApplicationsTimeSeries, error)
}

// StatsHandler exposes the /me/stats/* read endpoints. Each endpoint
// resolves the org id from the auth context, then delegates to the
// app service.
type StatsHandler struct {
	svc StatsService
}

// NewStatsHandler constructs the handler with a service implementation.
func NewStatsHandler(svc StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// GetVisibility renders the totals + daily series.
//
// GET /api/v1/me/stats/visibility?days=7|30|90
func (h *StatsHandler) GetVisibility(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok || orgID == uuid.Nil {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}
	days := parseDaysQuery(r, 30)

	vis, err := h.svc.GetVisibility(r.Context(), orgID, days)
	if err != nil {
		writeStatsError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": serializeVisibility(vis),
	})
}

// GetKeywords renders the top N keywords visitors typed.
//
// GET /api/v1/me/stats/keywords?days=30&limit=10
func (h *StatsHandler) GetKeywords(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok || orgID == uuid.Nil {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}
	days := parseDaysQuery(r, 30)
	limit := parseIntQuery(r, "limit", 10)

	rows, err := h.svc.GetKeywords(r.Context(), orgID, days, limit)
	if err != nil {
		writeStatsError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": serializeKeywords(rows),
	})
}

// GetEnterpriseApplications renders the per-day application counts.
//
// GET /api/v1/me/stats/enterprise-applications?days=30
func (h *StatsHandler) GetEnterpriseApplications(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok || orgID == uuid.Nil {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}
	days := parseDaysQuery(r, 30)

	out, err := h.svc.GetEnterpriseApplications(r.Context(), orgID, days)
	if err != nil {
		writeStatsError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": serializeApplicationsSeries(out),
	})
}

// parseDaysQuery reads ?days=N from the request, validating against
// the supported window sizes. Anything else returns the default
// (which the service then re-validates with a deterministic error).
func parseDaysQuery(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("days")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

// parseIntQuery reads an integer query param with a fallback.
func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// writeStatsError maps domain errors from the stats service onto
// stable HTTP status codes. Pure mapping — kept testable.
func writeStatsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainstats.ErrPeriodInvalid):
		res.Error(w, http.StatusBadRequest, "invalid_period", err.Error())
	case errors.Is(err, domainstats.ErrOrgIDRequired):
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, domainstats.ErrInvalidLimit):
		res.Error(w, http.StatusBadRequest, "invalid_limit", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// serializeVisibility shapes the domain aggregate for JSON output.
// Kept as a free function so the response shape is one place and
// agent B's frontend consumer has a single contract to follow.
func serializeVisibility(v *domainstats.Visibility) map[string]any {
	return map[string]any{
		"organization_id":     v.OrganizationID,
		"period_days":         int(v.PeriodDays),
		"total_views":         v.TotalViews,
		"unique_viewers":      v.UniqueViewers,
		"search_appearances":  v.SearchAppearances,
		"avg_search_position": v.AvgSearchPosition,
		"series":              serializeSeries(v.Series),
	}
}

// serializeApplicationsSeries shapes the applications time-series.
func serializeApplicationsSeries(s *domainstats.ApplicationsTimeSeries) map[string]any {
	return map[string]any{
		"organization_id": s.OrganizationID,
		"period_days":     int(s.PeriodDays),
		"total_count":     s.TotalCount,
		"series":          serializeSeries(s.Series),
	}
}

// serializeSeries converts the daily-bucket slice to the JSON shape.
// Always returns a non-nil slice so the contract is stable.
func serializeSeries(in []domainstats.DailyBucket) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, b := range in {
		out = append(out, map[string]any{
			"date":  b.Date.UTC().Format(time.RFC3339),
			"count": b.Count,
		})
	}
	return out
}

// serializeKeywords flattens KeywordRow slice to the JSON shape.
func serializeKeywords(in []domainstats.KeywordRow) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, k := range in {
		out = append(out, map[string]any{
			"keyword":      k.Keyword,
			"count":        k.Count,
			"avg_position": k.AvgPosition,
		})
	}
	return out
}
