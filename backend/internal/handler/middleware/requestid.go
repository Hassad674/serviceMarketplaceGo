package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyRole      contextKey = "role"
	ContextKeyIsAdmin   contextKey = "is_admin"
	ContextKeyRequestID contextKey = "request_id"
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
