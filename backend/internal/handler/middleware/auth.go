package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
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
					switch verifySessionVersion(r.Context(), sessionVersions, session.UserID, session.SessionVersion) {
					case sessionVersionMatch:
						// continue
					case sessionVersionUserGone:
						// User row deleted (e.g. operator left their org). The
						// session cookie is still present client-side but the
						// backing account is gone — tell the client their
						// session is invalid so it clears state and redirects
						// to login. See R16 zombie-session fix.
						response.Error(w, http.StatusUnauthorized, "session_invalid", "session is no longer valid — please sign in again")
						return
					case sessionVersionRevoked:
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
						switch verifySessionVersion(r.Context(), sessionVersions, claims.UserID, claims.SessionVersion) {
						case sessionVersionMatch:
							// continue
						case sessionVersionUserGone:
							response.Error(w, http.StatusUnauthorized, "session_invalid", "session is no longer valid — please sign in again")
							return
						case sessionVersionRevoked:
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

// sessionVersionOutcome is the tri-state result of a session_version
// check: the carried version matches the DB, the carried version is
// stale (explicit revoke), or the user row no longer exists at all
// (account deleted — e.g. operator left their org).
type sessionVersionOutcome int

const (
	sessionVersionMatch sessionVersionOutcome = iota
	sessionVersionRevoked
	sessionVersionUserGone
)

// verifySessionVersion returns sessionVersionMatch when the carried
// version matches the current DB value (or when the checker is nil —
// acceptable in dev/tests), sessionVersionUserGone when the backing
// user row has been deleted, and sessionVersionRevoked when the
// version was explicitly bumped. Transient errors (DB unreachable,
// Redis blip) fall through to sessionVersionMatch to avoid locking
// everyone out; the operator can tighten this to fail-closed if needed.
func verifySessionVersion(
	ctx context.Context,
	checker SessionVersionChecker,
	userID uuid.UUID,
	carriedVersion int,
) sessionVersionOutcome {
	if checker == nil {
		return sessionVersionMatch
	}
	current, err := checker.GetSessionVersion(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			// The user row has been deleted — the carried token/session
			// belongs to a zombie account (e.g. operator who left their
			// org). Treat this as a hard session invalidation so the
			// client clears state and logs the user out. R16 fix.
			return sessionVersionUserGone
		}
		// Fail-open on transient errors. A persistent inability to reach
		// the source would cause fake "session valid" for everyone, but
		// that's less severe than locking everyone out.
		return sessionVersionMatch
	}
	if current == carriedVersion {
		return sessionVersionMatch
	}
	return sessionVersionRevoked
}
