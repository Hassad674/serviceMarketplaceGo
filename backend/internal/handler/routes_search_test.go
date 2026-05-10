package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestSearchRoutes_PublicHybridSearchHasNoAuth is the regression
// guard for the bug where the public listing routes (/freelancers,
// /agencies, /referrers) hit `/api/v1/search` and got a 401 because
// the route was wrapped in the JWT auth middleware. The landing
// search bar funnels every visitor through this endpoint, so an
// auth gate here breaks discovery for incognito users.
//
// The shape we test for is the chi middleware chain length: when
// /api/v1/search is in a group with `Auth` + `NoCache`, len(mws) is
// 8 (request_id + http_obs + logger + recovery + security_headers +
// cors + global_ratelimit + auth = 7 globals + per-route NoCache
// minus the auth slot when the gate is dropped → 7). When auth is
// dropped, len(mws) drops by exactly one. We pin the deltas
// explicitly so a future refactor that re-introduces an auth
// middleware on /search will fail this test loudly.
func TestSearchRoutes_PublicHybridSearchHasNoAuth(t *testing.T) {
	r := NewRouter(snapshotDeps())

	mwCounts := collectMWCounts(t, r)

	// /api/v1/search MUST be public — auth removed in the May 2026
	// fix for the public-listing-search-broken regression.
	got, ok := mwCounts["GET /api/v1/search"]
	if !ok {
		t.Fatalf("GET /api/v1/search not registered")
	}
	// /api/v1/search/key and /api/v1/search/track stay protected
	// (they require an authenticated caller).
	keyMW, ok := mwCounts["GET /api/v1/search/key"]
	if !ok {
		t.Fatalf("GET /api/v1/search/key not registered")
	}
	trackMW, ok := mwCounts["GET /api/v1/search/track"]
	if !ok {
		t.Fatalf("GET /api/v1/search/track not registered")
	}

	// /search has exactly ONE less middleware than its sibling
	// /search/key because the auth gate is the only delta.
	assert.Equal(t, keyMW-1, got,
		"/api/v1/search must drop exactly one middleware (the auth gate) compared to /api/v1/search/key — got %d vs %d",
		got, keyMW)
	assert.Equal(t, trackMW-1, got,
		"/api/v1/search must drop exactly one middleware (the auth gate) compared to /api/v1/search/track — got %d vs %d",
		got, trackMW)
	// Sanity floor — the public route still carries the global
	// middleware stack (request id, logger, recovery, security
	// headers, CORS, rate limit) plus the per-route NoCache. If
	// the count drops below 7 we have lost a global guard.
	if got < 7 {
		t.Fatalf("GET /api/v1/search has lost a global middleware — got %d, want >= 7", got)
	}
}

// collectMWCounts walks the chi tree and returns a "METHOD PATH" →
// middleware-chain-length map.
func collectMWCounts(t *testing.T, r chi.Router) map[string]int {
	t.Helper()
	out := map[string]int{}
	err := chi.Walk(r, func(method string, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		key := strings.TrimSpace(method + " " + route)
		out[key] = len(mws)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	return out
}
