package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS_WhitelistedOrigin(t *testing.T) {
	origins := []string{"https://app.example.com", "http://localhost:3000"}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORS_NonWhitelistedOrigin(t *testing.T) {
	origins := []string{"https://app.example.com"}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled, "next must still be called for non-preflight")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"non-whitelisted origin must not receive Allow-Origin header")
	// SEC-24: Allow-Credentials/Methods/Headers/Max-Age MUST be absent
	// for non-allowlisted origins. Emitting them unconditionally is a
	// cache-poisoning hazard and a confusing CORS signal.
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"),
		"non-whitelisted origin must not receive Allow-Credentials")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Methods"),
		"non-whitelisted origin must not receive Allow-Methods")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Headers"),
		"non-whitelisted origin must not receive Allow-Headers")
	assert.Empty(t, rec.Header().Get("Access-Control-Max-Age"),
		"non-whitelisted origin must not receive Max-Age")
}

// TestCORS_VaryOriginAlwaysSet enforces SEC-24's primary rule:
// shared caches need `Vary: Origin` whether or not the origin is
// allow-listed, otherwise an attacker can poison a cached response.
func TestCORS_VaryOriginAlwaysSet(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{"allowlisted origin", "https://app.example.com"},
		{"non-allowlisted origin", "https://evil.com"},
		{"no origin header", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origins := []string{"https://app.example.com"}
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := CORS(origins)(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			vary := rec.Header().Values("Vary")
			assert.Contains(t, vary, "Origin",
				"Vary header MUST contain Origin in every response")
		})
	}
}

// TestCORS_AllowCredentialsConditional locks down the SEC-24 rule that
// Allow-Credentials must only be emitted for allowlisted origins. A
// permissive `*: true` is a CORS misconfiguration that can leak
// authenticated responses cross-origin.
func TestCORS_AllowCredentialsConditional(t *testing.T) {
	origins := []string{"https://app.example.com"}

	tests := []struct {
		name              string
		origin            string
		wantCredsTrue     bool
	}{
		{"allowlisted origin gets credentials", "https://app.example.com", true},
		{"non-allowlisted origin does NOT get credentials", "https://evil.com", false},
		{"empty origin does NOT get credentials", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := CORS(origins)(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			creds := rec.Header().Get("Access-Control-Allow-Credentials")
			if tt.wantCredsTrue {
				assert.Equal(t, "true", creds,
					"allowlisted origin must receive Allow-Credentials: true")
			} else {
				assert.Empty(t, creds,
					"non-allowlisted/empty origin must NOT receive Allow-Credentials")
			}
		})
	}
}

func TestCORS_OptionsPreflight(t *testing.T) {
	origins := []string{"https://app.example.com"}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/profile", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "preflight must not reach the next handler")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	// Audit SEC-36: lowered from 86400 (24h) to 600 (10 min) so allowlist
	// changes propagate to clients within minutes instead of a full day.
	assert.Equal(t, "600", rec.Header().Get("Access-Control-Max-Age"))
}

// Regression: V4 audit found that Idempotency-Key was missing from
// Access-Control-Allow-Headers, silently disabling the F.6+F.7 client
// idempotency wiring on cross-origin POSTs (browser would strip the
// header before the request even left). Lock the allowlist contents.
func TestCORS_AllowHeadersIncludesIdempotencyKey(t *testing.T) {
	origins := []string{"https://app.example.com"}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/proposals", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	for _, expected := range []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID", "X-Auth-Mode"} {
		assert.Contains(t, allowHeaders, expected,
			"Allow-Headers must include %q so the browser does not strip it on preflight", expected)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	origins := []string{"https://app.example.com"}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Origin header set.
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"no origin header means no Allow-Origin in response")
}

func TestCORS_MultipleOrigins(t *testing.T) {
	origins := []string{
		"https://app.example.com",
		"http://localhost:3000",
		"http://localhost:5173",
	}

	tests := []struct {
		name       string
		origin     string
		wantAllow  string
	}{
		{
			name:      "first origin",
			origin:    "https://app.example.com",
			wantAllow: "https://app.example.com",
		},
		{
			name:      "second origin",
			origin:    "http://localhost:3000",
			wantAllow: "http://localhost:3000",
		},
		{
			name:      "third origin",
			origin:    "http://localhost:5173",
			wantAllow: "http://localhost:5173",
		},
		{
			name:      "unknown origin",
			origin:    "https://unknown.com",
			wantAllow: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := CORS(origins)(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantAllow, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
