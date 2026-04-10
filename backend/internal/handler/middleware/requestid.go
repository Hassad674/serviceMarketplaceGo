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

// GetOrgRole returns the role the authenticated user holds within their
// organization (owner, admin, member, viewer), or "" if none.
func GetOrgRole(ctx context.Context) string {
	if role, ok := ctx.Value(ContextKeyOrgRole).(string); ok {
		return role
	}
	return ""
}
