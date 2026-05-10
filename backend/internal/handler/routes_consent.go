package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// mountConsentRoutes wires the single POST /consent/log endpoint.
//
// Auth is OPTIONAL — when an authenticated session is present the
// middleware annotates the context with the user id which the handler
// picks up automatically; anonymous visitors are also accepted because
// the cookie banner appears before any authentication step.
//
// nil handler = feature disabled, mounting is a no-op (per project
// modularity rule — every feature must be removable).
func mountConsentRoutes(r chi.Router, deps RouterDeps, _ func(http.Handler) http.Handler) {
	if deps.Consent == nil {
		return
	}
	r.Route("/consent", func(r chi.Router) {
		r.Post("/log", deps.Consent.Log)
	})
}
