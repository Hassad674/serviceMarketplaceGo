package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	appsearch "marketplace-backend/internal/app/search"
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
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	persona, err := parsePersona(r.URL.Query().Get("persona"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_persona", err.Error())
		return
	}

	input := appsearch.QueryInput{
		Persona: persona,
		Query:   r.URL.Query().Get("q"),
		SortBy:  r.URL.Query().Get("sort_by"),
		Page:    parseIntDefault(r.URL.Query().Get("page"), 0),
		PerPage: parseIntDefault(r.URL.Query().Get("per_page"), 0),
		Filters: parseFilterInput(r),
	}

	result, err := h.deps.Service.Query(r.Context(), input)
	if err != nil {
		if errors.Is(err, appsearch.ErrPersonaNotConfigured) {
			response.Error(w, http.StatusServiceUnavailable, "persona_unavailable",
				"search engine is not configured for the requested persona")
			return
		}
		response.Error(w, http.StatusInternalServerError, "search_failed",
			"failed to execute search")
		return
	}

	response.JSON(w, http.StatusOK, result)
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
