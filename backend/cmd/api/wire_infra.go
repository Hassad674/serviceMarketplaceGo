package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	resendadapter "marketplace-backend/internal/adapter/resend"
	s3adapter "marketplace-backend/internal/adapter/s3"
	"marketplace-backend/internal/adapter/ws"
	"marketplace-backend/internal/config"
	jobdomain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/crypto"

	goredis "github.com/redis/go-redis/v9"
)

// infrastructure bundles every backbone resource the application
// depends on: open *sql.DB and *redis.Client connections, the
// repositories that wrap them, the always-on output adapters
// (storage, email, session, hasher, JWT issuer, refresh-token
// blacklist), the messaging fan-out (presence + stream broadcaster
// + ws hub), and the cookie configuration shared by every auth
// handler.
//
// Holding all of this in a single struct keeps main.go focused on
// orchestration: it builds the infrastructure once, threads the
// fields into the various wireXxx helpers, and never touches the
// underlying connection lifecycles again.
type infrastructure struct {
	DB                         *sql.DB
	Redis                      *goredis.Client
	UserRepo                   *postgres.UserRepository
	ProfileRepo                *postgres.ProfileRepository
	ResetRepo                  *postgres.PasswordResetRepository
	OrganizationRepo           *postgres.OrganizationRepository
	OrganizationMemberRepo     *postgres.OrganizationMemberRepository
	OrganizationInvitationRepo *postgres.OrganizationInvitationRepository
	AuditRepo                  *postgres.AuditRepository
	ModerationResultsRepo      repository.ModerationResultsRepository
	MessageRepo                *postgres.ConversationRepository
	Hasher                     service.HasherService
	TokenSvc                   service.TokenService
	EmailSvc                   service.EmailService
	StorageSvc                 service.StorageService
	SessionSvc                 service.SessionService
	RefreshBlacklistSvc        service.RefreshBlacklistService
	PresenceSvc                service.PresenceService
	StreamBroadcaster          *redisadapter.StreamBroadcaster
	MessagingRateLimiter       *redisadapter.MessagingRateLimiter
	InvitationRateLimiter      *redisadapter.InvitationRateLimiter
	WSHub                      *ws.Hub
	CookieCfg                  *handler.CookieConfig
	SourceID                   string
}

