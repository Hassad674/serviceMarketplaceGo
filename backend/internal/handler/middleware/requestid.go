package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	ContextKeyUserID         contextKey = "user_id"
	ContextKeyRole           contextKey = "role"
	ContextKeyIsAdmin        contextKey = "is_admin"
	ContextKeyRequestID      contextKey = "request_id"
	ContextKeyOrganizationID contextKey = "organization_id"
	ContextKeyOrgRole        contextKey = "org_role"
	// ContextKeyPermissions carries the []string of effective
	// permissions for the authenticated user's org membership, as
	// resolved at login/refresh time with the org's role overrides
	// applied. Empty when the user has no org. The RequirePermission
	// middleware prefers this list over the static role-based lookup
	// so per-org customizations take effect on every endpoint.
	ContextKeyPermissions contextKey = "permissions"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}

		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ContextKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	if id, ok := ctx.Value(ContextKeyUserID).(uuid.UUID); ok {
		return id, true
	}
	return uuid.UUID{}, false
}

func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(ContextKeyRole).(string); ok {
		return role
	}
	return ""
}

func GetIsAdmin(ctx context.Context) bool {
	if isAdmin, ok := ctx.Value(ContextKeyIsAdmin).(bool); ok {
		return isAdmin
	}
	return false
}

// GetOrganizationID returns the organization id of the authenticated user
// for this request, if any. Providers and unauthenticated requests return
// (uuid.Nil, false).
func GetOrganizationID(ctx context.Context) (uuid.UUID, bool) {
	if id, ok := ctx.Value(ContextKeyOrganizationID).(uuid.UUID); ok {
		return id, true
	}
	return uuid.UUID{}, false
}

// MustGetOrgID is the panic-on-missing variant of GetOrganizationID.
// It is the contract used by app services that hit RLS-protected
// reads inside a request whose handler is supposed to have stamped
// the organization context — if the context is missing it is a
// programming bug (handler forgot to enforce auth, or a test
// forgot to populate the context), not a user error.
//
// Production code that legitimately runs without an authenticated
// org (cron schedulers, background workers) MUST instead use the
// system.WithSystemActor / system.IsSystemActor helpers — never
// MustGetOrgID with a fallback.
func MustGetOrgID(ctx context.Context) uuid.UUID {
	id, ok := GetOrganizationID(ctx)
	if !ok || id == uuid.Nil {
		panic("middleware.MustGetOrgID: organization id missing from context — " +
			"handler must enforce Auth + organization gate before calling this code path")
	}
	return id
}

// GetOrgRole returns the role the authenticated user holds within their
// organization (owner, admin, member, viewer), or "" if none.
func GetOrgRole(ctx context.Context) string {
	if role, ok := ctx.Value(ContextKeyOrgRole).(string); ok {
		return role
	}
	return ""
}

// GetPermissions returns the list of effective permissions attached
// to the authenticated user's session, as resolved with the org's
// role overrides at login/refresh time. Returns (nil, false) when
// no permissions were set on the context (e.g. a session created
// before the feature shipped, or a user with no org).
func GetPermissions(ctx context.Context) ([]string, bool) {
	if perms, ok := ctx.Value(ContextKeyPermissions).([]string); ok {
		return perms, true
	}
	return nil, false
}
