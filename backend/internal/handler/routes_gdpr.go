package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// mountGDPRRoutes wires the four right-to-erasure / right-to-export
// endpoints (P5).
//
// Auth-required: GET /me/export, POST /me/account/request-deletion,
// POST /me/account/cancel-deletion. ConfirmDeletion is intentionally
// NOT auth-gated because the link arrives in the user's email and
// the JWT in the query string is the auth token (purpose-scoped, 24h
// TTL — see app/gdpr/service.go).
//
// nil handler = feature disabled, mounting is a no-op so the GDPR
// surface stays fully removable per the project modularity rule.
func mountGDPRRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.GDPR == nil {
		return
	}
	r.Route("/me/account", func(r chi.Router) {
		// confirm-deletion is unauthenticated by design — the JWT
		// in the query string is the auth.
		r.Get("/confirm-deletion", deps.GDPR.ConfirmDeletion)

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Post("/request-deletion", deps.GDPR.RequestDeletion)
			r.Post("/cancel-deletion", deps.GDPR.CancelDeletion)
		})
	})

	r.Route("/me/export", func(r chi.Router) {
		r.Use(auth)
		r.Get("/", deps.GDPR.Export)
	})
}
