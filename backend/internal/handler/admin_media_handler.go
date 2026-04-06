package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

// ListMedia handles GET /api/v1/admin/media.
func (h *AdminHandler) ListMedia(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	page := parsePage(r.URL.Query().Get("page"))

	filters := repository.AdminMediaFilters{
		Status:  r.URL.Query().Get("status"),
		Type:    r.URL.Query().Get("type"),
		Context: r.URL.Query().Get("context"),
		Search:  r.URL.Query().Get("search"),
		Page:    page,
		Limit:   limit,
	}

	items, total, err := h.svc.ListMedia(r.Context(), filters)
	if err != nil {
		slog.Error("admin list media", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list media")
		return
	}

	data := make([]response.AdminMediaResponse, 0, len(items))
	for _, item := range items {
		data = append(data, response.NewAdminMediaResponse(item))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        data,
		"total":       total,
		"page":        page,
		"total_pages": totalPages(total, limit),
	})
}

// GetMediaDetail handles GET /api/v1/admin/media/{id}.
func (h *AdminHandler) GetMediaDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	m, err := h.svc.GetMedia(r.Context(), id)
	if err != nil {
		handleMediaError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewAdminMediaResponse(*m),
	})
}

// ApproveMedia handles POST /api/v1/admin/media/{id}/approve.
func (h *AdminHandler) ApproveMedia(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.ApproveMedia(r.Context(), id, adminID); err != nil {
		handleMediaError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "media approved"})
}

// RejectMedia handles POST /api/v1/admin/media/{id}/reject.
func (h *AdminHandler) RejectMedia(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.RejectMedia(r.Context(), id, adminID); err != nil {
		handleMediaError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "media rejected"})
}

// DeleteMedia handles DELETE /api/v1/admin/media/{id}.
func (h *AdminHandler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.DeleteMedia(r.Context(), id); err != nil {
		handleMediaError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleMediaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, mediadomain.ErrMediaNotFound):
		res.Error(w, http.StatusNotFound, "media_not_found", err.Error())
	default:
		slog.Error("unhandled media error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
