package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountProposalRoutes wires the proposal surface and its sibling
// "active projects" listing. The proposal lifecycle endpoints share
// PermProposalsRespond — accept / decline / pay / complete / cancel /
// per-milestone fund-submit-approve-reject all live here.
func mountProposalRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Proposal == nil {
		return
	}
	r.Route("/proposals", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.With(middleware.RequirePermission(organization.PermProposalsView)).Get("/{id}", deps.Proposal.GetProposal)
		r.With(middleware.RequirePermission(organization.PermProposalsCreate)).Post("/", deps.Proposal.CreateProposal)
		r.With(middleware.RequirePermission(organization.PermProposalsCreate)).Post("/{id}/modify", deps.Proposal.ModifyProposal)
		// Respond actions (accept, decline, pay, complete flow)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermProposalsRespond))
			r.Post("/{id}/accept", deps.Proposal.AcceptProposal)
			r.Post("/{id}/decline", deps.Proposal.DeclineProposal)
			// Legacy endpoints (one-time mode shortcut, kept for
			// backward compatibility — they delegate to the same
			// proposal service methods as the new milestone-explicit
			// routes below).
			r.Post("/{id}/pay", deps.Proposal.PayProposal)
			r.Post("/{id}/confirm-payment", deps.Proposal.ConfirmPayment)
			r.Post("/{id}/request-completion", deps.Proposal.RequestCompletion)
			r.Post("/{id}/complete", deps.Proposal.CompleteProposal)
			r.Post("/{id}/reject-completion", deps.Proposal.RejectCompletion)
			// Phase 5: milestone-explicit endpoints. The {mid}
			// segment is validated against the current active
			// milestone — a stale client view (someone else has
			// moved the proposal forward) yields 409 Conflict
			// instead of silently mutating the wrong milestone.
			r.Post("/{id}/milestones/{mid}/fund", deps.Proposal.FundMilestone)
			r.Post("/{id}/milestones/{mid}/submit", deps.Proposal.SubmitMilestone)
			r.Post("/{id}/milestones/{mid}/approve", deps.Proposal.ApproveMilestone)
			r.Post("/{id}/milestones/{mid}/reject", deps.Proposal.RejectMilestone)
			// Boundary cancel (no money in flight). Either side
			// may initiate — the proposal service's
			// requireOrgIsParticipant check handles auth.
			r.Post("/{id}/cancel", deps.Proposal.CancelProposal)
		})
	})
	r.Route("/projects", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Use(middleware.RequirePermission(organization.PermProposalsView))
		r.Get("/", deps.Proposal.ListActiveProjects)
	})
}
