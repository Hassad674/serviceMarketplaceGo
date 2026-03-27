package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
)

type RouterDeps struct {
	Auth           *AuthHandler
	Profile        *ProfileHandler
	Upload         *UploadHandler
	Health         *HealthHandler
	Messaging      *MessagingHandler
	Proposal       *ProposalHandler
	WSHandler      http.HandlerFunc
	Config         *config.Config
	TokenService   service.TokenService
	SessionService service.SessionService
}

func NewRouter(deps RouterDeps) chi.Router {
	r := chi.NewRouter()

	// Global middleware
	limiter := middleware.NewRateLimiter(10, 20) // 10 req/s, burst 20
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(deps.Config.AllowedOrigins))
	r.Use(limiter.Middleware)

	// Health routes
	r.Get("/health", deps.Health.Health)
	r.Get("/ready", deps.Health.Ready)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", deps.Auth.Register)
			r.Post("/login", deps.Auth.Login)
			r.Post("/refresh", deps.Auth.Refresh)
			r.Post("/forgot-password", deps.Auth.ForgotPassword)
			r.Post("/reset-password", deps.Auth.ResetPassword)

			// Protected
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Get("/me", deps.Auth.Me)
				r.Post("/logout", deps.Auth.Logout)
				r.Put("/referrer-enable", deps.Auth.EnableReferrer)
			})
		})

		// Profile routes (authenticated)
		r.Route("/profile", func(r chi.Router) {
			r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
			r.Get("/", deps.Profile.GetMyProfile)
			r.Put("/", deps.Profile.UpdateMyProfile)
		})

		// Upload routes (authenticated)
		r.Route("/upload", func(r chi.Router) {
			r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
			r.Post("/photo", deps.Upload.UploadPhoto)
			r.Post("/video", deps.Upload.UploadVideo)
			r.Delete("/video", deps.Upload.DeleteVideo)
			r.Post("/referrer-video", deps.Upload.UploadReferrerVideo)
			r.Delete("/referrer-video", deps.Upload.DeleteReferrerVideo)
		})

		// Public profiles
		r.Get("/profiles/search", deps.Profile.SearchProfiles)
		r.Get("/profiles/{userId}", deps.Profile.GetPublicProfile)

		// Messaging routes (authenticated)
		if deps.Messaging != nil {
			r.Route("/messaging", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Post("/conversations", deps.Messaging.StartConversation)
				r.Get("/conversations", deps.Messaging.ListConversations)
				r.Get("/conversations/{id}/messages", deps.Messaging.ListMessages)
				r.Post("/conversations/{id}/messages", deps.Messaging.SendMessage)
				r.Post("/conversations/{id}/read", deps.Messaging.MarkAsRead)
				r.Put("/messages/{id}", deps.Messaging.EditMessage)
				r.Delete("/messages/{id}", deps.Messaging.DeleteMessage)
				r.Post("/upload-url", deps.Messaging.GetPresignedURL)
				r.Get("/unread-count", deps.Messaging.GetTotalUnread)
			})
		}

		// Proposal routes (authenticated)
		if deps.Proposal != nil {
			r.Route("/proposals", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Post("/", deps.Proposal.CreateProposal)
				r.Get("/{id}", deps.Proposal.GetProposal)
				r.Post("/{id}/accept", deps.Proposal.AcceptProposal)
				r.Post("/{id}/decline", deps.Proposal.DeclineProposal)
				r.Post("/{id}/modify", deps.Proposal.ModifyProposal)
				r.Post("/{id}/pay", deps.Proposal.SimulatePayment)
			})
			r.Route("/projects", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Get("/", deps.Proposal.ListActiveProjects)
			})
		}

		// WebSocket (auth handled inside the handler)
		if deps.WSHandler != nil {
			r.Get("/ws", deps.WSHandler)
		}

		// Test routes (debug — backend & DB connectivity)
		r.Route("/test", func(r chi.Router) {
			r.Get("/health-check", deps.Health.HealthCheck)
			r.Get("/words", deps.Health.GetWords)
			r.Post("/words", deps.Health.AddWord)
		})
	})

	return r
}
