package middleware

import (
	"net/http"

	"marketplace-backend/pkg/response"
)

// RequireAdmin checks if the authenticated user has admin privileges.
// Must be chained AFTER middleware.Auth.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !GetIsAdmin(r.Context()) {
				response.Error(w, http.StatusForbidden, "forbidden", "admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
