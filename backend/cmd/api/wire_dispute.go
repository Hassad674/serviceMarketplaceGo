package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	anthropicadapter "marketplace-backend/internal/adapter/anthropic"
	"marketplace-backend/internal/adapter/noop"
	"marketplace-backend/internal/adapter/postgres"
	disputeapp "marketplace-backend/internal/app/dispute"
	"marketplace-backend/internal/app/messaging"
	notifapp "marketplace-backend/internal/app/notification"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/internal/system"
)

// disputeWiring carries the dispute feature's user-facing handlers.
// The scheduler runs in a goroutine started inside wireDispute and
// stops when the context cancels — no field exposed to the caller.
type disputeWiring struct {
	Handler      *handler.DisputeHandler
	AdminHandler *handler.AdminDisputeHandler
}

// disputeDeps captures the upstream services + repos the dispute
// feature reaches into. Bundled into a struct because the dispute
// surface touches enough collaborators (proposals, milestones, users,
// messaging, notifications, payments) that a positional argument list
// would be impossible to read.
type disputeDeps struct {
	Ctx            context.Context
	Cfg            *config.Config
	DB             *sql.DB
	Proposals      repository.ProposalRepository
	Milestones     repository.MilestoneRepository
	Users          repository.UserRepository
	MessageRepo    repository.MessageRepository
	Messaging      *messaging.Service
	Notifications  *notifapp.Service
	Payments       *paymentapp.Service
	ProposalSvcRef *proposalapp.Service // reserved for future cross-feature wiring
}

// wireDispute brings up the dispute feature: repository, optional AI
// analyzer (Anthropic Claude Haiku → noop fallback), service, HTTP +
// admin handlers, and the auto-resolve scheduler that runs in its own
// goroutine on a 1h cadence (1min in development).
//
// The scheduler shares the same disputeCtx as every other long-lived
// background job — passed in by main.go and cancelled at graceful
// shutdown so this function does not have to manage its own lifecycle.
func wireDispute(deps disputeDeps) disputeWiring {
	cfg := deps.Cfg
	db := deps.DB
	ctx := deps.Ctx
	// BUG-NEW-04 path 6/8: disputes is RLS-protected by migration 125
	// (USING client_organization_id = current_org OR provider_organization_id
	// = current_org). The txRunner wrap makes Create / Update /
	// GetByIDForOrg / ListByOrganization pass under prod NOSUPERUSER
	// NOBYPASSRLS. Sub-tables (dispute_evidence, dispute_counter_proposals,
	// dispute_ai_chat_messages) are NOT directly RLS-protected so they
	// stay on the legacy direct-db path; the application-level
	// authorization layer enforces access through the parent dispute.
	disputeRepo := postgres.NewDisputeRepository(db).WithTxRunner(postgres.NewTxRunner(db))
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
		Proposals:     deps.Proposals,
		Milestones:    deps.Milestones,
		Users:         deps.Users,
		MessageRepo:   deps.MessageRepo,
		Messages:      deps.Messaging,
		Notifications: deps.Notifications,
		Payments:      deps.Payments,
		AI:            aiAnalyzer,
	})
	disputeHandler := handler.NewDisputeHandler(disputeSvc)
	adminDisputeHandler := handler.NewAdminDisputeHandler(disputeSvc, disputeRepo, cfg.Env != "production")

	// Dispute scheduler — auto-resolve ghost (7d) + escalate to admin.
	// Escalation logic itself is fully delegated to disputeSvc.escalate
	// so the scheduler and the manual force-escalate endpoint share
	// the same code path (AI summary, system message, notifications
	// all included).
	disputeScheduler := disputeapp.NewScheduler(disputeapp.SchedulerDeps{
		Svc:           disputeSvc,
		Disputes:      disputeRepo,
		Proposals:     deps.Proposals,
		Messages:      deps.Messaging,
		Notifications: deps.Notifications,
		Payments:      deps.Payments,
	})
	disputeInterval := 1 * time.Hour
	if cfg.Env == "development" {
		disputeInterval = 1 * time.Minute
	}
	// The dispute scheduler runs auto-resolve + escalate flows
	// without an authenticated user — tag its goroutine context
	// as a system actor so downstream service / repository calls
	// take the non-tenant-aware code path.
	schedulerCtx := system.WithSystemActor(ctx)
	go disputeScheduler.Run(schedulerCtx, disputeInterval)
	slog.Info("dispute scheduler started", "interval", disputeInterval)

	return disputeWiring{
		Handler:      disputeHandler,
		AdminHandler: adminDisputeHandler,
	}
}
