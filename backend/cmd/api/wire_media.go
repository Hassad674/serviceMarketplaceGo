package main

import (
	"context"
	"database/sql"
	"log/slog"

	comprehendadapter "marketplace-backend/internal/adapter/comprehend"
	"marketplace-backend/internal/adapter/noop"
	openaiadapter "marketplace-backend/internal/adapter/openai"
	redisadapter "marketplace-backend/internal/adapter/redis"
	rekognitionadapter "marketplace-backend/internal/adapter/rekognition"
	"marketplace-backend/internal/adapter/s3transit"
	sqsadapter "marketplace-backend/internal/adapter/sqs"
	mediaapp "marketplace-backend/internal/app/media"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"

	goredis "github.com/redis/go-redis/v9"
)

// mediaWiring carries the products of the media moderation feature
// initialisation: the media app service, the AdminNotifier (used both
// for media moderation and for the report feature), and the optional
// transit storage + content/text moderation adapters.
type mediaWiring struct {
	MediaSvc       *mediaapp.Service
	AdminNotifier  *redisadapter.AdminNotifierService
	TextModeration service.TextModerationService
}

// mediaDeps captures the upstream dependencies needed to bring up
// media moderation: SQL pool (admin notifier writes through), Redis
// (admin notifier counters), broadcaster (stream events), email +
// session services (used by the media app for moderation
// notifications), and the storage adapter for upload persistence.
type mediaDeps struct {
	Ctx         context.Context
	Cfg         *config.Config
	DB          *sql.DB
	Redis       *goredis.Client
	Broadcaster service.MessageBroadcaster
	Email       service.EmailService
	SessionSvc  service.SessionService
	Storage     service.StorageService
	Users       repository.UserRepository
	Reports     mediaReportSetter // SetAdminNotifier(svc) — typed locally so the report app stays optional
	MediaRepo   repository.MediaRepository
}

// mediaReportSetter is the narrow contract used by the wire helper to
// thread the AdminNotifier into the report feature without exposing
// the report app's full surface.
type mediaReportSetter interface {
	SetAdminNotifier(svc service.AdminNotifierService)
}

// wireMediaModeration brings up content moderation (image/video via
// Rekognition or noop), text moderation (OpenAI / Comprehend / noop),
// the media app service, the AdminNotifier (with its Redis counter
// fan-out), and the SQS worker that finalises asynchronous Rekognition
// jobs. The block fully boots even when none of the AWS / OpenAI
// integrations are configured: every path falls back to the noop
// adapter and the feature stays optional.
func wireMediaModeration(deps mediaDeps) mediaWiring {
	cfg := deps.Cfg

	moderationSvc := buildContentModeration(cfg)
	transitStorage := buildTransitStorage(cfg)

	mediaSvc := mediaapp.NewService(mediaapp.ServiceDeps{
		Media:               deps.MediaRepo,
		Users:               deps.Users,
		Storage:             deps.Storage,
		Transit:             transitStorage,
		Moderation:          moderationSvc,
		Email:               deps.Email,
		SessionSvc:          deps.SessionSvc,
		Broadcaster:         deps.Broadcaster,
		FlagThreshold:       cfg.RekognitionThreshold,
		AutoRejectThreshold: cfg.RekognitionAutoRejectThreshold,
	})

	textModerationSvc := buildTextModeration(cfg)
	startSQSWorker(deps.Ctx, cfg, transitStorage, mediaSvc)

	// Admin notification counters (per-admin Redis counters)
	adminNotifier := redisadapter.NewAdminNotifierService(deps.Redis, deps.DB, deps.Broadcaster)
	if deps.Reports != nil {
		deps.Reports.SetAdminNotifier(adminNotifier)
	}
	mediaSvc.SetAdminNotifier(adminNotifier)
	slog.Info("admin notification counters enabled")

	return mediaWiring{
		MediaSvc:       mediaSvc,
		AdminNotifier:  adminNotifier,
		TextModeration: textModerationSvc,
	}
}

func buildContentModeration(cfg *config.Config) service.ContentModerationService {
	if !cfg.RekognitionConfigured() {
		slog.Info("content moderation disabled (noop)")
		return noop.NewModerationService()
	}
	rekSvc, err := rekognitionadapter.NewModerationService(rekognitionadapter.ModerationServiceDeps{
		Region:      cfg.RekognitionRegion,
		Threshold:   cfg.RekognitionThreshold,
		SNSTopicARN: cfg.SNSTopicARN,
		RoleARN:     cfg.RekognitionRoleARN,
	})
	if err != nil {
		slog.Error("failed to init Rekognition moderation service", "error", err)
		return noop.NewModerationService()
	}
	slog.Info("content moderation enabled (AWS Rekognition)")
	return rekSvc
}

func buildTransitStorage(cfg *config.Config) service.TransitStorageService {
	if !cfg.VideoModerationConfigured() {
		return nil
	}
	transit, err := s3transit.NewTransitStorage(cfg.RekognitionRegion, cfg.S3ModerationBucket)
	if err != nil {
		slog.Error("failed to init S3 transit storage", "error", err)
		return nil
	}
	slog.Info("video moderation transit storage enabled", "bucket", cfg.S3ModerationBucket)
	return transit
}

func buildTextModeration(cfg *config.Config) service.TextModerationService {
	// Text moderation — selected by TEXT_MODERATION_PROVIDER env var.
	// Defaults to OpenAI because it is free, multilingual (FR-native),
	// and returns the fine-grained category scores that
	// domain/moderation uses for zero-tolerance rules.
	switch cfg.TextModerationProviderOrDefault() {
	case "openai":
		slog.Info("text moderation enabled (OpenAI omni-moderation)")
		return openaiadapter.NewTextModerationService(cfg.OpenAIAPIKey)
	case "comprehend":
		comprehendSvc, err := comprehendadapter.NewTextModerationService(cfg.RekognitionRegion)
		if err != nil {
			slog.Error("failed to init Comprehend text moderation, falling back to noop", "error", err)
			return noop.NewTextModerationService()
		}
		slog.Info("text moderation enabled (AWS Comprehend)")
		return comprehendSvc
	default:
		slog.Info("text moderation disabled (noop)")
		return noop.NewTextModerationService()
	}
}

func startSQSWorker(ctx context.Context, cfg *config.Config, transitStorage service.TransitStorageService, finalizer sqsadapter.JobFinalizer) {
	// SQS worker polls Rekognition completion notifications and
	// finalizes jobs. Only spins up when video moderation is
	// configured and the transit storage came up cleanly.
	if !cfg.VideoModerationConfigured() || transitStorage == nil {
		return
	}
	worker, err := sqsadapter.NewWorker(sqsadapter.WorkerDeps{
		Region:    cfg.RekognitionRegion,
		QueueURL:  cfg.SQSQueueURL,
		Finalizer: finalizer,
	})
	if err != nil {
		slog.Error("failed to init SQS worker", "error", err)
		return
	}
	go worker.Start(ctx)
}
