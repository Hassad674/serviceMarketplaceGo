package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/config"
)

// TestSecurityHeaders_AllHeadersSet asserts every security header
// documented in backend/CLAUDE.md is present with the exact expected
// value when the middleware sees a typical request. Table-driven over
// HTTP methods so we cover OPTIONS preflights, GET reads and POST
// mutations in one pass — the middleware MUST stamp the headers on
// every response no matter the verb.
func TestSecurityHeaders_AllHeadersSet(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodOptions, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			cfg := &config.Config{Env: "production"}
			handler := SecurityHeaders(cfg)(next)
			req := httptest.NewRequest(method, "/api/v1/health", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			tests := []struct {
				header string
				want   string
			}{
				{"Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"},
				{"X-Content-Type-Options", "nosniff"},
				{"X-Frame-Options", "DENY"},
				{"X-Xss-Protection", "0"},
				{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
				{"Referrer-Policy", "strict-origin-when-cross-origin"},
				{"Permissions-Policy", "camera=(), microphone=(), geolocation=()"},
			}
			for _, tc := range tests {
				assert.Equal(t, tc.want, rec.Header().Get(tc.header),
					"header %s must be set to the expected value", tc.header)
			}
		})
	}
}

// TestSecurityHeaders_HSTSOnlyInProduction guards the rule that HSTS is
// emitted only when Env == "production". In development the header is
// dangerous because a stuck HSTS pin on http://localhost can prevent
// legitimate non-https local development for a year.
func TestSecurityHeaders_HSTSOnlyInProduction(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		wantHSTSSet bool
	}{
		{"production emits HSTS", "production", true},
		{"development omits HSTS", "development", false},
		{"staging omits HSTS", "staging", false},
		{"empty env omits HSTS", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			cfg := &config.Config{Env: tt.env}
			handler := SecurityHeaders(cfg)(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			got := rec.Header().Get("Strict-Transport-Security")
			if tt.wantHSTSSet {
				assert.NotEmpty(t, got, "HSTS must be set in %q", tt.env)
				assert.Contains(t, got, "max-age=31536000")
				assert.Contains(t, got, "includeSubDomains")
			} else {
				assert.Empty(t, got, "HSTS must NOT be set in %q", tt.env)
			}
		})
	}
}

// TestSecurityHeaders_PassesThrough verifies the middleware delegates
// to the next handler and does not consume the response body. A naive
// implementation that wraps the writer could break streaming.
func TestSecurityHeaders_PassesThrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	})

	cfg := &config.Config{Env: "production"}
	handler := SecurityHeaders(cfg)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, called, "next handler must be called")
	assert.Equal(t, http.StatusTeapot, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
	// And security headers still present
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}

// TestSecurityHeaders_NilConfigPanicsCleanly is a guard against future
// refactors that drop the config dependency: passing nil should fail
// fast at wiring time rather than silently downgrading security.
// We use a sub-test with assert.Panics so the regression is loud.
func TestSecurityHeaders_NilConfigPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = SecurityHeaders(nil)
	}, "constructor must panic when given a nil *config.Config")
}
