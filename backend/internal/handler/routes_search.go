package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/handler/middleware"
)

// mountSearchRoutes wires the Typesense-backed search surface (scoped
// API key, search query, click tracker). Mounted only when Typesense
// is configured — otherwise the deps.Search pointer stays nil and the
// route group is skipped.
func mountSearchRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Search == nil {
		return
	}
	// Typesense-backed search routes. Both endpoints require an
	// authenticated user. The legacy /profiles/search SQL path
	// was retired in phase 4 (30-day grace ended April 2026) —
	// the only remaining consumer is the referral provider
	// picker, which uses it as a simple directory read.
	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/search/key", deps.Search.ScopedKey)
		r.Get("/search", deps.Search.Search)
		// Click-through tracking. GET used instead of POST so
		// the browser beacon API can fire even on unload.
		r.Get("/search/track", deps.Search.Track)
	})
}
