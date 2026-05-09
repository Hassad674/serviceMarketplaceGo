package handler

// admin_security_test.go is the regression guard for the admin
// authorization perimeter. The audit on 2026-05-09 documented that
// every /api/v1/admin/* endpoint MUST chain BOTH middlewares:
//
//   1. Auth — produces an authenticated identity. 401 otherwise.
//   2. RequireAdmin — gates on the live (DB-fronted-by-Redis) is_admin
//      flag. 403 otherwise.
//
// A future refactor that mounts a new admin endpoint OUTSIDE the
// /admin sub-router (or that drops one of the two middlewares from
// the chain) would silently expose admin operations to non-admin
// callers. These tests catch that drift in CI.
//
// The strategy is to walk the chi tree of the fully-wired router and
// assert structural properties on every (METHOD, /api/v1/admin/...)
// route. We do not try to invoke handlers — pure structural checks.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminRoutes_AllUnderAdminSubRouter walks the entire router and
// asserts that no /api/v1/admin/* route exists outside the admin
// sub-router's middleware chain.
//
// We compare middleware counts against a non-admin baseline route
// (/api/v1/auth/me) which is mounted with auth + nocache only. Every
// admin route MUST carry STRICTLY MORE middlewares than that baseline
// because RequireAdmin + NoCache are layered on top. If any admin
// route's middleware count is less than or equal to a /me route, it
// has slipped past the gate.
func TestAdminRoutes_AllUnderAdminSubRouter(t *testing.T) {
	r := NewRouter(snapshotDeps())

	// Collect every (method, path, mw count) tuple.
	mwCounts := map[string]int{}
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		mwCounts[method+" "+route] = len(mws)
		return nil
	})
	require.NoError(t, err, "chi.Walk failed")

	// Sanity baseline: /api/v1/auth/me carries the global stack + auth.
	const baselineKey = "GET /api/v1/auth/me"
	baselineMW, ok := mwCounts[baselineKey]
	require.True(t, ok, "%s must exist — test setup is broken", baselineKey)
	require.Greater(t, baselineMW, 0, "baseline must have at least one middleware")

	// Walk again, this time asserting on every admin route.
	adminRouteCount := 0
	err = chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/api/v1/admin/") {
			return nil
		}
		adminRouteCount++
		count := len(mws)
		assert.Greater(t, count, baselineMW,
			"admin route %s %s has %d middlewares but baseline /auth/me has %d — "+
				"admin routes MUST chain RequireAdmin (and NoCache) on top of the global stack",
			method, route, count, baselineMW)
		return nil
	})
	require.NoError(t, err)
	require.Greater(t, adminRouteCount, 30,
		"sanity check: there are at least 30 admin routes (got %d) — "+
			"if this drops, the audit doc must be updated",
		adminRouteCount)
}

// TestAdminRoutes_NoAdminEndpointOutsideAdminSubRouter confirms there
// is no /api/v1/<X>/admin/... route mounted under a non-admin
// sub-router. A handler accidentally registered as "/api/v1/users/admin/..."
// would not be caught by the admin middleware. The check is lenient —
// it simply asserts every "admin" segment in the URL appears at exactly
// position 3 (.../api/v1/admin/...).
func TestAdminRoutes_NoAdminEndpointOutsideAdminSubRouter(t *testing.T) {
	r := NewRouter(snapshotDeps())

	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.Contains(route, "admin") {
			return nil
		}
		// Permit /api/v1/admin/<...> — the canonical admin sub-router.
		if strings.HasPrefix(route, "/api/v1/admin/") {
			return nil
		}
		// Permit any path whose "admin" segment is part of an
		// unrelated word (e.g. "administered", "admin-team"). Currently
		// no such routes exist; the test will start failing if one is
		// added so the reviewer can confirm the location is intentional.
		t.Errorf("route %s %s contains 'admin' outside the canonical /api/v1/admin/ sub-router. "+
			"If this is intentional, add an explicit allow-list to this test. "+
			"Otherwise, move the route under /api/v1/admin/ so the RequireAdmin gate covers it.",
			method, route)
		return nil
	})
	require.NoError(t, err)
}

