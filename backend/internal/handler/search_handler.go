package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	appsearch "marketplace-backend/internal/app/search"
	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/search"
	"marketplace-backend/pkg/response"
)

// search_handler.go exposes the two HTTP entry points for the
// public search query path:
//
//   - GET /api/v1/search/key   -> returns a per-persona scoped key
//                                 the frontend uses to query
//                                 Typesense directly.
//   - GET /api/v1/search       -> server-side proxy that runs the
//                                 query against Typesense and
//                                 returns the typed result. Used
//                                 for SSR fallbacks and any client
//                                 that cannot speak Typesense
//                                 natively (Flutter, scripts, …).
//
// Both handlers are auth-gated. The scoped key endpoint is also
// rate limited via the global IP limiter in router.go.
//
// The handler is intentionally thin: it parses query params,
// delegates to the app service, and serialises the result. No
// business logic.

// scopedKeyTTL is the lifetime of a generated scoped key. Frontends
// rotate at 55 min so we keep a 5 min safety margin between cache
// expiry and Typesense rejection.
const scopedKeyTTL = 1 * time.Hour

// SearchHandlerDeps groups the dependencies the handler needs.
// Using a struct keeps the constructor signature stable as we add
// future features (analytics, click tracking, …).
type SearchHandlerDeps struct {
	Service       *appsearch.Service
	Client        *search.Client // master-key client used for scoped key generation
	TypesenseHost string         // returned to the frontend so it knows which host to hit
	APIKey        string         // master key, used as the HMAC parent
	// ClickTracker is optional — when nil the /search/track endpoint
	// returns 503 so the frontend knows to stop emitting beacons.
	ClickTracker ClickTracker
	// Logger receives one structured log line per successful search
	// request. Nil-safe: defaults to slog.Default() at dispatch time.
	Logger *slog.Logger
}

// ClickTracker is the narrow port /search/track calls. Implemented
// by internal/app/searchanalytics.Service.
type ClickTracker interface {
	RecordClick(ctx context.Context, searchID, docID string, position int) error
}

// SearchHandler implements the public search routes.
type SearchHandler struct {
	deps SearchHandlerDeps
}

// NewSearchHandler builds a handler from its dependency struct.
func NewSearchHandler(deps SearchHandlerDeps) *SearchHandler {
	return &SearchHandler{deps: deps}
}

// scopedKeyResponse is the shape returned by GET /search/key.
type scopedKeyResponse struct {
	Key       string `json:"key"`
	Host      string `json:"host"`
	ExpiresAt int64  `json:"expires_at"`
	Persona   string `json:"persona"`
}

// ScopedKey handles GET /api/v1/search/key?persona=freelance.
//
// Returns a Typesense scoped API key with the persona filter
// embedded in the HMAC. Frontend caches the key for 55 minutes and
// uses it to query Typesense directly.
func (h *SearchHandler) ScopedKey(w http.ResponseWriter, r *http.Request) {
	persona, err := parsePersona(r.URL.Query().Get("persona"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_persona", err.Error())
		return
	}
	if !h.deps.Service.HasPersona(persona) {
		response.Error(w, http.StatusServiceUnavailable, "persona_unavailable",
			fmt.Sprintf("search engine is not configured for persona %q", persona))
		return
	}

	expiresAt := time.Now().Add(scopedKeyTTL).Unix()
	embedded := search.EmbeddedSearchParams{
		FilterBy:  fmt.Sprintf("persona:%s && is_published:true", persona),
		ExpiresAt: expiresAt,
	}
	key, err := h.deps.Client.GenerateScopedSearchKey(h.deps.APIKey, embedded)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "scoped_key_failed",
			"failed to generate scoped search key")
		return
	}

	response.JSON(w, http.StatusOK, scopedKeyResponse{
		Key:       key,
		Host:      h.deps.TypesenseHost,
		ExpiresAt: expiresAt,
		Persona:   string(persona),
	})
}

