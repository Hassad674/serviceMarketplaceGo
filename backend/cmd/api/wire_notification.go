package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"marketplace-backend/internal/adapter/fcm"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	kycapp "marketplace-backend/internal/app/kyc"
	notifapp "marketplace-backend/internal/app/notification"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"

	goredis "github.com/redis/go-redis/v9"
)

// notificationWiring carries the notification feature's products. The
// worker runs in a goroutine started inside wireNotificationFeature
// and stops when the supplied context cancels.
type notificationWiring struct {
	Service *notifapp.Service
	Handler *handler.NotificationHandler
	Push    service.PushService // nil when FCM is not configured
}

// notificationDeps captures the upstream dependencies the
// notification feature needs.
type notificationDeps struct {
	Ctx           context.Context
	Cfg           *config.Config
	DB            *sql.DB
	Redis         *goredis.Client
	SourceID      string
	Email         service.EmailService
	Users         repository.UserRepository
	Presence      service.PresenceService
	Broadcaster   service.MessageBroadcaster
}

// wireNotificationFeature brings up the notification feature: optional
// FCM push service, persistent repo + Redis-backed job queue, the
// notif app service, and the worker that processes push + email
// deliveries asynchronously.
//
// BUG-16: the worker pool now spawns N parallel processors so a single
// slow delivery cannot stall the queue. Concurrency comes from the
// config — defaults to 5 when unset / zero.
func wireNotificationFeature(deps notificationDeps) notificationWiring {
	pushSvc := buildPushService(deps.Cfg)

	// BUG-NEW-04 path 1/8: notifications is RLS-protected by migration
	// 125 with the policy
	//   USING (user_id = current_setting('app.current_user_id', true)::uuid)
	// Production rotates the application DB role to NOSUPERUSER
	// NOBYPASSRLS — without the txRunner wrap, INSERTs into notifications
	// are rejected and SELECT/UPDATE/DELETE silently return 0 rows.
	notifRepo := postgres.NewNotificationRepository(deps.DB).WithTxRunner(postgres.NewTxRunner(deps.DB))
	notifQueue := redisadapter.NewNotificationJobQueue(deps.Redis, deps.SourceID)
	if err := notifQueue.EnsureGroup(context.Background()); err != nil {
		slog.Error("failed to create notification job group", "error", err)
	}
	notifSvc := notifapp.NewService(notifapp.ServiceDeps{
		Notifications: notifRepo,
		Presence:      deps.Presence,
		Broadcaster:   deps.Broadcaster,
		Push:          pushSvc, // nil if FCM not configured
		Email:         deps.Email,
		Users:         deps.Users,
		Queue:         notifQueue,
	})

	// Start notification delivery worker (processes push + email async).
	notifWorker := notifapp.NewWorker(notifapp.WorkerDeps{
		Queue:    notifQueue,
		Presence: deps.Presence,
		Push:     pushSvc,
		Email:    deps.Email,
		Users:    deps.Users,
		Notifs:   notifRepo,
	}).WithConfig(notifapp.WorkerConfig{
		Concurrency: deps.Cfg.NotificationWorkerConcurrency,
	})
	go notifWorker.Run(deps.Ctx)
	slog.Info("notification feature enabled")

	return notificationWiring{
		Service: notifSvc,
		Handler: handler.NewNotificationHandler(notifSvc),
		Push:    pushSvc,
	}
}

// buildPushService selects the FCM push adapter when configured, or
// returns nil otherwise. Startup logs at INFO in both the "disabled"
// and the "init failed" paths so operators can tell the app booted
// without push without seeing scary ERRORs in their console — only
// truly unexpected failures would ever need ERROR, and none of the
// current init paths qualify.
func buildPushService(cfg *config.Config) service.PushService {
	if !cfg.FCMConfigured() {
		slog.Info("push notification service disabled (FCM_CREDENTIALS_PATH not set)")
		return nil
	}
	fcmSvc, err := fcm.NewPushService(cfg.FCMCredentialsPath)
	if err != nil {
		slog.Info("push notification service disabled (FCM init failed)",
			"error", err)
		return nil
	}
	slog.Info("push notification service enabled (FCM)")
	return fcmSvc
}

// kycSchedulerDeps captures the dependencies of the KYC enforcement
// scheduler that fans payout-blocked reminders to providers who
// haven't completed Stripe KYC yet.
type kycSchedulerDeps struct {
	Ctx           context.Context
	Cfg           *config.Config
	Organizations repository.OrganizationRepository
	Records       repository.PaymentRecordRepository
	Notifications *notifapp.Service
}

// startKYCScheduler launches the day-0/3/7/14 reminder scheduler. The
// goroutine stops when ctx is cancelled along with the rest of the
// background workers. Cadence is 1h in production and 1min in dev so
// the loop is testable without time travel.
func startKYCScheduler(deps kycSchedulerDeps) {
	scheduler := kycapp.NewScheduler(kycapp.SchedulerDeps{
		Organizations: deps.Organizations,
		Records:       deps.Records,
		Notifications: deps.Notifications,
	})
	interval := 1 * time.Hour
	if deps.Cfg.Env == "development" {
		interval = 1 * time.Minute
	}
	go scheduler.Run(deps.Ctx, interval)
	slog.Info("kyc enforcement scheduler started", "interval", interval)
}
