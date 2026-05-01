package main

import (
	"context"
	"database/sql"

	"marketplace-backend/internal/adapter/postgres"
	mediaapp "marketplace-backend/internal/app/media"
	"marketplace-backend/internal/app/messaging"
	appmoderation "marketplace-backend/internal/app/moderation"
	notifapp "marketplace-backend/internal/app/notification"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// messagingDeps captures the upstream dependencies the messaging app
// service needs at construction time. The service is built EARLY so
// other features (job, proposal, etc.) can reach into it; the
// MediaRecorder + ModerationOrchestrator setters are wired later by
// main.go because they cross feature boundaries.
type messagingDeps struct {
	MessageRepo       repository.MessageRepository
	UserRepo          repository.UserRepository
	OrganizationRepo  repository.OrganizationRepository
	OrgMembers        repository.OrganizationMemberRepository
	Presence          service.PresenceService
	Broadcaster       service.MessageBroadcaster
	Storage           service.StorageService
	RateLimiter       service.MessagingRateLimiter
}

// wireMessaging brings up the messaging app service. The
// MediaRecorder + ModerationOrchestrator setters are NOT applied
// here — they cross multiple wire boundaries (the recorder lives in
// the media wire, the orchestrator in the moderation wire) so
// main.go threads them in after construction. Returning the bare
// service keeps this helper free of those late-stage dependencies.
func wireMessaging(deps messagingDeps) *messaging.Service {
	return messaging.NewService(messaging.ServiceDeps{
		Messages:      deps.MessageRepo,
		Users:         deps.UserRepo,
		Organizations: deps.OrganizationRepo,
		OrgMembers:    deps.OrgMembers,
		Presence:      deps.Presence,
		Broadcaster:   deps.Broadcaster,
		Storage:       deps.Storage,
		RateLimiter:   deps.RateLimiter,
		// MediaRecorder is set below after mediaSvc is created.
	})
}

// uploadsWiring carries the upload-related HTTP handlers. The upload
// context is owned by main.go (it is cancelled at SIGTERM as part
// of the graceful shutdown sequence to wind down RecordUpload
// goroutines cleanly) so it is passed in by the caller.
type uploadsWiring struct {
	UploadHandler                *handler.UploadHandler
	FreelanceProfileVideoHandler *handler.FreelanceProfileVideoHandler
	ReferrerProfileVideoHandler  *handler.ReferrerProfileVideoHandler
	HealthHandler                *handler.HealthHandler
}

// uploadsDeps captures the resources the upload handlers reach into:
// the SQL pool (only needed by the health handler's DB ping), the
// storage adapter, the profile / freelance / referrer repositories,
// and the media app service that records each successful upload.
type uploadsDeps struct {
	UploadCtx            context.Context
	DB                   *sql.DB
	Storage              service.StorageService
	ProfileRepo          repository.ProfileRepository
	FreelanceProfileRepo *postgres.FreelanceProfileRepository
	ReferrerProfileRepo  *postgres.ReferrerProfileRepository
	MediaSvc             *mediaapp.Service
}

// wireUploads brings up the upload-related HTTP handlers (legacy
// profile, freelance video, referrer video) plus the health probe.
//
// uploadCtx is cancelled at SIGTERM so in-flight RecordUpload
// goroutines (fired by /upload/* endpoints) wind down their
// downstream Rekognition / S3 work cleanly. Closes BUG-17 — the
// previous detached goroutines were truncated mid-flight and left
// orphan media records.
//
// The Typesense ping wiring stays in main.go (next to where the
// typesense client is built) — coupling it here would force the
// helper to take the typesense client which only one of the
// returned handlers cares about.
func wireUploads(deps uploadsDeps) uploadsWiring {
	uploadHandler := handler.NewUploadHandler(deps.Storage, deps.ProfileRepo, deps.MediaSvc).
		WithShutdownContext(deps.UploadCtx)
	freelanceProfileVideoHandler := handler.NewFreelanceProfileVideoHandler(deps.Storage, deps.FreelanceProfileRepo, deps.MediaSvc)
	referrerProfileVideoHandler := handler.NewReferrerProfileVideoHandler(deps.Storage, deps.ReferrerProfileRepo, deps.MediaSvc)
	healthHandler := handler.NewHealthHandler(deps.DB)
	return uploadsWiring{
		UploadHandler:                uploadHandler,
		FreelanceProfileVideoHandler: freelanceProfileVideoHandler,
		ReferrerProfileVideoHandler:  referrerProfileVideoHandler,
		HealthHandler:                healthHandler,
	}
}

// moderationDeps captures the upstream dependencies the central
// moderation orchestrator reaches into: the text moderation
// adapter, the audit + results repositories, and the admin
// notifier that fans out flagged-content alerts.
type moderationDeps struct {
	TextModeration        service.TextModerationService
	ModerationResultsRepo repository.ModerationResultsRepository
	AuditRepo             repository.AuditRepository
	AdminNotifier         service.AdminNotifierService
}

// wireModeration brings up the central text moderation orchestrator.
// One instance fans every pipeline (messaging, reviews, profile
// blocking, jobs, …) through the same analyse → decide → persist →
// audit → notify chain so the policy lives in one place.
//
// The 6 cross-feature SetModerationOrchestrator setter calls
// (messaging, review, auth, profile, job, proposal) STAY in main.go
// — they bridge multiple wire boundaries and would defeat the
// purpose of the split if they lived here.
func wireModeration(deps moderationDeps) *appmoderation.Service {
	// Central text moderation orchestrator. One instance fans every
	// pipeline (messaging, reviews, profile blocking, jobs, …) through
	// the same analyse → decide → persist → audit → notify chain so
	// the policy lives in one place.
	return appmoderation.NewService(appmoderation.Deps{
		TextModeration: deps.TextModeration,
		Results:        deps.ModerationResultsRepo,
		Audit:          deps.AuditRepo,
		AdminNotifier:  deps.AdminNotifier,
	})
}

// kycDeps captures the upstream dependencies the KYC enforcement
// scheduler needs. Re-exported here as a thin wrapper around
// wire_notification's startKYCScheduler so this file stays the
// single landing page for "uploads + messaging + moderation + KYC"
// orchestration referenced by the brief.
type kycDeps struct {
	Ctx           context.Context
	Cfg           *config.Config
	Organizations repository.OrganizationRepository
	Records       repository.PaymentRecordRepository
	Notifications *notifapp.Service
}

// wireKYC starts the KYC enforcement scheduler — sends reminders at
// day 0/3/7/14 for providers with available funds who haven't
// completed Stripe KYC. Delegates to startKYCScheduler in
// wire_notification.go for the actual goroutine launch.
func wireKYC(deps kycDeps) {
	// KYC enforcement scheduler — sends reminders at day 0/3/7/14 for
	// providers with available funds who haven't completed Stripe KYC.
	// See startKYCScheduler in wire_notification.go.
	startKYCScheduler(kycSchedulerDeps{
		Ctx:           deps.Ctx,
		Cfg:           deps.Cfg,
		Organizations: deps.Organizations,
		Records:       deps.Records,
		Notifications: deps.Notifications,
	})
}
