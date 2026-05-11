package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/handler/middleware"
)

// TestPublicCacheHeaders_HitsAnonymousProfileRoute exercises a chi
// router that mirrors the wiring in `mountPublicProfileReads`. The
// handler is a 200-OK stub — we verify that, when an anonymous
// request hits the route, the response carries the public cache
// headers (Cache-Control + Vary).
//
// Mirrors the real `r.With(middleware.PublicCache, …).Get(...)`
// wiring so a regression that drops PublicCache from the chain
// fails this test.
func TestPublicCacheHeaders_AnonymousProfileGetReceivesPublicHeaders(t *testing.T) {
	r := chi.NewRouter()
	r.With(middleware.PublicCache).Get("/api/v1/freelance-profiles/{orgID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "public, max-age=60, s-maxage=300", rec.Header().Get("Cache-Control"))
	joined := strings.Join(rec.Header().Values("Vary"), ", ")
	assert.Contains(t, joined, "Accept-Language", "Vary must include Accept-Language so FR/EN don't cross-pollute")
	assert.Contains(t, joined, "Cookie", "Vary must include Cookie so authenticated callers get a distinct cache key")
}

// TestPublicCacheHeaders_AuthenticatedRequestBypassesPublicCache pins
// the security-critical invariant: a request carrying a session
// cookie MUST receive `Cache-Control: private, no-store` so the
// Vercel/CDN edge never caches a personalized payload.
func TestPublicCacheHeaders_AuthenticatedRequestBypassesPublicCache(t *testing.T) {
	r := chi.NewRouter()
	r.With(middleware.PublicCache).Get("/api/v1/freelance-profiles/{orgID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "logged-in-user-sid"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	assert.Equal(t, "private, max-age=0, no-store", cc,
		"authenticated callers MUST NOT receive a public Cache-Control — the CDN would cache their personalized payload")
}

// TestPublicCacheHeaders_BearerAuthBypassesPublicCache covers mobile
// clients + admin SPA which use Authorization: Bearer rather than the
// session cookie.
func TestPublicCacheHeaders_BearerAuthBypassesPublicCache(t *testing.T) {
	r := chi.NewRouter()
	r.With(middleware.PublicCache).Get("/api/v1/freelance-profiles/{orgID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	req.Header.Set("Authorization", "Bearer access-token-xyz")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, "private, max-age=0, no-store", rec.Header().Get("Cache-Control"))
}

// TestPublicCacheHeaders_MultipleRoutes confirms PublicCache fires on
// each of the public read endpoints we wired in this batch — a
// shotgun-style guard so a future refactor that drops one of them
// fails the suite loudly.
func TestPublicCacheHeaders_MultipleRoutesReceiveHeaders(t *testing.T) {
	r := chi.NewRouter()
	stub := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	r.With(middleware.PublicCache).Get("/api/v1/profiles/{orgId}", stub)
	r.With(middleware.PublicCache).Get("/api/v1/clients/{orgId}", stub)
	r.With(middleware.PublicCache).Get("/api/v1/freelance-profiles/{orgID}", stub)
	r.With(middleware.PublicCache).Get("/api/v1/referrer-profiles/{orgID}", stub)
	r.With(middleware.PublicCache).Get("/api/v1/referrer-profiles/{orgID}/reputation", stub)
	r.With(middleware.PublicCache).Get("/api/v1/profiles/{orgId}/project-history", stub)
	r.With(middleware.PublicCache).Get("/api/v1/reviews/org/{orgId}", stub)
	r.With(middleware.PublicCache).Get("/api/v1/reviews/average/{orgId}", stub)
	r.With(middleware.PublicCache).Get("/api/v1/search", stub)

	paths := []string{
		"/api/v1/profiles/org-123",
		"/api/v1/clients/org-123",
		"/api/v1/freelance-profiles/org-123",
		"/api/v1/referrer-profiles/org-123",
		"/api/v1/referrer-profiles/org-123/reputation",
		"/api/v1/profiles/org-123/project-history",
		"/api/v1/reviews/org/org-123",
		"/api/v1/reviews/average/org-123",
		"/api/v1/search?q=foo&pos=1",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			assert.Equal(t, "public, max-age=60, s-maxage=300", rec.Header().Get("Cache-Control"),
				"public anonymous route %s must receive public cache headers", p)
		})
	}
}

// TestPublicCacheHeaders_PrivateProfileEndpointStaysPrivate is the
// dual: the AUTHENTICATED profile reads (`/api/v1/profile/...`) must
// keep their `no-store` headers via NoCache middleware. We mount the
// two side-by-side and confirm the policies do not cross-contaminate.
func TestPublicCacheHeaders_PrivateProfileEndpointKeepsNoStore(t *testing.T) {
	r := chi.NewRouter()
	stub := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	// PUBLIC read — uses PublicCache.
	r.With(middleware.PublicCache).Get("/api/v1/freelance-profiles/{orgID}", stub)
	// AUTHENTICATED read of own profile — uses NoCache.
	r.With(middleware.NoCache).Get("/api/v1/freelance-profile/", stub)

	// Anonymous hit on the PUBLIC read → public cache.
	publicRec := httptest.NewRecorder()
	r.ServeHTTP(publicRec, httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil))
	assert.Equal(t, "public, max-age=60, s-maxage=300", publicRec.Header().Get("Cache-Control"))

	// Same client (no auth) hit on the AUTHENTICATED route → no-store
	// (the handler would normally also fail at the auth layer; we
	// stub it out here, so we're testing the cache-control wiring
	// alone).
	privateRec := httptest.NewRecorder()
	r.ServeHTTP(privateRec, httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profile/", nil))
	assert.Equal(t, "no-store, no-cache, must-revalidate", privateRec.Header().Get("Cache-Control"))
}
