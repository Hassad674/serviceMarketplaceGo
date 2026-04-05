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
	proposaldomain "marketplace-backend/internal/domain/proposal"
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

	p, err := h.proposalSvc.CreateProposal(r.Context(), proposalapp.CreateProposalInput{
		ConversationID: convID,
		SenderID:       userID,
		RecipientID:    recipientID,
		Title:          req.Title,
		Description:    req.Description,
		Amount:         req.Amount,
		Deadline:       deadline,
		Documents:      docs,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewProposalResponse(p, nil))
}

func (h *ProposalHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	p, docs, err := h.proposalSvc.GetProposal(r.Context(), userID, proposalID)
	if err != nil {
		handleProposalError(w, err)
		return
	}

	clientName, providerName := h.proposalSvc.GetParticipantNames(r.Context(), p.ClientID, p.ProviderID)
	res.JSON(w, http.StatusOK, response.NewProposalResponseWithNames(p, docs, clientName, providerName))
}

func (h *ProposalHandler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
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

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.DeclineProposal(r.Context(), proposalapp.DeclineProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
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

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	result, err := h.proposalSvc.InitiatePayment(r.Context(), proposalapp.PayProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
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
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	// Verify the user is the client
	p, err := h.proposalSvc.GetProposalByID(r.Context(), proposalID)
	if err != nil {
		handleProposalError(w, err)
		return
	}
	if p.ClientID != userID {
		res.Error(w, http.StatusForbidden, "not_authorized", "only the client can confirm payment")
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

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.RequestCompletion(r.Context(), proposalapp.RequestCompletionInput{
		ProposalID: proposalID,
		UserID:     userID,
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

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.CompleteProposal(r.Context(), proposalapp.CompleteProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
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

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.RejectCompletion(r.Context(), proposalapp.RejectCompletionInput{
		ProposalID: proposalID,
		UserID:     userID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func (h *ProposalHandler) ListActiveProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	proposals, nextCursor, err := h.proposalSvc.ListActiveProjects(r.Context(), userID, cursorStr, limit)
	if err != nil {
		handleProposalError(w, err)
		return
	}

	data := make([]response.ProposalResponse, len(proposals))
	for i, p := range proposals {
		cn, pn := h.proposalSvc.GetParticipantNames(r.Context(), p.ClientID, p.ProviderID)
		data[i] = response.NewProposalResponseWithNames(p, nil, cn, pn)
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
	default:
		slog.Error("unhandled proposal error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
