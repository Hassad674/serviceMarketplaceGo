package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log/slog"

	"marketplace-backend/internal/adapter/postgres"
	gdprapp "marketplace-backend/internal/app/gdpr"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// gdprWiring carries the GDPR feature's user-facing handler and the
// scheduler's lifecycle so main.go can wire both with one call.
type gdprWiring struct {
	Handler   *handler.GDPRHandler
	Scheduler *gdprapp.Scheduler // optional — nil when salt is missing
}

// gdprDeps captures the upstream services + repos the GDPR feature
// reaches into. Bundled into a struct because the surface touches a
// dozen collaborators (DB, users, hasher, email, frontend URL, salt)
// that would not fit comfortably as positional arguments.
type gdprDeps struct {
	Ctx         context.Context
	Cfg         *config.Config
	DB          *sql.DB
	Users       repository.UserRepository
	Hasher      service.HasherService
	Email       service.EmailService
}

// wireGDPR brings up the GDPR right-to-erasure + right-to-export
// feature: repository, signer (HS256 with a derived secret separate
// from the access-token JWT), service, handler, and the daily purge
// scheduler.
//
// The scheduler runs in a goroutine started here and stops when the
// shared context cancels — same pattern as wire_dispute.go. In dev
// it ticks every minute so end-to-end smoke tests don't have to wait
// 24 hours.
//
// SAFETY: when GDPR_ANONYMIZATION_SALT is the dev fallback, we still
// boot the scheduler in development (so devs can exercise the flow)
// but the audit row hashes will be predictable. In production
// config.Validate() refuses to boot without a fresh salt — so this
// branch does not ship insecure prod deployments.
func wireGDPR(deps gdprDeps) gdprWiring {
	repo := postgres.NewGDPRRepository(deps.DB)

	// Derive a dedicated signing key for the deletion JWT. We never
	// reuse cfg.JWTSecret directly: a leaked access-token signing
	// key MUST NOT let an attacker forge deletion-confirmation links
	// for arbitrary users. SHA256 of (JWT_SECRET || "gdpr-purpose")
	// gives us a deterministic, distinct key without adding another
	// env var.
	derivedSecret := deriveGDPRSigningSecret(deps.Cfg.JWTSecret)
	signer, err := gdprapp.NewHS256Signer(derivedSecret)
	if err != nil {
		// JWTSecret can never be empty after config.Validate so
		// this branch is for defense in depth only.
		slog.Error("gdpr: signer init failed", "error", err)
		return gdprWiring{}
	}

	svc := gdprapp.NewService(gdprapp.ServiceDeps{
		Repo:        repo,
		Users:       deps.Users,
		Hasher:      deps.Hasher,
		Email:       deps.Email,
		Signer:      signer,
		FrontendURL: deps.Cfg.FrontendURL,
	})
	gdprHandler := handler.NewGDPRHandler(svc)

	// Boot the purge scheduler. The dev interval is short so a fresh
	// checkout exercising the flow doesn't wait a full day; prod
	// keeps the standard 24h cadence.
	interval := gdprapp.SchedulerInterval
	if deps.Cfg.IsDevelopment() {
		interval = gdprapp.SchedulerDevInterval
	}
	sch := gdprapp.NewScheduler(svc, deps.Cfg.GDPRAnonymizationSalt, 100, interval)
	go sch.Run(deps.Ctx)
	slog.Info("gdpr scheduler started", "interval", interval)

	return gdprWiring{
		Handler:   gdprHandler,
		Scheduler: sch,
	}
}

// deriveGDPRSigningSecret hashes the configured JWT secret with a
// fixed purpose tag so the deletion JWT and access-token JWT use
// distinct signing keys without an extra env var. The output is hex
// so HS256 reads it as a 64-char ASCII string.
func deriveGDPRSigningSecret(jwtSecret string) string {
	sum := sha256.Sum256([]byte(jwtSecret + "|gdpr-deletion-confirmation"))
	return hex.EncodeToString(sum[:])
}
