package middleware

import (
	"net/http"
	"strings"

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
//
// B.3 (2026-05-10): hardening added on top of the legacy header set to
// raise the floor against tabnabbing/XS-Leaks (Cross-Origin-* trio),
// clickjacking via embed (frame-ancestors 'none'), <base> hijacking
// (base-uri 'self') and form-action injection (form-action 'self').
// The CSP now explicitly whitelists every third-party origin the web
// front-end actually contacts — Stripe (js.stripe.com + *.stripe.com),
// LiveKit (wss://*.livekit.cloud), PostHog (*.posthog.com,
// *.i.posthog.com), Google Analytics 4 (googletagmanager.com,
// google-analytics.com, *.analytics.google.com) and Cloudflare R2
// (*.r2.cloudflarestorage.com, *.r2.dev). The Next.js layer also
// emits a CSP — these two converge to the same whitelist so a request
// blocked at one layer is also blocked at the other.
//
// COEP rationale: we ship `Cross-Origin-Embedder-Policy: credentialless`
// instead of `require-corp`. require-corp would force every cross-origin
// resource (Stripe iframes, LiveKit WebSocket, PostHog/GA scripts, R2
// images) to opt-in via CORP — none of those vendors emit CORP today.
// `credentialless` keeps the cross-origin isolation guarantee for the
// document's own context (process isolation, SharedArrayBuffer eligible)
// while letting third-party subresources load without credentials. This
// is the same trade-off Stripe and LiveKit recommend in their CSP docs.
func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	if cfg == nil {
		panic("SecurityHeaders: config must not be nil")
	}
	isProd := cfg.IsProduction()
	csp := buildCSP()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			// Disable the legacy XSS auditor — modern browsers rely on
			// CSP and the auditor itself has been a source of XSS bugs.
			h.Set("X-XSS-Protection", "0")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			// Microphone + camera are allowed for same-origin only (used by
			// voice messages and LiveKit calls). Geolocation stays fully
			// disabled — the app does not use it. An empty allowlist for
			// microphone/camera (the original `()` value) silently blocks
			// getUserMedia without showing the browser permission prompt,
			// which broke voice messages and call audio in 2026-04-30.
			h.Set("Permissions-Policy", "camera=(self), microphone=(self), geolocation=()")

			// Cross-Origin-* trio — defense against tabnabbing/XS-Leaks.
			// COOP "same-origin": isolates this browsing context group
			// from cross-origin openers, neutralising window.opener
			// reverse-tabnabbing attacks.
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			// CORP "same-site": permits cross-subdomain loads
			// (api.example.com -> assets.example.com) while still
			// blocking arbitrary third-party origins from embedding
			// our responses.
			h.Set("Cross-Origin-Resource-Policy", "same-site")
			// COEP "credentialless": gives us cross-origin isolation
			// (eligibility for SharedArrayBuffer / high-resolution
			// timers) without breaking Stripe / LiveKit / PostHog / GA
			// embeds, which do not emit CORP. See package doc for the
			// full rationale.
			h.Set("Cross-Origin-Embedder-Policy", "credentialless")

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

// buildCSP assembles the Content-Security-Policy header value. The
// directives mirror web/src/shared/lib/csp.ts so both layers agree on
// the same allow-list — a request blocked by one layer is also blocked
// by the other. Static origins live in named constants to keep the
// directive list short and reviewable.
func buildCSP() string {
	directives := []string{
		"default-src 'self'",
		"script-src 'self' " + joinOrigins(stripeOrigins, posthogOrigins, ga4ScriptOrigins),
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob: " + joinOrigins(r2Origins, stripeOrigins, ga4ImgOrigins),
		"media-src 'self' blob: " + joinOrigins(r2Origins),
		"font-src 'self' data:",
		"connect-src 'self' " + joinOrigins(
			stripeOrigins,
			posthogOrigins,
			ga4ConnectOrigins,
			livekitOrigins,
			r2Origins,
		),
		"frame-src " + joinOrigins(stripeOrigins),
		// frame-ancestors 'none' is the modern equivalent of
		// X-Frame-Options: DENY. Both ship — XFO for legacy browsers
		// that ignore CSP, frame-ancestors for everything else.
		"frame-ancestors 'none'",
		"object-src 'none'",
		// base-uri 'self' prevents an attacker who can inject HTML from
		// rewriting <base href> to redirect every relative URL to a
		// hostile origin.
		"base-uri 'self'",
		// form-action 'self' stops <form action="https://evil"> when an
		// attacker manages to inject a form into our DOM.
		"form-action 'self'",
	}
	return strings.Join(directives, "; ")
}

// joinOrigins flattens N origin slices into a single space-separated
// string. Each slice is treated as a logical group (Stripe, PostHog,
// LiveKit…) so the directive composition reads top-down without
// repeating spread literals at each call site.
func joinOrigins(groups ...[]string) string {
	total := 0
	for _, g := range groups {
		total += len(g)
	}
	out := make([]string, 0, total)
	for _, g := range groups {
		out = append(out, g...)
	}
	return strings.Join(out, " ")
}

// stripeOrigins covers the three host buckets Stripe.js + Stripe
// Embedded actually contacts: the loader (js.stripe.com), the API
// (api.stripe.com / hooks.stripe.com / m.stripe.com), and the
// Embedded Components iframes that resolve under arbitrary
// *.stripe.com subdomains.
var stripeOrigins = []string{
	"https://js.stripe.com",
	"https://api.stripe.com",
	"https://hooks.stripe.com",
	"https://*.stripe.com",
}

// livekitOrigins is the WebSocket origin family used for the LiveKit
// signalling channel. The deployment lives on *.livekit.cloud; we
// also whitelist the https variant so the SDK can fetch its config.
var livekitOrigins = []string{
	"wss://*.livekit.cloud",
	"https://*.livekit.cloud",
}

// posthogOrigins covers both the eu/us hosts and the regional CDN
// that serves the SDK chunks (recorder, decide, array endpoint).
var posthogOrigins = []string{
	"https://*.posthog.com",
	"https://*.i.posthog.com",
}

// ga4ScriptOrigins is the gtag.js loader. ga4ConnectOrigins covers
// the analytics POST endpoints; ga4ImgOrigins covers the 1x1 pixel
// fallback that ships when the JS POST fails.
var ga4ScriptOrigins = []string{
	"https://www.googletagmanager.com",
}

var ga4ConnectOrigins = []string{
	"https://www.google-analytics.com",
	"https://*.analytics.google.com",
	"https://*.googletagmanager.com",
}

var ga4ImgOrigins = []string{
	"https://www.google-analytics.com",
	"https://*.analytics.google.com",
}

// r2Origins is the Cloudflare R2 bucket family. Public reads use
// *.r2.dev; signed URLs use *.r2.cloudflarestorage.com.
var r2Origins = []string{
	"https://*.r2.cloudflarestorage.com",
	"https://*.r2.dev",
}
