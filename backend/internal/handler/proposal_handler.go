package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	userdomain "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type ProposalHandler struct {
	proposalSvc *proposalapp.Service
	paymentSvc  *paymentapp.Service // nil if Stripe not configured
}

func NewProposalHandler(svc *proposalapp.Service, paymentSvc *paymentapp.Service) *ProposalHandler {
	return &ProposalHandler{proposalSvc: svc, paymentSvc: paymentSvc}
}

func (h *ProposalHandler) CreateProposal(w http.ResponseWriter, r *http.Request) {
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

	var deadline *time.Time
	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err != nil {
			t, err = time.Parse("2006-01-02", req.Deadline)
			if err != nil {
				res.Error(w, http.StatusBadRequest, "invalid_deadline", "deadline must be RFC3339 or YYYY-MM-DD")
				return
			}
		}
		deadline = &t
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

// convertMilestoneInputs maps the request DTO milestone slice onto the
// proposal app service's MilestoneInput type. Returns an error if a
// deadline string is malformed (RFC3339 or YYYY-MM-DD only).
func convertMilestoneInputs(in []request.MilestoneInputRequest) ([]proposalapp.MilestoneInput, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]proposalapp.MilestoneInput, 0, len(in))
	for _, m := range in {
		var deadline *time.Time
		if m.Deadline != "" {
			t, err := time.Parse(time.RFC3339, m.Deadline)
			if err != nil {
				t, err = time.Parse("2006-01-02", m.Deadline)
				if err != nil {
					return nil, errors.New("milestone deadline must be RFC3339 or YYYY-MM-DD")
				}
			}
			deadline = &t
		}
		out = append(out, proposalapp.MilestoneInput{
			Sequence:    m.Sequence,
			Title:       m.Title,
			Description: m.Description,
			Amount:      m.Amount,
			Deadline:    deadline,
		})
	}
	return out, nil
}

func (h *ProposalHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

func (h *ProposalHandler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

func (h *ProposalHandler) DeclineProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

func (h *ProposalHandler) ModifyProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

	var deadline *time.Time
	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err != nil {
			t, err = time.Parse("2006-01-02", req.Deadline)
			if err != nil {
				res.Error(w, http.StatusBadRequest, "invalid_deadline", "deadline must be RFC3339 or YYYY-MM-DD")
				return
			}
		}
		deadline = &t
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

func (h *ProposalHandler) PayProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	result, err := h.proposalSvc.InitiatePayment(r.Context(), proposalapp.PayProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	// Simulation mode: payment completed immediately
	if result == nil {
		res.JSON(w, http.StatusOK, map[string]string{"status": "paid"})
		return
	}

	// Stripe mode: return client_secret for Elements
	res.JSON(w, http.StatusOK, response.PaymentIntentResponse{
		ClientSecret:    result.ClientSecret,
		PaymentRecordID: result.PaymentRecordID.String(),
		Amounts: response.PaymentAmounts{
			ProposalAmount: result.ProposalAmount,
			StripeFee:      result.StripeFee,
			PlatformFee:    result.PlatformFee,
			ClientTotal:    result.ClientTotal,
			ProviderPayout: result.ProviderPayout,
		},
	})
}

// ConfirmPayment is called by the frontend after stripe.confirmPayment() succeeds.
// It serves as a fallback to the webhook — ensures the proposal transitions to paid/active.
// AdminActivateProposal forces a proposal to paid+active state (admin-only, for testing).
func (h *ProposalHandler) AdminActivateProposal(w http.ResponseWriter, r *http.Request) {
	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}
	if err := h.proposalSvc.ConfirmPaymentAndActivate(r.Context(), proposalID); err != nil {
		handleProposalError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "activated"})
}

