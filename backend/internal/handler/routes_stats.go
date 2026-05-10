package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/handler/middleware"
)

// mountStatsRoutes wires the /me/stats/* read endpoints. Auth is
// required and the rows are org-scoped by the underlying repository
// — no cross-org access is possible because the handler reads the
// org id from the JWT context only.
//
// The block is self-skipping when the handler is nil so the feature
// stays fully removable: deleting the wiring in cmd/api drops the
// route without touching the router file.
func mountStatsRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Stats == nil {
		return
	}
	r.Route("/me/stats", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/visibility", deps.Stats.GetVisibility)
		r.Get("/keywords", deps.Stats.GetKeywords)
		r.Get("/enterprise-applications", deps.Stats.GetEnterpriseApplications)
	})
}
