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

// AuthDeps groups the collaborators the auth middleware needs.
// Bundling them keeps the public factory's signature stable as we
// add same-layer checks (session_version, live user state, org
// overrides) without violating the project's 4-parameter rule.
//
// All checker fields are optional in tests (nil-safe). Production
// wiring always passes a non-nil checker for each.
type AuthDeps struct {
	TokenService    service.TokenService
	SessionService  service.SessionService
	SessionVersions SessionVersionChecker
	UserState       UserStateChecker
	OrgOverrides    OrgOverridesResolver
	// FailClosedInProd routes a transient DB/Redis failure during
	// the session_version OR user_state lookup to 503 instead of
	// silently trusting the cookie/JWT snapshot. Set to true in
	// production wiring; left false in dev/test so a contributor's
	// broken local DB does not lock everyone out.
	FailClosedInProd bool
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
//
// Backwards-compatible thin shim around AuthFromDeps. Existing call
// sites (tests + worktrees that haven't migrated yet) keep working
// without churn. New production code paths use AuthFromDeps directly
// so the live UserStateChecker can be wired in.
func Auth(
	tokenService service.TokenService,
	sessionService service.SessionService,
	sessionVersions SessionVersionChecker,
	overridesResolver OrgOverridesResolver,
) func(http.Handler) http.Handler {
	return AuthFromDeps(AuthDeps{
		TokenService:    tokenService,
		SessionService:  sessionService,
		SessionVersions: sessionVersions,
		OrgOverrides:    overridesResolver,
	})
}

// AuthWithFailClosed is the legacy production-ready factory. Kept for
// backwards compatibility with existing test wiring. New code should
// build an AuthDeps literal and call AuthFromDeps directly so the
// live UserStateChecker is wired alongside the session-version
// checker.
//
// F.5 S8 — when failClosedInProd is true, a transient lookup failure
// (DB outage / Redis blip while resolving session_version) returns
// 503 to the client instead of trusting the snapshot. Without this
// flag the middleware fell open: an attacker who triggered the
// upstream incident bypassed permission revocation. In dev/test the
// legacy "trust snapshot" behaviour is preserved so a contributor's
// broken local DB does not lock out everyone.
func AuthWithFailClosed(
	tokenService service.TokenService,
	sessionService service.SessionService,
	sessionVersions SessionVersionChecker,
	overridesResolver OrgOverridesResolver,
	failClosedInProd bool,
) func(http.Handler) http.Handler {
	return AuthFromDeps(AuthDeps{
		TokenService:     tokenService,
		SessionService:   sessionService,
		SessionVersions:  sessionVersions,
		OrgOverrides:     overridesResolver,
		FailClosedInProd: failClosedInProd,
	})
}

// AuthFromDeps is the canonical factory. It accepts the full AuthDeps
// bundle so the call site is explicit about every collaborator and
// new same-layer checks (live admin/status state) can be wired in
// without breaking existing constructors.
//
// is_admin propagation fix: the middleware now consults
// AuthDeps.UserState on every authenticated request and OVERRIDES the
// is_admin / status snapshot baked into the session/JWT at login.
// Without this override, a `UPDATE users SET is_admin = true` issued
// outside the application code path (e.g. operator promotion via
// SQL) would never reach the active sessions — they would keep
// returning 403 on /admin endpoints until each user logs out and
// back in. The 30-second Redis cache in the production checker
// absorbs the per-request DB cost.
func AuthFromDeps(deps AuthDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strategy 1: Session cookie (web clients)
			if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
				if handled := tryCookieAuth(w, r, cookie.Value, deps, next); handled {
					return
				}
			}

			// Strategy 2: Bearer token (mobile clients + admin SPA)
			if header := r.Header.Get("Authorization"); header != "" {
				if handled := tryBearerAuth(w, r, header, deps, next); handled {
					return
				}
			}

			response.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		})
	}
}

// tryCookieAuth attempts to authenticate via the session cookie. Returns
// true when the request has been fully handled (either dispatched to
// `next` or terminated with an error response). When false, the caller
// keeps trying the next strategy (bearer).
func tryCookieAuth(
	w http.ResponseWriter, r *http.Request, sessionID string,
	deps AuthDeps, next http.Handler,
) bool {
	session, err := deps.SessionService.Get(r.Context(), sessionID)
	if err != nil {
		// Cookie present but session lookup failed — fall through to
		// the bearer path (typically a stale cookie + a fresh token).
		return false
	}

	if !checkSessionVersion(w, r, deps, session.UserID, session.SessionVersion, "session has been revoked — please sign in again") {
		return true
	}
	live, ok := checkUserState(w, r, deps, session.UserID, UserState{
		IsAdmin: session.IsAdmin,
		Status:  user.StatusActive,
	})
	if !ok {
		return true
	}

	ctx := stampAuthContext(r.Context(), authStamp{
		UserID:      session.UserID,
		Role:        session.Role,
		IsAdmin:     live.IsAdmin,
		OrgID:       session.OrganizationID,
		OrgRole:     session.OrgRole,
		Permissions: session.Permissions,
	}, deps.OrgOverrides)
	next.ServeHTTP(w, r.WithContext(ctx))
	return true
}

