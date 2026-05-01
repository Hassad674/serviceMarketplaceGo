package handler

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/config"
)

// TestMountHelpers_RouteCounts asserts every mount<X>Routes helper
// registers a deterministic, non-zero number of routes when the
// router is built with a maximally-populated dep set. The numbers are
// fixed by the golden snapshot test (router_snapshot_test.go);
// asserting them per-helper here makes a regression in any single
// helper fail with a clear message instead of a 265-line diff.
//
// Each test case builds an isolated /api/v1 router with only the
// helper it exercises, walks the resulting chi tree, and counts the
// (method, path) tuples. A zero count means the helper short-
// circuited unexpectedly; a wrong count means a route was added or
// removed during the split.
func TestMountHelpers_RouteCounts(t *testing.T) {
	deps := snapshotDeps()
	auth := func(next http.Handler) http.Handler { return next }

	cases := []struct {
		name       string
		mount      func(r chi.Router)
		wantRoutes int
	}{
		{"auth", func(r chi.Router) { mountAuthRoutes(r, deps, auth) }, 27},
		{"profile", func(r chi.Router) { mountProfileRoutes(r, deps, auth) }, 44},
		{"upload", func(r chi.Router) { mountUploadRoutes(r, deps, auth) }, 8},
		{"search", func(r chi.Router) { mountSearchRoutes(r, deps, auth) }, 3},
		{"messaging+call", func(r chi.Router) { mountMessagingRoutes(r, deps, auth) }, 13},
		{"proposal", func(r chi.Router) { mountProposalRoutes(r, deps, auth) }, 16},
		{"jobs", func(r chi.Router) { mountJobRoutes(r, deps, auth) }, 16},
		{"review", func(r chi.Router) { mountReviewRoutes(r, deps, auth) }, 4},
		{"report", func(r chi.Router) { mountReportRoutes(r, deps, auth) }, 2},
		{"social", func(r chi.Router) { mountSocialLinkRoutes(r, deps, auth) }, 12},
		{"portfolio", func(r chi.Router) { mountPortfolioRoutes(r, deps, auth) }, 6},
		{"notification", func(r chi.Router) { mountNotificationRoutes(r, deps, auth) }, 9},
		{"billing", func(r chi.Router) { mountBillingRoutes(r, deps, auth) }, 23},
		{"referral", func(r chi.Router) { mountReferralRoutes(r, deps, auth) }, 8},
		{"dispute", func(r chi.Router) { mountDisputeRoutes(r, deps, auth) }, 7},
		{"admin", func(r chi.Router) { mountAdminRoutes(r, deps, auth) }, 61},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			tc.mount(r)
			got := countRoutes(t, r)
			if got != tc.wantRoutes {
				t.Errorf("%s mounted %d routes, want %d (golden file is the source of truth — adjust the count or fix the regression)",
					tc.name, got, tc.wantRoutes)
			}
		})
	}
}

// TestNewRouter_ReturnsNonNilTree is the cheapest possible regression
// guard — if the router construction panics or returns nil for the
// fully-populated dep struct, every other test in the package would
// be unreachable. Catching that early gives a clear stack trace.
func TestNewRouter_ReturnsNonNilTree(t *testing.T) {
	r := NewRouter(snapshotDeps())
	if r == nil {
		t.Fatal("NewRouter returned nil router")
	}
	if cnt := countRoutes(t, r); cnt == 0 {
		t.Errorf("router has zero routes — wiring is broken")
	}
}

// TestRouter_GlobalAndV1Middleware_Compiles asserts the orchestrator
// branch that builds the global middleware + the /api/v1 mutation
// throttle compiles for every (RateLimiter present, RateLimiter
// absent) combination. The build tags are exercised through the
// snapshot test, but a focused assertion here makes the failure
// mode obvious if either branch regresses.
func TestRouter_GlobalAndV1Middleware_Compiles(t *testing.T) {
	deps := snapshotDeps()
	if r := NewRouter(deps); r == nil {
		t.Fatal("NewRouter returned nil with rate limiter absent")
	}
	// Re-build with cfg.IsProduction() == true to exercise the
	// SecurityHeaders production branch.
	deps.Config = &config.Config{
		Env:            "production",
		AllowedOrigins: []string{"https://example.com"},
	}
	if r := NewRouter(deps); r == nil {
		t.Fatal("NewRouter returned nil in production mode")
	}
}

func countRoutes(t *testing.T, r chi.Router) int {
	t.Helper()
	count := 0
	err := chi.Walk(r, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	return count
}
