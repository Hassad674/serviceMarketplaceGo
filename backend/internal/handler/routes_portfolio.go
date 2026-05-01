package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountPortfolioRoutes wires the /portfolio surface — public reads on
// /portfolio/org/{orgId} and authenticated writes scoped by the org
// profile-edit permission.
func mountPortfolioRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Portfolio == nil {
		return
	}
	// Public: read portfolio for an organization
	r.Get("/portfolio/org/{orgId}", deps.Portfolio.ListPortfolioByOrganization)
	r.Get("/portfolio/{id}", deps.Portfolio.GetPortfolioItem)

	// Authenticated: manage own portfolio (org profile edit permission)
	r.Route("/portfolio", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Use(middleware.RequirePermission(organization.PermOrgProfileEdit))
		r.Post("/", deps.Portfolio.CreatePortfolioItem)
		r.Put("/reorder", deps.Portfolio.ReorderPortfolio)
		r.Put("/{id}", deps.Portfolio.UpdatePortfolioItem)
		r.Delete("/{id}", deps.Portfolio.DeletePortfolioItem)
	})
}
