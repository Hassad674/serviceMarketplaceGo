package middleware

import (
	"context"
	"net/http"
	"strings"

	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/response"
)

func Auth(tokenService service.TokenService, sessionService service.SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strategy 1: Session cookie (web clients)
			if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
				session, err := sessionService.Get(r.Context(), cookie.Value)
				if err == nil {
					ctx := context.WithValue(r.Context(), ContextKeyUserID, session.UserID)
					ctx = context.WithValue(ctx, ContextKeyRole, session.Role)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Strategy 2: Bearer token (mobile clients)
			header := r.Header.Get("Authorization")
			if header != "" {
				parts := strings.SplitN(header, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					claims, err := tokenService.ValidateAccessToken(parts[1])
					if err == nil {
						ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
						ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			response.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		})
	}
}
