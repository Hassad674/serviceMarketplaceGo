package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
				{"X-Content-Type-Options", "nosniff"},
				{"X-Frame-Options", "DENY"},
				{"X-Xss-Protection", "0"},
				{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
				{"Referrer-Policy", "strict-origin-when-cross-origin"},
				{"Permissions-Policy", "camera=(self), microphone=(self), geolocation=()"},
				// B.3 Cross-Origin-* trio — values are checked verbatim
				// because the policy choice (same-origin / same-site /
				// credentialless) is part of the documented contract
				// and any drift must be a deliberate, reviewed change.
				{"Cross-Origin-Opener-Policy", "same-origin"},
				{"Cross-Origin-Resource-Policy", "same-site"},
				{"Cross-Origin-Embedder-Policy", "credentialless"},
			}
			for _, tc := range tests {
				assert.Equal(t, tc.want, rec.Header().Get(tc.header),
					"header %s must be set to the expected value", tc.header)
			}

			// CSP is asserted separately because the directive list is
			// long and we want to reason about each directive
			// individually rather than against a single 600-char
			// string literal that is brittle to refactor.
			csp := rec.Header().Get("Content-Security-Policy")
			require.NotEmpty(t, csp, "Content-Security-Policy must be set")
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

// TestSecurityHeaders_CSP_HardenedDirectives asserts the B.3
// hardening: every injection-mitigation directive is present with the
// exact value, and the legacy `default-src 'self'` baseline survives.
// Each assertion is its own line so a regression points at the
// missing directive directly rather than at a 600-char diff.
func TestSecurityHeaders_CSP_HardenedDirectives(t *testing.T) {
	csp := captureCSP(t, &config.Config{Env: "production"})
	directives := parseCSP(csp)

	require.NotEmpty(t, directives, "CSP must contain at least one directive")

	// The baseline that existed before B.3 — keep it explicit so a
	// future refactor cannot silently widen `default-src`.
	assert.Equal(t, "'self'", directives["default-src"],
		"default-src must remain 'self'")

	// Injection-mitigation directives added in B.3.
	assert.Equal(t, "'none'", directives["frame-ancestors"],
		"frame-ancestors must be 'none' to defend against clickjacking "+
			"even when X-Frame-Options is ignored by the browser")
	assert.Equal(t, "'self'", directives["base-uri"],
		"base-uri must be 'self' to block <base href> hijacking on injected HTML")
	assert.Equal(t, "'self'", directives["form-action"],
		"form-action must be 'self' to block <form action=evil> on injected HTML")
	assert.Equal(t, "'none'", directives["object-src"],
		"object-src must be 'none' — the app embeds no <object>/<embed>/<applet>")
}

// TestSecurityHeaders_CSP_ThirdPartyWhitelists asserts every vendor the
// front-end actually contacts is whitelisted in the appropriate
// directive. The test is per-vendor rather than per-directive so a
// failure points at the integration that broke (Stripe / LiveKit /
// PostHog / GA / R2) instead of "CSP is wrong".
func TestSecurityHeaders_CSP_ThirdPartyWhitelists(t *testing.T) {
	csp := captureCSP(t, &config.Config{Env: "production"})
	directives := parseCSP(csp)

	t.Run("stripe", func(t *testing.T) {
		// Loader script + iframe + API endpoint all need a slot.
		assert.Contains(t, directives["script-src"], "https://js.stripe.com",
			"script-src must whitelist js.stripe.com (Stripe.js loader)")
		assert.Contains(t, directives["script-src"], "https://*.stripe.com",
			"script-src must whitelist *.stripe.com (Embedded Components)")
		assert.Contains(t, directives["frame-src"], "https://js.stripe.com",
			"frame-src must whitelist js.stripe.com (Checkout iframe)")
		assert.Contains(t, directives["frame-src"], "https://*.stripe.com",
			"frame-src must whitelist *.stripe.com (Embedded iframes)")
		assert.Contains(t, directives["connect-src"], "https://api.stripe.com",
			"connect-src must whitelist api.stripe.com (Stripe API)")
		assert.Contains(t, directives["connect-src"], "https://hooks.stripe.com",
			"connect-src must whitelist hooks.stripe.com (Stripe webhooks)")
	})

	t.Run("livekit", func(t *testing.T) {
		assert.Contains(t, directives["connect-src"], "wss://*.livekit.cloud",
			"connect-src must whitelist wss://*.livekit.cloud (LiveKit signalling)")
		assert.Contains(t, directives["connect-src"], "https://*.livekit.cloud",
			"connect-src must whitelist https://*.livekit.cloud (LiveKit config fetch)")
	})

	t.Run("posthog", func(t *testing.T) {
		assert.Contains(t, directives["script-src"], "https://*.posthog.com",
			"script-src must whitelist *.posthog.com (PostHog SDK)")
		assert.Contains(t, directives["connect-src"], "https://*.posthog.com",
			"connect-src must whitelist *.posthog.com (capture endpoint)")
		assert.Contains(t, directives["connect-src"], "https://*.i.posthog.com",
			"connect-src must whitelist *.i.posthog.com (regional ingest)")
	})

	t.Run("ga4", func(t *testing.T) {
		assert.Contains(t, directives["script-src"], "https://www.googletagmanager.com",
			"script-src must whitelist googletagmanager.com (gtag.js loader)")
		assert.Contains(t, directives["connect-src"], "https://www.google-analytics.com",
			"connect-src must whitelist google-analytics.com (GA4 events)")
		assert.Contains(t, directives["connect-src"], "https://*.analytics.google.com",
			"connect-src must whitelist *.analytics.google.com (regional GA4)")
		assert.Contains(t, directives["img-src"], "https://www.google-analytics.com",
			"img-src must whitelist google-analytics.com (1x1 pixel beacons)")
	})

	t.Run("cloudflare-r2", func(t *testing.T) {
		assert.Contains(t, directives["img-src"], "https://*.r2.cloudflarestorage.com",
			"img-src must whitelist *.r2.cloudflarestorage.com (signed-URL images)")
		assert.Contains(t, directives["img-src"], "https://*.r2.dev",
			"img-src must whitelist *.r2.dev (public bucket images)")
		assert.Contains(t, directives["connect-src"], "https://*.r2.cloudflarestorage.com",
			"connect-src must whitelist *.r2.cloudflarestorage.com (uploads)")
		assert.Contains(t, directives["media-src"], "https://*.r2.dev",
			"media-src must whitelist *.r2.dev (public audio/video)")
	})
}

