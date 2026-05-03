package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	stripeadapter "marketplace-backend/internal/adapter/stripe"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler/middleware"
)

// main is the API entry point. It performs three boot steps and
// nothing else — every wireXxx call lives in bootstrap.go so this
// file stays a high-signal orchestration script:
//
//  1. Load config + install the redaction-aware structured logger.
//  2. Call bootstrap(...) which assembles every adapter, app service,
//     handler, and worker into a single *App ready to serve traffic.
//  3. Hand the App to runServer, which owns the HTTP listener and
//     drives the 3-step graceful shutdown on SIGTERM (WS drain →
//     in-flight upload drain → workers + OTel flush).
func main() {
	// 1. Configuration + structured logger.
	cfg := config.Load()

	logLevel := slog.LevelInfo
	if cfg.IsDevelopment() {
		logLevel = slog.LevelDebug
	}
	// SlogReplaceAttr is wired globally so every emitted attribute
	// passes through the redaction pipeline (SEC-FINAL-13). A future
	// `slog.Info(..., "headers", r.Header)` would otherwise leak Bearer
	// tokens — the handler boundary now scrubs them before any sink
	// (stdout / file / OTLP) sees the payload.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       logLevel,
		ReplaceAttr: middleware.SlogReplaceAttr,
	}))
	slog.SetDefault(logger)

	// Fail-fast in production when secrets are missing or use the
	// open-source fallbacks. In development this only prints loud
	// warnings — see config.Validate for the policy.
	if err := cfg.Validate(); err != nil {
		slog.Error("config validation failed", "error", err)
		os.Exit(1)
	}

	// Install OTel-wrapped HTTP transports on third-party SDKs that
	// expose package-global clients. This must happen BEFORE any SDK
	// instance is created so the wrap is in place from the first
	// outbound call. bootstrap() handles the SDK instances; this
	// global needs to land before bootstrap fires the first wire.
	stripeadapter.InstallOTelBackends()

	// 2. Bootstrap — assembles the App.
	bootstrapCtx, bootstrapCancel := context.WithCancel(context.Background())
	defer bootstrapCancel()
	app, err := bootstrap(bootstrapCtx, cfg)
	if err != nil {
		slog.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}
	// otelShutdown is invoked by runServer's graceful-shutdown 3-step
	// (phase 3 — workers + flush). The deferred fallback below covers
	// abnormal exits where runServer never returns (panic, os.Exit
	// upstream) so spans are flushed even when the SIGTERM path is
	// not taken. The OTel SDK's Shutdown is idempotent — calling it
	// from both sites is safe.
	defer func() {
		if app == nil || app.OtelShutdown == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := app.OtelShutdown(ctx); err != nil {
			slog.Debug("otel deferred shutdown noop or already-flushed", "error", err)
		}
	}()

	// 3. Serve. runServer owns the HTTP listener and the SIGTERM
	// handler that drives the 3-step graceful shutdown. Every cancel
	// the App captured during bootstrap (workers, in-flight uploads,
	// infra teardown) is threaded through serveDeps so the SIGTERM
	// path drains them in phase 3 before the process exits.
	runServer(serveDeps{
		Cfg:           cfg,
		Router:        app.Router,
		WSHub:         app.WSHub,
		UploadCancel:  app.UploadCancel,
		UploadHandler: app.UploadHandler,
		WorkerCancels: app.WorkerCancels,
		OtelShutdown:  app.OtelShutdown,
	})
}
