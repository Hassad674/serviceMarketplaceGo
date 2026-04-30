package middleware

import (
	"net/http"

	"marketplace-backend/internal/config"
)

// SecurityHeaders adds the OWASP-recommended security headers to every
// response. The middleware is environment-aware: HSTS is only emitted in
// production because pinning HSTS on a developer's localhost can wedge
// non-https local development for the full max-age (a year).
//
// Header values match the spec in backend/CLAUDE.md ("HTTP security
// headers middleware") so a future audit can diff the runtime output
// against the documented reference without ambiguity.
//
// The constructor takes the typed *config.Config (not the env string)
// so callers can never fall back to a default of "production" — passing
// nil panics at wiring time, which is the loud failure mode we want.
//
// SEC-03 (audit 2026-04-29): this middleware closes the gap from the
// 2026-03-30 audit that flagged "SecurityHeaders shipped" as false. The
// middleware MUST run after Recovery (so even panics get headers) and
// before CORS (so CORS responses also carry the headers).
func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	if cfg == nil {
		panic("SecurityHeaders: config must not be nil")
	}
	isProd := cfg.IsProduction()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			// Disable the legacy XSS auditor — modern browsers rely on
			// CSP and the auditor itself has been a source of XSS bugs.
			h.Set("X-XSS-Protection", "0")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			if isProd {
				// 1 year HSTS with subdomain coverage. We deliberately
				// do NOT include "preload" — that requires a separate
				// policy review and an entry in the browser preload
				// list, neither of which is appropriate for an MVP.
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
