package middleware

import (
	"net/http"
	"strings"
)

// CORS implements the strict allow-list policy required by SEC-24:
//
//   - `Vary: Origin` is ALWAYS added so caches do not serve a response
//     allowed for one origin to a different origin.
//   - `Access-Control-Allow-Origin` is reflected only when the request's
//     Origin matches the allow-list.
//   - `Access-Control-Allow-Credentials`, `Allow-Methods`, and
//     `Allow-Headers` are emitted ONLY when the origin is allow-listed.
//     Sending these unconditionally on a non-allowlisted origin is a
//     subtle CORS smell: it suggests the request is permitted when it
//     is not, and it gives shared caches mixed signals that can lead
//     to cache poisoning.
//   - `Access-Control-Max-Age: 600` (SEC-36 reduction from the 24h
//     default) so allow-list changes propagate to clients within
//     minutes instead of a full day.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	originsMap := make(map[string]bool)
	for _, o := range allowedOrigins {
		originsMap[strings.TrimSpace(o)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Vary: Origin is required even when no Allow-Origin is
			// emitted — caches must understand that the response varies
			// based on the request Origin so they do not serve a
			// no-CORS response to an allow-listed origin and vice-versa.
			// Use Add (not Set) so we don't clobber a Vary header set
			// upstream by another middleware.
			w.Header().Add("Vary", "Origin")

			origin := r.Header.Get("Origin")
			allowed := origin != "" && originsMap[origin]

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID, X-Auth-Mode")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "600")
			}

			if r.Method == http.MethodOptions {
				// Preflight: end here whether or not the origin was
				// allowed. Browsers see the missing Allow-Origin and
				// block the actual request — exactly the desired
				// behaviour for non-allowlisted origins.
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
