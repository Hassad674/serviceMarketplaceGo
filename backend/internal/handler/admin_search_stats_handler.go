// admin_search_stats_handler.go exposes GET /api/v1/admin/search/stats
// — an admin-only view of aggregate search analytics. Surfaces the
// top queries, zero-result rate, and latency percentiles for a time
// range chosen by the caller (default: last 7 days).
//
// The handler is a thin bridge: parse + validate query params,
// delegate to searchanalytics.StatsService, wrap the result in the
// project's standard { data, meta } envelope.
//
// Authorization is enforced at router level (Auth + RequireAdmin).
// We add a second defensive check inside the handler so an operator
// miswiring the route still gets a clean 403 instead of leaking
// aggregate data to a non-admin.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/response"
)

// StatsComputer is the narrow port the handler depends on. Defined
// locally so unit tests can drop a fake in without coupling to the
// whole app-layer service.
type StatsComputer interface {
	Compute(ctx context.Context, q searchanalytics.StatsQuery) (*searchanalytics.Stats, error)
}

// AdminSearchStatsHandler serves the stats endpoint. One field so
// the struct stays trivial to wire.
type AdminSearchStatsHandler struct {
	Stats StatsComputer
}

// NewAdminSearchStatsHandler builds the handler. Panics if the
// StatsComputer is nil because wiring without it is always a bug —
// the alternative is a silent 503 that hides the misconfiguration.
func NewAdminSearchStatsHandler(stats StatsComputer) *AdminSearchStatsHandler {
	if stats == nil {
		panic("handler: AdminSearchStatsHandler requires a non-nil StatsComputer")
	}
	return &AdminSearchStatsHandler{Stats: stats}
}

// GetStats handles GET /api/v1/admin/search/stats?from=…&to=…&persona=…&limit=…
//
// Response shape (standard envelope):
//
//	{
//	  "data": {
//	    "total_searches":    1234,
//	    "zero_result_rate":  0.07,
//	    "avg_latency_ms":    84.2,
//	    "p95_latency_ms":    210.5,
//	    "top_queries":       [...],
//	    "zero_result_queries": [...],
//	    "from": "2026-04-10T00:00:00Z",
//	    "to":   "2026-04-17T00:00:00Z"
//	  },
//	  "meta": { "request_id": "..." }
//	}
func (h *AdminSearchStatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	// Defensive role check. Router middleware should have already
	// enforced RequireAdmin, but we re-check so a misrouted call
	// never leaks aggregates.
	if !middleware.GetIsAdmin(r.Context()) {
		response.Error(w, http.StatusForbidden, "forbidden", "admin role required")
		return
	}

	q, err := parseStatsQuery(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	stats, err := h.Stats.Compute(r.Context(), q)
	if err != nil {
		if errors.Is(err, searchanalytics.ErrInvalidRange) {
			response.Error(w, http.StatusBadRequest, "invalid_range", err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "stats_failed",
			"failed to compute search stats")
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// parseStatsQuery decodes the query-string parameters into a typed
// StatsQuery. Empty strings fall back to the service defaults.
//
// Rules:
//   - `from` / `to` are RFC3339. Either / both can be omitted.
//   - `persona` must be one of freelance/agency/referrer, or empty.
//     Anything else is rejected so a typo does not silently return
//     all personas.
//   - `limit` is 1..100. Invalid limits are rejected (not clamped)
//     so a misbehaving client gets a fast feedback loop; the
//     service's MaxStatsLimit still protects against overflow.
func parseStatsQuery(r *http.Request) (searchanalytics.StatsQuery, error) {
	qp := r.URL.Query()
	query := searchanalytics.StatsQuery{}

	if raw := strings.TrimSpace(qp.Get("from")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return query, errors.New("from must be an RFC3339 timestamp")
		}
		query.From = t
	}
	if raw := strings.TrimSpace(qp.Get("to")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return query, errors.New("to must be an RFC3339 timestamp")
		}
		query.To = t
	}
	if raw := strings.TrimSpace(qp.Get("persona")); raw != "" {
		if !isValidStatsPersona(raw) {
			return query, errors.New("persona must be one of freelance, agency, referrer")
		}
		query.Persona = raw
	}
	if raw := strings.TrimSpace(qp.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > searchanalytics.MaxStatsLimit {
			return query, errors.New("limit must be an integer between 1 and 100")
		}
		query.Limit = n
	}
	return query, nil
}

// isValidStatsPersona keeps the persona list in one place so tests
// can pin the accepted values without reading the handler body.
func isValidStatsPersona(p string) bool {
	switch p {
	case "freelance", "agency", "referrer":
		return true
	default:
		return false
	}
}
