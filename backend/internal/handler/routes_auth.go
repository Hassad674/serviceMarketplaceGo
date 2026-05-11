package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountAuthRoutes wires the /auth, /invitations and /organizations team
// routes onto the v1 sub-router. Extracted from NewRouter as part of
// phase-3-F so the orchestrator stays under the 200-line ceiling.
//
// The whole sub-tree is built around a single shared `auth` middleware
// passed in by the parent — that way every authenticated branch reuses
// the exact same closure (token + session + overrides resolver) and we
// avoid re-binding the dependencies in each helper.
func mountAuthRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	mountCoreAuth(r, deps, auth)
	mountInvitationRoutes(r, deps, auth)
	mountTeamRoutes(r, deps, auth)
}

func mountCoreAuth(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	idem := idempotencyMiddleware(deps)

	r.Route("/auth", func(r chi.Router) {
		// SEC-FINAL-02 idempotency on /register: while email-uniqueness
		// already prevents true double-creation, a retry today still
		// triggers a second password hash (CPU) + a second welcome
		// email + a second audit-log row. The middleware caches the
		// first 2xx response so retries land on a stable replay.
		r.With(idem).Post("/register", deps.Auth.Register)
		// RATE-LIMIT-PROD: dedicated per-IP cap on /login (10/min).
		// Sits on top of the existing per-email + per-IP brute-force
		// service so even a single IP rotating emails burns out fast.
		postWithClass(r, "/login", deps.Auth.Login, deps,
			AuthLoginRateLimitPolicy(deps.Config), ipKeyFromLimiter(deps))
		// B.6 Email 2FA: completes a login that was gated by the 2FA
		// flag. Public route — auth middleware would reject it because
		// no token has been issued yet (tokens come back IN the
		// response). Tight per-IP cap defeats 6-digit code
		// brute-forcing.
		postWithClass(r, "/login/verify-2fa", deps.Auth.VerifyTwoFactor, deps,
			Auth2FAVerifyRateLimitPolicy(deps.Config), ipKeyFromLimiter(deps))
		r.Post("/refresh", deps.Auth.Refresh)
		// RATE-LIMIT-PROD: 3/min per email-hash. Keying by email (not
		// by IP) is what defeats an attacker iterating thousands of
		// emails from a single IP — the per-IP global cap would let
		// that traffic through, the per-email gate stops it at the
		// boundary.
		postWithClass(r, "/forgot-password", deps.Auth.ForgotPassword, deps,
			PasswordResetRateLimitPolicy(deps.Config), middleware.EmailKey())
		r.Post("/reset-password", deps.Auth.ResetPassword)

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Use(middleware.NoCache)
			r.Get("/me", deps.Auth.Me)
			r.Get("/ws-token", deps.Auth.WSToken)
			r.Post("/web-session", deps.Auth.WebSession)
			r.Post("/logout", deps.Auth.Logout)
			r.Put("/referrer-enable", deps.Auth.EnableReferrer)
			// Account self-service: rotate credentials. Both endpoints
			// invalidate the caller's session_version on success — the
			// existing access token will be rejected on its next
			// authenticated request, forcing a fresh login.
			r.Post("/change-email", deps.Auth.ChangeEmail)
			r.Post("/change-password", deps.Auth.ChangePassword)
		})
	})

	// B.6 Email 2FA opt-in/opt-out. Mounted under /me so the user
	// owns the toggle implicitly — no orgID, no resource id. Both
	// endpoints require auth; the disable endpoint additionally
	// requires fresh password re-auth in the body.
	r.Route("/me/two-factor", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		// RATE-LIMIT-PROD: tight 5/min per-user cap on /enable to
		// neutralise email-bombing. A stolen session calling /enable
		// repeatedly would otherwise spam the user's inbox with 2FA
		// setup emails.
		postWithClass(r, "/enable", deps.Auth.EnableTwoFactor, deps,
			Auth2FAEnableRateLimitPolicy(deps.Config), middleware.UserKey())
		r.Post("/disable", deps.Auth.DisableTwoFactor)
	})
}

