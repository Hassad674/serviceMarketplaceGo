package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicCache_AnonymousRequestGetsPublicHeaders(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := PublicCache(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled, "next handler must be called")
	assert.Equal(t, "public, max-age=60, s-maxage=300", rec.Header().Get("Cache-Control"))

	// Vary header may be set twice (Accept-Language + Cookie). Both must appear.
	vary := rec.Header().Values("Vary")
	joined := strings.Join(vary, ", ")
	assert.Contains(t, joined, "Accept-Language", "Vary must include Accept-Language")
	assert.Contains(t, joined, "Cookie", "Vary must include Cookie")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPublicCache_RequestWithSessionCookieGetsPrivateHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := PublicCache(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "some-session"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	assert.Equal(t, "private, max-age=0, no-store", cc,
		"authenticated requests must NOT receive public cache headers")
	// Public Vary should not be added when bypassing
	vary := strings.Join(rec.Header().Values("Vary"), ", ")
	assert.NotContains(t, vary, "Accept-Language",
		"private bypass must not advertise public cache Vary")
}

func TestPublicCache_RequestWithEmptySessionCookieIsTreatedAnonymous(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := PublicCache(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: ""})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "public, max-age=60, s-maxage=300", rec.Header().Get("Cache-Control"),
		"empty cookie value should still be considered anonymous")
}

func TestPublicCache_RequestWithAuthorizationHeaderGetsPrivateHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := PublicCache(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "private, max-age=0, no-store", rec.Header().Get("Cache-Control"),
		"requests carrying a bearer token must bypass public cache")
}

func TestPublicCache_RequestWithRefreshTokenCookieGetsPrivateHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := PublicCache(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profiles/abc", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "rt-value"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "private, max-age=0, no-store", rec.Header().Get("Cache-Control"))
}

func TestPublicCache_DownstreamHandlerCanOverridePublicHeader(t *testing.T) {
	// Some specialized endpoints may want to set a longer cache TTL.
	// PublicCache writes the header BEFORE next.ServeHTTP, so handlers
	// can override by writing later.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=600, s-maxage=3600")
		w.WriteHeader(http.StatusOK)
	})

	handler := PublicCache(next)

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "public, max-age=600, s-maxage=3600", rec.Header().Get("Cache-Control"),
		"handlers must be able to override the default public cache-control")
}

func TestPublicCache_NonGetMethodStillSetsHeader(t *testing.T) {
	// We don't gate on method here — the routing layer is responsible for
	// only mounting PublicCache on GET routes. This test pins current
	// behaviour so a future change explicitly considers method handling.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := PublicCache(next)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/freelance-profiles/abc", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, "public, max-age=60, s-maxage=300", rec.Header().Get("Cache-Control"))
		})
	}
}
