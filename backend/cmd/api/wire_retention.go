package main

import (
	"context"
	"database/sql"
	"log/slog"

	"marketplace-backend/internal/adapter/postgres"
	retentionapp "marketplace-backend/internal/app/retention"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/domain/retention"
)

// retentionDeps captures the boot-time dependencies of the retention
// scheduler: the lifecycle context for the goroutine, the config
// (interval + env-derived overrides), and the privileged DB pool.
//
// The retention sweep is a system-actor: it reaches across every
// tenant's RLS-protected rows in audit_logs / messages / notifications
// to apply storage-limitation. We therefore wire it onto the admin
// (BYPASSRLS) pool, mirroring the GDPR scheduler.
type retentionDeps struct {
	Ctx context.Context
	Cfg *config.Config
	DB  *sql.DB
}

// wireRetention brings up the Phase B.1 retention scheduler.
//
// Returns the scheduler instance (so the caller can hold a reference
// for graceful shutdown / testing) plus a cancel func that the caller
// adds to its closeFns. When the cancel fires, the scheduler's Run
// returns and any in-flight Sweep returns ctx.Canceled cleanly.
//
// Boot is fail-OPEN: if NewService rejects the policy slice we log a
// loud error and return a nil scheduler. Retention is critical for
// the privacy contract but never load-bearing for serving requests —
// the API must boot even when one config knob is misset, otherwise
// every operator change carries a downtime risk that swamps the
// privacy benefit.
func wireRetention(deps retentionDeps) *retentionapp.Scheduler {
	repo := postgres.NewRetentionRepository(deps.DB)

	policies := retention.DefaultPolicies(retention.Overrides{})
	svc, err := retentionapp.NewService(repo, policies)
	if err != nil {
		slog.Error("retention scheduler: refusing to boot — invalid policy set", "error", err)
		return nil
	}

	interval := deps.Cfg.RetentionInterval
	if interval <= 0 {
		interval = retentionapp.SchedulerInterval
		if deps.Cfg.IsDevelopment() {
			interval = retentionapp.SchedulerDevInterval
		}
	}

	sch := retentionapp.NewScheduler(svc, interval)
	go sch.Run(deps.Ctx)
	slog.Info("retention scheduler started",
		"interval", interval,
		"policies", len(policies))
	return sch
}