// postWithClass registers a POST handler with an optional RATE-LIMIT
// class layered on top. When the RateLimiter dependency is nil (test
// stubs), the helper falls back to a plain r.Post so the router-snapshot
// test (which boots with a nil limiter to verify the route table) sees
// the same middleware-chain count as before the refactor.
//
// We deliberately do NOT use r.With(noopMiddleware).Post — chi counts
// every wrapper in the chain, which would inflate `mw=N` in the
// snapshot and force every consumer of the golden file to regen.
func postWithClass(r chi.Router, pattern string, h http.HandlerFunc, deps RouterDeps, policy middleware.RateLimitPolicy, key func(*http.Request) (string, bool)) {
	if deps.RateLimiter == nil {
		r.Post(pattern, h)
		return
	}
	r.With(deps.RateLimiter.Middleware(policy, key)).Post(pattern, h)
}

// ipKeyFromLimiter returns an IP-based keyFn for the public auth
// endpoints (/login + /verify-2fa). The user is anonymous when these
// fire, so keying by user_id is impossible — IP is the only signal
// available. Routed through the RateLimiter's ClientIP so trusted-proxy
// + IPv6 /64 normalisation are applied consistently with the global
// throttle.
func ipKeyFromLimiter(deps RouterDeps) func(*http.Request) (string, bool) {
	if deps.RateLimiter == nil {
		// Unused in practice (postWithClass short-circuits when the
		// limiter is nil), but a safe fallback prevents panics if the
		// wiring evolves.
		return func(*http.Request) (string, bool) { return "", false }
	}
	return deps.RateLimiter.IPKey()
}

func mountInvitationRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Invitation == nil {
		return
	}
	// Public: validate a token and accept an invitation.
	r.Get("/invitations/validate", deps.Invitation.Validate)
	r.Post("/invitations/accept", deps.Invitation.Accept)

	idem := idempotencyMiddleware(deps)
	r.Route("/organizations/{orgID}/invitations", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		// SEC-FINAL-02 idempotency on Send so a flaky network retry
		// does not deliver a duplicate invitation email or an extra
		// pending invitation row.
		r.With(idem).Post("/", deps.Invitation.Send)
		r.Get("/", deps.Invitation.List)
		r.Post("/{invID}/resend", deps.Invitation.Resend)
		r.Delete("/{invID}", deps.Invitation.Cancel)
	})
}

func mountTeamRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Team == nil {
		return
	}
	// Static org-scoped routes that do NOT take an orgID URL param go
	// above the {orgID} route group so chi resolves them correctly.
	// role-definitions is a global catalogue (R13: team page "About
	// roles" panel + edit modal preview).
	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/organizations/role-definitions", deps.Team.RoleDefinitions)
	})

	r.Route("/organizations/{orgID}", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)

		r.Get("/members", deps.Team.ListMembers)
		r.Patch("/members/{userID}", deps.Team.UpdateMember)
		r.Delete("/members/{userID}", deps.Team.RemoveMember)
		r.Post("/leave", deps.Team.Leave)

		r.Post("/transfer", deps.Team.InitiateTransfer)
		r.Delete("/transfer", deps.Team.CancelTransfer)
		r.Post("/transfer/accept", deps.Team.AcceptTransfer)
		r.Post("/transfer/decline", deps.Team.DeclineTransfer)

		// Role permissions editor (R17 — per-org customization).
		// GET is readable by any org member (every role holds
		// team.view in the defaults). PATCH is Owner-only and
		// additionally defense-in-depth gated by the service
		// layer. The middleware fast-path uses the Owner-only
		// PermTeamManageRolePermissions permission which is
		// itself non-overridable.
		if deps.RoleOverrides != nil {
			r.Get("/role-permissions", deps.RoleOverrides.GetMatrix)
			r.With(middleware.RequirePermission(organization.PermTeamManageRolePermissions)).
				Patch("/role-permissions", deps.RoleOverrides.UpdateMatrix)
		}
	})
}
