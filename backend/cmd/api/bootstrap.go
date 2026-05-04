package main

import (
	"context"
	"log/slog"
	"time"

	"marketplace-backend/internal/adapter/nominatim"
	"marketplace-backend/internal/adapter/postgres"
	profileapp "marketplace-backend/internal/app/profile"
	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/observability"
)

// otelInitTimeout caps the OpenTelemetry exporter handshake at boot.
const otelInitTimeout = 10 * time.Second

// setupInfraAndTracing brings up every backbone resource (DB, Redis,
// repos, output adapters, messaging fan-out, WS hub) and the OTel
// tracing pipeline. Returns the infrastructure bundle, the infra
// context cancel (for WorkerCancels), plus the OTel shutdown closure
// (held on App.OtelShutdown so runServer can flush pending spans
// during the phase-3 graceful exit).
//
// Side effects on app: closeFns are appended for the infra cancel +
// closer + the WSHub pointer is published.
func setupInfraAndTracing(ctx context.Context, cfg *config.Config, app *App) (infrastructure, context.CancelFunc, error) {
	infraCtx, infraCancel := context.WithCancel(ctx)
	app.closeFns = append(app.closeFns, infraCancel)
	infra, closeInfra := wireInfrastructure(infraCtx, cfg)
	app.closeFns = append(app.closeFns, closeInfra)
	app.WSHub = infra.WSHub

	otelCfg := observability.LoadFromEnv()
	if otelCfg.Environment == "" {
		otelCfg.Environment = cfg.Env
	}
	otelInitCtx, otelInitCancel := context.WithTimeout(ctx, otelInitTimeout)
	otelShutdown, otelErr := observability.Init(otelInitCtx, otelCfg)
	otelInitCancel()
	if otelErr != nil {
		// Tracing failure must never block boot — log and continue.
		slog.Warn("otel init failed, continuing without tracing", "error", otelErr)
	}
	app.OtelShutdown = otelShutdown
	return infra, infraCancel, nil
}

