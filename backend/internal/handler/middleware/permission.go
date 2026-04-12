package middleware

import (
	"net/http"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/pkg/response"
)

// RequirePermission checks whether the authenticated user's org role
// grants the given permission. Must be chained AFTER middleware.Auth
// which populates ContextKeyOrgRole.
//
// The check is purely in-memory: orgRole is already baked into the
// JWT/session at login time, and HasPermission is a map lookup.
// Zero database round-trips on the hot path.
//
// Users with no organization (orgRole == "") are denied — org-bound
// features require an org context. Solo providers who haven't been
// provisioned an org yet receive a dedicated error code so the
// frontend can show a targeted message.
func RequirePermission(perm organization.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgRole := GetOrgRole(r.Context())
			if orgRole == "" {
				response.Error(w, http.StatusForbidden, "no_organization", "you must be part of an organization to perform this action")
				return
			}

			// Fast path: session carries the pre-resolved permission set
			// (static defaults + per-org overrides applied at login).
			// This is the path that honors per-org customization.
			if perms, ok := GetPermissions(r.Context()); ok {
				wanted := string(perm)
				for _, p := range perms {
					if p == wanted {
						next.ServeHTTP(w, r)
						return
					}
				}
				response.Error(w, http.StatusForbidden, "permission_denied", "you do not have permission to perform this action")
				return
			}

			// Legacy fallback: no permissions list on the session (e.g.
			// a cookie created before R17 shipped, or a unit test that
			// injects only the OrgRole into context). Use the static
			// role-based lookup so existing clients keep working until
			// they refresh.
			if !organization.HasPermission(organization.Role(orgRole), perm) {
				response.Error(w, http.StatusForbidden, "permission_denied", "you do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
