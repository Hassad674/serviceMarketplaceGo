package main

import (
	"context"
	"database/sql"
	"log/slog"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/adapter/r2"
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

	// B.2: optionally install the R2 cold-storage writer. The B.2
	// sweep dumps audit_logs_archive rows to R2 once they are older
	// than the cold-tier cutoff. Wiring is fail-OPEN: if the bucket
	// env var is empty (e.g., dev), the writer is left nil and the
	// archive_to_r2 sweep skips itself with a one-line WARN per tick
	// — deliberately noisy so the operator notices a missing config
	// in staging/prod.
	if writer := buildAuditArchiveWriter(deps.Cfg); writer != nil {
		repo = repo.WithAuditArchiveWriter(writer)
		slog.Info("retention: audit cold-storage writer wired",
			"bucket", deps.Cfg.AuditColdStorageBucket)
	} else {
		slog.Warn("retention: audit cold-storage writer NOT wired — STORAGE_AUDIT_COLD_BUCKET unset; archive_to_r2 policy will skip")
	}

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

// buildAuditArchiveWriter constructs the R2 writer used by the B.2
// cold-tier sweep. Returns nil (skip wiring) when the operator has
// not configured a cold-storage bucket — keeps dev / local boots
// from blocking on a missing env var.
//
// The bucket override defaults to the existing storage bucket so
// deployments that want to reuse one bucket and segregate by prefix
// (audit-cold/<year>/<month>/...) need only set the feature flag.
// When STORAGE_AUDIT_COLD_BUCKET is set explicitly, that bucket
// wins — useful for separating retention costs from user uploads.
func buildAuditArchiveWriter(cfg *config.Config) *r2.AuditArchiveWriter {
	bucket := cfg.AuditColdStorageBucket
	if bucket == "" {
		return nil
	}
	if cfg.StorageEndpoint == "" {
		slog.Warn("retention: cold-storage bucket configured but STORAGE_ENDPOINT empty; skipping wiring")
		return nil
	}
	w, err := r2.NewAuditArchiveWriter(r2.Config{
		Endpoint:  cfg.StorageEndpoint,
		AccessKey: cfg.StorageAccessKey,
		SecretKey: cfg.StorageSecretKey,
		Bucket:    bucket,
		UseSSL:    cfg.StorageUseSSL,
	})
	if err != nil {
		slog.Error("retention: failed to build audit cold-storage writer", "error", err)
		return nil
	}
	return w
}