// TestSecurityHeaders_CrossOriginTrio asserts the COOP/CORP/COEP
// values explicitly. These three are a single security feature
// (cross-origin isolation) and changing any one in isolation breaks
// the guarantee — we want a regression to fail loudly with the
// intended value in the error.
func TestSecurityHeaders_CrossOriginTrio(t *testing.T) {
	tests := []struct {
		header string
		want   string
		why    string
	}{
		{
			"Cross-Origin-Opener-Policy",
			"same-origin",
			"COOP must be same-origin to neutralise window.opener tabnabbing/XS-Leaks",
		},
		{
			"Cross-Origin-Resource-Policy",
			"same-site",
			"CORP must be same-site to allow cross-subdomain loads while blocking arbitrary origins",
		},
		{
			"Cross-Origin-Embedder-Policy",
			"credentialless",
			"COEP must be credentialless — require-corp would break Stripe/LiveKit/PostHog/GA embeds " +
				"because none of those vendors emit a CORP header today",
		},
	}

	csp := captureCSP // ensure compile reference unused vars don't drift
	_ = csp

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := SecurityHeaders(&config.Config{Env: "production"})(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			assert.Equal(t, tt.want, rec.Header().Get(tt.header), tt.why)
		})
	}
}

// TestSecurityHeaders_CrossOriginTrio_NonProduction asserts the
// Cross-Origin-* trio is emitted in EVERY environment, not just
// production. HSTS is the only header that gets the "production-only"
// treatment because it can wedge localhost. The COOP/CORP/COEP trio
// has no such risk and must guard dev/staging too — an attacker who
// finds an XS-Leak in dev can exfiltrate dev credentials.
func TestSecurityHeaders_CrossOriginTrio_NonProduction(t *testing.T) {
	for _, env := range []string{"development", "staging", ""} {
		t.Run(env, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := SecurityHeaders(&config.Config{Env: env})(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, "same-origin", rec.Header().Get("Cross-Origin-Opener-Policy"))
			assert.Equal(t, "same-site", rec.Header().Get("Cross-Origin-Resource-Policy"))
			assert.Equal(t, "credentialless", rec.Header().Get("Cross-Origin-Embedder-Policy"))
		})
	}
}

// captureCSP runs the middleware once and returns the CSP header.
// Centralised so every CSP-targeted test has identical setup and a
// single place to evolve if the constructor signature changes.
func captureCSP(t *testing.T, cfg *config.Config) string {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := SecurityHeaders(cfg)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Header().Get("Content-Security-Policy")
}

// parseCSP splits a CSP header value into a directive -> value map.
// The value is the rest of the directive's source list as a single
// space-separated string. Tests then `assert.Contains(directives[X], origin)`
// to verify membership without locking the origin order.
func parseCSP(header string) map[string]string {
	out := make(map[string]string)
	for _, entry := range strings.Split(header, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		sp := strings.Index(entry, " ")
		if sp < 0 {
			// Directive with no source list (e.g. `upgrade-insecure-requests`).
			out[entry] = ""
			continue
		}
		name := strings.TrimSpace(entry[:sp])
		value := strings.TrimSpace(entry[sp+1:])
		out[name] = value
	}
	return out
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
