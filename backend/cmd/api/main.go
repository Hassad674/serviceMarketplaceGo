package main

import (
	"context"
	"log/slog"
	"os"

	"marketplace-backend/internal/adapter/nominatim"
	"marketplace-backend/internal/adapter/postgres"
	profileapp "marketplace-backend/internal/app/profile"
	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
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

	// Fail-fast in production when secrets are missing or use the
	// open-source fallbacks. In development this only prints loud
	// warnings — see config.Validate for the policy.
	if err := cfg.Validate(); err != nil {
		slog.Error("config validation failed", "error", err)
		os.Exit(1)
	}

	// Bring up every backbone resource (DB, Redis, repos, output
	// adapters, messaging fan-out, WS hub) — see wire_infra.go.
	infraCtx, infraCancel := context.WithCancel(context.Background())
	defer infraCancel()
	infra, closeInfra := wireInfrastructure(infraCtx, cfg)
	defer closeInfra()

	// Auth feature — see wire_auth.go.
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
	// Profile service + Tier 1 geocoder (migration 083). The
	// Nominatim adapter is used as-is in every environment because
	// the public endpoint is free and the profile save flow
	// gracefully degrades on any geocoding failure — see
	// adapter/nominatim/geocoder.go and app/profile/service.go.
	profileGeocoder := nominatim.NewGeocoder("marketplace-backend/1.0 (contact@marketplace.local)")
	profileSvc := profileapp.NewService(infra.ProfileRepo).WithGeocoder(profileGeocoder)

	// Messaging service (initial wiring — MediaRecorder + Moderation
	// orchestrator setters are applied below after their respective
	// services exist). See wire_uploads_messaging_moderation_kyc.go.
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

	// Proposal feature (early-stage repos + searchPublisher + txRunner).
	// See wire_proposal.go. The matching app service + handler are
	// wired below by wireProposalService once notification / messaging /
	// payment have been built.
	proposalRepos := wireProposalRepos(proposalReposDeps{
		Cfg: cfg,
		DB:  infra.DB,
	})
	proposalRepo := proposalRepos.ProposalRepo
	milestoneRepo := proposalRepos.MilestoneRepo
	milestoneSvc := proposalRepos.MilestoneSvc
	paymentRecordRepo := proposalRepos.PaymentRecordRepo
	bonusLogRepo := proposalRepos.BonusLogRepo
	pendingEventsRepo := proposalRepos.PendingEventsRepo
	milestoneTransitionsRepo := proposalRepos.MilestoneTransitionsRepo
	searchPublisher := proposalRepos.SearchPublisher
	txRunner := proposalRepos.TxRunner
	_ = milestoneSvc // wired into proposal service deps below

	// Job feature — see wire_review_jobs_portfolio_report.go.
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

	// Review repository (early-stage). The app service is wired below
	// once the notification feature exists.
	reviewRepoWire := wireReviewRepo(infra.DB)
	reviewRepo := reviewRepoWire.Repo

	// Social links — see wire_social.go.
	socialLinks := wireSocialLinks(infra.DB)
	socialLinkHandler := socialLinks.Agency
	freelanceSocialLinkHandler := socialLinks.Freelance
	referrerSocialLinkHandler := socialLinks.Referrer

	// Portfolio feature — see wire_review_jobs_portfolio_report.go.
	portfolioWire := wirePortfolio(infra.DB)
	portfolioHandler := portfolioWire.Handler

	// Project history feature — see wire_review_jobs_portfolio_report.go.
	projectHistoryWire := wireProjectHistory(projectHistoryDeps{
		ProposalRepo: proposalRepo,
		ReviewRepo:   reviewRepo,
	})
	projectHistoryHandler := projectHistoryWire.Handler

	// Call feature (optional — only when LiveKit is configured).
	// See wire_call.go.
	callHandler := wireCall(callDeps{
		Cfg:          cfg,
		Redis:        infra.Redis,
		Presence:     infra.PresenceSvc,
		Broadcaster:  infra.StreamBroadcaster,
		MessagingSvc: messagingSvc,
		UserRepo:     infra.UserRepo,
	})

	// Stripe payment adapter — see wire_payment.go.
	stripe := wireStripe(cfg)
	stripeSvc := stripe.Charges
	stripeReversalSvc := stripe.Reversals
	stripeKYCReader := stripe.KYCReader

	// Notification feature (push + email + WS) — see wire_notification.go.
	notifWorkerCtx, notifWorkerCancel := context.WithCancel(context.Background())
	defer notifWorkerCancel()
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
	notifHandler := notification.Handler

	// Organization team services — wired here so they can dispatch
	// team_* notifications through the same notifSvc used by the rest
	// of the app. See wire_team.go for the body.
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
	roleOverridesSvc := team.RoleOverridesSvc
	_ = roleOverridesSvc // used only via roleOverridesHandler below

	// KYC enforcement scheduler — see
	// wire_uploads_messaging_moderation_kyc.go (delegates to
	// startKYCScheduler in wire_notification.go).
	kycCtx, kycCancel := context.WithCancel(context.Background())
	defer kycCancel()
	wireKYC(kycDeps{
		Ctx:           kycCtx,
		Cfg:           cfg,
		Organizations: infra.OrganizationRepo,
		Records:       paymentRecordRepo,
		Notifications: notifSvc,
	})

	// Payment service — see wire_payment.go.
	paymentInfoSvc := wirePayment(paymentDeps{
		Cfg:               cfg,
		PaymentRecordRepo: paymentRecordRepo,
		UserRepo:          infra.UserRepo,
		OrganizationRepo:  infra.OrganizationRepo,
		StripeSvc:         stripeSvc,
		Notifications:     notifSvc,
	})

	// Proposal service + worker + handler (late-stage). See
	// wire_proposal.go. Runs AFTER notification / messaging / payment
	// because the service deps reach into all three.
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

	// Wire proposal → payment status lookup so RequestPayout only
	// releases escrow funds for missions whose proposal has reached
	// "completed". Setter pattern because the dependency runs the wrong
	// way for constructor injection (payment is built before proposal).
	paymentInfoSvc.SetProposalStatusReader(newProposalStatusAdapter(proposalSvc))

	// Search engine — Typesense indexer + query service + analytics.
	// See wire_search.go: wireSearchIndexer brings up the Typesense
	// client and registers indexer handlers on the pending-events
	// worker; wireSearchQuery composes the query-side service and
	// admin stats handler. Both return nil products when Typesense
	// is not configured, which keeps every downstream consumer's
	// `if x != nil` short-circuit working.
	typesenseClient := wireSearchIndexer(cfg, infra.DB, pendingEventsWorker)
	searchHandler, adminSearchStatsHandler := wireSearchQuery(cfg, infra.DB, typesenseClient)
	pendingEventsCtx, pendingEventsCancel := context.WithCancel(context.Background())
	defer pendingEventsCancel()
	go func() {
		if err := pendingEventsWorker.Run(pendingEventsCtx); err != nil {
			slog.Error("pending events worker exited", "error", err)
		}
	}()
	slog.Info("phase 6: pending events worker started")

	// Review service + handler (late-stage) — see
	// wire_review_jobs_portfolio_report.go. Runs AFTER notification so
	// the service can fire submission events through the notif pipeline.
	reviewServiceWire := wireReviewService(reviewServiceDeps{
		ReviewRepo:    reviewRepo,
		ProposalRepo:  proposalRepo,
		UserRepo:      infra.UserRepo,
		Notifications: notifSvc,
	})
	reviewSvc := reviewServiceWire.Svc

	// Report feature — see wire_review_jobs_portfolio_report.go.
	reportWire := wireReport(reportDeps{
		DB:          infra.DB,
		UserRepo:    infra.UserRepo,
		MessageRepo: infra.MessageRepo,
		JobRepo:     jobRepo,
		JobAppRepo:  jobAppRepo,
	})
	reportRepo := reportWire.Repo
	reportSvc := reportWire.Svc
	reportHandler := reportWire.Handler

	// Media moderation feature — see wire_media.go.
	mediaWorkerCtx, mediaWorkerCancel := context.WithCancel(context.Background())
	defer mediaWorkerCancel()
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

	// Wire media recorder into messaging so file/voice messages are tracked.
	messagingSvc.SetMediaRecorder(mediaSvc)

	// Central text moderation orchestrator — see
	// wire_uploads_messaging_moderation_kyc.go. The 6
	// SetModerationOrchestrator setters below STAY in main.go because
	// they cross multiple wire boundaries.
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

	// Admin feature — see wire_admin.go.
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

	// SEC-11 HTTP rate limiter — see wire_router.go.
	httpRateLimiter := wireRateLimiter(rateLimiterDeps{
		Cfg:   cfg,
		Redis: infra.Redis,
	})

	// Team handlers were wired alongside the team services in
	// wire_team.go — pull them out of the team wiring struct so the
	// router builder reads as a flat list of handler bindings below.
	invitationHandler := team.InvitationHandler
	teamHandler := team.TeamHandler
	roleOverridesHandler := team.RoleOverridesHandler
	// Expertise + skills + profile pricing — see wire_skills.go.
	skills := wireSkillsAndPricing(infra.DB, infra.OrganizationRepo, infra.UserRepo, searchPublisher)
	expertiseSvc := skills.ExpertiseSvc
	skillSvc := skills.SkillSvc
	skillHandler := skills.SkillHandler
	profilePricingSvc := skills.ProfilePricingSvc
	profilePricingHandler := skills.ProfilePricingHandler

	// Split-profile feature (migrations 096-104) — see wire_personas.go.
	// Freelance / referrer / freelance pricing / referrer pricing
	// aggregates. The legacy profileSvc is re-bound via a fluent
	// setter when the search publisher is wired; main.go must keep
	// the new pointer for every downstream consumer.
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

	// Phase 4-M Redis cache-aside on hot read paths — see wire_caches.go.
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
	// Re-wire skillHandler with the cached service (wireSkillsAndPricing
	// produced it with the uncached service so the search publisher
	// could be attached; rebuild here so both the cache AND the
	// publisher are in play).
	skillHandler = handler.NewSkillHandler(cachingSkillSvc)
	if searchPublisher != nil {
		skillHandler = skillHandler.WithSearchIndexPublisher(searchPublisher)
	}

	// Re-bind freelanceProfileHandler with the public freelance cache
	// reader. wirePersonas built the handler without a public reader so
	// the cache (which depends on the cache-aware service above) could
	// be wired here without leaking into the persona helper.
	freelanceProfileHandler = freelanceProfileHandler.
		WithPublicReader(publicFreelanceProfileCache)

	// Referral (apport d'affaires) feature — see wire_referral.go.
	// Wired AFTER proposal/payment/freelanceProfile because it plugs
	// into them via setters to break the import cycle.
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

	// Apporteur reputation aggregate — wired after the referral repo
	// exists. See finaliseReferrerHandlers in wire_personas.go.
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
	_ = referrer.Service

	// Organization shared-profile handler — see wire_late_handlers.go.
	organizationSharedHandler := wireOrganizationShared(orgSharedDeps{
		OrganizationRepo: infra.OrganizationRepo,
		ProfileGeocoder:  profileGeocoder,
		SearchPublisher:  searchPublisher,
	})

	// Client profile (migration 114) — see wire_late_handlers.go.
	clientProfile := wireClientProfile(clientProfileDeps{
		ProfileRepo:      infra.ProfileRepo,
		OrganizationRepo: infra.OrganizationRepo,
		ProposalRepo:     proposalRepo,
		ReviewRepo:       reviewRepo,
	})
	clientProfileReadSvc := clientProfile.ReadSvc
	clientProfileHandler := clientProfile.Handler

	// Unified legacy profile handler — see wire_late_handlers.go.
	profileHandler := wireProfileHandler(profileHandlerDeps{
		ProfileSvc:         profileSvc,
		ExpertiseSvc:       expertiseSvc,
		PublicProfileCache: publicProfileCache,
		ExpertiseCache:     expertiseCache,
		SkillsReader:       skillSvc,
		ProfilePricingSvc:  profilePricingSvc,
		ClientStatsReader:  clientProfileReadSvc,
	})
	// uploadCtx is cancelled at SIGTERM so in-flight RecordUpload
	// goroutines (fired by /upload/* endpoints) wind down their
	// downstream Rekognition / S3 work cleanly. Closes BUG-17 — the
	// previous detached goroutines were truncated mid-flight and left
	// orphan media records.
	uploadCtx, uploadCancel := context.WithCancel(context.Background())
	defer uploadCancel()
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
		// Typesense is MANDATORY since phase 4 — the listing pages
		// have no SQL fallback. A failed ping takes /ready red so
		// load balancers rotate the misbehaving instance out.
		healthHandler = healthHandler.WithSearchPinger(typesenseClient, true)
	}
	messagingHandler := handler.NewMessagingHandler(messagingSvc)
	proposalHandler := proposalWire.ProposalHandler
	jobHandler := jobsWire.JobHandler
	jobAppHandler := jobsWire.JobAppHandler
	reviewHandler := reviewServiceWire.Handler

	// Stripe HTTP handler — see wire_late_handlers.go.
	stripeHandler := wireStripeHandler(stripeHandlerDeps{
		Cfg:               cfg,
		PaymentInfoSvc:    paymentInfoSvc,
		ProposalSvc:       proposalSvc,
		OrganizationRepo:  infra.OrganizationRepo,
		Notifications:     notifSvc,
		ReferralSvc:       referralSvc,
		PendingEventsRepo: pendingEventsRepo,
	})

	// Wallet + billing handlers — see wire_payment.go.
	walletHandler, billingHandler := wirePaymentHandlers(paymentHandlersDeps{
		PaymentInfoSvc: paymentInfoSvc,
		ProposalSvc:    proposalSvc,
	})

	// Subscription (Premium) feature — see wire_subscription.go.
	subscription := wireSubscription(subscriptionDeps{
		Cfg:            cfg,
		DB:             infra.DB,
		Redis:          infra.Redis,
		Users:          infra.UserRepo,
		Stripe:         stripeSvc,
		PaymentInfoSvc: paymentInfoSvc,
		StripeHandler:  stripeHandler,
	})
	subscriptionHandler := subscription.Handler
	subscriptionAppSvc := subscription.AppSvc
	stripeHandler = subscription.StripeHandler

	// Invoicing feature — outbound customer-facing invoices for
	// successful subscription payments. See wire_invoicing.go.
	// The block is optional: if Stripe is absent or the issuer/PDF
	// renderer init fails, every returned handler stays nil and the
	// router skips the corresponding routes. StripeHandler and
	// WalletHandler are re-bound so the router uses the invoicing-
	// aware variants.
	var billingProfileHandler *handler.BillingProfileHandler
	var invoiceHandler *handler.InvoiceHandler
	var adminCreditNoteHandler *handler.AdminCreditNoteHandler
	var adminInvoiceHandler *handler.AdminInvoiceHandler
	if stripeHandler != nil {
		invoicing := wireInvoicing(invoicingDeps{
			DB:              infra.DB,
			Redis:           infra.Redis,
			Email:           infra.EmailSvc,
			Storage:         infra.StorageSvc,
			Organizations:   infra.OrganizationRepo,
			Users:           infra.UserRepo,
			StripeKYC:       stripeKYCReader,
			StripeHandler:   stripeHandler,
			WalletHandler:   walletHandler,
			SubscriptionSvc: subscriptionAppSvc,
		})
		billingProfileHandler = invoicing.BillingProfile
		invoiceHandler = invoicing.Invoice
		adminCreditNoteHandler = invoicing.AdminCreditNote
		adminInvoiceHandler = invoicing.AdminInvoice
		stripeHandler = invoicing.StripeHandler
		walletHandler = invoicing.WalletHandler
	}

	// Dispute feature — see wire_dispute.go.
	disputeCtx, disputeCancel := context.WithCancel(context.Background())
	defer disputeCancel()
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

	// GDPR feature (P5) — right-to-erasure + right-to-export.
	// See wire_gdpr.go. The purge scheduler runs in its own goroutine
	// on a 24h cadence (1min in development) and stops with the
	// gdprCtx — same lifecycle pattern as wire_dispute.
	gdprCtx, gdprCancel := context.WithCancel(context.Background())
	defer gdprCancel()
	gdpr := wireGDPR(gdprDeps{
		Ctx:    gdprCtx,
		Cfg:    cfg,
		DB:     infra.DB,
		Users:  infra.UserRepo,
		Hasher: infra.Hasher,
		Email:  infra.EmailSvc,
	})
	gdprHandler := gdpr.Handler

	// WebSocket connection handler — see wire_router.go.
	wsHandler := wireWSHandler(wsHandlerDeps{
		Cfg:          cfg,
		WSHub:        infra.WSHub,
		MessagingSvc: messagingSvc,
		TokenSvc:     infra.TokenSvc,
		SessionSvc:   infra.SessionSvc,
		PresenceSvc:  infra.PresenceSvc,
		Broadcaster:  infra.StreamBroadcaster,
	})

	// Prometheus metrics registry. Exposed at GET /metrics by the
	// router. Instrumentation points (search handler, reindex CLI,
	// drift-check) receive this pointer via constructor injection.
	metrics := handler.NewMetrics()

	// Setup router — see wire_router.go for the full handler bundle.
	r := wireRouter(routerDepsBundle{
		Auth:                  authHandler,
		Invitation:            invitationHandler,
		Team:                  teamHandler,
		RoleOverrides:         roleOverridesHandler,
		Profile:               profileHandler,
		ClientProfile:         clientProfileHandler,
		ProfilePricing:        profilePricingHandler,
		FreelanceProfile:      freelanceProfileHandler,
		FreelancePricing:      freelancePricingHandler,
		FreelanceProfileVideo: freelanceProfileVideoHandler,
		ReferrerProfile:       referrerProfileHandler,
		ReferrerPricing:       referrerPricingHandler,
		ReferrerProfileVideo:  referrerProfileVideoHandler,
		OrganizationShared:    organizationSharedHandler,
		Upload:                uploadHandler,
		Health:                healthHandler,
		Messaging:             messagingHandler,
		Proposal:              proposalHandler,
		Job:                   jobHandler,
		JobApplication:        jobAppHandler,
		Review:                reviewHandler,
		Report:                reportHandler,
		Call:                  callHandler,
		SocialLink:            socialLinkHandler,
		FreelanceSocialLink:   freelanceSocialLinkHandler,
		ReferrerSocialLink:    referrerSocialLinkHandler,
		Embedded: wireEmbeddedHandler(embeddedHandlerDeps{
			OrganizationRepo: infra.OrganizationRepo,
			FrontendURL:      cfg.FrontendURL,
		}),
		Notification:         notifHandler,
		Stripe:               stripeHandler,
		Wallet:               walletHandler,
		Billing:              billingHandler,
		Subscription:         subscriptionHandler,
		BillingProfile:       billingProfileHandler,
		Invoice:              invoiceHandler,
		AdminCreditNote:      adminCreditNoteHandler,
		AdminInvoice:         adminInvoiceHandler,
		Admin:                adminHandler,
		Portfolio:            portfolioHandler,
		ProjectHistory:       projectHistoryHandler,
		Dispute:              disputeHandler,
		AdminDispute:         adminDisputeHandler,
		GDPR:                 gdprHandler,
		Skill:                skillHandler,
		Referral:             referralHandler,
		Search:               searchHandler,
		AdminSearchStats:     adminSearchStatsHandler,
		WSHandler:            wsHandler,
		Cfg:                  cfg,
		TokenService:         infra.TokenSvc,
		SessionService:       infra.SessionSvc,
		UserRepo:             infra.UserRepo,
		OrgOverridesResolver: orgOverridesAdapter{repo: infra.OrganizationRepo},
		Metrics:              metrics,
		RateLimiter:          httpRateLimiter,
	})

	// Run server + drive graceful shutdown — see wire_serve.go.
	runServer(serveDeps{
		Cfg:           cfg,
		Router:        r,
		UploadCancel:  uploadCancel,
		UploadHandler: uploadHandler,
	})
}