// tryBearerAuth attempts to authenticate via a Bearer token. Same
// return contract as tryCookieAuth.
func tryBearerAuth(
	w http.ResponseWriter, r *http.Request, header string,
	deps AuthDeps, next http.Handler,
) bool {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return false
	}
	claims, err := deps.TokenService.ValidateAccessToken(parts[1])
	if err != nil {
		return false
	}

	if !checkSessionVersion(w, r, deps, claims.UserID, claims.SessionVersion, "token has been revoked — please sign in again") {
		return true
	}
	live, ok := checkUserState(w, r, deps, claims.UserID, UserState{
		IsAdmin: claims.IsAdmin,
		Status:  user.StatusActive,
	})
	if !ok {
		return true
	}

	ctx := stampAuthContext(r.Context(), authStamp{
		UserID:      claims.UserID,
		Role:        claims.Role,
		IsAdmin:     live.IsAdmin,
		OrgID:       claims.OrganizationID,
		OrgRole:     claims.OrgRole,
		Permissions: claims.Permissions,
	}, deps.OrgOverrides)
	next.ServeHTTP(w, r.WithContext(ctx))
	return true
}

// checkSessionVersion runs the session_version revocation check and
// writes the appropriate error response on failure. Returns true when
// the request should keep flowing through the middleware, false when
// it has been terminated and the caller must return.
func checkSessionVersion(
	w http.ResponseWriter, r *http.Request,
	deps AuthDeps, userID uuid.UUID, carriedVersion int, revokedMsg string,
) bool {
	switch verifySessionVersion(r.Context(), deps.SessionVersions, userID, carriedVersion) {
	case sessionVersionMatch:
		return true
	case sessionVersionUserGone:
		response.Error(w, http.StatusUnauthorized, "session_invalid",
			"session is no longer valid — please sign in again")
		return false
	case sessionVersionRevoked:
		response.Error(w, http.StatusUnauthorized, "session_revoked", revokedMsg)
		return false
	case sessionVersionLookupFailed:
		if deps.FailClosedInProd {
			response.Error(w, http.StatusServiceUnavailable, "auth_unavailable",
				"authentication backend is degraded — retry shortly")
			return false
		}
		// Dev/test: trust the snapshot.
		return true
	default:
		return true
	}
}

// checkUserState consults the live UserState checker (for is_admin /
// status) and writes the appropriate error response on failure.
// Returns the authoritative UserState alongside a boolean signalling
// whether the request should keep flowing.
//
// Why this is the central fix for the is_admin propagation bug: the
// snapshot baked into session.IsAdmin / claims.IsAdmin is captured at
// login time and never refreshed. By overlaying the live value here,
// any toggle of users.is_admin (including direct SQL updates from an
// operator console) propagates within at most the cache TTL — without
// forcing the user to log out and back in.
func checkUserState(
	w http.ResponseWriter, r *http.Request,
	deps AuthDeps, userID uuid.UUID, snapshot UserState,
) (UserState, bool) {
	live, outcome := resolveUserState(r.Context(), deps.UserState, userID, snapshot)
	switch outcome {
	case userStateOK:
		return live, true
	case userStateUserGone:
		response.Error(w, http.StatusUnauthorized, "session_invalid",
			"session is no longer valid — please sign in again")
		return UserState{}, false
	case userStateBanned:
		response.Error(w, http.StatusForbidden, "account_banned",
			"this account has been banned")
		return UserState{}, false
	case userStateLookupFailed:
		if deps.FailClosedInProd {
			response.Error(w, http.StatusServiceUnavailable, "auth_unavailable",
				"authentication backend is degraded — retry shortly")
			return UserState{}, false
		}
		// Dev/test: trust the snapshot.
		return snapshot, true
	default:
		return snapshot, true
	}
}

// authStamp groups the fields the middleware writes onto the request
// context after a successful authentication. Bundling them keeps the
// stamp helper under the 4-parameter ceiling.
type authStamp struct {
	UserID      uuid.UUID
	Role        string
	IsAdmin     bool
	OrgID       *uuid.UUID
	OrgRole     string
	Permissions []string
}

// stampAuthContext writes the standard auth context keys onto ctx and
// runs the live-permissions resolver. Centralising it here eliminates
// the duplicate stamping logic that previously lived in both the
// cookie and bearer code paths.
func stampAuthContext(
	ctx context.Context,
	stamp authStamp,
	overrides OrgOverridesResolver,
) context.Context {
	ctx = context.WithValue(ctx, ContextKeyUserID, stamp.UserID)
	ctx = context.WithValue(ctx, ContextKeyRole, stamp.Role)
	ctx = context.WithValue(ctx, ContextKeyIsAdmin, stamp.IsAdmin)
	if stamp.OrgID != nil {
		ctx = context.WithValue(ctx, ContextKeyOrganizationID, *stamp.OrgID)
	}
	if stamp.OrgRole != "" {
		ctx = context.WithValue(ctx, ContextKeyOrgRole, stamp.OrgRole)
	}
	return injectLivePermissions(ctx, overrides, stamp.OrgID, stamp.OrgRole, stamp.Permissions)
}

// sessionVersionOutcome is the result of a session_version
// check. The middleware branches on it to issue the right HTTP
// response.
type sessionVersionOutcome int

const (
	sessionVersionMatch sessionVersionOutcome = iota
	sessionVersionRevoked
	sessionVersionUserGone
	// F.5 S8 — sessionVersionLookupFailed signals a transient
	// upstream failure (DB outage, Redis blip). The middleware maps
	// this to 503 in production and to "trust snapshot" in dev.
	sessionVersionLookupFailed
)

// verifySessionVersion returns sessionVersionMatch when the carried
// version matches the current DB value (or when the checker is nil —
// acceptable in dev/tests), sessionVersionUserGone when the backing
// user row has been deleted, sessionVersionRevoked when the version
// was explicitly bumped, and sessionVersionLookupFailed when the
// resolver itself errors out (DB unreachable, Redis blip). The
// caller decides how to handle the latter (F.5 S8 fail-closed in
// production).
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
		// Transient error — let the caller decide. Production should
		// return 503 (F.5 S8); dev can keep trusting the snapshot.
		slog.Error("auth: session_version lookup failed",
			"user_id", userID, "error", err)
		return sessionVersionLookupFailed
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