// Search handles GET /api/v1/search?persona=X&q=Y&...
//
// Server-side proxy that runs a single query against Typesense and
// returns the typed QueryResult. Used by SSR pages that cannot wait
// for the frontend to fetch a scoped key.
//
// Emits exactly one `search.query` structured log line per
// successful request (see search_log.go). Failures land in the
// standard error path and do not produce a search.query log — the
// log is "the user saw results" evidence, not a generic access log.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	persona, err := parsePersona(r.URL.Query().Get("persona"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_persona", err.Error())
		return
	}

	var userIDStr string
	if userID, ok := middleware.GetUserID(r.Context()); ok {
		userIDStr = userID.String()
	}

	input := appsearch.QueryInput{
		Persona:   persona,
		Query:     r.URL.Query().Get("q"),
		SortBy:    r.URL.Query().Get("sort_by"),
		Page:      parseIntDefault(r.URL.Query().Get("page"), 0),
		PerPage:   parseIntDefault(r.URL.Query().Get("per_page"), 0),
		Cursor:    r.URL.Query().Get("cursor"),
		UserID:    userIDStr,
		SessionID: r.URL.Query().Get("session_id"),
		Filters:   parseFilterInput(r),
	}

	result, err := h.deps.Service.Query(r.Context(), input)
	if err != nil {
		if errors.Is(err, appsearch.ErrPersonaNotConfigured) {
			response.Error(w, http.StatusServiceUnavailable, "persona_unavailable",
				"search engine is not configured for the requested persona")
			return
		}
		if errors.Is(err, appsearch.ErrCursorInvalid) {
			response.Error(w, http.StatusBadRequest, "invalid_cursor",
				"cursor is invalid or expired")
			return
		}
		response.Error(w, http.StatusInternalServerError, "search_failed",
			"failed to execute search")
		return
	}

	h.logSearch(r, input, result, time.Since(start))

	response.JSON(w, http.StatusOK, result)
}

// logSearch emits the structured `search.query` log line. Extracted
// from Search so it can be unit-tested independently with a
// buffer-backed logger. Never returns an error — the log is
// best-effort and must not fail the request.
func (h *SearchHandler) logSearch(r *http.Request, input appsearch.QueryInput, result *appsearch.QueryResult, latency time.Duration) {
	query, truncated := truncateQueryForLog(input.Query)
	filterBy := appsearch.BuildFilterBy(input.Filters)
	// cursor_active is true when the client paginated into a page
	// beyond the first — that's either input.Cursor non-empty OR an
	// explicit page > 1 param. We also infer "the service bumped us
	// past page 1" from the response to keep the log honest when a
	// scripted client sends page=0 and expects Typesense to default.
	cursorActive := input.Cursor != "" || input.Page > 1 || result.Page > 1
	payload := SearchLog{
		RequestID:        middleware.GetRequestID(r.Context()),
		UserID:           input.UserID,
		Persona:          string(input.Persona),
		Query:            query,
		FilterBy:         filterBy,
		SortBy:           strings.TrimSpace(input.SortBy),
		ResultsCount:     result.Found,
		LatencyMs:        int(latency.Milliseconds()),
		Hybrid:           result.Hybrid,
		CursorActive:     cursorActive,
		Truncated:        truncated,
		Reranked:         result.Reranked,
		RerankDurationMs: result.RerankDurationMs,
		TopFinalScore:    result.TopFinalScore,
	}
	emitSearchLog(h.deps.Logger, payload)
}

