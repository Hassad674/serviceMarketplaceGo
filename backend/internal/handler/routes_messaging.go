package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountMessagingRoutes wires the conversation + message surface
// (/messaging) and the LiveKit-backed call control plane (/calls).
func mountMessagingRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	mountConversationRoutes(r, deps, auth)
	mountCallRoutes(r, deps, auth)
}

func mountConversationRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Messaging == nil {
		return
	}
	r.Route("/messaging", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		// Read operations
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermMessagingView))
			r.Get("/conversations", deps.Messaging.ListConversations)
			r.Get("/conversations/{id}/messages", deps.Messaging.ListMessages)
			r.Post("/conversations/{id}/read", deps.Messaging.MarkAsRead)
			r.Get("/unread-count", deps.Messaging.GetTotalUnread)
		})
		// Write operations
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermMessagingSend))
			r.Post("/conversations", deps.Messaging.StartConversation)
			r.Post("/conversations/{id}/messages", deps.Messaging.SendMessage)
			r.Put("/messages/{id}", deps.Messaging.EditMessage)
			r.Delete("/messages/{id}", deps.Messaging.DeleteMessage)
			r.Post("/upload-url", deps.Messaging.GetPresignedURL)
		})
	})
}

func mountCallRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Call == nil {
		return
	}
	r.Route("/calls", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.With(middleware.RequirePermission(organization.PermMessagingSend)).Post("/initiate", deps.Call.InitiateCall)
		// Accept/decline/end are receiving-side actions — view permission is sufficient
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermMessagingView))
			r.Post("/{id}/accept", deps.Call.AcceptCall)
			r.Post("/{id}/decline", deps.Call.DeclineCall)
			r.Post("/{id}/end", deps.Call.EndCall)
		})
	})
}
