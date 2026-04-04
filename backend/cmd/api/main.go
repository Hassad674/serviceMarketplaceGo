package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"marketplace-backend/internal/adapter/fcm"
	"marketplace-backend/internal/adapter/livekit"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	resendadapter "marketplace-backend/internal/adapter/resend"
	s3adapter "marketplace-backend/internal/adapter/s3"
	stripeadapter "marketplace-backend/internal/adapter/stripe"
	"marketplace-backend/internal/adapter/ws"
	adminapp "marketplace-backend/internal/app/admin"
	"marketplace-backend/internal/app/auth"
	callapp "marketplace-backend/internal/app/call"
	jobapp "marketplace-backend/internal/app/job"
	"marketplace-backend/internal/app/messaging"
	notifapp "marketplace-backend/internal/app/notification"
	paymentapp "marketplace-backend/internal/app/payment"
	profileapp "marketplace-backend/internal/app/profile"
	proposalapp "marketplace-backend/internal/app/proposal"
	reportapp "marketplace-backend/internal/app/report"
	reviewapp "marketplace-backend/internal/app/review"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/service"
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
	// In production (cross-origin: Railway backend + Vercel frontend),
	// SameSite=None is required for cookies to be sent cross-origin.
	// SameSite=None requires Secure=true.
	sameSite := http.SameSiteLaxMode
	if cfg.IsProduction() {
		sameSite = http.SameSiteNoneMode
	}
	cookieCfg := &handler.CookieConfig{
		Secure:   cfg.CookieSecure,
		Domain:   "",
		MaxAge:   int(cfg.SessionTTL.Seconds()),
		SameSite: sameSite,
	}

	// Messaging adapters
	messageRepo := postgres.NewConversationRepository(db)
	presenceSvc := redisadapter.NewPresenceService(redisClient, 45*time.Second)
	// Use HOSTNAME env var (set by Railway/Docker) or fallback to a fixed name.
	// This prevents dead consumer accumulation on redeploys.
	sourceID := os.Getenv("HOSTNAME")
	if sourceID == "" {
		sourceID = "api-main"
	}
	streamBroadcaster := redisadapter.NewStreamBroadcaster(redisClient, sourceID)
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

	// Job feature
	jobRepo := postgres.NewJobRepository(db)
	jobAppRepo := postgres.NewJobApplicationRepository(db)
	jobViewRepo := postgres.NewJobViewRepository(db)
	jobCreditRepo := postgres.NewJobCreditRepository(db)
	jobSvc := jobapp.NewService(jobapp.ServiceDeps{
		Jobs:         jobRepo,
		Applications: jobAppRepo,
		Users:        userRepo,
		Profiles:     profileRepo,
		Messages:     messagingSvc,
		JobViews:     jobViewRepo,
		Credits:      jobCreditRepo,
	})

	// Review feature
	reviewRepo := postgres.NewReviewRepository(db)

	// Social links feature
	socialLinkRepo := postgres.NewSocialLinkRepository(db)
	socialLinkSvc := profileapp.NewSocialLinkService(socialLinkRepo)
	socialLinkHandler := handler.NewSocialLinkHandler(socialLinkSvc)

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

	// Stripe payment adapter (optional — only when Stripe is configured)
	var stripeSvc service.StripeService
	if cfg.StripeConfigured() {
		stripeAdapter := stripeadapter.NewService(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
		stripeSvc = stripeAdapter
		slog.Info("stripe payment adapter enabled")
	} else {
		slog.Info("stripe payment adapter disabled (not configured)")
	}

	// Payment info + records + identity documents feature
	paymentInfoRepo := postgres.NewPaymentInfoRepository(db)
	paymentRecordRepo := postgres.NewPaymentRecordRepository(db)
	identityDocRepo := postgres.NewIdentityDocumentRepository(db)
	businessPersonRepo := postgres.NewBusinessPersonRepository(db)

	// Push notification service (optional — only when FCM is configured)
	var pushSvc service.PushService
	if cfg.FCMConfigured() {
		fcmSvc, fcmErr := fcm.NewPushService(cfg.FCMCredentialsPath)
		if fcmErr != nil {
			slog.Error("failed to init FCM push service", "error", fcmErr)
		} else {
			pushSvc = fcmSvc
			slog.Info("push notification service enabled (FCM)")
		}
	}

	// Notification feature
	notifRepo := postgres.NewNotificationRepository(db)
	notifQueue := redisadapter.NewNotificationJobQueue(redisClient, sourceID)
	if err := notifQueue.EnsureGroup(context.Background()); err != nil {
		slog.Error("failed to create notification job group", "error", err)
	}
	notifSvc := notifapp.NewService(notifapp.ServiceDeps{
		Notifications: notifRepo,
		Presence:      presenceSvc,
		Broadcaster:   streamBroadcaster,
		Push:          pushSvc, // nil if FCM not configured
		Email:         emailSvc,
		Users:         userRepo,
		Queue:         notifQueue,
	})

	// Start notification delivery worker (processes push + email async)
	notifWorker := notifapp.NewWorker(notifapp.WorkerDeps{
		Queue:    notifQueue,
		Presence: presenceSvc,
		Push:     pushSvc,
		Email:    emailSvc,
		Users:    userRepo,
		Notifs:   notifRepo,
	})
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go notifWorker.Run(workerCtx)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	slog.Info("notification feature enabled")

	// Country spec cache (optional — only when Stripe is configured)
	var countrySpecCache *redisadapter.CountrySpecCache
	if stripeSvc != nil {
		countrySpecCache = redisadapter.NewCountrySpecCache(redisClient, stripeSvc)
		if err := countrySpecCache.WarmCache(context.Background()); err != nil {
			slog.Warn("failed to warm country spec cache", "error", err)
		}
	}

	// Payment info service (depends on notifications)
	paymentInfoSvc := paymentapp.NewService(paymentapp.ServiceDeps{
		Payments:      paymentInfoRepo,
		Records:       paymentRecordRepo,
		Documents:     identityDocRepo,
		Persons:       businessPersonRepo,
		Stripe:        stripeSvc,
		Storage:       storageSvc,
		Notifications: notifSvc,
		CountrySpecs:  countrySpecCache,
		FrontendURL:   cfg.FrontendURL,
	})
	paymentInfoHandler := handler.NewPaymentInfoHandler(paymentInfoSvc)
	identityDocHandler := handler.NewIdentityDocumentHandler(paymentInfoSvc)

	// Wire services that depend on notifications
	proposalSvc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:     proposalRepo,
		Users:         userRepo,
		Messages:      messagingSvc,
		Storage:       storageSvc,
		Notifications: notifSvc,
		Payments:      paymentProcessor(paymentInfoSvc, cfg),
		Credits:       jobCreditRepo,
	})
	reviewSvc := reviewapp.NewService(reviewapp.ServiceDeps{
		Reviews:       reviewRepo,
		Proposals:     proposalRepo,
		Notifications: notifSvc,
	})

	// Report feature
	reportRepo := postgres.NewReportRepository(db)
	reportSvc := reportapp.NewService(reportapp.ServiceDeps{
		Reports:  reportRepo,
		Users:    userRepo,
		Messages: messageRepo,
	})
	reportHandler := handler.NewReportHandler(reportSvc)

	// Admin feature
	adminSvc := adminapp.NewService(userRepo, db)
	adminHandler := handler.NewAdminHandler(adminSvc)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc, sessionSvc, cookieCfg)
	profileHandler := handler.NewProfileHandler(profileSvc)
	uploadHandler := handler.NewUploadHandler(storageSvc, profileRepo)
	healthHandler := handler.NewHealthHandler(db)
	messagingHandler := handler.NewMessagingHandler(messagingSvc)
	proposalHandler := handler.NewProposalHandler(proposalSvc, paymentInfoSvc)
	jobHandler := handler.NewJobHandler(jobSvc)
	jobAppHandler := handler.NewJobApplicationHandler(jobSvc)
	reviewHandler := handler.NewReviewHandler(reviewSvc)

	// Stripe handler (optional)
	var stripeHandler *handler.StripeHandler
	if cfg.StripeConfigured() {
		stripeHandler = handler.NewStripeHandler(paymentInfoSvc, proposalSvc, cfg.StripePublishableKey)
	}

	// Wallet handler
	walletHandler := handler.NewWalletHandler(paymentInfoSvc, proposalSvc)

	wsHandler := ws.ServeWS(ws.ConnDeps{
		Hub:              wsHub,
		MessagingSvc:     messagingSvc,
		TokenSvc:         tokenSvc,
		SessionSvc:       sessionSvc,
		PresenceSvc:      presenceSvc,
		Broadcaster:      streamBroadcaster,
		AllowedWSOrigins: wsOriginPatterns(cfg.AllowedOrigins),
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
		JobApplication: jobAppHandler,
		Review:         reviewHandler,
		Report:         reportHandler,
		Call:           callHandler,
		SocialLink:     socialLinkHandler,
		PaymentInfo:    paymentInfoHandler,
		Notification:   notifHandler,
		Stripe:         stripeHandler,
		Wallet:         walletHandler,
		IdentityDoc:    identityDocHandler,
		Admin:          adminHandler,
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

// wsOriginPatterns converts full origin URLs (e.g. "https://example.com")
// to hostname patterns (e.g. "example.com") for coder/websocket OriginPatterns,
// and adds a wildcard for local development.
// paymentProcessor returns the payment service as PaymentProcessor if Stripe is configured, nil otherwise.
func paymentProcessor(svc *paymentapp.Service, cfg *config.Config) service.PaymentProcessor {
	if cfg.StripeConfigured() {
		return svc
	}
	return nil
}

func wsOriginPatterns(origins []string) []string {
	patterns := make([]string, 0, len(origins)+1)
	for _, o := range origins {
		// Strip scheme — coder/websocket matches on hostname only.
		host := strings.TrimPrefix(o, "https://")
		host = strings.TrimPrefix(host, "http://")
		if host != "" {
			patterns = append(patterns, host)
		}
	}
	// Always allow localhost for dev.
	patterns = append(patterns, "localhost:*")
	return patterns
}
