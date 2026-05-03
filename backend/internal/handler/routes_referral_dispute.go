package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountReferralRoutes wires the apport d'affaires endpoints. No
// per-route permission gate: ownership (referrer / provider / client
// party of the referral) is enforced inside the service layer by
// loadAndAuthorise on every state transition.
func mountReferralRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Referral == nil {
		return
	}
	r.Route("/referrals", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Post("/", deps.Referral.Create)
		r.Get("/me", deps.Referral.ListMine)
		r.Get("/incoming", deps.Referral.ListIncoming)
		r.Get("/{id}", deps.Referral.Get)
		r.Post("/{id}/respond", deps.Referral.Respond)
		r.Get("/{id}/negotiations", deps.Referral.ListNegotiations)
		r.Get("/{id}/attributions", deps.Referral.ListAttributions)
		r.Get("/{id}/commissions", deps.Referral.ListCommissions)
	})
}

// mountDisputeRoutes wires the proposal-level dispute surface. Read
// uses PermProposalsView; write uses PermProposalsRespond — both gates
// already cover the underlying proposal so no extra dispute-specific
// permission is needed.
func mountDisputeRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Dispute == nil {
		return
	}
	idem := idempotencyMiddleware(deps)
	r.Route("/disputes", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		// Read
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermProposalsView))
			r.Get("/mine", deps.Dispute.ListMyDisputes)
			r.Get("/{id}", deps.Dispute.GetDispute)
		})
		// Write (disputes are proposal-level actions)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermProposalsRespond))
			// SEC-FINAL-02 idempotency on the OpenDispute creation
			// POST so a retry on flaky network does not open a
			// second dispute against the same proposal.
			r.With(idem).Post("/", deps.Dispute.OpenDispute)
			r.Post("/{id}/counter-propose", deps.Dispute.CounterPropose)
			r.Post("/{id}/counter-proposals/{cpId}/respond", deps.Dispute.RespondToCounter)
			r.Post("/{id}/cancel", deps.Dispute.CancelDispute)
			r.Post("/{id}/cancellation/respond", deps.Dispute.RespondToCancellation)
		})
	})
}
