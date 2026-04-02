package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	adminapp "marketplace-backend/internal/app/admin"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

type AdminHandler struct {
	svc *adminapp.Service
}

func NewAdminHandler(svc *adminapp.Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListUsers handles GET /api/v1/admin/users.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	filters := repository.AdminUserFilters{
		Role:   r.URL.Query().Get("role"),
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("search"),
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  parseLimit(r.URL.Query().Get("limit"), 20),
	}

	users, nextCursor, total, err := h.svc.ListUsers(r.Context(), filters)
	if err != nil {
		slog.Error("admin list users", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list users")
		return
	}

	items := make([]response.AdminUserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, response.NewAdminUserResponse(u))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
		"total":       total,
	})
}

// GetUser handles GET /api/v1/admin/users/{id}.
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	u, err := h.svc.GetUser(r.Context(), id)
	if err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewAdminUserResponse(u),
	})
}

// SuspendUser handles POST /api/v1/admin/users/{id}/suspend.
func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var body struct {
		Reason    string  `json:"reason"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if body.Reason == "" {
		res.Error(w, http.StatusBadRequest, "validation_error", "reason is required")
		return
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "validation_error", "expires_at must be RFC3339 format")
			return
		}
		expiresAt = &t
	}

	if err := h.svc.SuspendUser(r.Context(), id, body.Reason, expiresAt); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user suspended"})
}

// UnsuspendUser handles POST /api/v1/admin/users/{id}/unsuspend.
func (h *AdminHandler) UnsuspendUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.UnsuspendUser(r.Context(), id); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user unsuspended"})
}

// BanUser handles POST /api/v1/admin/users/{id}/ban.
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if body.Reason == "" {
		res.Error(w, http.StatusBadRequest, "validation_error", "reason is required")
		return
	}

	if err := h.svc.BanUser(r.Context(), id, body.Reason); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user banned"})
}

// UnbanUser handles POST /api/v1/admin/users/{id}/unban.
func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.UnbanUser(r.Context(), id); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user unbanned"})
}

func parseAdminUserID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}

func handleAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		res.Error(w, http.StatusNotFound, "user_not_found", err.Error())
	default:
		slog.Error("unhandled admin error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
