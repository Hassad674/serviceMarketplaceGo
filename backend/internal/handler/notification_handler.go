package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	notifapp "marketplace-backend/internal/app/notification"
	notifdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type NotificationHandler struct {
	notifSvc *notifapp.Service
}

func NewNotificationHandler(svc *notifapp.Service) *NotificationHandler {
	return &NotificationHandler{notifSvc: svc}
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursorParam := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	notifs, nextCursor, err := h.notifSvc.List(r.Context(), userID, cursorParam, limit)
	if err != nil {
		slog.Error("list notifications", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list notifications")
		return
	}

	items := make([]response.NotificationResponse, 0, len(notifs))
	for _, n := range notifs {
		items = append(items, response.NotificationFromDomain(n))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	count, err := h.notifSvc.GetUnreadCount(r.Context(), userID)
	if err != nil {
		slog.Error("get unread count", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get unread count")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": map[string]int{"count": count},
	})
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.notifSvc.MarkAsRead(r.Context(), id, userID); err != nil {
		handleNotificationError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	if err := h.notifSvc.MarkAllAsRead(r.Context(), userID); err != nil {
		slog.Error("mark all as read", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to mark all as read")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *NotificationHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.notifSvc.Delete(r.Context(), id, userID); err != nil {
		handleNotificationError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	prefs, err := h.notifSvc.GetPreferences(r.Context(), userID)
	if err != nil {
		slog.Error("get preferences", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get preferences")
		return
	}

	items := make([]response.NotificationPreferenceResponse, 0, len(prefs))
	for _, p := range prefs {
		items = append(items, response.PreferenceFromDomain(p))
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.UpdateNotificationPreferencesRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	prefs := make([]*notifdomain.Preferences, 0, len(req.Preferences))
	for _, p := range req.Preferences {
		prefs = append(prefs, &notifdomain.Preferences{
			UserID:           userID,
			NotificationType: notifdomain.NotificationType(p.Type),
			InApp:            p.InApp,
			Push:             p.Push,
			Email:            p.Email,
		})
	}

	if err := h.notifSvc.UpdatePreferences(r.Context(), userID, prefs); err != nil {
		slog.Error("update preferences", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to update preferences")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *NotificationHandler) RegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.RegisterDeviceTokenRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.Token == "" {
		res.Error(w, http.StatusBadRequest, "missing_token", "token is required")
		return
	}
	if req.Platform != "android" && req.Platform != "ios" && req.Platform != "web" {
		res.Error(w, http.StatusBadRequest, "invalid_platform", "platform must be android, ios, or web")
		return
	}

	if err := h.notifSvc.RegisterDevice(r.Context(), userID, req.Token, req.Platform); err != nil {
		slog.Error("register device token", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to register device")
		return
	}

	res.JSON(w, http.StatusCreated, map[string]any{"data": map[string]string{"status": "ok"}})
}

func handleNotificationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, notifdomain.ErrNotFound):
		res.Error(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, notifdomain.ErrNotOwner):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		slog.Error("unhandled notification error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
