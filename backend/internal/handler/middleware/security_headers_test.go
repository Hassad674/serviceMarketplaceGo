package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
				{"Permissions-Policy", "camera=(self), microphone=(self), geolocation=()"},
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

// TestSecurityHeaders_PermissionsPolicy_AllowsMicrophoneCamera is the
// invariant that documents the intent of the Permissions-Policy header
// and prevents the 2026-04-30 regression from coming back.
//
// Background: between 2026-04-30 and 2026-05-07 the header shipped as
// `camera=(), microphone=(), geolocation=()`. An empty allowlist `()`
// blocks getUserMedia for ALL origins (including same-origin) and the
// browser refuses to show the permission prompt, silently breaking
// voice messages AND LiveKit calls. The fix is `(self)` — same-origin
// is allowed, third parties remain blocked.
//
// This test asserts the SEMANTIC invariants — not a string match —
// so a future contributor can re-order directives or add a new one
// without breaking the test, but cannot accidentally re-introduce an
// empty allowlist on microphone or camera.
func TestSecurityHeaders_PermissionsPolicy_AllowsMicrophoneCamera(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &config.Config{Env: "production"}
	handler := SecurityHeaders(cfg)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	policy := rec.Header().Get("Permissions-Policy")
	assert.NotEmpty(t, policy, "Permissions-Policy header must be set")

	directives := parsePermissionsPolicy(t, policy)

	// microphone MUST allow at least same-origin. An empty allowlist
	// `()` would silently block getUserMedia (no browser prompt) and
	// break voice messages + LiveKit calls.
	mic, ok := directives["microphone"]
	assert.True(t, ok, "Permissions-Policy must declare a microphone directive")
	assert.NotEqual(t, "()", mic,
		"microphone allowlist must NOT be empty — `()` blocks getUserMedia "+
			"silently. Use `(self)` to allow same-origin (voice messages, LiveKit).")
	assert.Contains(t, mic, "self",
		"microphone directive must include `self` so getUserMedia works on the app's own origin")

	// camera: same invariant as microphone (LiveKit video).
	cam, ok := directives["camera"]
	assert.True(t, ok, "Permissions-Policy must declare a camera directive")
	assert.NotEqual(t, "()", cam,
		"camera allowlist must NOT be empty — `()` blocks getUserMedia "+
			"silently. Use `(self)` to allow same-origin (LiveKit calls).")
	assert.Contains(t, cam, "self",
		"camera directive must include `self` so getUserMedia works on the app's own origin")

	// geolocation: explicitly closed. The app does not request the
	// user's location and we want to keep the attack surface minimal.
	geo, ok := directives["geolocation"]
	assert.True(t, ok, "Permissions-Policy must declare a geolocation directive")
	assert.Equal(t, "()", geo,
		"geolocation must remain disabled — the app does not use it. "+
			"If you add a feature that needs it, update this test alongside the policy.")
}

// parsePermissionsPolicy splits the header value into directive ->
// allowlist pairs. The header format is a comma-separated list of
// `directive=allowlist` entries where the allowlist is either `*`,
// `()`, or `(origin1 origin2 ...)`. We keep the allowlist verbatim
// (including parentheses) so the test can assert on its full shape.
func parsePermissionsPolicy(t *testing.T, header string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for _, entry := range strings.Split(header, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		eq := strings.Index(entry, "=")
		if eq < 0 {
			t.Fatalf("malformed Permissions-Policy directive %q (missing `=`)", entry)
		}
		name := strings.TrimSpace(entry[:eq])
		value := strings.TrimSpace(entry[eq+1:])
		out[name] = value
	}
	return out
}
