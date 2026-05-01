package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/handler/middleware"
)

// mountNotificationRoutes wires the user-facing /notifications surface.
// All endpoints require auth; per-route permission is unnecessary
// because every notification row is already filtered by user_id at the
// repository level.
func mountNotificationRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Notification == nil {
		return
	}
	r.Route("/notifications", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/", deps.Notification.ListNotifications)
		r.Get("/unread-count", deps.Notification.GetUnreadCount)
		r.Post("/{id}/read", deps.Notification.MarkAsRead)
		r.Post("/read-all", deps.Notification.MarkAllAsRead)
		r.Delete("/{id}", deps.Notification.DeleteNotification)
		r.Get("/preferences", deps.Notification.GetPreferences)
		r.Put("/preferences", deps.Notification.UpdatePreferences)
		r.Patch("/preferences/bulk-email", deps.Notification.BulkUpdateEmailPreferences)
		r.Post("/device-token", deps.Notification.RegisterDeviceToken)
	})
}
