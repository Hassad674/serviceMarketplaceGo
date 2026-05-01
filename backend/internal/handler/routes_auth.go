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
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", deps.Auth.Register)
		r.Post("/login", deps.Auth.Login)
		r.Post("/refresh", deps.Auth.Refresh)
		r.Post("/forgot-password", deps.Auth.ForgotPassword)
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
		})
	})
}

func mountInvitationRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Invitation == nil {
		return
	}
	// Public: validate a token and accept an invitation.
	r.Get("/invitations/validate", deps.Invitation.Validate)
	r.Post("/invitations/accept", deps.Invitation.Accept)

	r.Route("/organizations/{orgID}/invitations", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Post("/", deps.Invitation.Send)
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
