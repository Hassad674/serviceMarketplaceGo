package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	reportapp "marketplace-backend/internal/app/report"
	reportdomain "marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ReportHandler handles HTTP requests for reports.
type ReportHandler struct {
	reportSvc *reportapp.Service
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(svc *reportapp.Service) *ReportHandler {
	return &ReportHandler{reportSvc: svc}
}

// CreateReport handles POST /api/v1/reports.
func (h *ReportHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.CreateReportRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_target_id", "target_id must be a valid UUID")
		return
	}

	var conversationID uuid.UUID
	if req.ConversationID != "" {
		cid, parseErr := uuid.Parse(req.ConversationID)
		if parseErr != nil {
			res.Error(w, http.StatusBadRequest, "invalid_conversation_id", "conversation_id must be a valid UUID")
			return
		}
		conversationID = cid
	}

	rp, err := h.reportSvc.CreateReport(r.Context(), reportapp.CreateReportInput{
		ReporterID:     userID,
		TargetType:     req.TargetType,
		TargetID:       targetID,
		ConversationID: conversationID,
		Reason:         req.Reason,
		Description:    req.Description,
	})
	if err != nil {
		handleReportError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, map[string]any{
		"data": response.ReportFromDomain(rp),
	})
}

// ListMyReports handles GET /api/v1/reports/mine.
func (h *ReportHandler) ListMyReports(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	reports, nextCursor, err := h.reportSvc.ListMyReports(r.Context(), userID, cursor, limit)
	if err != nil {
		slog.Error("list my reports", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list reports")
		return
	}

	items := make([]response.ReportResponse, 0, len(reports))
	for _, rp := range reports {
		items = append(items, response.ReportFromDomain(rp))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

func handleReportError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reportdomain.ErrInvalidTargetType):
		res.Error(w, http.StatusBadRequest, "invalid_target_type", err.Error())
	case errors.Is(err, reportdomain.ErrInvalidReason):
		res.Error(w, http.StatusBadRequest, "invalid_reason", err.Error())
	case errors.Is(err, reportdomain.ErrReasonNotAllowedForType):
		res.Error(w, http.StatusBadRequest, "invalid_reason", err.Error())
	case errors.Is(err, reportdomain.ErrDescriptionTooLong):
		res.Error(w, http.StatusBadRequest, "description_too_long", err.Error())
	case errors.Is(err, reportdomain.ErrSelfReport):
		res.Error(w, http.StatusBadRequest, "self_report", err.Error())
	case errors.Is(err, reportdomain.ErrAlreadyReported):
		res.Error(w, http.StatusConflict, "already_reported", err.Error())
	case errors.Is(err, reportdomain.ErrMissingReporter):
		res.Error(w, http.StatusBadRequest, "missing_reporter", err.Error())
	case errors.Is(err, reportdomain.ErrMissingTarget):
		res.Error(w, http.StatusBadRequest, "missing_target", err.Error())
	default:
		slog.Error("unhandled report error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