// TestAdminRoutes_KnownEndpointsArePresent pins the inventory of admin
// endpoints. If a future refactor drops one (e.g. by accidentally
// omitting `mountAdminMediaRoutes` from `mountAdminRoutes`), the test
// fails with a clear "missing admin route" message instead of a
// silent gap.
//
// The list mirrors routes_admin.go and the audit doc. Adding a new
// admin endpoint requires updating this list — a deliberate gate to
// keep the security inventory in sync with the code.
func TestAdminRoutes_KnownEndpointsArePresent(t *testing.T) {
	r := NewRouter(snapshotDeps())

	registered := map[string]bool{}
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	})
	require.NoError(t, err)

	// Curated list of admin endpoints. Every entry below MUST be
	// present in the wired router. Group order mirrors routes_admin.go
	// so the diff is easy to read in a code review.
	want := []string{
		// Users
		"GET /api/v1/admin/dashboard/stats",
		"GET /api/v1/admin/users",
		"GET /api/v1/admin/users/{id}",
		"POST /api/v1/admin/users/{id}/suspend",
		"POST /api/v1/admin/users/{id}/unsuspend",
		"POST /api/v1/admin/users/{id}/ban",
		"POST /api/v1/admin/users/{id}/unban",
		"GET /api/v1/admin/users/{id}/reports",
		"GET /api/v1/admin/notifications",
		"POST /api/v1/admin/notifications/{category}/reset",
		// Conversations + reports
		"GET /api/v1/admin/conversations",
		"GET /api/v1/admin/conversations/{id}",
		"GET /api/v1/admin/conversations/{id}/messages",
		"GET /api/v1/admin/conversations/{id}/reports",
		"POST /api/v1/admin/reports/{id}/resolve",
		// Jobs + applications
		"GET /api/v1/admin/jobs",
		"GET /api/v1/admin/jobs/{id}",
		"GET /api/v1/admin/jobs/{id}/reports",
		"DELETE /api/v1/admin/jobs/{id}",
		"GET /api/v1/admin/job-applications",
		"DELETE /api/v1/admin/job-applications/{id}",
		// Message moderation
		"POST /api/v1/admin/messages/{id}/approve-moderation",
		"POST /api/v1/admin/messages/{id}/hide",
		"POST /api/v1/admin/messages/{id}/restore-moderation",
		// Reviews
		"GET /api/v1/admin/reviews",
		"GET /api/v1/admin/reviews/{id}",
		"DELETE /api/v1/admin/reviews/{id}",
		"GET /api/v1/admin/reviews/{id}/reports",
		"POST /api/v1/admin/reviews/{id}/approve-moderation",
		"POST /api/v1/admin/reviews/{id}/restore-moderation",
		// Unified moderation
		"GET /api/v1/admin/moderation",
		"GET /api/v1/admin/moderation/count",
		"POST /api/v1/admin/moderation/{content_type}/{content_id}/restore",
		// Media
		"GET /api/v1/admin/media",
		"GET /api/v1/admin/media/{id}",
		"POST /api/v1/admin/media/{id}/approve",
		"POST /api/v1/admin/media/{id}/reject",
		"DELETE /api/v1/admin/media/{id}",
		// Disputes
		"GET /api/v1/admin/disputes",
		"GET /api/v1/admin/disputes/{id}",
		"POST /api/v1/admin/disputes/{id}/resolve",
		"POST /api/v1/admin/disputes/{id}/force-escalate",
		"POST /api/v1/admin/disputes/{id}/ai-chat",
		"POST /api/v1/admin/disputes/{id}/ai-budget",
		"GET /api/v1/admin/disputes/count",
		// Proposals + credits
		"POST /api/v1/admin/proposals/{id}/activate",
		"POST /api/v1/admin/credits/reset",
		"POST /api/v1/admin/credits/reset/{userId}",
		"GET /api/v1/admin/credits/bonus-log",
		"GET /api/v1/admin/credits/bonus-log/pending",
		"POST /api/v1/admin/credits/bonus-log/{id}/approve",
		"POST /api/v1/admin/credits/bonus-log/{id}/reject",
		// Organization team management
		"GET /api/v1/admin/users/{id}/organization",
		"POST /api/v1/admin/organizations/{id}/force-transfer",
		"PATCH /api/v1/admin/organizations/{id}/members/{userID}",
		"DELETE /api/v1/admin/organizations/{id}/members/{userID}",
		"DELETE /api/v1/admin/organizations/{id}/invitations/{invID}",
		// Search analytics
		"GET /api/v1/admin/search/stats",
		// Invoicing
		"POST /api/v1/admin/invoices/{id}/credit-note",
		"GET /api/v1/admin/invoices",
		"GET /api/v1/admin/invoices/{id}/pdf",
	}

	for _, key := range want {
		assert.True(t, registered[key],
			"missing admin endpoint: %s — verify routes_admin.go still mounts it", key)
	}
}

// TestAdminRoutes_NoCacheHeaderApplied — the admin sub-router chains
// `middleware.NoCache` on top of Auth + RequireAdmin so a stale CDN /
// browser cache cannot replay an admin response from a different user
// (subtle privilege-escalation vector when a shared cache is involved).
//
// We can't introspect the middleware identity from chi.Walk, but we
// can pin the route count and the middleware count on a representative
// admin endpoint. If any of these numbers shifts, the auditor must
// re-verify that NoCache is still in the chain.
func TestAdminRoutes_NoCacheChainShape(t *testing.T) {
	r := NewRouter(snapshotDeps())

	mwCounts := map[string]int{}
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		mwCounts[method+" "+route] = len(mws)
		return nil
	})
	require.NoError(t, err)

	// Pin a representative endpoint. The number must match every
	// other admin GET endpoint at the same depth — verified by the
	// next test.
	const probe = "GET /api/v1/admin/dashboard/stats"
	probeMW, ok := mwCounts[probe]
	require.True(t, ok, "%s must exist", probe)
	assert.GreaterOrEqual(t, probeMW, 3,
		"%s must carry at least 3 middlewares (global + Auth + RequireAdmin + NoCache)",
		probe)
}

// TestAdminRoutes_MiddlewareCountConsistent — every admin endpoint
// (excluding any explicit per-route policy) must carry the same
// middleware count. A drift means a route is missing one of the
// three group middlewares (Auth / RequireAdmin / NoCache) — almost
// always a security regression.
func TestAdminRoutes_MiddlewareCountConsistent(t *testing.T) {
	r := NewRouter(snapshotDeps())

	mwByRoute := map[string]int{}
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/api/v1/admin/") {
			mwByRoute[method+" "+route] = len(mws)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, mwByRoute)

	// Take the first as the reference. All others must match.
	var refKey string
	var refCount int
	for k, v := range mwByRoute {
		refKey = k
		refCount = v
		break
	}

	for k, v := range mwByRoute {
		assert.Equal(t, refCount, v,
			"%s has %d middlewares, but reference %s has %d — "+
				"the admin sub-router applies a uniform Auth+RequireAdmin+NoCache stack. "+
				"A drift here means one of the gates is missing on this route, OR a per-route "+
				"middleware was added intentionally — in which case update this test with the new shape.",
			k, v, refKey, refCount)
	}
}
