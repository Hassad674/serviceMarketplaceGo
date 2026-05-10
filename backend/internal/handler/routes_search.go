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
	// `/search` is the discovery entry point used by the public
	// listing routes (`/freelancers`, `/agencies`, `/referrers`)
	// — incognito visitors arriving from the landing search bar
	// must hit it without a session cookie. The handler already
	// reads the JWT user_id optionally (analytics tagging only),
	// so dropping auth here only removes the gate, not the
	// behaviour. NoCache is kept so the response stays fresh.
	r.Group(func(r chi.Router) {
		r.Use(middleware.NoCache)
		r.Get("/search", deps.Search.Search)
	})
	// Scoped-key minting and click tracking remain authenticated:
	//   - /search/key returns a Typesense scoped API key tied to
	//     the caller. We never want to mint that for an
	//     anonymous client.
	//   - /search/track captures CTR analytics tied to a logged-
	//     in user; an anonymous beacon would just pollute the
	//     attribution table.
	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/search/key", deps.Search.ScopedKey)
		// Click-through tracking. GET used instead of POST so
		// the browser beacon API can fire even on unload.
		r.Get("/search/track", deps.Search.Track)
	})
}
