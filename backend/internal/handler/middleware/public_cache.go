package middleware

import "net/http"

// PublicCache sets HTTP cache headers for public, anonymous read endpoints so
// the Vercel/CDN edge can serve cached responses on subsequent hits without
// reaching the backend.
//
// Strategy:
//   - max-age=60      → browser caches for 60s.
//   - s-maxage=300    → shared caches (Vercel CDN) cache for 5 min.
//   - Vary: Accept-Language, Cookie → prevents cross-locale or cross-user
//     bleed. Authenticated requests (with a session cookie or Authorization
//     header) get a distinct cache key — and in fact bypass the public
//     cache entirely (see below), so they never poison the public CDN.
//
// Authenticated routes (which run after Auth middleware) MUST NOT chain
// through PublicCache — they should use NoCache. PublicCache is intended for
// strictly anonymous read endpoints (public profiles, search, public review
// lists).
//
// If a downstream handler decides the resource is private/personalized, it
// may overwrite Cache-Control — the last write wins because handlers run
// before the response is flushed.
func PublicCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Defensive: if the caller already presents a session cookie or
		// bearer token, skip public caching entirely so we never leak a
		// user-specific payload onto the CDN edge — even if the handler
		// happens to serve identical content for anonymous and
		// authenticated callers.
		if hasSessionCredential(r) {
			w.Header().Set("Cache-Control", "private, max-age=0, no-store")
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=300")
		// Vary on Accept-Language so localized pages (FR / EN) do not
		// cross-pollute. Vary on Cookie so an authenticated request that
		// somehow reaches this path gets a separate cache entry.
		w.Header().Add("Vary", "Accept-Language")
		w.Header().Add("Vary", "Cookie")
		next.ServeHTTP(w, r)
	})
}

// hasSessionCredential reports whether the request carries any credential
// used by the backend or frontend to identify a session — a session cookie
// (web clients), a legacy refresh/access-token cookie, or an Authorization
// header (mobile clients + admin SPA).
func hasSessionCredential(r *http.Request) bool {
	for _, name := range []string{"session_id", "refresh_token", "access_token"} {
		if c, err := r.Cookie(name); err == nil && c.Value != "" {
			return true
		}
	}
	if r.Header.Get("Authorization") != "" {
		return true
	}
	return false
}
