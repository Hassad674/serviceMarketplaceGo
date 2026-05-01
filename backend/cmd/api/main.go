package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marketplace-backend/internal/adapter/livekit"
	"marketplace-backend/internal/adapter/nominatim"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	"marketplace-backend/internal/adapter/ws"
	callapp "marketplace-backend/internal/app/call"
	clientprofileapp "marketplace-backend/internal/app/clientprofile"
	embeddedapp "marketplace-backend/internal/app/embedded"
	jobapp "marketplace-backend/internal/app/job"
	"marketplace-backend/internal/app/messaging"
	appmoderation "marketplace-backend/internal/app/moderation"
	milestoneapp "marketplace-backend/internal/app/milestone"
	paymentapp "marketplace-backend/internal/app/payment"
	portfolioapp "marketplace-backend/internal/app/portfolio"
	profileapp "marketplace-backend/internal/app/profile"
	projecthistoryapp "marketplace-backend/internal/app/projecthistory"
	proposalapp "marketplace-backend/internal/app/proposal"
	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	reportapp "marketplace-backend/internal/app/report"
	reviewapp "marketplace-backend/internal/app/review"
	"marketplace-backend/internal/config"
	jobdomain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
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

	// Local aliases keep the rest of main.go readable. Each name
	// matches the variable that lived inline before phase-3-F.
	db := infra.DB
	redisClient := infra.Redis
	userRepo := infra.UserRepo
	profileRepo := infra.ProfileRepo
	resetRepo := infra.ResetRepo
	organizationRepo := infra.OrganizationRepo
	organizationMemberRepo := infra.OrganizationMemberRepo
	organizationInvitationRepo := infra.OrganizationInvitationRepo
	auditRepo := infra.AuditRepo
	moderationResultsRepo := infra.ModerationResultsRepo
	hasher := infra.Hasher
	tokenSvc := infra.TokenSvc
	emailSvc := infra.EmailSvc
	storageSvc := infra.StorageSvc
	sessionSvc := infra.SessionSvc
	refreshBlacklistSvc := infra.RefreshBlacklistSvc
	messageRepo := infra.MessageRepo
	presenceSvc := infra.PresenceSvc
	streamBroadcaster := infra.StreamBroadcaster
	rateLimiter := infra.MessagingRateLimiter
	wsHub := infra.WSHub
	cookieCfg := infra.CookieCfg
	sourceID := infra.SourceID
	invitationRateLimiter := infra.InvitationRateLimiter

	// Auth feature — see wire_auth.go.
	authWire := wireAuth(authDeps{
		Cfg:                        cfg,
		Redis:                      redisClient,
		UserRepo:                   userRepo,
		ResetRepo:                  resetRepo,
		OrganizationRepo:           organizationRepo,
		OrganizationMemberRepo:     organizationMemberRepo,
		OrganizationInvitationRepo: organizationInvitationRepo,
		AuditRepo:                  auditRepo,
		Hasher:                     hasher,
		TokenSvc:                   tokenSvc,
		EmailSvc:                   emailSvc,
		SessionSvc:                 sessionSvc,
		RefreshBlacklistSvc:        refreshBlacklistSvc,
		CookieCfg:                  cookieCfg,
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
	// BUG-NEW-04 path 4/8: proposals is RLS-protected by migration 125
	// (USING client_organization_id = current_org OR provider_organization_id
	// = current_org). The txRunner wrap makes Create / Update /
	// GetByIDForOrg / List* pass under prod NOSUPERUSER NOBYPASSRLS.
	// Legacy GetByID stays for system-actor scheduler paths that run
	// with a privileged DB connection.
	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// Milestone — per-step funding/delivery sub-aggregate of a proposal.
	// The proposal app service consumes milestoneSvc to delegate the
	// Fund/Submit/Approve/Release transitions, and the dispute service
	// (phase 8) delegates OpenDispute/RestoreFromDispute to it as well.
	// BUG-NEW-04 path 5/8: proposal_milestones is RLS-protected by
	// migration 125 — milestones inherit security from the parent
	// proposal via a JOIN on the policy. The txRunner wrap makes
	// CreateBatch / Update / GetByIDForOrg / ListByProposalForOrg pass
	// under prod NOSUPERUSER NOBYPASSRLS. Each operation resolves the
	// parent proposal's stakeholder org via a defensive lookup before
	// opening the tenant tx.
	milestoneRepo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
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

	// Social links — see wire_social.go.
	socialLinks := wireSocialLinks(db)
	socialLinkHandler := socialLinks.Agency
	freelanceSocialLinkHandler := socialLinks.Freelance
	referrerSocialLinkHandler := socialLinks.Referrer

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

	// Stripe payment adapter — see wire_payment.go.
	stripe := wireStripe(cfg)
	stripeSvc := stripe.Charges
	stripeReversalSvc := stripe.Reversals
	stripeKYCReader := stripe.KYCReader

	// Payment records (custom KYC repos removed — see migration 040/041)
	// BUG-NEW-04 path 7/8: payment_records is RLS-protected by migration
	// 125 (USING organization_id = current_setting('app.current_org_id',
	// true)). The txRunner wrap makes Create / Update / GetByIDForOrg /
	// ListByOrganization pass under prod NOSUPERUSER NOBYPASSRLS. The
	// client's org (resolved from organization_members at INSERT time)
	// is the access boundary; provider-side reads of money received go
	// through the tenant-isolated proposal path instead.
	paymentRecordRepo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// Notification feature (push + email + WS) — see wire_notification.go.
	notifWorkerCtx, notifWorkerCancel := context.WithCancel(context.Background())
	defer notifWorkerCancel()
	notification := wireNotificationFeature(notificationDeps{
		Ctx:         notifWorkerCtx,
		Cfg:         cfg,
		DB:          db,
		Redis:       redisClient,
		SourceID:    sourceID,
		Email:       emailSvc,
		Users:       userRepo,
		Presence:    presenceSvc,
		Broadcaster: streamBroadcaster,
	})
	notifSvc := notification.Service
	notifHandler := notification.Handler

	// Organization team services — wired here so they can dispatch
	// team_* notifications through the same notifSvc used by the rest
	// of the app. See wire_team.go for the body.
	team := wireTeam(teamDeps{
		Cfg:                   cfg,
		DB:                    db,
		Redis:                 redisClient,
		Orgs:                  organizationRepo,
		Members:               organizationMemberRepo,
		Invitations:           organizationInvitationRepo,
		Users:                 userRepo,
		UserBatch:             userRepo,
		Hasher:                hasher,
		Email:                 emailSvc,
		Audits:                auditRepo,
		Notifications:         notifSvc,
		OrganizationSvc:       organizationSvc,
		SessionService:        sessionSvc,
		Cookie:                cookieCfg,
		InvitationRateLimiter: invitationRateLimiter,
		TokenService:          tokenSvc,
	})
	invitationSvc := team.InvitationSvc
	membershipSvc := team.MembershipSvc
	roleOverridesSvc := team.RoleOverridesSvc
	_ = roleOverridesSvc // used only via roleOverridesHandler below

	// KYC enforcement scheduler — sends reminders at day 0/3/7/14 for
	// providers with available funds who haven't completed Stripe KYC.
	// See startKYCScheduler in wire_notification.go.
	kycCtx, kycCancel := context.WithCancel(context.Background())
	defer kycCancel()
	startKYCScheduler(kycSchedulerDeps{
		Ctx:           kycCtx,
		Cfg:           cfg,
		Organizations: organizationRepo,
		Records:       paymentRecordRepo,
		Notifications: notifSvc,
	})

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
	// the outbox without re-wiring the whole chain. See wire_search.go.
	searchPublisher := wireSearchPublisher(cfg, pendingEventsRepo)

	// Outbox transaction runner (BUG-05). Used by the freelance and
	// legacy profile services to commit a profile mutation and the
	// matching `search.reindex` pending event in a single atomic
	// transaction — preventing permanent Postgres / Typesense drift
	// when the publisher Schedule path would otherwise fail after
	// the profile UPDATE has already committed. Cheap to construct:
	// holds only a *sql.DB pointer.
	txRunner := postgres.NewTxRunner(db)

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
		// Same concrete *postgres.UserRepository — it satisfies both
		// the wide UserRepository contract and the segregated
		// UserBatchReader (GetByIDs). The duplicate field exists so
		// the service can declare the segregated dep without forcing
		// every other caller to bring it in.
		UsersBatch:           userRepo,
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

	// Phase 6: pending_events worker — see wire_pending_events.go.
	// The worker handles milestone auto-approve, fund reminders, and
	// proposal auto-close; search reindex/delete handlers are added
	// later by wireSearchIndexer when Typesense is configured.
	pendingEventsWorker := newPendingEventsWorker(pendingEventsRepo, proposalSvc)

	// Search engine — Typesense indexer + query service + analytics.
	// See wire_search.go: wireSearchIndexer brings up the Typesense
	// client and registers indexer handlers on the pending-events
	// worker; wireSearchQuery composes the query-side service and
	// admin stats handler. Both return nil products when Typesense
	// is not configured, which keeps every downstream consumer's
	// `if x != nil` short-circuit working.
	typesenseClient := wireSearchIndexer(cfg, db, pendingEventsWorker)
	searchHandler, adminSearchStatsHandler := wireSearchQuery(cfg, db, typesenseClient)
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

	// Media moderation feature — see wire_media.go.
	mediaWorkerCtx, mediaWorkerCancel := context.WithCancel(context.Background())
	defer mediaWorkerCancel()
	mediaRepo := postgres.NewMediaRepository(db)
	media := wireMediaModeration(mediaDeps{
		Ctx:         mediaWorkerCtx,
		Cfg:         cfg,
		DB:          db,
		Redis:       redisClient,
		Broadcaster: streamBroadcaster,
		Email:       emailSvc,
		SessionSvc:  sessionSvc,
		Storage:     storageSvc,
		Users:       userRepo,
		Reports:     reportSvc,
		MediaRepo:   mediaRepo,
	})
	mediaSvc := media.MediaSvc
	textModerationSvc := media.TextModeration
	adminNotifierSvc := media.AdminNotifier

	// Wire media recorder into messaging so file/voice messages are tracked.
	messagingSvc.SetMediaRecorder(mediaSvc)

	// Central text moderation orchestrator. One instance fans every
	// pipeline (messaging, reviews, profile blocking, jobs, …) through
	// the same analyse → decide → persist → audit → notify chain so
	// the policy lives in one place.
	moderationOrchestrator := appmoderation.NewService(appmoderation.Deps{
		TextModeration: textModerationSvc,
		Results:        moderationResultsRepo,
		Audit:          auditRepo,
		AdminNotifier:  adminNotifierSvc,
	})
	messagingSvc.SetModerationOrchestrator(moderationOrchestrator)
	reviewSvc.SetModerationOrchestrator(moderationOrchestrator)
	authSvc.SetModerationOrchestrator(moderationOrchestrator)
	profileSvc.WithModerationOrchestrator(moderationOrchestrator)
	jobSvc.SetModerationOrchestrator(moderationOrchestrator)
	proposalSvc.SetModerationOrchestrator(moderationOrchestrator)

	// Admin feature — see wire_admin.go.
	adminHandler := wireAdmin(adminDeps{
		DB:                  db,
		Users:               userRepo,
		Reports:             reportRepo,
		Reviews:             reviewRepo,
		Jobs:                jobRepo,
		Applications:        jobAppRepo,
		Proposals:           proposalRepo,
		Media:               mediaRepo,
		ModerationResults:   moderationResultsRepo,
		Audit:               auditRepo,
		Storage:             storageSvc,
		Session:             sessionSvc,
		Broadcaster:         streamBroadcaster,
		AdminNotifier:       adminNotifierSvc,
		Organizations:       organizationRepo,
		OrganizationMembers: organizationMemberRepo,
		OrganizationInvites: organizationInvitationRepo,
		Membership:          membershipSvc,
		Invitation:          invitationSvc,
	})

	// SEC-11: Redis-backed sliding-window rate limiter. The same
	// instance hosts every quota class — the per-route policy and key
	// extractor are passed at the route definition site.
	trustedProxies, err := middleware.ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		slog.Error("invalid TRUSTED_PROXIES", "error", err)
		os.Exit(1)
	}
	httpRateLimiter := middleware.NewRateLimiter(redisClient, trustedProxies)

	// Team handlers were wired alongside the team services in
	// wire_team.go — pull them out of the team wiring struct so the
	// router builder reads as a flat list of handler bindings below.
	invitationHandler := team.InvitationHandler
	teamHandler := team.TeamHandler
	roleOverridesHandler := team.RoleOverridesHandler
	// Expertise + skills + profile pricing — see wire_skills.go.
	skills := wireSkillsAndPricing(db, organizationRepo, userRepo, searchPublisher)
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
		DB:              db,
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

	// ---- Phase 4-M: Redis cache-aside on hot read paths ----
	//
	// Each cache wraps the underlying app service via the decorator
	// pattern (see adapter/redis/profile_cache.go for the rationale).
	// Reads first consult Redis; misses fall through to the service
	// and back-fill the entry. Writes go through the service directly
	// — the service fires the cache's Invalidate hook AFTER a
	// successful DB write (cache-aside contract — DB write succeeds
	// → cache delete; reverse order opens a split-brain window).
	//
	// TTLs are tuned per signal volatility:
	//   - profile:agency:{org}      60s (operator edits are rare)
	//   - profile:freelance:{org}   60s (same)
	//   - expertise:org:{org}       5min (lists change very rarely)
	//   - skills:curated:{key}:{n}  10min (catalog is curator-seeded)
	//
	// Stampede protection: every cache uses a singleflight.Group so
	// a thundering herd on a cold key triggers exactly one DB call.
	//
	// Negative caching: per-org profile caches absorb 404 spam by
	// caching the not-found signal for 30s.
	//
	// Wired here (after wireSkillsAndPricing + wirePersonas) so the
	// caches see the search-publisher-bound services produced by those
	// helpers, then re-bind the affected handlers downstream.
	publicProfileCache := redisadapter.NewCachedPublicProfileReader(
		redisClient, profileSvc,
		redisadapter.DefaultPublicProfileCacheTTL,
		redisadapter.DefaultPublicProfileNegativeTTL,
	)
	profileSvc = profileSvc.WithCacheInvalidator(publicProfileCache)

	freelanceProfileSvc := personas.FreelanceProfileSvc
	publicFreelanceProfileCache := redisadapter.NewCachedPublicFreelanceProfileReader(
		redisClient, freelanceProfileSvc,
		redisadapter.DefaultPublicProfileCacheTTL,
		redisadapter.DefaultPublicProfileNegativeTTL,
	)
	freelanceProfileSvc = freelanceProfileSvc.WithCacheInvalidator(publicFreelanceProfileCache)

	expertiseCache := redisadapter.NewCachedExpertiseReader(
		redisClient, expertiseSvc, redisadapter.DefaultExpertiseCacheTTL,
	)
	expertiseSvc = expertiseSvc.WithCacheInvalidator(expertiseCache)

	skillCatalogCache := redisadapter.NewCachedSkillCatalogReader(
		redisClient, skillSvc, redisadapter.DefaultSkillCatalogCacheTTL,
	)
	// The skill handler needs every method on the skill service —
	// the cache only covers the two highest-traffic catalog reads.
	// A tiny composite routes the cached methods through Redis and
	// delegates everything else to the underlying service. See
	// caching_skill_service.go for the wrapper definition.
	cachingSkillSvc := newCachingSkillService(skillSvc, skillCatalogCache)
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
		DB:               db,
		Users:            userRepo,
		Organizations:    organizationRepo,
		OrganizationMems: organizationMemberRepo,
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
			Users:     userRepo,
		},
		OrgOwnerLookup:  &orgOwnerLookupAdapter{orgs: organizationRepo},
		SearchPublisher: searchPublisher,
	})
	referrerProfileHandler := referrer.ProfileHandler
	referrerPricingHandler := referrer.PricingHandler
	_ = referrer.Service

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

	// Client profile (migration 114) — the client-facing facet of the
	// organization's public profile. Two services orchestrate the
	// feature: the write path (ClientProfileService, co-located with
	// the profile aggregate) and the read path (clientprofile.Service,
	// its own package). Splitting write vs. read keeps each service
	// under the SRP cap and makes the feature fully removable by
	// dropping these few lines.
	clientProfileWriteSvc := profileapp.NewClientProfileService(profileRepo, organizationRepo)
	clientProfileReadSvc := clientprofileapp.NewService(clientprofileapp.ServiceDeps{
		Organizations: organizationRepo,
		Profiles:      profileRepo,
		Proposals:     proposalRepo,
		Reviews:       reviewRepo,
	})
	clientProfileHandler := handler.NewClientProfileHandler(clientProfileWriteSvc, clientProfileReadSvc)

	profileHandler := handler.
		NewProfileHandler(profileSvc, expertiseSvc).
		WithPublicReader(publicProfileCache).
		WithExpertiseReader(expertiseCache).
		WithSkillsReader(skillSvc).
		WithPricingReader(profilePricingSvc).
		WithClientStatsReader(clientProfileReadSvc)
	// uploadCtx is cancelled at SIGTERM so in-flight RecordUpload
	// goroutines (fired by /upload/* endpoints) wind down their
	// downstream Rekognition / S3 work cleanly. Closes BUG-17 — the
	// previous detached goroutines were truncated mid-flight and left
	// orphan media records.
	uploadCtx, uploadCancel := context.WithCancel(context.Background())
	defer uploadCancel()
	uploadHandler := handler.NewUploadHandler(storageSvc, profileRepo, mediaSvc).
		WithShutdownContext(uploadCtx)
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

	// Subscription (Premium) feature — see wire_subscription.go.
	subscription := wireSubscription(subscriptionDeps{
		Cfg:            cfg,
		DB:             db,
		Redis:          redisClient,
		Users:          userRepo,
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
			DB:              db,
			Redis:           redisClient,
			Email:           emailSvc,
			Storage:         storageSvc,
			Organizations:   organizationRepo,
			Users:           userRepo,
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
		DB:             db,
		Proposals:      proposalRepo,
		Milestones:     milestoneRepo,
		Users:          userRepo,
		MessageRepo:    messageRepo,
		Messaging:      messagingSvc,
		Notifications:  notifSvc,
		Payments:       paymentInfoSvc,
		ProposalSvcRef: proposalSvc,
	})
	disputeHandler := dispute.Handler
	adminDisputeHandler := dispute.AdminHandler

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
		ClientProfile:  clientProfileHandler,
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
		BillingProfile:      billingProfileHandler,
		Invoice:             invoiceHandler,
		AdminCreditNote:     adminCreditNoteHandler,
		AdminInvoice:        adminInvoiceHandler,
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
		TokenService:         tokenSvc,
		SessionService:       sessionSvc,
		UserRepo:             userRepo,
		OrgOverridesResolver: orgOverridesAdapter{repo: organizationRepo},
		Metrics:              metrics,
		RateLimiter:          httpRateLimiter,
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

	// BUG-17: drain in-flight upload goroutines (max 30s budget shared
	// with the HTTP shutdown above). uploadCancel above triggers the
	// individual goroutine's WithCancel so they observe the shutdown
	// signal; Stop() then waits for them to exit cleanly.
	uploadCancel()
	if err := uploadHandler.Stop(ctx); err != nil {
		slog.Warn("upload handler shutdown timed out", "error", err)
	}

	slog.Info("server stopped")
}
