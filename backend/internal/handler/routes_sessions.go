package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/handler/middleware"
)

// mountSessionsRoutes wires the user-facing Sécurité-page session
// management endpoints (SEC-SESSIONS):
//
//   GET    /api/v1/me/sessions                — list active sessions
//   DELETE /api/v1/me/sessions/{id}           — revoke one session
//   POST   /api/v1/me/sessions/revoke-others  — revoke every session except the current one
//
// Auth is required and the rows are user-scoped by the underlying
// repository — no organization permission gate applies because the
// data is strictly the caller's own session audit trail.
//
// The block is self-skipping when the handler is nil so the feature
// stays fully removable.
func mountSessionsRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Sessions == nil {
		return
	}
	r.Route("/me/sessions", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/", deps.Sessions.List)
		r.Post("/revoke-others", deps.Sessions.RevokeAllExceptCurrent)
		r.Delete("/{id}", deps.Sessions.Revoke)
	})
}
