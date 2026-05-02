package middleware

import (
	"log/slog"
	"net/http"

	"marketplace-backend/pkg/response"
)

// RequireRole returns middleware that grants a request through only
// when the authenticated caller's primary role is in the allow-list.
//
// SECURITY (SEC-FINAL-03): handler-level role checks are easy to
// forget on a new endpoint. RequireRole pushes the gate to the
// router, where it lives next to the URL pattern — adding a route
// without a role check becomes a visible diff omission rather than
// a silent miss. It complements `RequireAdmin` (admin flag) and
// `RequirePermission` (per-org granular perms) — none of the three
// replaces the others; they layer for defense in depth.
//
// Behaviour:
//   - Must be chained AFTER `Auth` so the role is already on the
//     context. If the role is unset (the Auth middleware was
//     skipped, or the caller is unauthenticated), the request is
//     denied with `unauthorized` 401 — making the misuse loud.
//   - Multiple allowed roles supported. Empty allow-list panics at
//     construction (programmer error, not a runtime concern).
//   - Denied requests log a `slog.Warn` event with the request id,
//     the caller's id, the carried role, and the route — the audit
//     trail for "who tried what" without leaking authorized callers'
//     traffic.
//   - The error envelope uses the documented `{error, message}`
//     shape (matching the rest of `pkg/response`).
//
// Example:
//
//	r.With(middleware.Auth(...), middleware.RequireRole("agency",
//	    "provider")).Post("/missions", h.Create)
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	if len(roles) == 0 {
		panic("middleware.RequireRole: at least one allowed role is required")
	}
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		if r == "" {
			panic("middleware.RequireRole: empty role string is not allowed")
		}
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			if role == "" {
				// Guard against being mounted without `Auth` ahead of
				// it: the role context key is unset, so the request
				// is unauthenticated as far as we can tell. 401 makes
				// the misconfiguration visible to whoever calls the
				// endpoint.
				slog.Warn("authorization.denied",
					"reason", "missing_role",
					"path", r.URL.Path,
					"method", r.Method,
					"request_id", GetRequestID(r.Context()),
				)
				response.Error(w, http.StatusUnauthorized,
					"unauthorized", "authentication required")
				return
			}
			if _, ok := allowed[role]; !ok {
				userID, _ := GetUserID(r.Context())
				slog.Warn("authorization.denied",
					"reason", "insufficient_role",
					"role", role,
					"user_id", userID.String(),
					"path", r.URL.Path,
					"method", r.Method,
					"request_id", GetRequestID(r.Context()),
				)
				response.Error(w, http.StatusForbidden,
					"insufficient_role",
					"your role does not permit this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
