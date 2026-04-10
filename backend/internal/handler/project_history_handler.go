package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	projecthistoryapp "marketplace-backend/internal/app/projecthistory"
	"marketplace-backend/internal/handler/dto/response"

	res "marketplace-backend/pkg/response"
)

// ProjectHistoryHandler serves the public project history of a provider.
type ProjectHistoryHandler struct {
	svc *projecthistoryapp.Service
}

// NewProjectHistoryHandler creates a new ProjectHistoryHandler.
func NewProjectHistoryHandler(svc *projecthistoryapp.Service) *ProjectHistoryHandler {
	return &ProjectHistoryHandler{svc: svc}
}

// ListByProvider handles GET /api/v1/profiles/{userId}/project-history.
// Public endpoint — no authentication required.
func (h *ProjectHistoryHandler) ListByProvider(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "userId must be a valid UUID")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	entries, nextCursor, err := h.svc.ListByProvider(r.Context(), userID, cursor, limit)
	if err != nil {
		slog.Error("list project history", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list project history")
		return
	}

	res.JSON(w, http.StatusOK, response.NewProjectHistoryListResponse(entries, nextCursor))
}