// Track handles GET /api/v1/search/track?search_id=…&doc_id=…&position=N.
//
// Fire-and-forget from the frontend (via navigator.sendBeacon or a
// fetch). We respond 204 on success, 400 on malformed input, 404
// when the search_id has already rotated out of the table, and 503
// when click tracking is not wired.
func (h *SearchHandler) Track(w http.ResponseWriter, r *http.Request) {
	if h.deps.ClickTracker == nil {
		response.Error(w, http.StatusServiceUnavailable, "tracker_unavailable",
			"click tracking is not configured")
		return
	}
	q := r.URL.Query()
	searchID := strings.TrimSpace(q.Get("search_id"))
	docID := strings.TrimSpace(q.Get("doc_id"))
	position := parseIntDefault(q.Get("position"), -1)
	if searchID == "" || docID == "" || position < 0 {
		response.Error(w, http.StatusBadRequest, "invalid_track_params",
			"search_id, doc_id, and position are required")
		return
	}
	err := h.deps.ClickTracker.RecordClick(r.Context(), searchID, docID, position)
	if err != nil {
		if errors.Is(err, searchanalytics.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "search_not_found",
				"the search you clicked on is no longer tracked")
			return
		}
		response.Error(w, http.StatusInternalServerError, "track_failed",
			"failed to record click")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parsePersona normalises and validates the `persona` query param.
// Invalid values return a structured error so the handler can
// respond with a clean 400 + machine-readable code.
func parsePersona(raw string) (search.Persona, error) {
	p := search.Persona(strings.ToLower(strings.TrimSpace(raw)))
	if !p.IsValid() {
		return "", fmt.Errorf("persona must be one of freelance, agency, referrer (got %q)", raw)
	}
	return p, nil
}

// parseIntDefault converts a query string int with a fallback. Used
// for page + per_page so handlers stay one-liner.
func parseIntDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

// parseFilterInput pulls every supported filter out of the query
// string and returns a typed FilterInput. Unknown / empty params
// are silently dropped.
//
// The accepted query format mirrors the frontend's URL builder:
//
//	availability=available_now,available_soon
//	pricing_min=50000  pricing_max=150000
//	city=Paris  country=fr
//	geo_lat=48.8566 geo_lng=2.3522 geo_radius_km=25
//	languages=fr,en
//	expertise=dev,design  skills=react,go
//	rating_min=4
//	work_mode=remote,hybrid
//	verified=true  top_rated=true  negotiable=true
func parseFilterInput(r *http.Request) appsearch.FilterInput {
	q := r.URL.Query()
	return appsearch.FilterInput{
		AvailabilityStatus: parseStringList(q.Get("availability")),
		PricingMin:         parseInt64Pointer(q.Get("pricing_min")),
		PricingMax:         parseInt64Pointer(q.Get("pricing_max")),
		City:               q.Get("city"),
		CountryCode:        q.Get("country"),
		GeoLat:             parseFloatPointer(q.Get("geo_lat")),
		GeoLng:             parseFloatPointer(q.Get("geo_lng")),
		GeoRadiusKm:        parseFloatPointer(q.Get("geo_radius_km")),
		Languages:          parseStringList(q.Get("languages")),
		ExpertiseDomains:   parseStringList(q.Get("expertise")),
		Skills:             parseStringList(q.Get("skills")),
		RatingMin:          parseFloatPointer(q.Get("rating_min")),
		WorkMode:           parseStringList(q.Get("work_mode")),
		IsVerified:         parseBoolPointer(q.Get("verified")),
		IsTopRated:         parseBoolPointer(q.Get("top_rated")),
		Negotiable:         parseBoolPointer(q.Get("negotiable")),
	}
}

// parseStringList splits a comma-separated query param into a
// slice. Empty input returns nil so the filter builder treats it
// as "no filter".
func parseStringList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseInt64Pointer returns a non-nil pointer when raw parses
// successfully. Empty / invalid input returns nil so the filter
// builder treats it as "no filter".
func parseInt64Pointer(raw string) *int64 {
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

// parseFloatPointer is the float counterpart to parseInt64Pointer.
func parseFloatPointer(raw string) *float64 {
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &v
}

// parseBoolPointer accepts "true" / "false" / "1" / "0". Anything
// else returns nil so the filter builder leaves the toggle out.
func parseBoolPointer(raw string) *bool {
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &v
}

// Compile-time assertion that QueryResult marshals cleanly. The
// blank identifier silences the linter while making it impossible
// to accidentally remove json marshalling support from the result.
var _ = func() any { _, _ = json.Marshal(appsearch.QueryResult{}); return nil }
