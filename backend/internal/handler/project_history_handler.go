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

// ProjectHistoryHandler serves the public project history of an org.
type ProjectHistoryHandler struct {
	svc *projecthistoryapp.Service
}

// NewProjectHistoryHandler creates a new ProjectHistoryHandler.
func NewProjectHistoryHandler(svc *projecthistoryapp.Service) *ProjectHistoryHandler {
	return &ProjectHistoryHandler{svc: svc}
}

// ListByOrganization handles GET /api/v1/profiles/{orgId}/project-history.
// Public endpoint — no authentication required. Returns the organization's
// completed deliveries (provider-side of the proposal).
func (h *ProjectHistoryHandler) ListByOrganization(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "orgId must be a valid UUID")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	entries, nextCursor, err := h.svc.ListByOrganization(r.Context(), orgID, cursor, limit)
	if err != nil {
		slog.Error("list project history", "error", err, "org_id", orgID)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list project history")
		return
	}

	res.JSON(w, http.StatusOK, response.NewProjectHistoryListResponse(entries, nextCursor))
}
