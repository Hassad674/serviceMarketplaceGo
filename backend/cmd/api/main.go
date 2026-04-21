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

	anthropicadapter "marketplace-backend/internal/adapter/anthropic"
	comprehendadapter "marketplace-backend/internal/adapter/comprehend"
	"marketplace-backend/internal/adapter/fcm"
	"marketplace-backend/internal/adapter/livekit"
	"marketplace-backend/internal/adapter/nominatim"
	"marketplace-backend/internal/adapter/noop"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	rekognitionadapter "marketplace-backend/internal/adapter/rekognition"
	resendadapter "marketplace-backend/internal/adapter/resend"
	s3adapter "marketplace-backend/internal/adapter/s3"
	"marketplace-backend/internal/adapter/s3transit"
	sqsadapter "marketplace-backend/internal/adapter/sqs"
	stripeadapter "marketplace-backend/internal/adapter/stripe"
	"marketplace-backend/internal/adapter/worker"
	"marketplace-backend/internal/adapter/worker/handlers"
	"marketplace-backend/internal/adapter/ws"
	adminapp "marketplace-backend/internal/app/admin"
	"marketplace-backend/internal/app/auth"
	callapp "marketplace-backend/internal/app/call"
	disputeapp "marketplace-backend/internal/app/dispute"
	embeddedapp "marketplace-backend/internal/app/embedded"
	freelancepricingapp "marketplace-backend/internal/app/freelancepricing"
	freelanceprofileapp "marketplace-backend/internal/app/freelanceprofile"
	jobapp "marketplace-backend/internal/app/job"
	kycapp "marketplace-backend/internal/app/kyc"
	mediaapp "marketplace-backend/internal/app/media"
	"marketplace-backend/internal/app/messaging"
	milestoneapp "marketplace-backend/internal/app/milestone"
	notifapp "marketplace-backend/internal/app/notification"
	organizationapp "marketplace-backend/internal/app/organization"
	paymentapp "marketplace-backend/internal/app/payment"
	portfolioapp "marketplace-backend/internal/app/portfolio"
	profileapp "marketplace-backend/internal/app/profile"
	profilepricingapp "marketplace-backend/internal/app/profilepricing"
	projecthistoryapp "marketplace-backend/internal/app/projecthistory"
	proposalapp "marketplace-backend/internal/app/proposal"
	referralapp "marketplace-backend/internal/app/referral"
	referrerpricingapp "marketplace-backend/internal/app/referrerpricing"
	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	reportapp "marketplace-backend/internal/app/report"
	reviewapp "marketplace-backend/internal/app/review"
	appsearch "marketplace-backend/internal/app/search"
	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/app/searchindex"
	skillapp "marketplace-backend/internal/app/skill"
	subscriptionapp "marketplace-backend/internal/app/subscription"
	"marketplace-backend/internal/config"
	jobdomain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/pendingevent"
	profiledomain "marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
	"marketplace-backend/pkg/crypto"

	"github.com/google/uuid"
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
	// The organization repository seeds every new org with
	// jobdomain.WeeklyQuota application credits at creation time. The
	// starter value flows through main.go (this file) so the
	// organization package stays free of any cross-feature import —
	// hexagonal wiring, not modular coupling.
	organizationRepo := postgres.NewOrganizationRepository(db, jobdomain.WeeklyQuota)
	organizationMemberRepo := postgres.NewOrganizationMemberRepository(db)
	organizationInvitationRepo := postgres.NewOrganizationInvitationRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	hasher := crypto.NewBcryptHasher()
	tokenSvc := crypto.NewJWTService(cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)
	emailSvc := resendadapter.NewEmailService(cfg.ResendAPIKey, cfg.ResendDevRedirectTo)
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
	invitationRateLimiter := redisadapter.NewInvitationRateLimiter(redisClient)
	organizationSvc := organizationapp.NewService(organizationRepo, organizationMemberRepo, organizationInvitationRepo)
	// invitationSvc and membershipSvc are constructed below, AFTER the
	// notification feature is set up — they depend on notifSvc so the
	// team events (invitation accepted, role changed, transfer, …) can
	// fire notifications through the same pipeline as the rest of the app.
	authSvc := auth.NewServiceWithDeps(auth.ServiceDeps{
		Users:       userRepo,
		Resets:      resetRepo,
		Hasher:      hasher,
		Tokens:      tokenSvc,
		Email:       emailSvc,
		Orgs:        organizationSvc,
		FrontendURL: cfg.FrontendURL,
	})
	// Profile service + Tier 1 geocoder (migration 083). The
	// Nominatim adapter is used as-is in every environment because
	// the public endpoint is free and the profile save flow
	// gracefully degrades on any geocoding failure — see
	// adapter/nominatim/geocoder.go and app/profile/service.go.
	profileGeocoder := nominatim.NewGeocoder("marketplace-backend/1.0 (contact@marketplace.local)")
	profileSvc := profileapp.NewService(profileRepo).WithGeocoder(profileGeocoder)
	messagingSvc := messaging.NewService(messaging.ServiceDeps{
		Messages:      messageRepo,
		Users:         userRepo,
		Organizations: organizationRepo,
		OrgMembers:    organizationMemberRepo,
		Presence:      presenceSvc,
		Broadcaster:   streamBroadcaster,
		Storage:       storageSvc,
		RateLimiter:   rateLimiter,
		// MediaRecorder is set below after mediaSvc is created.
	})

	// Proposal
	proposalRepo := postgres.NewProposalRepository(db)

	// Milestone — per-step funding/delivery sub-aggregate of a proposal.
	// The proposal app service consumes milestoneSvc to delegate the
	// Fund/Submit/Approve/Release transitions, and the dispute service
	// (phase 8) delegates OpenDispute/RestoreFromDispute to it as well.
	milestoneRepo := postgres.NewMilestoneRepository(db)
	milestoneSvc := milestoneapp.NewService(milestoneapp.ServiceDeps{
		Milestones: milestoneRepo,
	})
	_ = milestoneSvc // wired into proposal service deps below

	// Job feature
	jobRepo := postgres.NewJobRepository(db)
	jobAppRepo := postgres.NewJobApplicationRepository(db)
	jobViewRepo := postgres.NewJobViewRepository(db)
	// The credit repository drives a lazy weekly refill from its
	// GetOrCreate method — every read on an org whose pool has aged
	// past RefillPeriod floor-bumps the balance back up to WeeklyQuota
	// atomically. No cron, no background worker, self-healing after
	// downtime.
	jobCreditRepo := postgres.NewJobCreditRepository(db, jobdomain.WeeklyQuota, jobdomain.RefillPeriod)
	jobSvc := jobapp.NewService(jobapp.ServiceDeps{
		Jobs:          jobRepo,
		Applications:  jobAppRepo,
		Users:         userRepo,
		Organizations: organizationRepo,
		Profiles:      profileRepo,
		Messages:      messagingSvc,
		JobViews:      jobViewRepo,
		Credits:       jobCreditRepo,
	})

	// Review feature
	reviewRepo := postgres.NewReviewRepository(db)

	// Social links feature — one service instance per persona,
	// each bound at construction time so the downstream handler
	// stays unaware of the persona dimension.
	socialLinkRepo := postgres.NewSocialLinkRepository(db)
	agencySocialLinkSvc, err := profileapp.NewSocialLinkService(socialLinkRepo, profiledomain.PersonaAgency)
	if err != nil {
		slog.Error("failed to init agency social link service", "error", err)
		os.Exit(1)
	}
	freelanceSocialLinkSvc, err := profileapp.NewSocialLinkService(socialLinkRepo, profiledomain.PersonaFreelance)
	if err != nil {
		slog.Error("failed to init freelance social link service", "error", err)
		os.Exit(1)
	}
	referrerSocialLinkSvc, err := profileapp.NewSocialLinkService(socialLinkRepo, profiledomain.PersonaReferrer)
	if err != nil {
		slog.Error("failed to init referrer social link service", "error", err)
		os.Exit(1)
	}
	socialLinkHandler := handler.NewSocialLinkHandler(agencySocialLinkSvc)
	freelanceSocialLinkHandler := handler.NewSocialLinkHandler(freelanceSocialLinkSvc)
	referrerSocialLinkHandler := handler.NewSocialLinkHandler(referrerSocialLinkSvc)

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

	// Stripe payment adapter (optional — only when Stripe is configured).
	// The concrete stripeAdapter satisfies BOTH service.StripeService and
	// service.StripeTransferReversalService, so we keep a typed reference
	// to inject it into the referral feature alongside the narrower interface.
	var stripeSvc service.StripeService
	var stripeReversalSvc service.StripeTransferReversalService
	if cfg.StripeConfigured() {
		stripeAdapter := stripeadapter.NewService(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
		stripeSvc = stripeAdapter
		stripeReversalSvc = stripeAdapter
		slog.Info("stripe payment adapter enabled")
	} else {
		slog.Info("stripe payment adapter disabled (not configured)")
	}

	// Payment records (custom KYC repos removed — see migration 040/041)
	paymentRecordRepo := postgres.NewPaymentRecordRepository(db)

	// Push notification service (optional — only when FCM is configured)
	// FCM push notifications are optional — the backend starts with pushSvc
	// left nil when credentials are missing or invalid. Startup must log at
	// INFO in both the "disabled" and the "init failed" paths so operators
	// can tell the app booted without push without seeing scary ERRORs in
	// their console. Only truly unexpected failures would ever need ERROR,
	// and none of the current init paths qualify.
	var pushSvc service.PushService
	if !cfg.FCMConfigured() {
		slog.Info("push notification service disabled (FCM_CREDENTIALS_PATH not set)")
	} else {
		fcmSvc, fcmErr := fcm.NewPushService(cfg.FCMCredentialsPath)
		if fcmErr != nil {
			slog.Info("push notification service disabled (FCM init failed)",
				"error", fcmErr)
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

	// Organization team services — wired here so they can dispatch
	// team_* notifications through the same notifSvc used by the rest
	// of the app. Kept at the same indentation level as the other
	// services so the intent stays obvious.
	invitationSvc := organizationapp.NewInvitationService(organizationapp.InvitationServiceDeps{
		Orgs:          organizationRepo,
		Members:       organizationMemberRepo,
		Invitations:   organizationInvitationRepo,
		Users:         userRepo,
		Hasher:        hasher,
		Email:         emailSvc,
		RateLimiter:   invitationRateLimiter,
		Notifications: notifSvc,
		FrontendURL:   cfg.FrontendURL,
	})
	membershipSvc := organizationapp.NewMembershipService(organizationapp.MembershipServiceDeps{
		Orgs:          organizationRepo,
		Members:       organizationMemberRepo,
		Users:         userRepo,
		Notifications: notifSvc,
	})

	// Role permissions editor (R17 — per-org customization). Uses a
	// dedicated Redis-backed rate limiter so the audit tail and the
	// Owner email notification stay independent from the rest of the
	// invitation rate limit.
	rolePermsRateLimiter := redisadapter.NewRolePermissionsRateLimiter(redisClient)
	roleOverridesSvc := organizationapp.NewRoleOverridesService(organizationapp.RoleOverridesServiceDeps{
		Orgs:        organizationRepo,
		Members:     organizationMemberRepo,
		Users:       userRepo,
		Audits:      auditRepo,
		Email:       emailSvc,
		RateLimiter: rolePermsRateLimiter,
	})

	// KYC enforcement scheduler — sends reminders at day 0/3/7/14 for
	// providers with available funds who haven't completed Stripe KYC.
	kycScheduler := kycapp.NewScheduler(kycapp.SchedulerDeps{
		Organizations: organizationRepo,
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
		Organizations: organizationRepo,
		Stripe:        stripeSvc,
		Notifications: notifSvc,
		FrontendURL:   cfg.FrontendURL,
	})

	// Credit bonus fraud log
	bonusLogRepo := postgres.NewCreditBonusLogRepository(db)

	// Pending events queue (phase 6 — unified scheduler + Stripe outbox).
	// The proposal service writes events here when a milestone is
	// submitted (auto-approve), released (fund-reminder + auto-close),
	// or released into the Stripe outbox (phase 7).
	pendingEventsRepo := postgres.NewPendingEventRepository(db)

	// Search engine publisher — built once so every service that
	// mutates actor signals (freelance profile, referrer profile,
	// pricing, skills, etc.) can emit a `search.reindex` event on
	// the outbox without re-wiring the whole chain. The publisher
	// debounces rapid repeats so a storm of profile updates does
	// not translate to a storm of index rebuilds.
	//
	// Nil when Typesense is not configured — services receive the
	// nil publisher and silently skip publishing. Removing the
	// search feature entirely is a matter of deleting this block
	// and the `.WithSearchIndexPublisher(searchPublisher)` calls.
	var searchPublisher *searchindex.Publisher
	if cfg.TypesenseConfigured() {
		var pubErr error
		searchPublisher, pubErr = searchindex.NewPublisher(searchindex.PublisherConfig{
			Events: pendingEventsRepo,
		})
		if pubErr != nil {
			slog.Error("search: failed to build publisher", "error", pubErr)
			os.Exit(1)
		}
	}

	// Milestone audit trail (phase 9 — append-only). Every successful
	// withMilestoneLock writes one row recording from→to status pair,
	// actor id + org, and an optional reason string. The DB user
	// holds INSERT/SELECT only on this table (Update/Delete are
	// forbidden so the timeline cannot be rewritten).
	milestoneTransitionsRepo := postgres.NewMilestoneTransitionRepository(db)

	// Wire services that depend on notifications
	proposalSvc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:            proposalRepo,
		Milestones:           milestoneRepo,
		MilestoneTransitions: milestoneTransitionsRepo,
		PendingEvents:        pendingEventsRepo,
		Users:                userRepo,
		Organizations:        organizationRepo,
		Messages:             messagingSvc,
		Storage:              storageSvc,
		Notifications:        notifSvc,
		Payments:             paymentProcessor(paymentInfoSvc, cfg),
		Credits:              jobCreditRepo,
		BonusLog:             bonusLogRepo,
		// Phase 6 timer defaults (override via env in production):
		// 7-day auto-approval, 7-day fund reminder, 14-day auto-close.
	})

	// Wire proposal → payment status lookup so RequestPayout only
	// releases escrow funds for missions whose proposal has reached
	// "completed". Setter pattern because the dependency runs the wrong
	// way for constructor injection (payment is built before proposal).
	paymentInfoSvc.SetProposalStatusReader(newProposalStatusAdapter(proposalSvc))

	// Phase 6: pending_events worker. Runs in a background goroutine
	// alongside the API server, ticks every 30 seconds, and drives the
	// auto-approval, fund-reminder, and auto-close timers. Multiple
	// instances of this binary are safe to run side by side — PopDue
	// uses FOR UPDATE SKIP LOCKED so workers never claim the same row.
	pendingEventsWorker := worker.New(pendingEventsRepo, worker.Config{
		TickInterval: 30 * time.Second,
		BatchSize:    20,
	})
	pendingEventsWorker.Register(pendingevent.TypeMilestoneAutoApprove, handlers.NewMilestoneAutoApproveHandler(proposalSvc))
	pendingEventsWorker.Register(pendingevent.TypeMilestoneFundReminder, handlers.NewMilestoneFundReminderHandler(proposalSvc))
	pendingEventsWorker.Register(pendingevent.TypeProposalAutoClose, handlers.NewProposalAutoCloseHandler(proposalSvc))
	pendingEventsWorker.Register(pendingevent.TypeStripeTransfer, handlers.NewStripeTransferHandler(proposalSvc))

	// Search engine (Typesense) — phase 1 infrastructure. Always
	// wires the indexer + event handlers when TYPESENSE_* config
	// is present, even when SEARCH_ENGINE=sql, so the outbox
	// pipeline can populate the index ahead of the query-path
	// switch over. If Typesense is not configured we silently
	// skip registration — the outbox events will land as "no
	// handler registered" and stay in failed status until an
	// operator re-enables indexing.
	var typesenseClient *search.Client // nil when TYPESENSE_* env vars are absent
	if cfg.TypesenseConfigured() {
		tsClient, err := search.NewClient(cfg.TypesenseHost, cfg.TypesenseAPIKey)
		if err != nil {
			slog.Error("search: invalid typesense configuration", "error", err)
			os.Exit(1)
		}
		if err := search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
			Client: tsClient,
			Logger: slog.Default(),
		}); err != nil {
			slog.Warn("search: ensure schema failed, continuing without indexing", "error", err)
		}
		// Bootstrap the search-only parent key used as the HMAC
		// parent for scoped search keys. Typesense refuses to derive
		// scoped keys from the master admin key — we MUST use a key
		// whose `actions` list contains `documents:search`. We cycle
		// the key on every startup because Typesense only exposes
		// the full value on creation.
		if err := tsClient.EnsureSearchAPIKey(context.Background()); err != nil {
			slog.Error("search: failed to bootstrap search API key", "error", err)
			os.Exit(1)
		}
		slog.Info("search: search-only parent key bootstrapped")

		// Phase 3: when OPENAI_API_KEY is set, live embeddings become
		// MANDATORY — a transient 5xx or 429 no longer silently falls
		// back to the mock (which would ship near-duplicate vectors
		// and destroy semantic ranking). Wrap the live client in
		// RetryingEmbeddingsClient so transient failures retry with
		// exponential backoff (500ms / 1s / 2s, matching the spec).
		var embedder search.EmbeddingsClient
		if cfg.OpenAIAPIKey != "" {
			openaiClient, openaiErr := search.NewOpenAIEmbeddings(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingsModel)
			if openaiErr != nil {
				slog.Error("search: OPENAI_API_KEY set but client invalid — aborting to surface config error",
					"error", openaiErr)
				os.Exit(1)
			}
			embedder = search.NewRetryingEmbeddings(openaiClient)
			slog.Info("search: live OpenAI embeddings enabled (with retry)",
				"model", cfg.OpenAIEmbeddingsModel)
		} else {
			slog.Warn("search: OPENAI_API_KEY not set, using mock embeddings — search quality will be degraded")
			embedder = search.NewMockEmbeddings()
		}

		searchDataRepo := postgres.NewSearchDocumentRepository(db)
		searchIndexer, err := search.NewIndexer(searchDataRepo, embedder)
		if err != nil {
			slog.Error("search: failed to build indexer", "error", err)
			os.Exit(1)
		}
		searchIndexSvc, err := searchindex.NewService(searchindex.Config{
			Client:  tsClient,
			Indexer: searchIndexer,
			Logger:  slog.Default(),
		})
		if err != nil {
			slog.Error("search: failed to build indexing service", "error", err)
			os.Exit(1)
		}

		pendingEventsWorker.Register(pendingevent.TypeSearchReindex, handlers.NewSearchReindexHandler(searchIndexSvc))
		pendingEventsWorker.Register(pendingevent.TypeSearchDelete, handlers.NewSearchDeleteHandler(searchIndexSvc))
		typesenseClient = tsClient
		slog.Info("search: typesense indexer wired")
	} else {
		slog.Warn("search: typesense not configured — the listing pages will return 503 until TYPESENSE_* env vars are set")
	}

	// Search query service (phase 2+). Wired only when Typesense is
	// configured. Lives outside the previous block because the
	// indexer + the query path are independent — we can index
	// without serving and vice versa.
	var searchQuerySvc *appsearch.Service
	var searchHandler *handler.SearchHandler
	var adminSearchStatsHandler *handler.AdminSearchStatsHandler
	var searchAnalyticsSvc *searchanalytics.Service
	if typesenseClient != nil {
		// Phase 3: wire the analytics service so every search is
		// captured and the /search/track endpoint has somewhere to
		// persist clicks. Nil-safe — if the repo fails to build
		// search keeps working without analytics.
		analyticsRepo := postgres.NewSearchAnalyticsRepository(db)
		analyticsSvc, analyticsErr := searchanalytics.NewService(searchanalytics.Config{
			Repository: analyticsRepo,
			Logger:     slog.Default(),
		})
		if analyticsErr != nil {
			slog.Error("search: analytics service disabled", "error", analyticsErr)
		} else {
			searchAnalyticsSvc = analyticsSvc
		}

		// Phase 4: admin stats dashboard. Reuses the same repository
		// (which now implements both Repository and StatsRepository)
		// so there's no extra dependency. The handler is gated by
		// RequireAdmin at the router level.
		statsSvc, statsErr := searchanalytics.NewStatsService(searchanalytics.StatsServiceConfig{
			Repository: analyticsRepo,
			Logger:     slog.Default(),
		})
		if statsErr != nil {
			slog.Error("search: stats service disabled", "error", statsErr)
		} else {
			adminSearchStatsHandler = handler.NewAdminSearchStatsHandler(statsSvc)
		}

		// Phase 3: hybrid search needs a live embedder on the query
		// path. Reuse the same OpenAI client (with retry wrapper)
		// we built for indexing — the rate limits live on the API
		// key, so sharing the client matters.
		var queryEmbedder search.EmbeddingsClient
		if cfg.OpenAIAPIKey != "" {
			openaiClient, openaiErr := search.NewOpenAIEmbeddings(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingsModel)
			if openaiErr == nil {
				queryEmbedder = search.NewRetryingEmbeddings(openaiClient)
			} else {
				slog.Warn("search: query-time embedder disabled",
					"error", openaiErr)
			}
		}

		analyticsAdapter := newSearchAnalyticsRecorder(searchAnalyticsSvc)

		// Ranking V1 pipeline wiring (phase 6F) — composition of the
		// four Stage 2-5 packages. Every knob lives in RANKING_*
		// environment variables (see docs/ranking-tuning.md). Boot
		// fails loud on malformed env: a typo in a float weight must
		// never limp into prod with a silent zero.
		rankingPipeline := buildRankingPipeline()

		// LTR capture wiring — the repo is the same SearchAnalyticsRepository
		// already built above. The service holds the goroutine that writes
		// result_features_json; the repo runs the UPDATE under a 3s deadline.
		var ltrRepo searchanalytics.LTRRepository = analyticsRepo

		searchQuerySvc = appsearch.NewService(appsearch.ServiceDeps{
			Freelance:        search.NewFreelanceClient(typesenseClient),
			Agency:           search.NewAgencyClient(typesenseClient),
			Referrer:         search.NewReferrerClient(typesenseClient),
			Embedder:         queryEmbedder,
			Analytics:        analyticsAdapter,
			Logger:           slog.Default(),
			RankingPipeline:  rankingPipeline,
			LTRRepository:    ltrRepo,
			AnalyticsService: searchAnalyticsSvc,
		})
		searchHandler = handler.NewSearchHandler(handler.SearchHandlerDeps{
			Service:       searchQuerySvc,
			Client:        typesenseClient,
			TypesenseHost: cfg.TypesenseHost,
			// Use the bootstrapped search-only key as the HMAC parent
			// for scoped key generation. Typesense rejects scoped keys
			// derived from the master admin key.
			APIKey:       typesenseClient.SearchAPIKey(),
			ClickTracker: searchAnalyticsSvc,
			Logger:       slog.Default(),
		})
		slog.Info("search: query service wired",
			"hybrid_enabled", queryEmbedder != nil,
			"analytics_enabled", searchAnalyticsSvc != nil,
			"admin_stats_enabled", adminSearchStatsHandler != nil,
			"ranking_enabled", rankingPipeline != nil,
			"ltr_capture_enabled", ltrRepo != nil && searchAnalyticsSvc != nil)
	}
	pendingEventsCtx, pendingEventsCancel := context.WithCancel(context.Background())
	defer pendingEventsCancel()
	go func() {
		if err := pendingEventsWorker.Run(pendingEventsCtx); err != nil {
			slog.Error("pending events worker exited", "error", err)
		}
	}()
	slog.Info("phase 6: pending events worker started")
	reviewSvc := reviewapp.NewService(reviewapp.ServiceDeps{
		Reviews:       reviewRepo,
		Proposals:     proposalRepo,
		Users:         userRepo,
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
		// Phase 6 team admin wiring — these power the GET team detail
		// endpoint and the four force actions. Repositories come from
		// the organization wiring block above. The membership +
		// invitation services already carry notifSvc so team events
		// triggered by force actions still land in the notifications
		// table through the same pipeline as user-driven actions.
		Orgs:           organizationRepo,
		OrgMembers:     organizationMemberRepo,
		OrgInvitations: organizationInvitationRepo,
		Membership:     membershipSvc,
		Invitation:     invitationSvc,
	})
	adminHandler := handler.NewAdminHandler(adminSvc)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc, organizationSvc, sessionSvc, cookieCfg)
	invitationHandler := handler.NewInvitationHandler(handler.InvitationHandlerDeps{
		InvitationService: invitationSvc,
		OrgService:        organizationSvc,
		TokenService:      tokenSvc,
		SessionService:    sessionSvc,
		Cookie:            cookieCfg,
	})
	teamHandler := handler.NewTeamHandler(handler.TeamHandlerDeps{
		Membership:     membershipSvc,
		OrgService:     organizationSvc,
		UserBatch:      userRepo,
		SessionService: sessionSvc,
		Cookie:         cookieCfg,
		Users:          userRepo,
	})
	roleOverridesHandler := handler.NewRoleOverridesHandler(roleOverridesSvc)
	// Expertise feature (org-scoped domain specializations). Shares the
	// profile application package and is co-located in the profile
	// handler because expertise is part of the org's public profile.
	expertiseRepo := postgres.NewExpertiseRepository(db)
	expertiseSvc := profileapp.NewExpertiseService(expertiseRepo, organizationRepo)

	// Skills feature (hybrid catalog + per-org profile attachments).
	// Uses a small org-type-resolver adapter (org_type_resolver.go) to
	// bridge the existing organization repo to the skill service's
	// dependency contract, keeping the skill package independent of
	// domain/organization.
	//
	// The profile handler receives the skill service via
	// WithSkillsReader so the public profile / search endpoints can
	// decorate responses with each org's declared skills. The skill
	// service satisfies the handler's local SkillsReader contract.
	skillCatalogRepo := postgres.NewSkillCatalogRepository(db)
	profileSkillRepo := postgres.NewProfileSkillRepository(db)
	skillSvc := skillapp.NewService(
		skillCatalogRepo,
		profileSkillRepo,
		newOrgTypeResolverAdapter(organizationRepo),
	)
	skillHandler := handler.NewSkillHandler(skillSvc)
	if searchPublisher != nil {
		skillHandler = skillHandler.WithSearchIndexPublisher(searchPublisher)
	}

	// Profile pricing feature (migration 083). Uses a local
	// org-info resolver adapter (profile_pricing_org_info_resolver.go)
	// to bridge the existing organization + user repos to the
	// pricing service's dependency contract, keeping the
	// profilepricing package independent of domain/organization
	// and domain/user.
	profilePricingRepo := postgres.NewProfilePricingRepository(db)
	profilePricingSvc := profilepricingapp.NewService(
		profilePricingRepo,
		newProfilePricingOrgInfoResolverAdapter(organizationRepo, userRepo),
	)
	profilePricingHandler := handler.NewProfilePricingHandler(profilePricingSvc)
	if searchPublisher != nil {
		profilePricingHandler = profilePricingHandler.WithSearchIndexPublisher(searchPublisher)
	}

	// Split-profile feature (migrations 096-104). The freelance /
	// referrer / freelance pricing / referrer pricing aggregates
	// are the new home for provider_personal profiles; the legacy
	// profile / profilepricing stays in place for agency orgs
	// until the agency refactor ships. Each feature is wired as a
	// separate chain (repo -> service -> handler) so deleting the
	// split means removing these lines only.
	freelanceProfileRepo := postgres.NewFreelanceProfileRepository(db)
	freelanceProfileSvc := freelanceprofileapp.NewService(freelanceProfileRepo)
	if searchPublisher != nil {
		freelanceProfileSvc = freelanceProfileSvc.WithSearchIndexPublisher(searchPublisher)
		// Phase 2 carry-over: the legacy agency profile service
		// also publishes reindex events. Done via a setter here
		// because profileSvc is created earlier (line ~191) for
		// other downstream wiring; this keeps the publisher
		// dependency optional and isolated.
		profileSvc = profileSvc.WithSearchIndexPublisher(searchPublisher)
	}
	freelancePricingRepo := postgres.NewFreelancePricingRepository(db)
	freelancePricingSvc := freelancepricingapp.NewService(freelancePricingRepo)
	freelanceProfileHandler := handler.
		NewFreelanceProfileHandler(freelanceProfileSvc).
		WithSkillsReader(skillSvc).
		WithPricingReader(freelancePricingSvc)
	freelancePricingHandler := handler.NewFreelancePricingHandler(freelancePricingSvc, freelanceProfileSvc)
	if searchPublisher != nil {
		freelancePricingHandler = freelancePricingHandler.WithSearchIndexPublisher(searchPublisher)
	}

	referrerProfileRepo := postgres.NewReferrerProfileRepository(db)
	referrerProfileSvc := referrerprofileapp.NewService(referrerProfileRepo)
	if searchPublisher != nil {
		referrerProfileSvc = referrerProfileSvc.WithSearchIndexPublisher(searchPublisher)
	}
	referrerPricingRepo := postgres.NewReferrerPricingRepository(db)
	referrerPricingSvc := referrerpricingapp.NewService(referrerPricingRepo)

	// Referral (apport d'affaires) feature — wired AFTER proposal/payment/
	// freelanceProfile because it plugs into them via setters to break the
	// import cycle. The feature is purely optional: startup with no
	// referral service leaves every exposed port nil, and every call site
	// short-circuits on that check.
	referralRepo := postgres.NewReferralRepository(db)
	referralSvc := referralapp.NewService(referralapp.ServiceDeps{
		Referrals:        referralRepo,
		Users:            userRepo,
		Messages:         messagingSvc,
		Notifications:    notifSvc,
		Stripe:           stripeSvc,
		Reversals:        stripeReversalSvc,
		SnapshotProfiles: referralapp.NewThinSnapshotLoader(freelanceProfileRepo),
		StripeAccounts:    referralapp.NewOrgStripeAccountResolver(organizationRepo),
		OrgMembers:        referralapp.NewOrgDirectoryMemberResolver(organizationRepo, organizationMemberRepo),
		ProposalSummaries: referralapp.NewProposalRepoSummaryResolver(proposalRepo, milestoneRepo),
	})
	// Setter-based wiring to avoid import cycles between proposal/payment/embedded.
	proposalSvc.SetReferralAttributor(referralSvc)
	paymentInfoSvc.SetReferralDistributor(referralSvc)
	paymentInfoSvc.SetReferralClawback(referralSvc)
	paymentInfoSvc.SetReferralWalletReader(referralSvc)
	referralHandler := handler.NewReferralHandler(referralSvc)

	// Referral scheduler — hourly tick running ExpireStaleIntros (14 days
	// of silence on pending_* rows) and ExpireMaturedReferrals (active rows
	// past expires_at). Runs in its own goroutine; stops when pendingEventsCtx
	// is cancelled along with the rest of the background workers.
	referralScheduler := referralapp.NewScheduler(referralSvc, 0)
	go referralScheduler.Run(pendingEventsCtx)
	slog.Info("referral scheduler started")

	// Apporteur reputation aggregate — wired after the referral repo
	// exists. Kept as a fluent setter so the persona service stays
	// independent at the type level and the reputation surface can
	// be disabled by omitting this line.
	referrerProfileSvc = referrerProfileSvc.WithReputationDeps(referrerprofileapp.ReputationDeps{
		Referrals: referralRepo,
		Proposals: proposalRepo,
		Reviews:   reviewRepo,
		Users:     userRepo,
	})
	referrerProfileHandler := handler.
		NewReferrerProfileHandler(referrerProfileSvc).
		WithPricingReader(referrerPricingSvc).
		WithOrgOwnerLookup(&orgOwnerLookupAdapter{orgs: organizationRepo})
	referrerPricingHandler := handler.NewReferrerPricingHandler(referrerPricingSvc, referrerProfileSvc)
	if searchPublisher != nil {
		referrerPricingHandler = referrerPricingHandler.WithSearchIndexPublisher(searchPublisher)
	}

	// Organization shared-profile handler — writes the photo /
	// location / languages columns that both personas JOIN at read
	// time. Reuses the optional Nominatim geocoder from the legacy
	// profile flow so behaviour stays byte-identical.
	organizationSharedHandler := handler.
		NewOrganizationSharedProfileHandler(organizationRepo).
		WithGeocoder(profileGeocoder)
	if searchPublisher != nil {
		organizationSharedHandler = organizationSharedHandler.WithSearchIndexPublisher(searchPublisher)
	}

	profileHandler := handler.
		NewProfileHandler(profileSvc, expertiseSvc).
		WithSkillsReader(skillSvc).
		WithPricingReader(profilePricingSvc)
	uploadHandler := handler.NewUploadHandler(storageSvc, profileRepo, mediaSvc)
	freelanceProfileVideoHandler := handler.NewFreelanceProfileVideoHandler(storageSvc, freelanceProfileRepo, mediaSvc)
	referrerProfileVideoHandler := handler.NewReferrerProfileVideoHandler(storageSvc, referrerProfileRepo, mediaSvc)
	healthHandler := handler.NewHealthHandler(db)
	if typesenseClient != nil {
		// Typesense is MANDATORY since phase 4 — the listing pages
		// have no SQL fallback. A failed ping takes /ready red so
		// load balancers rotate the misbehaving instance out.
		healthHandler = healthHandler.WithSearchPinger(typesenseClient, true)
	}
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
		// Backed by the organizations table since phase R5 — the Stripe
		// Connect account lives on the org (the merchant of record).
		embeddedNotifier := embeddedapp.NewNotifier(
			embeddedapp.NewNotificationSenderAdapter(notifSvc),
			organizationRepo,
			5*time.Minute,
		)
		// Wire the referral KYC listener on the embedded notifier so parked
		// pending_kyc commissions are drained the moment the referrer's
		// Stripe account becomes payable.
		embeddedNotifier.SetReferralKYCListener(referralSvc)
		stripeHandler = stripeHandler.WithEmbeddedNotifier(embeddedNotifier)
	}

	// Wallet handler
	walletHandler := handler.NewWalletHandler(paymentInfoSvc, proposalSvc)

	// Billing handler — read-only fee preview endpoint for the proposal
	// creation flow. Shares the payment service (no new dependencies) so
	// the fee schedule stays the single source of truth across CreatePaymentIntent
	// and the client-facing simulator.
	billingHandler := handler.NewBillingHandler(paymentInfoSvc)

	// Subscription (Premium) feature. Wires the cached reader BEFORE the
	// handlers because payment.SetSubscriptionReader must be called so
	// subsequent milestone releases see the waiver. The whole block is
	// optional: when Stripe is not configured the feature stays off and
	// payment falls back to the full grid fee everywhere.
	var subscriptionHandler *handler.SubscriptionHandler
	if stripeSvc != nil {
		stripeSubSvc := stripeadapter.NewSubscriptionService(cfg.StripeSecretKey)
		subRepo := postgres.NewSubscriptionRepository(db)
		amountsRepo := postgres.NewProviderMilestoneAmountsRepository(db)

		subscriptionAppSvc := subscriptionapp.NewService(subscriptionapp.ServiceDeps{
			Subscriptions: subRepo,
			Users:         userRepo,
			Amounts:       amountsRepo,
			Stripe:        stripeSubSvc,
			LookupKeys:    subscriptionapp.DefaultLookupKeys(),
			URLs: subscriptionapp.URLs{
				CheckoutSuccess: cfg.FrontendURL + "/billing/success?session_id={CHECKOUT_SESSION_ID}",
				CheckoutCancel:  cfg.FrontendURL + "/billing/cancel",
				PortalReturn:    cfg.FrontendURL + "/billing",
			},
		})

		// The payment feature reads Premium status through the cached
		// reader — the app service answers on cache miss, Redis serves
		// subsequent calls within 60s, and every webhook invalidates
		// the user's entry so state changes surface immediately.
		subscriptionReader := redisadapter.NewCachedSubscriptionReader(
			redisClient, subscriptionAppSvc, redisadapter.DefaultSubscriptionCacheTTL,
		)
		paymentInfoSvc.SetSubscriptionReader(subscriptionReader)

		subscriptionHandler = handler.NewSubscriptionHandler(subscriptionAppSvc)

		// Wire subscription events into the Stripe webhook dispatcher
		// along with the Redis-backed idempotency guard that dedupes
		// Stripe's own retry behaviour. The cache reader does double
		// duty as the invalidator the dispatcher flushes on each state
		// change.
		if stripeHandler != nil {
			idempotencyStore := redisadapter.NewWebhookIdempotencyStore(redisClient, redisadapter.DefaultWebhookIdempotencyTTL)
			stripeHandler = stripeHandler.WithSubscription(subscriptionAppSvc, subscriptionReader, idempotencyStore)
		}

		slog.Info("subscription feature enabled (premium plan)")
	} else {
		slog.Info("subscription feature disabled (stripe not configured)")
	}

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
		Milestones:    milestoneRepo,
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

	// Prometheus metrics registry. Exposed at GET /metrics by the
	// router. Instrumentation points (search handler, reindex CLI,
	// drift-check) receive this pointer via constructor injection.
	metrics := handler.NewMetrics()

	// Setup router
	r := handler.NewRouter(handler.RouterDeps{
		Auth:           authHandler,
		Invitation:     invitationHandler,
		Team:           teamHandler,
		RoleOverrides:  roleOverridesHandler,
		Profile:        profileHandler,
		ProfilePricing: profilePricingHandler,

		// Split-profile handlers (migrations 096-104).
		FreelanceProfile:      freelanceProfileHandler,
		FreelancePricing:      freelancePricingHandler,
		FreelanceProfileVideo: freelanceProfileVideoHandler,
		ReferrerProfile:       referrerProfileHandler,
		ReferrerPricing:       referrerPricingHandler,
		ReferrerProfileVideo:  referrerProfileVideoHandler,
		OrganizationShared:    organizationSharedHandler,

		Upload:              uploadHandler,
		Health:              healthHandler,
		Messaging:           messagingHandler,
		Proposal:            proposalHandler,
		Job:                 jobHandler,
		JobApplication:      jobAppHandler,
		Review:              reviewHandler,
		Report:              reportHandler,
		Call:                callHandler,
		SocialLink:          socialLinkHandler,
		FreelanceSocialLink: freelanceSocialLinkHandler,
		ReferrerSocialLink:  referrerSocialLinkHandler,
		Embedded:            handler.NewEmbeddedHandler(organizationRepo, cfg.FrontendURL),
		Notification:        notifHandler,
		Stripe:              stripeHandler,
		Wallet:              walletHandler,
		Billing:             billingHandler,
		Subscription:        subscriptionHandler,
		Admin:               adminHandler,
		Portfolio:           portfolioHandler,
		ProjectHistory:      projectHistoryHandler,
		Dispute:             disputeHandler,
		AdminDispute:        adminDisputeHandler,
		Skill:               skillHandler,
		Referral:            referralHandler,
		Search:              searchHandler,
		AdminSearchStats:    adminSearchStatsHandler,
		WSHandler:           wsHandler,
		Config:              cfg,
		TokenService:        tokenSvc,
		SessionService:      sessionSvc,
		UserRepo:            userRepo,
		Metrics:             metrics,
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

// orgOwnerLookupAdapter implements handler.OrgOwnerLookup on top of
// the existing OrganizationRepository. Lives in main.go because it is
// a one-line wiring detail that should not bloat the handler package
// nor the organization domain.
type orgOwnerLookupAdapter struct {
	orgs repository.OrganizationRepository
}

func (a *orgOwnerLookupAdapter) OwnerUserIDForOrg(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error) {
	org, err := a.orgs.FindByID(ctx, orgID)
	if err != nil {
		return uuid.Nil, err
	}
	return org.OwnerUserID, nil
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

// buildRankingPipeline composes the four Stage 2-5 ranking packages
// into the RankingPipeline consumed by app/search.Service. All knobs
// live in RANKING_* environment variables; see docs/ranking-tuning.md
// for the operator playbook. Missing env vars fall back to the safe
// public defaults published in docs/ranking-v1.md §11.
//
// Boot-time fail-loud policy : scorer + rules configs return an error
// on malformed values so a typo in a weight raises slog.Error +
// os.Exit(1) rather than silently zeroing the ranking.
//
// Extract-time configs (features + antigaming) swallow malformed
// values by design — their individual extractors handle zero values
// gracefully, so a mistyped threshold just falls back to the default
// rather than taking down the search path.
func buildRankingPipeline() *appsearch.RankingPipeline {
	fcfg := features.LoadConfigFromEnv()
	agCfg := antigaming.LoadConfigFromEnv()
	scCfg, scErr := scorer.LoadConfigFromEnv()
	if scErr != nil {
		slog.Error("ranking: scorer config invalid", "error", scErr)
		os.Exit(1)
	}
	rlCfg, rlErr := rules.LoadConfigFromEnv()
	if rlErr != nil {
		slog.Error("ranking: rules config invalid", "error", rlErr)
		os.Exit(1)
	}

	ext := features.NewDefaultExtractor(fcfg)
	ag := antigaming.NewPipeline(agCfg, antigaming.NoopLinkedReviewersDetector{}, antigaming.SlogLogger{})
	rer := scorer.NewWeightedScorer(scCfg)
	br := rules.NewBusinessRules(rlCfg)

	return appsearch.NewRankingPipeline(ext, ag, rer, br)
}
