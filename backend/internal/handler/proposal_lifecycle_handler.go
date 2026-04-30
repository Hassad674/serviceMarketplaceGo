package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ProposalLifecycleHandler owns the proposal life-cycle endpoints —
// the create/read/respond/modify/cancel surface that does NOT touch
// the payment or completion state machines.
//
// SRP rationale: every method here mutates ONLY the proposal aggregate
// (or reads it). Funding (PayProposal), confirmation
// (ConfirmPayment) and milestone state-machine endpoints live on
// ProposalPaymentHandler / ProposalCompletionHandler / ProposalAdminHandler.
//
// Dependencies (DIP — minimum interface):
//   - proposalSvc: the proposal app service (facet of the wider service
//     used elsewhere — kept as the full *Service for now since further
//     port segregation is out of scope for Phase 3).
type ProposalLifecycleHandler struct {
	proposalSvc *proposalapp.Service
}

// NewProposalLifecycleHandler wires the lifecycle handler.
func NewProposalLifecycleHandler(svc *proposalapp.Service) *ProposalLifecycleHandler {
	return &ProposalLifecycleHandler{proposalSvc: svc}
}

// CreateProposal handles POST /api/v1/proposals.
func (h *ProposalLifecycleHandler) CreateProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.CreateProposalRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_recipient_id", "recipient_id must be a valid UUID")
		return
	}

	convID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_conversation_id", "conversation_id must be a valid UUID")
		return
	}

	deadline, err := parseOptionalDeadline(req.Deadline)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_deadline", err.Error())
		return
	}

	docs := convertDocumentInputs(req.Documents)
	milestoneInputs, mErr := convertMilestoneInputs(req.Milestones)
	if mErr != nil {
		res.Error(w, http.StatusBadRequest, "invalid_milestone", mErr.Error())
		return
	}

	p, err := h.proposalSvc.CreateProposal(r.Context(), proposalapp.CreateProposalInput{
		ConversationID: convID,
		SenderID:       userID,
		RecipientID:    recipientID,
		Title:          req.Title,
		Description:    req.Description,
		Amount:         req.Amount,
		Deadline:       deadline,
		Documents:      docs,
		PaymentMode:    req.PaymentMode,
		Milestones:     milestoneInputs,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	// Fetch the just-persisted milestones so the response reflects the
	// final shape (synthesised single milestone for one-time mode, or
	// the full N-milestone batch for milestone mode).
	createdMilestones, mErr := h.proposalSvc.ListMilestones(r.Context(), p.ID)
	if mErr != nil {
		slog.Error("list milestones after create", "proposal_id", p.ID, "error", mErr)
		createdMilestones = nil
	}
	res.JSON(w, http.StatusCreated, response.NewProposalResponseWithMilestones(p, nil, createdMilestones))
}

// GetProposal handles GET /api/v1/proposals/{id}.
func (h *ProposalLifecycleHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	p, docs, err := h.proposalSvc.GetProposal(r.Context(), userID, orgID, proposalID)
	if err != nil {
		handleProposalError(w, err)
		return
	}

	clientName, providerName := h.proposalSvc.GetParticipantNames(r.Context(), p.ClientID, p.ProviderID)

	// Phase 5: enrich the response with the milestone list so the
	// frontend can render the tracker without a second round trip.
	// Failure here is non-blocking — the proposal still ships with
	// an empty milestones slice and the frontend falls back to the
	// legacy single-amount UX.
	milestones, mErr := h.proposalSvc.ListMilestones(r.Context(), p.ID)
	if mErr != nil {
		slog.Error("list milestones for proposal", "proposal_id", p.ID, "error", mErr)
		milestones = nil
	}
	res.JSON(w, http.StatusOK, response.NewProposalResponseWithNames(p, docs, milestones, clientName, providerName))
}

// AcceptProposal handles POST /api/v1/proposals/{id}/accept.
func (h *ProposalLifecycleHandler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.AcceptProposal(r.Context(), proposalapp.AcceptProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

// DeclineProposal handles POST /api/v1/proposals/{id}/decline.
func (h *ProposalLifecycleHandler) DeclineProposal(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.DeclineProposal(r.Context(), proposalapp.DeclineProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "declined"})
}

// ModifyProposal handles POST /api/v1/proposals/{id}/modify.
func (h *ProposalLifecycleHandler) ModifyProposal(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	var req request.ModifyProposalRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	deadline, err := parseOptionalDeadline(req.Deadline)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_deadline", err.Error())
		return
	}

	docs := convertDocumentInputs(req.Documents)

	p, err := h.proposalSvc.ModifyProposal(r.Context(), proposalapp.ModifyProposalInput{
		ProposalID:  proposalID,
		UserID:      userID,
		OrgID:       orgID,
		Title:       req.Title,
		Description: req.Description,
		Amount:      req.Amount,
		Deadline:    deadline,
		Documents:   docs,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewProposalResponse(p, nil))
}

// CancelProposal handles POST /proposals/{id}/cancel. Either party may
// initiate cancellation at a milestone boundary (no active milestone in
// flight). Already-released milestones stay released; pending_funding
// milestones become cancelled.
func (h *ProposalLifecycleHandler) CancelProposal(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "proposal id must be a valid UUID")
		return
	}

	if err := h.proposalSvc.CancelProposal(r.Context(), proposalapp.CancelProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	}); err != nil {
		handleProposalError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListActiveProjects handles GET /api/v1/projects.
func (h *ProposalLifecycleHandler) ListActiveProjects(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	proposals, nextCursor, err := h.proposalSvc.ListActiveProjectsByOrganization(r.Context(), orgID, cursorStr, limit)
	if err != nil {
		handleProposalError(w, err)
		return
	}

	// Batch-fetch milestones for every listed proposal in a single
	// round trip — see ListByProposals which uses ANY($1::uuid[]) to
	// sidestep N+1.
	proposalIDs := make([]uuid.UUID, len(proposals))
	for i, p := range proposals {
		proposalIDs[i] = p.ID
	}
	milestonesByProposal, mErr := h.proposalSvc.ListMilestonesForProposals(r.Context(), proposalIDs)
	if mErr != nil {
		slog.Error("list milestones for proposals batch", "count", len(proposalIDs), "error", mErr)
		milestonesByProposal = nil
	}

	data := make([]response.ProposalResponse, len(proposals))
	for i, p := range proposals {
		cn, pn := h.proposalSvc.GetParticipantNames(r.Context(), p.ClientID, p.ProviderID)
		data[i] = response.NewProposalResponseWithNames(p, nil, milestonesByProposal[p.ID], cn, pn)
	}
	res.JSON(w, http.StatusOK, response.ProjectListResponse{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	})
}

// ---------------------------------------------------------------------------
// Helpers shared with the other proposal_*_handler.go files
// ---------------------------------------------------------------------------

// parseOptionalDeadline parses an optional deadline string.
// Accepts RFC3339 or YYYY-MM-DD. Empty input returns nil, nil. Centralised
// here so create + modify share the same parsing rules.
func parseOptionalDeadline(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02", s)
		if err != nil {
			return nil, errors.New("deadline must be RFC3339 or YYYY-MM-DD")
		}
	}
	return &t, nil
}
