package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountReviewRoutes wires the /reviews surface (mixed public reads +
// authenticated writes) and the /reports authoring surface.
func mountReviewRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Review == nil {
		return
	}
	r.Route("/reviews", func(r chi.Router) {
		// Public: read reviews and average ratings (keyed by org).
		// PublicCache lets the CDN edge serve subsequent hits without
		// reaching the backend. Authenticated callers bypass the public
		// cache automatically (see `middleware/public_cache.go`).
		r.With(middleware.PublicCache).Get("/org/{orgId}", deps.Review.ListByOrganization)
		r.With(middleware.PublicCache).Get("/average/{orgId}", deps.Review.GetAverageRating)

		// Authenticated: create reviews and check eligibility
		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Use(middleware.NoCache)
			r.With(middleware.RequirePermission(organization.PermReviewsRespond)).Post("/", deps.Review.CreateReview)
			r.With(middleware.RequirePermission(organization.PermProposalsView)).Get("/can-review/{proposalId}", deps.Review.CanReview)
		})
	})
}

func mountReportRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Report == nil {
		return
	}
	r.Route("/reports", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Post("/", deps.Report.CreateReport)
		r.Get("/mine", deps.Report.ListMyReports)
	})
}
