package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	res "marketplace-backend/pkg/response"
)

// ApproveMessageModeration handles POST /api/v1/admin/messages/{id}/approve-moderation.
func (h *AdminHandler) ApproveMessageModeration(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.ApproveMessageModeration(r.Context(), id); err != nil {
		slog.Error("admin approve message moderation", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to approve message")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "message moderation approved"})
}

// HideMessage handles POST /api/v1/admin/messages/{id}/hide.
func (h *AdminHandler) HideMessage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.HideMessage(r.Context(), id); err != nil {
		slog.Error("admin hide message", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to hide message")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "message hidden"})
}

// ApproveReviewModeration handles POST /api/v1/admin/reviews/{id}/approve-moderation.
func (h *AdminHandler) ApproveReviewModeration(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.ApproveReviewModeration(r.Context(), id); err != nil {
		slog.Error("admin approve review moderation", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to approve review")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "review moderation approved"})
}
