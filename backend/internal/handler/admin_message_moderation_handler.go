package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// ApproveMessageModeration handles POST /api/v1/admin/messages/{id}/approve-moderation.
// Resolves the (id, adminID) pair from the URL + JWT context and
// delegates the dual-store write to the admin service. The handler
// stays thin — every business decision lives in app/admin.
func (h *AdminHandler) ApproveMessageModeration(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}
	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.ApproveMessageModeration(r.Context(), id, adminID); err != nil {
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
	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.HideMessage(r.Context(), id, adminID); err != nil {
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
	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.ApproveReviewModeration(r.Context(), id, adminID); err != nil {
		slog.Error("admin approve review moderation", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to approve review")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "review moderation approved"})
}

// RestoreMessageModeration handles POST /api/v1/admin/messages/{id}/restore-moderation.
// Clears an auto-applied soft-delete (or hidden) status so the message
// becomes visible again. Used by the admin to unwind false positives.
func (h *AdminHandler) RestoreMessageModeration(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}
	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.RestoreMessageModeration(r.Context(), id, adminID); err != nil {
		slog.Error("admin restore message moderation", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to restore message")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "message moderation restored"})
}

// RestoreReviewModeration handles POST /api/v1/admin/reviews/{id}/restore-moderation.
func (h *AdminHandler) RestoreReviewModeration(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}
	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.RestoreReviewModeration(r.Context(), id, adminID); err != nil {
		slog.Error("admin restore review moderation", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to restore review")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "review moderation restored"})
}

// RestoreModerationGeneric handles
//   POST /api/v1/admin/moderation/{content_type}/{content_id}/restore
//
// Generic restore covering every Phase 2 content type via a single
// route, so the admin frontend does not need a hand-rolled endpoint
// per content type. The handler delegates to the admin service which
// resolves the target on (content_type, content_id) in
// moderation_results and writes the override.
func (h *AdminHandler) RestoreModerationGeneric(w http.ResponseWriter, r *http.Request) {
	contentType := chi.URLParam(r, "content_type")
	if contentType == "" {
		res.Error(w, http.StatusBadRequest, "invalid_content_type", "content_type is required")
		return
	}
	contentID, err := uuid.Parse(chi.URLParam(r, "content_id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "content_id must be a valid UUID")
		return
	}
	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.RestoreModeration(r.Context(), contentType, contentID, adminID); err != nil {
		slog.Error("admin restore moderation (generic)", "error", err,
			"content_type", contentType, "content_id", contentID)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to restore content")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "moderation restored"})
}