// wireInfrastructure brings up every backbone resource. Returns a
// closer that the caller defers to release the open connections at
// shutdown.
//
// The function fails the process loud (os.Exit(1)) on any unrecoverable
// init error — the application has no business booting without a DB
// connection, a Redis connection, or a valid messaging fan-out. The
// caller does not need to thread a returned error.
//
// Side effects: launches the WebSocket hub goroutine and the stream
// broadcaster's Redis subscriber goroutine. Both are bound to the
// supplied lifecycle context — when ctx cancels, both unwind cleanly.
func wireInfrastructure(ctx context.Context, cfg *config.Config) (infrastructure, func()) {
	// Connect to database
	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// Connect to Redis
	redisClient, err := redisadapter.NewClient(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	slog.Info("redis connected")

	// Initialize repositories (output ports). The organization
	// repository seeds every new org with jobdomain.WeeklyQuota
	// application credits at creation time. The starter value flows
	// through this wiring so the organization package stays free of
	// any cross-feature import — hexagonal wiring, not modular
	// coupling.
	infra := infrastructure{
		DB:                         db,
		Redis:                      redisClient,
		UserRepo:                   postgres.NewUserRepository(db),
		ProfileRepo:                postgres.NewProfileRepository(db),
		ResetRepo:                  postgres.NewPasswordResetRepository(db),
		OrganizationRepo:           postgres.NewOrganizationRepository(db, jobdomain.WeeklyQuota),
		OrganizationMemberRepo:     postgres.NewOrganizationMemberRepository(db),
		OrganizationInvitationRepo: postgres.NewOrganizationInvitationRepository(db),
		// BUG-NEW-04 path 2/8: audit_logs is RLS-protected by migration 125
		// (USING user_id = current_setting('app.current_user_id', true)).
		// Migration 129 added WITH CHECK (true) so INSERTs pass even without
		// context, but the explicit txRunner wrap keeps parity with the rest
		// of the RLS migration and makes the read paths usable when the prod
		// DB role rotates to NOSUPERUSER NOBYPASSRLS.
		AuditRepo:             postgres.NewAuditRepository(db).WithTxRunner(postgres.NewTxRunner(db)),
		ModerationResultsRepo: postgres.NewModerationResultsRepository(db),
		Hasher:                     crypto.NewBcryptHasher(),
		TokenSvc:                   crypto.NewJWTService(cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry),
		EmailSvc:                   resendadapter.NewEmailService(cfg.ResendAPIKey, cfg.ResendDevRedirectTo),
		StorageSvc: s3adapter.NewStorageService(
			cfg.StorageEndpoint,
			cfg.StorageAccessKey,
			cfg.StorageSecretKey,
			cfg.StorageBucket,
			cfg.StoragePublicURL,
			cfg.StorageUseSSL,
		),
		SessionSvc: redisadapter.NewSessionService(redisClient, cfg.SessionTTL),
		// SEC-06: refresh-token rotation. Each /auth/refresh
		// blacklists the JTI of the consumed token; replays are
		// detected and rejected. The blacklist is Redis-backed with
		// per-entry TTLs that match the original token's remaining
		// expiry, so memory use is automatically bounded as old
		// tokens age out.
		RefreshBlacklistSvc:   redisadapter.NewRefreshBlacklistService(redisClient),
		MessagingRateLimiter:  redisadapter.NewMessagingRateLimiter(redisClient),
		InvitationRateLimiter: redisadapter.NewInvitationRateLimiter(redisClient),
	}

	infra.CookieCfg = buildCookieConfig(cfg)

	// Messaging adapters. The TxRunner is wired in here so the
	// conversation repository can install the RLS tenant context
	// (app.current_org_id / app.current_user_id) on the transactions
	// that INSERT into conversations and messages. Both tables are
	// RLS-protected by migration 125 and would otherwise reject
	// INSERTs from a non-superuser DB role with "new row violates
	// row-level security policy". TxRunner is allocated again
	// downstream for other consumers — both calls share the same
	// *sql.DB pool, so this is just a thin wrapper held twice.
	infra.MessageRepo = postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	infra.PresenceSvc = redisadapter.NewPresenceService(redisClient, 45*time.Second)

	// Use HOSTNAME env var (set by Railway/Docker) or fallback to a
	// fixed name. This prevents dead consumer accumulation on
	// redeploys.
	infra.SourceID = os.Getenv("HOSTNAME")
	if infra.SourceID == "" {
		infra.SourceID = "api-main"
	}
	infra.StreamBroadcaster = redisadapter.NewStreamBroadcaster(redisClient, infra.SourceID)

	// WebSocket hub
	infra.WSHub = ws.NewHub()
	hubCtx, hubCancel := context.WithCancel(ctx)
	go infra.WSHub.Run(hubCtx)

	// Start stream subscriber (distributes Redis stream events to
	// local WS clients).
	streamCtx, streamCancel := context.WithCancel(ctx)
	go infra.StreamBroadcaster.Subscribe(streamCtx, func(event redisadapter.StreamEvent) {
		infra.WSHub.HandleStreamEvent(ws.StreamEvent{
			Type:         event.Type,
			RecipientIDs: event.RecipientIDs,
			Payload:      event.Payload,
			SourceID:     event.SourceID,
		})
	})

	closer := func() {
		streamCancel()
		hubCancel()
		_ = redisClient.Close()
		_ = db.Close()
	}
	return infra, closer
}

// buildCookieConfig produces the per-environment session cookie
// settings. In production (cross-origin: Railway backend + Vercel
// frontend), SameSite=None is required for cookies to be sent
// cross-origin. SameSite=None requires Secure=true.
func buildCookieConfig(cfg *config.Config) *handler.CookieConfig {
	sameSite := http.SameSiteLaxMode
	if cfg.IsProduction() {
		sameSite = http.SameSiteNoneMode
	}
	return &handler.CookieConfig{
		Secure:   cfg.CookieSecure,
		Domain:   "",
		MaxAge:   int(cfg.SessionTTL.Seconds()),
		SameSite: sameSite,
	}
}
