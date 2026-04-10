package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/response"
)

// SessionVersionChecker is the minimal interface the auth middleware
// needs to enforce immediate revocation. Any implementation that can
// return the current users.session_version for a user id satisfies it.
//
// Defined locally here (not in port/) because it is a same-layer
// collaboration between the auth middleware and the user repository,
// not an external port to the outside world.
type SessionVersionChecker interface {
	GetSessionVersion(ctx context.Context, userID uuid.UUID) (int, error)
}

// Auth validates an incoming request's credentials (session cookie
// first, Bearer token second), injects the authenticated user context,
// and enforces the session_version revocation check.
//
// sessionVersions may be nil — in that case the revocation check is
// skipped. This is useful for unit tests and for deployments that
// have not yet wired the revocation system. In production, always
// pass a non-nil checker so role changes take effect immediately.
func Auth(
	tokenService service.TokenService,
	sessionService service.SessionService,
	sessionVersions SessionVersionChecker,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strategy 1: Session cookie (web clients)
			if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
				session, err := sessionService.Get(r.Context(), cookie.Value)
				if err == nil {
					if !checkSessionVersion(r.Context(), sessionVersions, session.UserID, session.SessionVersion) {
						response.Error(w, http.StatusUnauthorized, "session_revoked", "session has been revoked — please sign in again")
						return
					}
					ctx := context.WithValue(r.Context(), ContextKeyUserID, session.UserID)
					ctx = context.WithValue(ctx, ContextKeyRole, session.Role)
					ctx = context.WithValue(ctx, ContextKeyIsAdmin, session.IsAdmin)
					if session.OrganizationID != nil {
						ctx = context.WithValue(ctx, ContextKeyOrganizationID, *session.OrganizationID)
					}
					if session.OrgRole != "" {
						ctx = context.WithValue(ctx, ContextKeyOrgRole, session.OrgRole)
					}
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
						if !checkSessionVersion(r.Context(), sessionVersions, claims.UserID, claims.SessionVersion) {
							response.Error(w, http.StatusUnauthorized, "session_revoked", "token has been revoked — please sign in again")
							return
						}
						ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
						ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
						ctx = context.WithValue(ctx, ContextKeyIsAdmin, claims.IsAdmin)
						if claims.OrganizationID != nil {
							ctx = context.WithValue(ctx, ContextKeyOrganizationID, *claims.OrganizationID)
						}
						if claims.OrgRole != "" {
							ctx = context.WithValue(ctx, ContextKeyOrgRole, claims.OrgRole)
						}
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			response.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		})
	}
}

// checkSessionVersion returns true when the carried version matches
// the current DB value, or when the checker is nil (revocation system
// disabled — acceptable in dev/tests). Any error from the checker is
// treated as a pass to avoid locking users out on transient DB blips;
// the operator can tighten this to fail-closed if needed.
func checkSessionVersion(
	ctx context.Context,
	checker SessionVersionChecker,
	userID uuid.UUID,
	carriedVersion int,
) bool {
	if checker == nil {
		return true
	}
	current, err := checker.GetSessionVersion(ctx, userID)
	if err != nil {
		// Fail-open on transient errors. A persistent inability to reach
		// the source would cause fake "session valid" for everyone, but
		// that's less severe than locking everyone out.
		return true
	}
	return current == carriedVersion
}
