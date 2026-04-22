package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
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

// OrgOverridesResolver returns the per-org role-permission overrides
// used to compute the caller's effective permissions on every request.
// Keeping it narrow (just the overrides, not the full org row) means
// the auth middleware does not need to know the organization entity's
// shape and the implementation is free to front a DB query with a
// Redis / in-process cache without changing this contract.
//
// Return value semantics:
//   - A nil map is acceptable — the domain's EffectivePermissionsFor
//     treats it as "no overrides, fall back to the static defaults",
//     which is the desired behaviour for brand-new orgs.
//   - An error fails open: the middleware keeps the session's cached
//     perms list. We'd rather briefly trust the snapshot than lock
//     every user out on a transient DB blip.
type OrgOverridesResolver interface {
	GetRoleOverrides(ctx context.Context, orgID uuid.UUID) (organization.RoleOverrides, error)
}

// Auth validates an incoming request's credentials (session cookie
// first, Bearer token second), injects the authenticated user context,
// and enforces the session_version revocation check.
//
// sessionVersions may be nil — in that case the revocation check is
// skipped. This is useful for unit tests and for deployments that
// have not yet wired the revocation system. In production, always
// pass a non-nil checker so role changes take effect immediately.
//
// overridesResolver may be nil — in that case the middleware falls
// back to the perms snapshot baked into the session/JWT at login
// time. In production, always pass a non-nil resolver so new
// permissions added to the catalog after a deploy propagate to every
// live session without requiring everyone to log out.
func Auth(
	tokenService service.TokenService,
	sessionService service.SessionService,
	sessionVersions SessionVersionChecker,
	overridesResolver OrgOverridesResolver,
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
					ctx = injectLivePermissions(
						ctx, overridesResolver,
						session.OrganizationID, session.OrgRole, session.Permissions,
					)
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
						ctx = injectLivePermissions(
							ctx, overridesResolver,
							claims.OrganizationID, claims.OrgRole, claims.Permissions,
						)
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

// injectLivePermissions populates ContextKeyPermissions with the set of
// permissions effective for the caller's role + the org's live role
// overrides, computed fresh on every request.
//
// Why live instead of the snapshot baked into the session/JWT at login:
// the session stores the permission list verbatim, so any permission
// added to the static catalogue after a session was created never
// reaches the middleware unless the user re-logs-in. Existing users
// kept getting 403s on brand-new features (e.g. `org_client_profile.edit`
// after migration 114). Resolving against the catalogue on every
// request closes that drift at a cost of one cheap read per request
// (indexed lookup of `organizations.role_overrides`, Redis-cachable by
// the adapter) — the correctness gain far outweighs the µs cost.
//
// Fallbacks, in order:
//  1. No resolver wired OR session carries no org / role → use the
//     snapshot as-is (or leave the key unset if the snapshot is empty).
//     Keeps unit tests and legacy deployments working unchanged.
//  2. Resolver returns an error → fail open: trust the snapshot rather
//     than lock the user out on a transient DB blip. The error is
//     logged so operators can notice a persistent outage.
//  3. Role string on the session is unknown to the domain → use the
//     snapshot. Should never happen in production (we only sign known
//     roles into the session) but the safe default matters.
func injectLivePermissions(
	ctx context.Context,
	resolver OrgOverridesResolver,
	orgID *uuid.UUID,
	orgRole string,
	snapshot []string,
) context.Context {
	// Nothing to compute: no resolver, no org, or no role → keep the
	// snapshot if any so the RequirePermission middleware still has
	// something to match against.
	if resolver == nil || orgID == nil || orgRole == "" {
		if len(snapshot) > 0 {
			return context.WithValue(ctx, ContextKeyPermissions, snapshot)
		}
		return ctx
	}

	role := organization.Role(orgRole)
	overrides, err := resolver.GetRoleOverrides(ctx, *orgID)
	if err != nil {
		slog.Warn("auth: live perms lookup failed, falling back to session snapshot",
			"org_id", orgID.String(), "error", err)
		if len(snapshot) > 0 {
			return context.WithValue(ctx, ContextKeyPermissions, snapshot)
		}
		return ctx
	}

	effective := organization.EffectivePermissionsFor(role, overrides)
	perms := make([]string, 0, len(effective))
	for _, p := range effective {
		perms = append(perms, string(p))
	}
	if len(perms) == 0 {
		// Unknown role → EffectivePermissionsFor returned nil. Fall
		// back to the snapshot rather than silently deny everything.
		if len(snapshot) > 0 {
			return context.WithValue(ctx, ContextKeyPermissions, snapshot)
		}
		return ctx
	}
	return context.WithValue(ctx, ContextKeyPermissions, perms)
}
