package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// mountAutomatedDecisionAppealRoutes wires the single
// POST /me/automated-decision-appeals endpoint that lets an authenticated
// user file a request for human review of an automated decision (RGPD
// art. 22).
//
// nil handler = feature disabled, mounting is a no-op so the surface
// stays fully removable per the project modularity rule.
func mountAutomatedDecisionAppealRoutes(
	r chi.Router,
	deps RouterDeps,
	auth func(http.Handler) http.Handler,
) {
	if deps.AutomatedDecisionAppeal == nil {
		return
	}
	r.Route("/me/automated-decision-appeals", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", deps.AutomatedDecisionAppeal.FileAppeal)
	})
}
