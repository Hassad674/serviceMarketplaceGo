package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	proposalapp "marketplace-backend/internal/app/proposal"

	res "marketplace-backend/pkg/response"
)

// ProposalCompletionHandler owns the completion endpoints — the
// surface that transitions an active proposal through the
// request-completion → release / reject state machine.
//
// SRP rationale: every method here mutates the completion side of the
// proposal aggregate. Lifecycle (create/accept/cancel) lives on
// ProposalLifecycleHandler; funding (pay/confirm) lives on
// ProposalPaymentHandler.
//
// Dependencies:
//   - proposalSvc: drives the RequestCompletion / CompleteProposal /
//     RejectCompletion app-service methods. The "milestone-explicit"
//     SubmitMilestone / ApproveMilestone / RejectMilestone endpoints
//     also live here because they delegate to the same use-cases with
//     a milestone-id verification step at the URL layer.
type ProposalCompletionHandler struct {
	proposalSvc *proposalapp.Service
}

// NewProposalCompletionHandler wires the completion handler.
func NewProposalCompletionHandler(svc *proposalapp.Service) *ProposalCompletionHandler {
	return &ProposalCompletionHandler{proposalSvc: svc}
}

// RequestCompletion handles POST /api/v1/proposals/{id}/request-completion.
// Legacy one-time mode endpoint — delegates to RequestCompletion which
// transitions the milestone from funded → submitted.
func (h *ProposalCompletionHandler) RequestCompletion(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.RequestCompletion(r.Context(), proposalapp.RequestCompletionInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "completion_requested"})
}

// CompleteProposal handles POST /api/v1/proposals/{id}/complete. Legacy
// one-time mode endpoint — delegates to CompleteProposal which marks
// the milestone approved + released and (when payouts are auto-consented)
// fires the platform→connected transfer.
func (h *ProposalCompletionHandler) CompleteProposal(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.CompleteProposal(r.Context(), proposalapp.CompleteProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

// RejectCompletion handles POST /api/v1/proposals/{id}/reject-completion.
// Legacy one-time mode endpoint — sends the milestone back from
// submitted → funded so the provider can iterate.
func (h *ProposalCompletionHandler) RejectCompletion(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.RejectCompletion(r.Context(), proposalapp.RejectCompletionInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "active"})
}

// SubmitMilestone handles POST /proposals/{id}/milestones/{mid}/submit.
// The provider transitions the milestone from funded → submitted.
// Validates the {mid} matches the current active milestone before
// delegating to RequestCompletion.
func (h *ProposalCompletionHandler) SubmitMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	if !validateMilestoneMatchesCurrent(w, r, h.proposalSvc, proposalID, milestoneID) {
		return
	}
	if err := h.proposalSvc.RequestCompletion(r.Context(), proposalapp.RequestCompletionInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	}); err != nil {
		handleProposalError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "submitted"})
}

// ApproveMilestone handles POST /proposals/{id}/milestones/{mid}/approve.
// The client transitions the milestone from submitted → approved → released.
func (h *ProposalCompletionHandler) ApproveMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	if !validateMilestoneMatchesCurrent(w, r, h.proposalSvc, proposalID, milestoneID) {
		return
	}
	if err := h.proposalSvc.CompleteProposal(r.Context(), proposalapp.CompleteProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	}); err != nil {
		handleProposalError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "released"})
}

// RejectMilestone handles POST /proposals/{id}/milestones/{mid}/reject.
// The client sends the milestone back from submitted → funded so the
// provider can iterate.
func (h *ProposalCompletionHandler) RejectMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	if !validateMilestoneMatchesCurrent(w, r, h.proposalSvc, proposalID, milestoneID) {
		return
	}
	if err := h.proposalSvc.RejectCompletion(r.Context(), proposalapp.RejectCompletionInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	}); err != nil {
		handleProposalError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}
