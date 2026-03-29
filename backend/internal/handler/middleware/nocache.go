package middleware

import "net/http"

// NoCache sets HTTP headers that prevent browsers and intermediary caches from
// storing responses. Applied to authenticated route groups so that a subsequent
// user on the same browser never sees stale data from a previous session.
func NoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}