// bootstrap performs every wireXxx call + setter pattern required to
// produce a fully assembled App. main.go shrinks to "load config →
// setup logger → init OTel → bootstrap → serve → shutdown".
//
// Every dependency lifecycle (worker contexts, infra teardown, OTel
// shutdown) is captured on the returned App so the caller can drive
// it without re-deriving the wiring.
func bootstrap(ctx context.Context, cfg *config.Config) (*App, error) {
	app := &App{Cfg: cfg}

	infra, infraCancel, err := setupInfraAndTracing(ctx, cfg, app)
	if err != nil {
		return nil, err
	}

	authWire := wireAuth(authDeps{
		Cfg:                        cfg,
		Redis:                      infra.Redis,
		UserRepo:                   infra.UserRepo,
		ResetRepo:                  infra.ResetRepo,
		OrganizationRepo:           infra.OrganizationRepo,
		OrganizationMemberRepo:     infra.OrganizationMemberRepo,
		OrganizationInvitationRepo: infra.OrganizationInvitationRepo,
		AuditRepo:                  infra.AuditRepo,
		Hasher:                     infra.Hasher,
		TokenSvc:                   infra.TokenSvc,
		EmailSvc:                   infra.EmailSvc,
		SessionSvc:                 infra.SessionSvc,
		RefreshBlacklistSvc:        infra.RefreshBlacklistSvc,
		CookieCfg:                  infra.CookieCfg,
	})
	organizationSvc := authWire.OrganizationSvc
	authSvc := authWire.AuthSvc
	authHandler := authWire.AuthHandler

	// Profile + Tier-1 geocoder.
	profileGeocoder := nominatim.NewGeocoder("marketplace-backend/1.0 (contact@marketplace.local)")
	profileSvc := profileapp.NewService(infra.ProfileRepo).WithGeocoder(profileGeocoder)

	messagingSvc := wireMessaging(messagingDeps{
		MessageRepo:      infra.MessageRepo,
		UserRepo:         infra.UserRepo,
		OrganizationRepo: infra.OrganizationRepo,
		OrgMembers:       infra.OrganizationMemberRepo,
		Presence:         infra.PresenceSvc,
		Broadcaster:      infra.StreamBroadcaster,
		Storage:          infra.StorageSvc,
		RateLimiter:      infra.MessagingRateLimiter,
	})

	proposalRepos := wireProposalRepos(proposalReposDeps{Cfg: cfg, DB: infra.DB})
	proposalRepo := proposalRepos.ProposalRepo
	milestoneRepo := proposalRepos.MilestoneRepo
	paymentRecordRepo := proposalRepos.PaymentRecordRepo
	bonusLogRepo := proposalRepos.BonusLogRepo
	pendingEventsRepo := proposalRepos.PendingEventsRepo
	milestoneTransitionsRepo := proposalRepos.MilestoneTransitionsRepo
	searchPublisher := proposalRepos.SearchPublisher
	txRunner := proposalRepos.TxRunner

	jobsWire := wireJobs(jobsDeps{
		DB:               infra.DB,
		UserRepo:         infra.UserRepo,
		OrganizationRepo: infra.OrganizationRepo,
		ProfileRepo:      infra.ProfileRepo,
		MessagingSvc:     messagingSvc,
	})
	jobRepo := jobsWire.JobRepo
	jobAppRepo := jobsWire.JobAppRepo
	jobCreditRepo := jobsWire.JobCreditRepo
	jobSvc := jobsWire.JobSvc

	reviewRepoWire := wireReviewRepo(infra.DB)
	reviewRepo := reviewRepoWire.Repo

	socialLinks := wireSocialLinks(infra.DB)
	socialLinkHandler := socialLinks.Agency
	freelanceSocialLinkHandler := socialLinks.Freelance
	referrerSocialLinkHandler := socialLinks.Referrer

	portfolioHandler := wirePortfolio(infra.DB).Handler
	projectHistoryHandler := wireProjectHistory(projectHistoryDeps{
		ProposalRepo: proposalRepo,
		ReviewRepo:   reviewRepo,
	}).Handler

	callHandler := wireCall(callDeps{
		Cfg:          cfg,
		Redis:        infra.Redis,
		Presence:     infra.PresenceSvc,
		Broadcaster:  infra.StreamBroadcaster,
		MessagingSvc: messagingSvc,
		UserRepo:     infra.UserRepo,
	})

	stripe := wireStripe(cfg)
	stripeSvc := stripe.Charges
	stripeReversalSvc := stripe.Reversals
	stripeKYCReader := stripe.KYCReader

	notifWorkerCtx, notifWorkerCancel := context.WithCancel(ctx)
	app.closeFns = append(app.closeFns, notifWorkerCancel)
	notification := wireNotificationFeature(notificationDeps{
		Ctx:         notifWorkerCtx,
		Cfg:         cfg,
		DB:          infra.DB,
		Redis:       infra.Redis,
		SourceID:    infra.SourceID,
		Email:       infra.EmailSvc,
		Users:       infra.UserRepo,
		Presence:    infra.PresenceSvc,
		Broadcaster: infra.StreamBroadcaster,
	})
	notifSvc := notification.Service

	team := wireTeam(teamDeps{
		Cfg:                   cfg,
		DB:                    infra.DB,
		Redis:                 infra.Redis,
		Orgs:                  infra.OrganizationRepo,
		Members:               infra.OrganizationMemberRepo,
		Invitations:           infra.OrganizationInvitationRepo,
		Users:                 infra.UserRepo,
		UserBatch:             infra.UserRepo,
		Hasher:                infra.Hasher,
		Email:                 infra.EmailSvc,
		Audits:                infra.AuditRepo,
		Notifications:         notifSvc,
		OrganizationSvc:       organizationSvc,
		SessionService:        infra.SessionSvc,
		Cookie:                infra.CookieCfg,
		InvitationRateLimiter: infra.InvitationRateLimiter,
		TokenService:          infra.TokenSvc,
	})
	invitationSvc := team.InvitationSvc
	membershipSvc := team.MembershipSvc

	kycCtx, kycCancel := context.WithCancel(ctx)
	app.closeFns = append(app.closeFns, kycCancel)
	wireKYC(kycDeps{
		Ctx:           kycCtx,
		Cfg:           cfg,
		Organizations: infra.OrganizationRepo,
		Records:       paymentRecordRepo,
		Notifications: notifSvc,
	})

	paymentInfoSvc := wirePayment(paymentDeps{
		Cfg:               cfg,
		PaymentRecordRepo: paymentRecordRepo,
		UserRepo:          infra.UserRepo,
		OrganizationRepo:  infra.OrganizationRepo,
		StripeSvc:         stripeSvc,
		Notifications:     notifSvc,
	})

	proposalWire := wireProposalService(proposalServiceDeps{
		Cfg:                      cfg,
		ProposalRepo:             proposalRepo,
		MilestoneRepo:            milestoneRepo,
		MilestoneTransitionsRepo: milestoneTransitionsRepo,
		PendingEventsRepo:        pendingEventsRepo,
		BonusLogRepo:             bonusLogRepo,
		UserRepo:                 infra.UserRepo,
		UserBatch:                infra.UserRepo,
		OrganizationRepo:         infra.OrganizationRepo,
		JobCreditRepo:            jobCreditRepo,
		StorageSvc:               infra.StorageSvc,
		MessagingSvc:             messagingSvc,
		NotifSvc:                 notifSvc,
		PaymentInfoSvc:           paymentInfoSvc,
	})
	proposalSvc := proposalWire.ProposalSvc
	pendingEventsWorker := proposalWire.PendingEventsWorker

	paymentInfoSvc.SetProposalStatusReader(newProposalStatusAdapter(proposalSvc))

	typesenseClient := wireSearchIndexer(cfg, infra.DB, pendingEventsWorker)
	searchHandler, adminSearchStatsHandler := wireSearchQuery(cfg, infra.DB, typesenseClient)
	pendingEventsCtx, pendingEventsCancel := context.WithCancel(ctx)
	app.closeFns = append(app.closeFns, pendingEventsCancel)

	reviewServiceWire := wireReviewService(reviewServiceDeps{
		ReviewRepo:    reviewRepo,
		ProposalRepo:  proposalRepo,
		UserRepo:      infra.UserRepo,
		Notifications: notifSvc,
	})
	reviewSvc := reviewServiceWire.Svc

	reportWire := wireReport(reportDeps{
		DB:          infra.DB,
		UserRepo:    infra.UserRepo,
		MessageRepo: infra.MessageRepo,
		JobRepo:     jobRepo,
		JobAppRepo:  jobAppRepo,
	})
	reportRepo := reportWire.Repo
	reportSvc := reportWire.Svc

	mediaWorkerCtx, mediaWorkerCancel := context.WithCancel(ctx)
	app.closeFns = append(app.closeFns, mediaWorkerCancel)
	mediaRepo := postgres.NewMediaRepository(infra.DB)
	media := wireMediaModeration(mediaDeps{
		Ctx:         mediaWorkerCtx,
		Cfg:         cfg,
		DB:          infra.DB,
		Redis:       infra.Redis,
		Broadcaster: infra.StreamBroadcaster,
		Email:       infra.EmailSvc,
		SessionSvc:  infra.SessionSvc,
		Storage:     infra.StorageSvc,
		Users:       infra.UserRepo,
		Reports:     reportSvc,
		MediaRepo:   mediaRepo,
	})
	mediaSvc := media.MediaSvc
	textModerationSvc := media.TextModeration
	adminNotifierSvc := media.AdminNotifier

	messagingSvc.SetMediaRecorder(mediaSvc)

	moderationOrchestrator := wireModeration(moderationDeps{
		TextModeration:        textModerationSvc,
		ModerationResultsRepo: infra.ModerationResultsRepo,
		AuditRepo:             infra.AuditRepo,
		AdminNotifier:         adminNotifierSvc,
	})
	messagingSvc.SetModerationOrchestrator(moderationOrchestrator)
	reviewSvc.SetModerationOrchestrator(moderationOrchestrator)
	authSvc.SetModerationOrchestrator(moderationOrchestrator)
	profileSvc.WithModerationOrchestrator(moderationOrchestrator)
	jobSvc.SetModerationOrchestrator(moderationOrchestrator)
	proposalSvc.SetModerationOrchestrator(moderationOrchestrator)

	adminHandler := wireAdmin(adminDeps{
		DB:                  infra.DB,
		Users:               infra.UserRepo,
		Reports:             reportRepo,
		Reviews:             reviewRepo,
		Jobs:                jobRepo,
		Applications:        jobAppRepo,
		Proposals:           proposalRepo,
		Media:               mediaRepo,
		ModerationResults:   infra.ModerationResultsRepo,
		Audit:               infra.AuditRepo,
		Storage:             infra.StorageSvc,
		Session:             infra.SessionSvc,
		Broadcaster:         infra.StreamBroadcaster,
		AdminNotifier:       adminNotifierSvc,
		Organizations:       infra.OrganizationRepo,
		OrganizationMembers: infra.OrganizationMemberRepo,
		OrganizationInvites: infra.OrganizationInvitationRepo,
		Membership:          membershipSvc,
		Invitation:          invitationSvc,
	})

	httpRateLimiter := wireRateLimiter(rateLimiterDeps{Cfg: cfg, Redis: infra.Redis})

	// N4: wire the per-IP brute-force gate. The auth handler reuses
	// the rate-limiter's IP extraction (trusted-proxy XFF + IPv6 /64
	// mask) so the per-IP lockout key is identical to the throttle
	// key — no risk of one gate seeing a different IP space than the
	// other.
	authHandler.WithIPExtractor(httpRateLimiter.ClientIP)

	invitationHandler := team.InvitationHandler
	teamHandler := team.TeamHandler
	roleOverridesHandler := team.RoleOverridesHandler

	skills := wireSkillsAndPricing(infra.DB, infra.OrganizationRepo, infra.UserRepo, searchPublisher)
	expertiseSvc := skills.ExpertiseSvc
	skillSvc := skills.SkillSvc
	skillHandler := skills.SkillHandler
	profilePricingSvc := skills.ProfilePricingSvc
	profilePricingHandler := skills.ProfilePricingHandler

	personas := wirePersonas(personasDeps{
		DB:              infra.DB,
		ProfileSvc:      profileSvc,
		SearchPublisher: searchPublisher,
		TxRunner:        txRunner,
		SkillsReader:    skillSvc,
	})
	profileSvc = personas.ProfileSvc
	freelanceProfileRepo := personas.FreelanceProfileRepo
	freelanceProfileHandler := personas.FreelanceProfileHandler
	freelancePricingHandler := personas.FreelancePricingHandler
	referrerProfileRepo := personas.ReferrerProfileRepo
	referrerProfileSvc := personas.ReferrerProfileSvc
	referrerPricingSvc := personas.ReferrerPricingSvc

	caches := wireCaches(cachesDeps{
		Redis:               infra.Redis,
		ProfileSvc:          profileSvc,
		FreelanceProfileSvc: personas.FreelanceProfileSvc,
		ExpertiseSvc:        expertiseSvc,
		SkillSvc:            skillSvc,
	})
	publicProfileCache := caches.PublicProfileCache
	publicFreelanceProfileCache := caches.PublicFreelanceProfileCache
	expertiseCache := caches.ExpertiseCache
	profileSvc = caches.ProfileSvc
	expertiseSvc = caches.ExpertiseSvc
	cachingSkillSvc := caches.CachingSkill
	skillHandler = handler.NewSkillHandler(cachingSkillSvc)
	if searchPublisher != nil {
		skillHandler = skillHandler.WithSearchIndexPublisher(searchPublisher)
	}
	freelanceProfileHandler = freelanceProfileHandler.WithPublicReader(publicFreelanceProfileCache)

	referral := wireReferral(referralDeps{
		Ctx:              pendingEventsCtx,
		DB:               infra.DB,
		Users:            infra.UserRepo,
		Organizations:    infra.OrganizationRepo,
		OrganizationMems: infra.OrganizationMemberRepo,
		Proposals:        proposalRepo,
		Milestones:       milestoneRepo,
		Messaging:        messagingSvc,
		Notifications:    notifSvc,
		Stripe:           stripeSvc,
		StripeReversals:  stripeReversalSvc,
		FreelanceProfile: freelanceProfileRepo,
		Proposal:         proposalSvc,
		Payment:          paymentInfoSvc,
	})
	referralSvc := referral.Service
	referralHandler := referral.Handler
	referralRepo := referral.Repo

	referrer := finaliseReferrerHandlers(referrerReputationDeps{
		ReferrerProfileSvc: referrerProfileSvc,
		ReferrerPricingSvc: referrerPricingSvc,
		Reputation: referrerprofileapp.ReputationDeps{
			Referrals: referralRepo,
			Proposals: proposalRepo,
			Reviews:   reviewRepo,
			Users:     infra.UserRepo,
		},
		OrgOwnerLookup:  &orgOwnerLookupAdapter{orgs: infra.OrganizationRepo},
		SearchPublisher: searchPublisher,
	})
	referrerProfileHandler := referrer.ProfileHandler
	referrerPricingHandler := referrer.PricingHandler

	organizationSharedHandler := wireOrganizationShared(orgSharedDeps{
		OrganizationRepo: infra.OrganizationRepo,
		ProfileGeocoder:  profileGeocoder,
		SearchPublisher:  searchPublisher,
	})

	clientProfile := wireClientProfile(clientProfileDeps{
		ProfileRepo:      infra.ProfileRepo,
		OrganizationRepo: infra.OrganizationRepo,
		ProposalRepo:     proposalRepo,
		ReviewRepo:       reviewRepo,
	})
	clientProfileReadSvc := clientProfile.ReadSvc
	clientProfileHandler := clientProfile.Handler

	profileHandler := wireProfileHandler(profileHandlerDeps{
		ProfileSvc:         profileSvc,
		ExpertiseSvc:       expertiseSvc,
		PublicProfileCache: publicProfileCache,
		ExpertiseCache:     expertiseCache,
		SkillsReader:       skillSvc,
		ProfilePricingSvc:  profilePricingSvc,
		ClientStatsReader:  clientProfileReadSvc,
	})

	uploadCtx, uploadCancel := context.WithCancel(ctx)
	app.UploadCancel = uploadCancel
	app.closeFns = append(app.closeFns, uploadCancel)
	uploadsWire := wireUploads(uploadsDeps{
		UploadCtx:            uploadCtx,
		DB:                   infra.DB,
		Storage:              infra.StorageSvc,
		ProfileRepo:          infra.ProfileRepo,
		FreelanceProfileRepo: freelanceProfileRepo,
		ReferrerProfileRepo:  referrerProfileRepo,
		MediaSvc:             mediaSvc,
	})
	uploadHandler := uploadsWire.UploadHandler
	freelanceProfileVideoHandler := uploadsWire.FreelanceProfileVideoHandler
	referrerProfileVideoHandler := uploadsWire.ReferrerProfileVideoHandler
	healthHandler := uploadsWire.HealthHandler
	if typesenseClient != nil {
		healthHandler = healthHandler.WithSearchPinger(typesenseClient, true)
	}
	app.UploadHandler = uploadHandler

	messagingHandler := handler.NewMessagingHandler(messagingSvc)
	proposalHandler := proposalWire.ProposalHandler
	jobHandler := jobsWire.JobHandler
	jobAppHandler := jobsWire.JobAppHandler
	reviewHandler := reviewServiceWire.Handler

	billing := wireBillingFeatures(billingFeatureDeps{
		Cfg:               cfg,
		Infra:             infra,
		StripeSvc:         stripeSvc,
		StripeKYCReader:   stripeKYCReader,
		NotifSvc:          notifSvc,
		ProposalSvc:       proposalSvc,
		PaymentInfoSvc:    paymentInfoSvc,
		ReferralSvc:       referralSvc,
		PendingEventsRepo: pendingEventsRepo,
	})
	stripeHandler := billing.StripeHandler
	walletHandler := billing.WalletHandler
	billingHandler := billing.BillingHandler
	subscriptionHandler := billing.SubscriptionHandler
	billingProfileHandler := billing.BillingProfileHandler
	invoiceHandler := billing.InvoiceHandler
	adminCreditNoteHandler := billing.AdminCreditNote
	adminInvoiceHandler := billing.AdminInvoice

	registerStripeWebhookWorker(pendingEventsWorker, stripeHandler)

	go func() {
		if err := pendingEventsWorker.Run(pendingEventsCtx); err != nil {
			slog.Error("pending events worker exited", "error", err)
		}
	}()
	slog.Info("phase 6: pending events worker started")

	disputeCtx, disputeCancel := context.WithCancel(ctx)
	app.closeFns = append(app.closeFns, disputeCancel)
	dispute := wireDispute(disputeDeps{
		Ctx:            disputeCtx,
		Cfg:            cfg,
		DB:             infra.DB,
		Proposals:      proposalRepo,
		Milestones:     milestoneRepo,
		Users:          infra.UserRepo,
		MessageRepo:    infra.MessageRepo,
		Messaging:      messagingSvc,
		Notifications:  notifSvc,
		Payments:       paymentInfoSvc,
		ProposalSvcRef: proposalSvc,
	})
	disputeHandler := dispute.Handler
	adminDisputeHandler := dispute.AdminHandler

	gdprCtx, gdprCancel := context.WithCancel(ctx)
	app.closeFns = append(app.closeFns, gdprCancel)
	gdpr := wireGDPR(gdprDeps{
		Ctx:    gdprCtx,
		Cfg:    cfg,
		DB:     infra.DB,
		Users:  infra.UserRepo,
		Hasher: infra.Hasher,
		Email:  infra.EmailSvc,
	})
	gdprHandler := gdpr.Handler

	wsHandler := wireWSHandler(wsHandlerDeps{
		Cfg:          cfg,
		WSHub:        infra.WSHub,
		MessagingSvc: messagingSvc,
		TokenSvc:     infra.TokenSvc,
		SessionSvc:   infra.SessionSvc,
		PresenceSvc:  infra.PresenceSvc,
		Broadcaster:  infra.StreamBroadcaster,
	})

	metrics := handler.NewMetrics()

	app.Router = assembleRouter(bootstrappedRouter{
		Handlers: buildRouterHandlers(finalHandlers{
			Auth: authHandler, Invitation: invitationHandler, Team: teamHandler,
			RoleOverrides: roleOverridesHandler, Profile: profileHandler,
			ClientProfile: clientProfileHandler, ProfilePricing: profilePricingHandler,
			FreelanceProfile: freelanceProfileHandler, FreelancePricing: freelancePricingHandler,
			FreelanceProfileVideo: freelanceProfileVideoHandler,
			ReferrerProfile:       referrerProfileHandler, ReferrerPricing: referrerPricingHandler,
			ReferrerProfileVideo: referrerProfileVideoHandler,
			OrganizationShared:   organizationSharedHandler,
			Upload:               uploadHandler, Health: healthHandler,
			Messaging: messagingHandler, Proposal: proposalHandler,
			Job: jobHandler, JobApplication: jobAppHandler,
			Review: reviewHandler, Report: reportWire.Handler, Call: callHandler,
			SocialLink: socialLinkHandler, FreelanceSocialLink: freelanceSocialLinkHandler,
			ReferrerSocialLink: referrerSocialLinkHandler,
			Embedded: wireEmbeddedHandler(embeddedHandlerDeps{
				OrganizationRepo: infra.OrganizationRepo, FrontendURL: cfg.FrontendURL,
			}),
			Notification:    notification.Handler,
			Stripe:          stripeHandler, Wallet: walletHandler, Billing: billingHandler,
			Subscription:    subscriptionHandler,
			BillingProfile:  billingProfileHandler, Invoice: invoiceHandler,
			AdminCreditNote: adminCreditNoteHandler, AdminInvoice: adminInvoiceHandler,
			Admin:           adminHandler, Portfolio: portfolioHandler,
			ProjectHistory:  projectHistoryHandler,
			Dispute:         disputeHandler, AdminDispute: adminDisputeHandler,
			GDPR:            gdprHandler, Skill: skillHandler, Referral: referralHandler,
			Search:          searchHandler, AdminSearchStats: adminSearchStatsHandler,
		}),
		WSHandler:   wsHandler,
		Cfg:         cfg,
		Infra:       infra,
		Metrics:     metrics,
		RateLimiter: httpRateLimiter,
	})

	// WorkerCancels collects every long-running goroutine's context
	// cancel so runServer's 3-step graceful shutdown can drain them
	// in a deterministic order. Sequence mirrors the original main.go.
	app.WorkerCancels = []context.CancelFunc{
		notifWorkerCancel,
		pendingEventsCancel,
		mediaWorkerCancel,
		kycCancel,
		disputeCancel,
		gdprCancel,
		infraCancel,
	}

	return app, nil
}