func (h *ProposalHandler) ConfirmPayment(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	// Verify the caller's org is on the client side — any operator of
	// the client org can confirm payment on behalf of their team. Only
	// the client side is authorized because they are the party who
	// released the funds. The provider-side org must NOT be able to
	// bounce the payment record to "succeeded" on its own.
	if err := h.proposalSvc.AuthorizeClientOrg(r.Context(), proposalID, orgID); err != nil {
		handleProposalError(w, err)
		return
	}

	// Mark the payment record as succeeded
	if h.paymentSvc != nil {
		if err := h.paymentSvc.MarkPaymentSucceeded(r.Context(), proposalID); err != nil {
			slog.Error("mark payment succeeded", "proposal_id", proposalID, "error", err)
		}
	}

	// Confirm payment and activate the proposal
	if err := h.proposalSvc.ConfirmPaymentAndActivate(r.Context(), proposalID); err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func (h *ProposalHandler) RequestCompletion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

func (h *ProposalHandler) CompleteProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

func (h *ProposalHandler) RejectCompletion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

func (h *ProposalHandler) ListActiveProjects(w http.ResponseWriter, r *http.Request) {
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

// AdminListBonusLog handles GET /api/v1/admin/credits/bonus-log.
func (h *ProposalHandler) AdminListBonusLog(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	entries, nextCursor, err := h.proposalSvc.ListBonusLog(r.Context(), cursor, limit)
	if err != nil {
		slog.Error("list bonus log", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list bonus log")
		return
	}

	items := make([]response.BonusLogResponse, 0, len(entries))
	for _, e := range entries {
		items = append(items, response.NewBonusLogResponse(e))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// AdminListPendingBonusLog handles GET /api/v1/admin/credits/bonus-log/pending.
func (h *ProposalHandler) AdminListPendingBonusLog(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	entries, nextCursor, err := h.proposalSvc.ListPendingBonusLog(r.Context(), cursor, limit)
	if err != nil {
		slog.Error("list pending bonus log", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list pending bonus log")
		return
	}

	items := make([]response.BonusLogResponse, 0, len(entries))
	for _, e := range entries {
		items = append(items, response.NewBonusLogResponse(e))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// AdminApproveBonusEntry handles POST /api/v1/admin/credits/bonus-log/{id}/approve.
func (h *ProposalHandler) AdminApproveBonusEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.proposalSvc.ApproveBonusEntry(r.Context(), entryID); err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// AdminRejectBonusEntry handles POST /api/v1/admin/credits/bonus-log/{id}/reject.
func (h *ProposalHandler) AdminRejectBonusEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.proposalSvc.RejectBonusEntry(r.Context(), entryID); err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func convertDocumentInputs(inputs []request.DocumentInput) []proposalapp.DocumentInput {
	docs := make([]proposalapp.DocumentInput, len(inputs))
	for i, d := range inputs {
		docs[i] = proposalapp.DocumentInput{
			Filename: d.Filename,
			URL:      d.URL,
			Size:     d.Size,
			MimeType: d.MimeType,
		}
	}
	return docs
}

// ---------------------------------------------------------------------------
// Phase 5: milestone-explicit endpoints
//
// These handlers expose URL-level resource hierarchy for the new
// milestone-aware frontend:
//
//	POST /api/v1/proposals/{id}/milestones/{mid}/fund
//	POST /api/v1/proposals/{id}/milestones/{mid}/submit
//	POST /api/v1/proposals/{id}/milestones/{mid}/approve
//	POST /api/v1/proposals/{id}/milestones/{mid}/reject
//	POST /api/v1/proposals/{id}/cancel
//
// The {mid} segment is informational — the server still always operates
// on the current active milestone (because the strict-sequential rule
// guarantees there's only one). The mid is validated against
// GetCurrentActive: a mismatch returns 409 Conflict so a stale client
// view (someone else has moved the proposal forward) surfaces clearly
// instead of silently mutating the wrong milestone.
//
// The legacy endpoints (/pay, /request-completion, /complete,
// /reject-completion) keep their signatures and still work — they are
// thin shims that delegate to the same proposal service methods, just
// without the milestone-id verification.
// ---------------------------------------------------------------------------

// validateMilestoneMatchesCurrent fetches the current active milestone
// of the proposal and asserts that its id matches the one carried in
// the URL. Returns the milestone on success; on mismatch, writes a 409
// Conflict to the response and returns nil.
func (h *ProposalHandler) validateMilestoneMatchesCurrent(w http.ResponseWriter, r *http.Request, proposalID, expectedMilestoneID uuid.UUID) bool {
	current, err := h.proposalSvc.ListMilestones(r.Context(), proposalID)
	if err != nil {
		handleProposalError(w, err)
		return false
	}
	active := findFirstActiveMilestone(current)
	if active == nil {
		res.Error(w, http.StatusConflict, "no_active_milestone", "no active milestone on this proposal")
		return false
	}
	if active.ID != expectedMilestoneID {
		res.Error(w, http.StatusConflict, "stale_milestone",
			"the milestone id in the URL does not match the current active milestone — refresh and retry")
		return false
	}
	return true
}

// findFirstActiveMilestone is a tiny inline helper to avoid importing
// the milestone domain package solely for FindCurrentActive — the
// proposal service exposes ListMilestones which already returns the
// slice we need to scan. Mirrors milestone.FindCurrentActive.
func findFirstActiveMilestone(milestones []*milestonedomain.Milestone) *milestonedomain.Milestone {
	var current *milestonedomain.Milestone
	for _, m := range milestones {
		if m.IsTerminal() {
			continue
		}
		if current == nil || m.Sequence < current.Sequence {
			current = m
		}
	}
	return current
}

// FundMilestone handles POST /proposals/{id}/milestones/{mid}/fund.
// Validates that {mid} matches the current active milestone, then
// calls InitiatePayment which routes through the milestone state
// machine to create a Stripe PaymentIntent or simulate the payment.
func (h *ProposalHandler) FundMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	if !h.validateMilestoneMatchesCurrent(w, r, proposalID, milestoneID) {
		return
	}
	result, err := h.proposalSvc.InitiatePayment(r.Context(), proposalapp.PayProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}
	if result == nil {
		res.JSON(w, http.StatusOK, map[string]string{"status": "funded"})
		return
	}
	res.JSON(w, http.StatusOK, response.PaymentIntentResponse{
		ClientSecret:    result.ClientSecret,
		PaymentRecordID: result.PaymentRecordID.String(),
		Amounts: response.PaymentAmounts{
			ProposalAmount: result.ProposalAmount,
			StripeFee:      result.StripeFee,
			PlatformFee:    result.PlatformFee,
			ClientTotal:    result.ClientTotal,
			ProviderPayout: result.ProviderPayout,
		},
	})
}

// SubmitMilestone handles POST /proposals/{id}/milestones/{mid}/submit.
// The provider transitions the milestone from funded → submitted.
func (h *ProposalHandler) SubmitMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	if !h.validateMilestoneMatchesCurrent(w, r, proposalID, milestoneID) {
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
func (h *ProposalHandler) ApproveMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	if !h.validateMilestoneMatchesCurrent(w, r, proposalID, milestoneID) {
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
func (h *ProposalHandler) RejectMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	if !h.validateMilestoneMatchesCurrent(w, r, proposalID, milestoneID) {
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

// requireAuthContext is a tiny helper used by the new milestone-scoped
// handlers to extract user_id + organization_id from the JWT context
// in one call. Mirrors the pattern in every existing handler.
func requireAuthContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, orgID, true
}

// CancelProposal handles POST /proposals/{id}/cancel. Either party may
// initiate cancellation at a milestone boundary (no active milestone in
// flight). Already-released milestones stay released; pending_funding
// milestones become cancelled.
func (h *ProposalHandler) CancelProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
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

// parseProposalAndMilestoneID extracts and validates the two URL params.
// Writes a 400 to the response on parse failure and returns ok=false.
func parseProposalAndMilestoneID(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "proposal id must be a valid UUID")
		return uuid.Nil, uuid.Nil, false
	}
	milestoneID, err := uuid.Parse(chi.URLParam(r, "mid"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_milestone_id", "milestone id must be a valid UUID")
		return uuid.Nil, uuid.Nil, false
	}
	return proposalID, milestoneID, true
}

func handleProposalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, proposaldomain.ErrProposalNotFound):
		res.Error(w, http.StatusNotFound, "proposal_not_found", err.Error())
	case errors.Is(err, proposaldomain.ErrNotAuthorized):
		res.Error(w, http.StatusForbidden, "not_authorized", err.Error())
	case errors.Is(err, proposaldomain.ErrCannotModify):
		res.Error(w, http.StatusForbidden, "cannot_modify", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidStatus):
		res.Error(w, http.StatusConflict, "invalid_status", err.Error())
	case errors.Is(err, proposaldomain.ErrSameUser):
		res.Error(w, http.StatusBadRequest, "same_user", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidRoleCombination):
		res.Error(w, http.StatusBadRequest, "invalid_role_combination", err.Error())
	case errors.Is(err, proposaldomain.ErrEmptyTitle):
		res.Error(w, http.StatusBadRequest, "empty_title", err.Error())
	case errors.Is(err, proposaldomain.ErrEmptyDescription):
		res.Error(w, http.StatusBadRequest, "empty_description", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidAmount):
		res.Error(w, http.StatusBadRequest, "invalid_amount", err.Error())
	case errors.Is(err, proposaldomain.ErrBelowMinimumAmount):
		res.Error(w, http.StatusBadRequest, "below_minimum_amount", err.Error())
	case errors.Is(err, proposaldomain.ErrNotProvider):
		res.Error(w, http.StatusForbidden, "not_provider", err.Error())
	case errors.Is(err, proposaldomain.ErrNotClient):
		res.Error(w, http.StatusForbidden, "not_client", err.Error())
	case errors.Is(err, proposaldomain.ErrProviderKYCNotReady):
		// 412 Precondition Failed: the resource state is fine, but a
		// pre-condition (provider has finished Stripe onboarding) is
		// not met. The client should ask the provider to complete
		// payouts setup and retry. Message in French to match the
		// existing user-facing messaging style on the proposal flow.
		res.Error(w, http.StatusPreconditionFailed, "provider_kyc_incomplete",
			"Le prestataire doit terminer son onboarding Stripe avant que ce jalon puisse être libéré. Demande-lui de finaliser sa configuration de paiement.")
	case errors.Is(err, userdomain.ErrKYCRestricted):
		res.Error(w, http.StatusForbidden, "kyc_restricted", "Your account is restricted. Set up your payment info to lift this restriction.")
	default:
		slog.Error("unhandled proposal error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
