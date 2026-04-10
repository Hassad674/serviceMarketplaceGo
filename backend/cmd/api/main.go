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
	"marketplace-backend/internal/adapter/noop"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	comprehendadapter "marketplace-backend/internal/adapter/comprehend"
	rekognitionadapter "marketplace-backend/internal/adapter/rekognition"
	resendadapter "marketplace-backend/internal/adapter/resend"
	s3adapter "marketplace-backend/internal/adapter/s3"
	"marketplace-backend/internal/adapter/s3transit"
	sqsadapter "marketplace-backend/internal/adapter/sqs"
	stripeadapter "marketplace-backend/internal/adapter/stripe"
	"marketplace-backend/internal/adapter/ws"
	anthropicadapter "marketplace-backend/internal/adapter/anthropic"
	adminapp "marketplace-backend/internal/app/admin"
	"marketplace-backend/internal/app/auth"
	callapp "marketplace-backend/internal/app/call"
	disputeapp "marketplace-backend/internal/app/dispute"
	embeddedapp "marketplace-backend/internal/app/embedded"
	kycapp "marketplace-backend/internal/app/kyc"
	jobapp "marketplace-backend/internal/app/job"
	mediaapp "marketplace-backend/internal/app/media"
	"marketplace-backend/internal/app/messaging"
	notifapp "marketplace-backend/internal/app/notification"
	organizationapp "marketplace-backend/internal/app/organization"
	paymentapp "marketplace-backend/internal/app/payment"
	portfolioapp "marketplace-backend/internal/app/portfolio"
	profileapp "marketplace-backend/internal/app/profile"
	projecthistoryapp "marketplace-backend/internal/app/projecthistory"
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
	organizationRepo := postgres.NewOrganizationRepository(db)
	organizationMemberRepo := postgres.NewOrganizationMemberRepository(db)
	organizationInvitationRepo := postgres.NewOrganizationInvitationRepository(db)
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
	organizationSvc := organizationapp.NewService(organizationRepo, organizationMemberRepo, organizationInvitationRepo)
	authSvc := auth.NewServiceWithDeps(auth.ServiceDeps{
		Users:       userRepo,
		Resets:      resetRepo,
		Hasher:      hasher,
		Tokens:      tokenSvc,
		Email:       emailSvc,
		Orgs:        organizationSvc,
		FrontendURL: cfg.FrontendURL,
	})
	profileSvc := profileapp.NewService(profileRepo)
	messagingSvc := messaging.NewService(messaging.ServiceDeps{
		Messages:    messageRepo,
		Users:       userRepo,
		Presence:    presenceSvc,
		Broadcaster: streamBroadcaster,
		Storage:     storageSvc,
		RateLimiter: rateLimiter,
		// MediaRecorder is set below after mediaSvc is created.
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

	// Portfolio feature
	portfolioRepo := postgres.NewPortfolioRepository(db)
	portfolioSvc := portfolioapp.NewService(portfolioapp.ServiceDeps{
		Portfolios: portfolioRepo,
	})
	portfolioHandler := handler.NewPortfolioHandler(portfolioSvc)

	// Project history feature (orchestrates proposal + review reads for the
	// public provider profile page).
	projectHistorySvc := projecthistoryapp.NewService(projecthistoryapp.ServiceDeps{
		Proposals: proposalRepo,
		Reviews:   reviewRepo,
	})
	projectHistoryHandler := handler.NewProjectHistoryHandler(projectHistorySvc)

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

	// Payment records (custom KYC repos removed — see migration 040/041)
	paymentRecordRepo := postgres.NewPaymentRecordRepository(db)

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

	// KYC enforcement scheduler — sends reminders at day 0/3/7/14 for
	// providers with available funds who haven't completed Stripe KYC.
	kycScheduler := kycapp.NewScheduler(kycapp.SchedulerDeps{
		Users:         userRepo,
		Records:       paymentRecordRepo,
		Notifications: notifSvc,
	})
	kycCtx, kycCancel := context.WithCancel(context.Background())
	defer kycCancel()
	kycInterval := 1 * time.Hour
	if cfg.Env == "development" {
		kycInterval = 1 * time.Minute
	}
	go kycScheduler.Run(kycCtx, kycInterval)
	slog.Info("kyc enforcement scheduler started", "interval", kycInterval)

	// Payment service — charge creation + transfers + wallet overview.
	// KYC onboarding lives in internal/app/embedded (Embedded Components).
	paymentInfoSvc := paymentapp.NewService(paymentapp.ServiceDeps{
		Records:       paymentRecordRepo,
		Users:         userRepo,
		Stripe:        stripeSvc,
		Notifications: notifSvc,
		FrontendURL:   cfg.FrontendURL,
	})

	// Credit bonus fraud log
	bonusLogRepo := postgres.NewCreditBonusLogRepository(db)

	// Wire services that depend on notifications
	proposalSvc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:     proposalRepo,
		Users:         userRepo,
		Messages:      messagingSvc,
		Storage:       storageSvc,
		Notifications: notifSvc,
		Payments:      paymentProcessor(paymentInfoSvc, cfg),
		Credits:       jobCreditRepo,
		BonusLog:      bonusLogRepo,
	})
	reviewSvc := reviewapp.NewService(reviewapp.ServiceDeps{
		Reviews:       reviewRepo,
		Proposals:     proposalRepo,
		Notifications: notifSvc,
	})

	// Report feature
	reportRepo := postgres.NewReportRepository(db)
	reportSvc := reportapp.NewService(reportapp.ServiceDeps{
		Reports:      reportRepo,
		Users:        userRepo,
		Messages:     messageRepo,
		Jobs:         jobRepo,
		Applications: jobAppRepo,
	})
	reportHandler := handler.NewReportHandler(reportSvc)

	// Media moderation feature
	mediaRepo := postgres.NewMediaRepository(db)
	var moderationSvc service.ContentModerationService
	if cfg.RekognitionConfigured() {
		rekSvc, rekErr := rekognitionadapter.NewModerationService(rekognitionadapter.ModerationServiceDeps{
			Region:      cfg.RekognitionRegion,
			Threshold:   cfg.RekognitionThreshold,
			SNSTopicARN: cfg.SNSTopicARN,
			RoleARN:     cfg.RekognitionRoleARN,
		})
		if rekErr != nil {
			slog.Error("failed to init Rekognition moderation service", "error", rekErr)
			moderationSvc = noop.NewModerationService()
		} else {
			moderationSvc = rekSvc
			slog.Info("content moderation enabled (AWS Rekognition)")
		}
	} else {
		moderationSvc = noop.NewModerationService()
		slog.Info("content moderation disabled (noop)")
	}

	// Video moderation transit storage (optional, requires all AWS video vars)
	var transitStorage service.TransitStorageService
	if cfg.VideoModerationConfigured() {
		transit, transitErr := s3transit.NewTransitStorage(cfg.RekognitionRegion, cfg.S3ModerationBucket)
		if transitErr != nil {
			slog.Error("failed to init S3 transit storage", "error", transitErr)
		} else {
			transitStorage = transit
			slog.Info("video moderation transit storage enabled",
				"bucket", cfg.S3ModerationBucket)
		}
	}

	mediaSvc := mediaapp.NewService(mediaapp.ServiceDeps{
		Media:               mediaRepo,
		Users:               userRepo,
		Storage:             storageSvc,
		Transit:             transitStorage,
		Moderation:          moderationSvc,
		Email:               emailSvc,
		SessionSvc:          sessionSvc,
		Broadcaster:         streamBroadcaster,
		FlagThreshold:       cfg.RekognitionThreshold,
		AutoRejectThreshold: cfg.RekognitionAutoRejectThreshold,
	})

	// Wire media recorder into messaging so file/voice messages are tracked.
	messagingSvc.SetMediaRecorder(mediaSvc)

	// Text moderation (AWS Comprehend) — moderates messages and reviews for toxicity.
	var textModerationSvc service.TextModerationService
	if cfg.ComprehendConfigured() {
		comprehendSvc, compErr := comprehendadapter.NewTextModerationService(cfg.RekognitionRegion)
		if compErr != nil {
			slog.Error("failed to init Comprehend text moderation", "error", compErr)
			textModerationSvc = noop.NewTextModerationService()
		} else {
			textModerationSvc = comprehendSvc
			slog.Info("text moderation enabled (AWS Comprehend)")
		}
	} else {
		textModerationSvc = noop.NewTextModerationService()
		slog.Info("text moderation disabled (noop)")
	}
	messagingSvc.SetTextModeration(textModerationSvc)
	reviewSvc.SetTextModeration(textModerationSvc)

	// SQS worker polls Rekognition completion notifications and finalizes jobs.
	if cfg.VideoModerationConfigured() && transitStorage != nil {
		worker, workerErr := sqsadapter.NewWorker(sqsadapter.WorkerDeps{
			Region:    cfg.RekognitionRegion,
			QueueURL:  cfg.SQSQueueURL,
			Finalizer: mediaSvc,
		})
		if workerErr != nil {
			slog.Error("failed to init SQS worker", "error", workerErr)
		} else {
			workerCtx, workerCancel := context.WithCancel(context.Background())
			defer workerCancel()
			go worker.Start(workerCtx)
		}
	}

	// Admin notification counters (per-admin Redis counters)
	adminNotifierSvc := redisadapter.NewAdminNotifierService(redisClient, db, streamBroadcaster)
	reportSvc.SetAdminNotifier(adminNotifierSvc)
	mediaSvc.SetAdminNotifier(adminNotifierSvc)
	messagingSvc.SetAdminNotifier(adminNotifierSvc)
	reviewSvc.SetAdminNotifier(adminNotifierSvc)
	slog.Info("admin notification counters enabled")

	// Admin feature
	adminConvRepo := postgres.NewAdminConversationRepository(db)
	adminModerationRepo := postgres.NewAdminModerationRepository(db)
	adminSvc := adminapp.NewService(adminapp.ServiceDeps{
		Users:              userRepo,
		Reports:            reportRepo,
		Reviews:            reviewRepo,
		Jobs:               jobRepo,
		Applications:       jobAppRepo,
		Proposals:          proposalRepo,
		AdminConversations: adminConvRepo,
		MediaRepo:          mediaRepo,
		ModerationRepo:     adminModerationRepo,
		StorageSvc:         storageSvc,
		SessionSvc:         sessionSvc,
		Broadcaster:        streamBroadcaster,
		AdminNotifier:      adminNotifierSvc,
	})
	adminHandler := handler.NewAdminHandler(adminSvc)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc, organizationSvc, sessionSvc, cookieCfg)
	profileHandler := handler.NewProfileHandler(profileSvc)
	uploadHandler := handler.NewUploadHandler(storageSvc, profileRepo, mediaSvc)
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

		// Embedded Components notifier — diff-based multi-channel notifications
		// for Stripe account.* webhooks (activation, requirements, docs rejected).
		// Backed by the users table (migration 040) for both lookup + state.
		embeddedNotifier := embeddedapp.NewNotifier(
			embeddedapp.NewNotificationSenderAdapter(notifSvc),
			userRepo, // satisfies UserStore via the 3 Stripe methods on UserRepository
			5*time.Minute,
		)
		stripeHandler = stripeHandler.WithEmbeddedNotifier(embeddedNotifier)
	}

	// Wallet handler
	walletHandler := handler.NewWalletHandler(paymentInfoSvc, proposalSvc)

	// Dispute feature
	disputeRepo := postgres.NewDisputeRepository(db)
	var aiAnalyzer service.AIAnalyzer
	if cfg.AnthropicAPIKey != "" {
		aiAnalyzer = anthropicadapter.NewAnalyzer(cfg.AnthropicAPIKey)
		slog.Info("AI analyzer enabled (Anthropic Claude Haiku)")
	} else {
		aiAnalyzer = noop.NewAnalyzer()
		slog.Info("AI analyzer disabled (no ANTHROPIC_API_KEY)")
	}
	disputeSvc := disputeapp.NewService(disputeapp.ServiceDeps{
		Disputes:      disputeRepo,
		Proposals:     proposalRepo,
		Users:         userRepo,
		MessageRepo:   messageRepo,
		Messages:      messagingSvc,
		Notifications: notifSvc,
		Payments:      paymentInfoSvc,
		AI:            aiAnalyzer,
	})
	disputeHandler := handler.NewDisputeHandler(disputeSvc)
	adminDisputeHandler := handler.NewAdminDisputeHandler(disputeSvc, disputeRepo, cfg.Env != "production")

	// Dispute scheduler — auto-resolve ghost (7d) + escalate to admin.
	// Escalation logic itself is fully delegated to disputeSvc.escalate so
	// the scheduler and the manual force-escalate endpoint share the same
	// code path (AI summary, system message, notifications all included).
	disputeScheduler := disputeapp.NewScheduler(disputeapp.SchedulerDeps{
		Svc:           disputeSvc,
		Disputes:      disputeRepo,
		Proposals:     proposalRepo,
		Messages:      messagingSvc,
		Notifications: notifSvc,
		Payments:      paymentInfoSvc,
	})
	disputeCtx, disputeCancel := context.WithCancel(context.Background())
	defer disputeCancel()
	disputeInterval := 1 * time.Hour
	if cfg.Env == "development" {
		disputeInterval = 1 * time.Minute
	}
	go disputeScheduler.Run(disputeCtx, disputeInterval)
	slog.Info("dispute scheduler started", "interval", disputeInterval)

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
		Embedded:       handler.NewEmbeddedHandler(userRepo, cfg.FrontendURL),
		Notification:   notifHandler,
		Stripe:         stripeHandler,
		Wallet:         walletHandler,
		Admin:          adminHandler,
		Portfolio:      portfolioHandler,
		ProjectHistory: projectHistoryHandler,
		Dispute:        disputeHandler,
		AdminDispute:   adminDisputeHandler,
		WSHandler:      wsHandler,
		Config:         cfg,
		TokenService:   tokenSvc,
		SessionService: sessionSvc,
		UserRepo:       userRepo,
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
