package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

// GetNotificationCounters handles GET /api/v1/admin/notifications.
// Returns per-admin notification counters for all categories.
func (h *AdminHandler) GetNotificationCounters(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "missing user ID")
		return
	}

	counters, err := h.svc.GetNotificationCounters(r.Context(), adminID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get notification counters")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": counters})
}

// ResetNotificationCounter handles POST /api/v1/admin/notifications/{category}/reset.
// Resets the counter for a specific category for the requesting admin.
func (h *AdminHandler) ResetNotificationCounter(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "missing user ID")
		return
	}

	category := chi.URLParam(r, "category")
	if !isValidNotifCategory(category) {
		res.Error(w, http.StatusBadRequest, "validation_error", "invalid notification category")
		return
	}

	if err := h.svc.ResetNotificationCounter(r.Context(), adminID, category); err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to reset notification counter")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "counter reset"})
}

func isValidNotifCategory(category string) bool {
	for _, c := range portservice.AdminNotifCategories {
		if c == category {
			return true
		}
	}
	return false
}
