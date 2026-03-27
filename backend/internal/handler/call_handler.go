package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	callapp "marketplace-backend/internal/app/call"
	calldomain "marketplace-backend/internal/domain/call"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type CallHandler struct {
	callSvc *callapp.Service
}

func NewCallHandler(svc *callapp.Service) *CallHandler {
	return &CallHandler{callSvc: svc}
}

func (h *CallHandler) InitiateCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.InitiateCallRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	convID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_conversation_id", "conversation_id must be a valid UUID")
		return
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_recipient_id", "recipient_id must be a valid UUID")
		return
	}

	callType := calldomain.Type(req.Type)
	if !callType.IsValid() {
		res.Error(w, http.StatusBadRequest, "invalid_call_type", "type must be audio or video")
		return
	}

	result, err := h.callSvc.Initiate(r.Context(), callapp.InitiateInput{
		ConversationID: convID,
		InitiatorID:    userID,
		RecipientID:    recipientID,
		Type:           callType,
	})
	if err != nil {
		handleCallError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, result)
}

func (h *CallHandler) AcceptCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	callID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_call_id", "id must be a valid UUID")
		return
	}

	result, err := h.callSvc.Accept(r.Context(), callID, userID)
	if err != nil {
		handleCallError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, result)
}

func (h *CallHandler) DeclineCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	callID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_call_id", "id must be a valid UUID")
		return
	}

	if err := h.callSvc.Decline(r.Context(), callID, userID); err != nil {
		handleCallError(w, err)
		return
	}

	res.NoContent(w)
}

func (h *CallHandler) EndCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	callID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_call_id", "id must be a valid UUID")
		return
	}

	var req request.EndCallRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.callSvc.End(r.Context(), callapp.EndInput{
		CallID:   callID,
		UserID:   userID,
		Duration: req.Duration,
	}); err != nil {
		handleCallError(w, err)
		return
	}

	res.NoContent(w)
}

func handleCallError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, calldomain.ErrCallNotFound):
		res.Error(w, http.StatusNotFound, "call_not_found", "call not found")
	case errors.Is(err, calldomain.ErrUserBusy):
		res.Error(w, http.StatusConflict, "user_busy", "user is already in a call")
	case errors.Is(err, calldomain.ErrRecipientOffline):
		res.Error(w, http.StatusUnprocessableEntity, "recipient_offline", "recipient is offline")
	case errors.Is(err, calldomain.ErrNotParticipant):
		res.Error(w, http.StatusForbidden, "not_participant", "you are not a participant of this call")
	case errors.Is(err, calldomain.ErrInvalidTransition):
		res.Error(w, http.StatusConflict, "invalid_transition", "invalid call status transition")
	case errors.Is(err, calldomain.ErrNotConfigured):
		res.Error(w, http.StatusServiceUnavailable, "call_not_configured", "call service is not available")
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
