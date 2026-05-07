package handler

import (
	"errors"
	"net/http"
	"time"

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

// MyActiveCallResponse describes the caller's currently active call,
// used by the front-end at mount to reconcile orphan Redis state
// (browser closed mid-call, network loss, hangup race). Fields mirror
// the call entity but expose only what the UI needs to decide the
// fallback action — never the LiveKit token (a fresh accept/end is
// always required to (re)join a room).
type MyActiveCallResponse struct {
	CallID            string  `json:"call_id"`
	ConversationID    string  `json:"conversation_id"`
	RoomName          string  `json:"room_name"`
	Type              string  `json:"type"`
	Status            string  `json:"status"`
	StartedAt         *string `json:"started_at,omitempty"`
	OtherParticipantID string `json:"other_participant_id"`
}

// GetMyActiveCall is the reconciliation read for the caller's current
// call. Returns `{ "data": null }` when there is no active call —
// preferred over 404 because the absence of a call is the expected
// case for almost every page load, not a "not found" condition.
func (h *CallHandler) GetMyActiveCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	c, err := h.callSvc.GetActiveCallForUser(r.Context(), userID)
	if err != nil {
		handleCallError(w, err)
		return
	}
	if c == nil {
		res.JSON(w, http.StatusOK, map[string]any{"data": nil})
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": newMyActiveCallResponse(c, userID),
	})
}

// newMyActiveCallResponse builds the wire DTO from the domain call.
// `other_participant_id` is computed relative to the caller so the
// front-end can render a "Tu as un appel en cours avec X" message
// without a second round-trip.
func newMyActiveCallResponse(c *calldomain.Call, callerID uuid.UUID) MyActiveCallResponse {
	other := c.RecipientID
	if c.RecipientID == callerID {
		other = c.InitiatorID
	}
	resp := MyActiveCallResponse{
		CallID:             c.ID.String(),
		ConversationID:     c.ConversationID.String(),
		RoomName:           c.RoomName,
		Type:               string(c.Type),
		Status:             string(c.Status),
		OtherParticipantID: other.String(),
	}
	if c.StartedAt != nil {
		t := c.StartedAt.UTC().Format(time.RFC3339)
		resp.StartedAt = &t
	}
	return resp
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
