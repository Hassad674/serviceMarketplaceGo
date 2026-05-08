package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/handler/middleware"
)

// mountSecurityRoutes wires the /me/security/* read-only endpoints.
// Auth is required and the rows are user-scoped by the underlying
// repository — no organization permission gate applies because the
// data is strictly the caller's own audit trail.
//
// The block is self-skipping when the handler is nil so the feature
// stays fully removable.
func mountSecurityRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Security == nil {
		return
	}
	r.Route("/me/security", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/activity", deps.Security.ListActivity)
	})
}
