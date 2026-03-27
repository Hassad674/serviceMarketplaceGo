package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/adapter/livekit"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	resendadapter "marketplace-backend/internal/adapter/resend"
	s3adapter "marketplace-backend/internal/adapter/s3"
	"marketplace-backend/internal/adapter/ws"
	"marketplace-backend/internal/app/auth"
	callapp "marketplace-backend/internal/app/call"
	jobapp "marketplace-backend/internal/app/job"
	reviewapp "marketplace-backend/internal/app/review"
	"marketplace-backend/internal/app/messaging"
	profileapp "marketplace-backend/internal/app/profile"
	proposalapp "marketplace-backend/internal/app/proposal"
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

	// Connect to Redis
	redisClient, err := redisadapter.NewClient(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	slog.Info("redis connected")

	// Initialize adapters (output ports)
	userRepo := postgres.NewUserRepository(db)
	profileRepo := postgres.NewProfileRepository(db)
	resetRepo := postgres.NewPasswordResetRepository(db)
	hasher := crypto.NewBcryptHasher()
	tokenSvc := crypto.NewJWTService(cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)
	emailSvc := resendadapter.NewEmailService(cfg.ResendAPIKey)
	storageSvc := s3adapter.NewStorageService(
		cfg.StorageEndpoint,
		cfg.StorageAccessKey,
		cfg.StorageSecretKey,
		cfg.StorageBucket,
		cfg.StoragePublicURL,
		cfg.StorageUseSSL,
	)
	sessionSvc := redisadapter.NewSessionService(redisClient, cfg.SessionTTL)

	// Cookie configuration
	cookieCfg := &handler.CookieConfig{
		Secure: cfg.CookieSecure,
		Domain: "",
		MaxAge: int(cfg.SessionTTL.Seconds()),
	}

	// Messaging adapters
	messageRepo := postgres.NewConversationRepository(db)
	presenceSvc := redisadapter.NewPresenceService(redisClient, 45*time.Second)
	streamBroadcaster := redisadapter.NewStreamBroadcaster(redisClient, uuid.New().String())
	rateLimiter := redisadapter.NewMessagingRateLimiter(redisClient)

	// WebSocket hub
	wsHub := ws.NewHub()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	go wsHub.Run(hubCtx)

	// Start stream subscriber (distributes Redis stream events to local WS clients)
	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()
	go streamBroadcaster.Subscribe(streamCtx, func(event redisadapter.StreamEvent) {
		wsHub.HandleStreamEvent(ws.StreamEvent{
			Type:         event.Type,
			RecipientIDs: event.RecipientIDs,
			Payload:      event.Payload,
			SourceID:     event.SourceID,
		})
	})

	// Initialize application services
	authSvc := auth.NewService(userRepo, resetRepo, hasher, tokenSvc, emailSvc, cfg.FrontendURL)
	profileSvc := profileapp.NewService(profileRepo)
	messagingSvc := messaging.NewService(messaging.ServiceDeps{
		Messages:    messageRepo,
		Users:       userRepo,
		Presence:    presenceSvc,
		Broadcaster: streamBroadcaster,
		Storage:     storageSvc,
		RateLimiter: rateLimiter,
	})

	// Proposal
	proposalRepo := postgres.NewProposalRepository(db)
	proposalSvc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals: proposalRepo,
		Users:     userRepo,
		Messages:  messagingSvc,
		Storage:   storageSvc,
	})

	// Job feature
	jobRepo := postgres.NewJobRepository(db)
	jobSvc := jobapp.NewService(jobapp.ServiceDeps{
		Jobs:  jobRepo,
		Users: userRepo,
	})

	// Review feature
	reviewRepo := postgres.NewReviewRepository(db)
	reviewSvc := reviewapp.NewService(reviewapp.ServiceDeps{
		Reviews:   reviewRepo,
		Proposals: proposalRepo,
	})

	// Call feature (optional — only when LiveKit is configured)
	var callHandler *handler.CallHandler
	if cfg.LiveKitConfigured() {
		lkClient := livekit.NewClient(cfg.LiveKitURL, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret)
		callStateSvc := redisadapter.NewCallStateService(redisClient)
		callSvc := callapp.NewService(callapp.ServiceDeps{
			LiveKit:     lkClient,
			CallState:   callStateSvc,
			Presence:    presenceSvc,
			Broadcaster: streamBroadcaster,
			Messages:    messagingSvc,
			Users:       userRepo,
		})
		callHandler = handler.NewCallHandler(callSvc)
		slog.Info("call feature enabled (LiveKit configured)")
	} else {
		slog.Info("call feature disabled (LiveKit not configured)")
	}

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc, sessionSvc, cookieCfg)
	profileHandler := handler.NewProfileHandler(profileSvc)
	uploadHandler := handler.NewUploadHandler(storageSvc, profileRepo)
	healthHandler := handler.NewHealthHandler(db)
	messagingHandler := handler.NewMessagingHandler(messagingSvc)
	proposalHandler := handler.NewProposalHandler(proposalSvc)
	jobHandler := handler.NewJobHandler(jobSvc)
	reviewHandler := handler.NewReviewHandler(reviewSvc)

	wsHandler := ws.ServeWS(ws.ConnDeps{
		Hub:              wsHub,
		MessagingSvc:     messagingSvc,
		TokenSvc:         tokenSvc,
		SessionSvc:       sessionSvc,
		PresenceSvc:      presenceSvc,
		Broadcaster:      streamBroadcaster,
		AllowedWSOrigins: cfg.AllowedOrigins,
	})

	// Setup router
	r := handler.NewRouter(handler.RouterDeps{
		Auth:           authHandler,
		Profile:        profileHandler,
		Upload:         uploadHandler,
		Health:         healthHandler,
		Messaging:      messagingHandler,
		Proposal:       proposalHandler,
		Job:            jobHandler,
		Review:         reviewHandler,
		Call:           callHandler,
		WSHandler:      wsHandler,
		Config:         cfg,
		TokenService:   tokenSvc,
		SessionService: sessionSvc,
	})

	// Create HTTP server
	// WriteTimeout is 0 to allow long-lived WebSocket connections.
	// Handler-level timeouts protect regular HTTP endpoints instead.
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
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
