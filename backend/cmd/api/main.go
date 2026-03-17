package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marketplace-backend/internal/adapter/postgres"
	resendadapter "marketplace-backend/internal/adapter/resend"
	"marketplace-backend/internal/app/auth"
	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/pkg/crypto"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup structured logger
	logLevel := slog.LevelInfo
	if cfg.IsDevelopment() {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Connect to database
	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	// Initialize adapters (output ports)
	userRepo := postgres.NewUserRepository(db)
	profileRepo := postgres.NewProfileRepository(db)
	resetRepo := postgres.NewPasswordResetRepository(db)
	hasher := crypto.NewBcryptHasher()
	tokenSvc := crypto.NewJWTService(cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)
	emailSvc := resendadapter.NewEmailService(cfg.ResendAPIKey)

	// Initialize application services
	authSvc := auth.NewService(userRepo, resetRepo, hasher, tokenSvc, emailSvc, cfg.FrontendURL)
	profileSvc := profileapp.NewService(profileRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc)
	profileHandler := handler.NewProfileHandler(profileSvc)
	healthHandler := handler.NewHealthHandler(db)

	// Setup router
	r := handler.NewRouter(handler.RouterDeps{
		Auth:         authHandler,
		Profile:      profileHandler,
		Health:       healthHandler,
		Config:       cfg,
		TokenService: tokenSvc,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}
