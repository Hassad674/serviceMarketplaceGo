package handler

import (
	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
)

type RouterDeps struct {
	Auth         *AuthHandler
	Profile      *ProfileHandler
	Health       *HealthHandler
	Config       *config.Config
	TokenService service.TokenService
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

			// Protected
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService))
				r.Get("/me", deps.Auth.Me)
			})
		})

		// Profile routes (authenticated)
		r.Route("/profile", func(r chi.Router) {
			r.Use(middleware.Auth(deps.TokenService))
			r.Get("/", deps.Profile.GetMyProfile)
			r.Put("/", deps.Profile.UpdateMyProfile)
		})

		// Public profiles
		r.Get("/profiles/{userId}", deps.Profile.GetPublicProfile)

		// Test routes (debug — backend & DB connectivity)
		r.Route("/test", func(r chi.Router) {
			r.Get("/health-check", deps.Health.HealthCheck)
			r.Get("/words", deps.Health.GetWords)
			r.Post("/words", deps.Health.AddWord)
		})
	})

	return r
}
