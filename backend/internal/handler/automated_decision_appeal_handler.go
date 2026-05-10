package handler

import (
	"errors"
	"log/slog"
	"net/http"

	app "marketplace-backend/internal/app/automateddecision"
	"marketplace-backend/internal/domain/automateddecision"
	"marketplace-backend/internal/handler/middleware"
	jsondec "marketplace-backend/pkg/decode"
	res "marketplace-backend/pkg/response"
)

// AutomatedDecisionAppealHandler exposes the single endpoint
// POST /api/v1/me/automated-decision-appeals used by an authenticated
// user to file a request for human review of an automated decision
// (RGPD art. 22).
//
// The handler is intentionally thin — DTO decode, ownership is implicit
// (the user_id comes from the JWT context), service call, JSON response.
type AutomatedDecisionAppealHandler struct {
	svc *app.Service
}

// NewAutomatedDecisionAppealHandler wires the handler to the FileAppeal
// service.
func NewAutomatedDecisionAppealHandler(svc *app.Service) *AutomatedDecisionAppealHandler {
	return &AutomatedDecisionAppealHandler{svc: svc}
}

// fileAppealRequest is the JSON body shape accepted by FileAppeal.
// Field names mirror the domain to keep the contract obvious to a
// reader of the OpenAPI schema.
type fileAppealRequest struct {
	DecisionType string `json:"decision_type"`
	ReferenceID  string `json:"reference_id"`
	Reason       string `json:"reason"`
}

// fileAppealResponse is the canonical JSON shape returned to clients.
// Echoes the persisted ID and Status so the user can poll an admin
// queue surface (out of scope for B.5).
type fileAppealResponse struct {
	ID           string `json:"id"`
	DecisionType string `json:"decision_type"`
	ReferenceID  string `json:"reference_id"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// FileAppeal records a new request for human review.
//
//	POST /api/v1/me/automated-decision-appeals
//	{ "decision_type": "moderation", "reference_id": "...", "reason": "..." }
//
// Returns 201 + the persisted entity on success, 400 on validation,
// 401 when the request lacks an authenticated user, 503 when the
// feature is disabled (svc nil).
func (h *AutomatedDecisionAppealHandler) FileAppeal(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "feature_disabled",
			"automated decision appeals are not configured")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	// 8 KiB cap covers the 5_000-byte reason ceiling with comfortable
	// headroom for the surrounding JSON envelope. Anything bigger is a
	// DoS attempt — short-circuit at the transport layer.
	var req fileAppealRequest
	if err := jsondec.DecodeBody(w, r, &req, 8<<10); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	appeal, err := h.svc.FileAppeal(r.Context(), app.FileAppealInput{
		UserID:       userID,
		DecisionType: req.DecisionType,
		ReferenceID:  req.ReferenceID,
		Reason:       req.Reason,
	})
	if err != nil {
		switch {
		case errors.Is(err, automateddecision.ErrInvalidDecisionType),
			errors.Is(err, automateddecision.ErrReferenceIDRequired),
			errors.Is(err, automateddecision.ErrReasonRequired),
			errors.Is(err, automateddecision.ErrReasonTooLong):
			res.Error(w, http.StatusBadRequest, "invalid_input", err.Error())
		default:
			slog.Error("automated_decision: file appeal", "error", err.Error())
			res.Error(w, http.StatusInternalServerError, "internal_error",
				"could not record appeal")
		}
		return
	}

	res.JSON(w, http.StatusCreated, fileAppealResponse{
		ID:           appeal.ID.String(),
		DecisionType: string(appeal.DecisionType),
		ReferenceID:  appeal.ReferenceID,
		Status:       string(appeal.Status),
		CreatedAt:    appeal.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}
