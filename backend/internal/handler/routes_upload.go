package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountUploadRoutes wires the /upload endpoint group. Stacks the
// SEC-11 upload-class limiter on top of the global IP throttle so
// every upload endpoint shares the same per-user quota.
func mountUploadRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	r.Route("/upload", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		// SEC-11: upload-class limiter (10/min/user) on top of the
		// global IP throttle. Stacked here on the whole subtree so
		// every upload endpoint shares the same quota.
		if deps.RateLimiter != nil {
			r.Use(deps.RateLimiter.Middleware(middleware.DefaultUploadPolicy, middleware.UserKey()))
		}
		// Profile-related uploads require org profile edit permission
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermOrgProfileEdit))
			r.Post("/photo", deps.Upload.UploadPhoto)
			r.Post("/video", deps.Upload.UploadVideo)
			r.Delete("/video", deps.Upload.DeleteVideo)
			r.Post("/referrer-video", deps.Upload.UploadReferrerVideo)
			r.Delete("/referrer-video", deps.Upload.DeleteReferrerVideo)
			r.Post("/portfolio-image", deps.Upload.UploadPortfolioImage)
			r.Post("/portfolio-video", deps.Upload.UploadPortfolioVideo)
		})
		// Review video upload requires review permission
		r.With(middleware.RequirePermission(organization.PermReviewsRespond)).Post("/review-video", deps.Upload.UploadReviewVideo)
	})
}
