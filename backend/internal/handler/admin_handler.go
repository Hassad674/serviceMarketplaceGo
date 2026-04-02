package handler

import (
	"errors"
	"log/slog"
	"net/http"

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
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
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

func handleAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		res.Error(w, http.StatusNotFound, "user_not_found", err.Error())
	default:
		slog.Error("unhandled admin error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
