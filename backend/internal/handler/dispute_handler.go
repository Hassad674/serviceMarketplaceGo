package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	disputeapp "marketplace-backend/internal/app/dispute"
	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

type DisputeHandler struct {
	svc *disputeapp.Service
}

func NewDisputeHandler(svc *disputeapp.Service) *DisputeHandler {
	return &DisputeHandler{svc: svc}
}

func (h *DisputeHandler) OpenDispute(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.OpenDisputeRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	proposalID, err := uuid.Parse(req.ProposalID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "invalid proposal ID")
		return
	}

	var attachments []disputeapp.AttachmentInput
	for _, a := range req.Attachments {
		attachments = append(attachments, disputeapp.AttachmentInput{
			Filename: a.Filename, URL: a.URL, Size: a.Size, MimeType: a.MimeType,
		})
	}

	d, err := h.svc.OpenDispute(r.Context(), disputeapp.OpenDisputeInput{
		ProposalID:      proposalID,
		InitiatorID:     userID,
		Reason:          req.Reason,
		Description:     req.Description,
		MessageToParty:  req.MessageToParty,
		RequestedAmount: req.RequestedAmount,
		Attachments:     attachments,
	})
	if err != nil {
		slog.Error("open dispute failed", "error", err, "proposal_id", req.ProposalID, "reason", req.Reason, "amount", req.RequestedAmount)
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewDisputeListItem(d))
}

func (h *DisputeHandler) GetDispute(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	detail, err := h.svc.GetDispute(r.Context(), userID, disputeID)
	if err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewDisputeResponse(
		detail.Dispute, detail.Evidence, detail.CounterProposals))
}

func (h *DisputeHandler) CounterPropose(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	var req request.CounterProposeRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	var attachments []disputeapp.AttachmentInput
	for _, a := range req.Attachments {
		attachments = append(attachments, disputeapp.AttachmentInput{
			Filename: a.Filename, URL: a.URL, Size: a.Size, MimeType: a.MimeType,
		})
	}

	cp, err := h.svc.CounterPropose(r.Context(), disputeapp.CounterProposeInput{
		DisputeID:      disputeID,
		ProposerID:     userID,
		AmountClient:   req.AmountClient,
		AmountProvider: req.AmountProvider,
		Message:        req.Message,
		Attachments:    attachments,
	})
	if err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, map[string]string{"id": cp.ID.String()})
}

func (h *DisputeHandler) RespondToCounter(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	cpID, err := uuid.Parse(chi.URLParam(r, "cpId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid counter-proposal ID")
		return
	}

	var req request.RespondToCounterRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	if err := h.svc.RespondToCounter(r.Context(), disputeapp.RespondToCounterInput{
		DisputeID:         disputeID,
		CounterProposalID: cpID,
		UserID:            userID,
		Accept:            req.Accept,
	}); err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *DisputeHandler) CancelDispute(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	result, err := h.svc.CancelDispute(r.Context(), disputeapp.CancelDisputeInput{
		DisputeID: disputeID,
		UserID:    userID,
	})
	if err != nil {
		handleDisputeError(w, err)
		return
	}

	status := "cancelled"
	if result.Requested {
		status = "cancellation_requested"
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": status})
}

func (h *DisputeHandler) RespondToCancellation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	var req request.RespondToCancellationRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	if err := h.svc.RespondToCancellation(r.Context(), disputeapp.RespondToCancellationInput{
		DisputeID: disputeID,
		UserID:    userID,
		Accept:    req.Accept,
	}); err != nil {
		handleDisputeError(w, err)
		return
	}

	status := "refused"
	if req.Accept {
		status = "cancelled"
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": status})
}

func (h *DisputeHandler) ListMyDisputes(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	disputes, nextCursor, err := h.svc.ListMyDisputes(r.Context(), userID, cursor, 20)
	if err != nil {
		handleDisputeError(w, err)
		return
	}

	items := make([]response.DisputeResponse, 0, len(disputes))
	for _, d := range disputes {
		items = append(items, response.NewDisputeListItem(d))
	}

	res.JSON(w, http.StatusOK, response.DisputeListResponse{
		Data: items, NextCursor: nextCursor, HasMore: nextCursor != "",
	})
}

func handleDisputeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, disputedomain.ErrDisputeNotFound):
		res.Error(w, http.StatusNotFound, "dispute_not_found", err.Error())
	case errors.Is(err, disputedomain.ErrProposalNotDisputable):
		res.Error(w, http.StatusConflict, "proposal_not_disputable", err.Error())
	case errors.Is(err, disputedomain.ErrAlreadyDisputed):
		res.Error(w, http.StatusConflict, "already_disputed", err.Error())
	case errors.Is(err, disputedomain.ErrInvalidStatus):
		res.Error(w, http.StatusConflict, "invalid_status", err.Error())
	case errors.Is(err, disputedomain.ErrInvalidReason):
		res.Error(w, http.StatusBadRequest, "invalid_reason", err.Error())
	case errors.Is(err, disputedomain.ErrEmptyDescription):
		res.Error(w, http.StatusBadRequest, "empty_description", err.Error())
	case errors.Is(err, disputedomain.ErrDescriptionTooLong):
		res.Error(w, http.StatusBadRequest, "description_too_long", err.Error())
	case errors.Is(err, disputedomain.ErrInvalidAmount):
		res.Error(w, http.StatusBadRequest, "invalid_amount", err.Error())
	case errors.Is(err, disputedomain.ErrAmountMismatch):
		res.Error(w, http.StatusBadRequest, "amount_mismatch", err.Error())
	case errors.Is(err, disputedomain.ErrNotParticipant):
		res.Error(w, http.StatusForbidden, "not_participant", err.Error())
	case errors.Is(err, disputedomain.ErrNotAuthorized):
		res.Error(w, http.StatusForbidden, "not_authorized", err.Error())
	case errors.Is(err, disputedomain.ErrCancellationRequiresConsent):
		res.Error(w, http.StatusConflict, "cancellation_requires_consent", err.Error())
	case errors.Is(err, disputedomain.ErrCancellationAlreadyRequested):
		res.Error(w, http.StatusConflict, "cancellation_already_requested", err.Error())
	case errors.Is(err, disputedomain.ErrNoCancellationPending):
		res.Error(w, http.StatusConflict, "no_cancellation_pending", err.Error())
	case errors.Is(err, disputedomain.ErrAIBudgetSummaryExceeded):
		res.Error(w, http.StatusTooManyRequests, "ai_budget_summary_exceeded", err.Error())
	case errors.Is(err, disputedomain.ErrAIBudgetChatExceeded):
		res.Error(w, http.StatusTooManyRequests, "ai_budget_chat_exceeded", err.Error())
	case errors.Is(err, disputedomain.ErrCounterProposalNotFound):
		res.Error(w, http.StatusNotFound, "counter_proposal_not_found", err.Error())
	case errors.Is(err, disputedomain.ErrCounterProposalNotPending):
		res.Error(w, http.StatusConflict, "counter_proposal_not_pending", err.Error())
	case errors.Is(err, disputedomain.ErrCannotRespondToOwnProposal):
		res.Error(w, http.StatusForbidden, "cannot_respond_own_proposal", err.Error())
	default:
		slog.Error("unhandled dispute error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
