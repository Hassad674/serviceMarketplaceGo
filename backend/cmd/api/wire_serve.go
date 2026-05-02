package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/adapter/ws"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/observability"
)

// serveDeps captures the resources the HTTP server lifecycle hooks
// reach into. Server bootstrap (the *config.Config + the wired
// router) sits on the left; graceful-shutdown hooks (every
// CancelFunc + the WS hub + the OTel shutdown closure) sit on the
// right. The graceful path runs the three sub-budgets defined in
// docs/plans/P11_brief.md:
//
//  1. 15s — HTTP server drain via srv.Shutdown
//  2. 10s — WS hub drain via WSHub.GracefulShutdown (1001 frames)
//  3.  5s — workers + flush via every CancelFunc + UploadHandler.Stop
//           and OtelShutdown
//
// Total budget stays at 30s — the documented soft cap on Kubernetes
// preStop hooks before the kubelet escalates to SIGKILL. Sub-budgets
// are wired with context.WithTimeout so a slow phase does not eat
// the next phase's allowance.
type serveDeps struct {
	Cfg    *config.Config
	Router chi.Router

	// WSHub.GracefulShutdown closes every active WS conn with the
	// 1001 "Going Away" status frame so clients can re-connect to
	// the next instance instead of timing out on a dropped TCP
	// connection. Optional — when nil the WS phase is a no-op.
	WSHub *ws.Hub

	// UploadCancel signals the upload goroutines to wind down their
	// downstream Rekognition / S3 work. Stop blocks until they
	// confirm exit; the cancel is fired ahead of Stop so the work
	// observes the shutdown without polling.
	UploadCancel  context.CancelFunc
	UploadHandler *handler.UploadHandler

	// WorkerCancels is the bag of context.CancelFunc returned by
	// every worker / scheduler wired in main.go (notification,
	// pending events, kyc, dispute, gdpr, media moderation). Each
	// cancel is fired during phase 3 — the workers stop processing
	// their current task and return so the parent goroutine exits.
	// Order does not matter; the cancel call is idempotent.
	WorkerCancels []context.CancelFunc

	// OtelShutdown drains the OpenTelemetry SpanProcessor and the
	// OTLP exporter. Always non-nil — observability.Init returns a
	// no-op closure when tracing is disabled. Invoked at the tail of
	// phase 3 so spans recorded during shutdown are flushed before
	// the process exits.
	OtelShutdown observability.ShutdownFunc
}

// 3-step graceful shutdown sub-budgets. Sums to 30s — the Kubernetes
// preStop default. Tuning: bump httpDrainBudget if long-running
// requests need more time; bump wsDrainBudget for high-density WS
// workloads. workerDrainBudget is the smallest because workers tick
// on relatively short intervals (<= 30s) — anything longer than 5s
// means the worker is wedged and dragging it longer wastes the
// budget on the rest of the shutdown.
const (
	totalShutdownBudget = 30 * time.Second
	httpDrainBudget     = 15 * time.Second
	wsDrainBudget       = 10 * time.Second
	workerDrainBudget   = 5 * time.Second
)

// buildHTTPServer constructs the *http.Server with the project's
// timeout policy. Extracted from runServer so the slowloris guard and
// the existing timeouts can be asserted in a unit test without spinning
// up a real listener.
//
// Timeouts:
//   - ReadHeaderTimeout 5s — PERF-FINAL-B-01 / P10 slowloris guard.
//     Caps the time a client can take to send the request HEADERS;
//     does NOT cover the body. Legitimate slow uploads still work
//     because the body window is governed by ReadTimeout (15s) and,
//     for chunked multipart uploads, by per-handler deadlines.
//     Without this cap a malicious client can hold the connection
//     open indefinitely, exhausting the server's connection pool.
//   - ReadTimeout 15s — overall body window for non-streaming
//     requests.
//   - WriteTimeout 0 — long-lived WebSocket connections.
//     Handler-level timeouts protect regular HTTP endpoints instead.
//   - IdleTimeout 60s — keep-alive idle window.
//
// Any timeout change MUST be reflected in TestBuildHTTPServer_Timeouts
// (cmd/api/wire_serve_test.go) — that test is the single source of
// truth for the server's timeout configuration.
func buildHTTPServer(cfg *config.Config, router http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,  // PERF-FINAL-B-01 / P10 slowloris guard
		ReadTimeout:       15 * time.Second, // body window — does not affect WebSocket reads (handler-level)
		WriteTimeout:      0,                // long-lived WebSocket; per-handler deadlines elsewhere
		IdleTimeout:       60 * time.Second,
	}
}

// runServer brings up the HTTP server and waits for SIGINT/SIGTERM
// to drive the 3-step graceful shutdown documented above. Behaviour
// is otherwise byte-identical to the legacy inline block in main.go:
// ReadHeaderTimeout 5s (slowloris guard), ReadTimeout 15s, WriteTimeout 0
// (long-lived WS), IdleTimeout 60s.
func runServer(deps serveDeps) {
	srv := buildHTTPServer(deps.Cfg, deps.Router)

	go func() {
		slog.Info("server starting", "port", deps.Cfg.Port, "env", deps.Cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("graceful shutdown initiated",
		"total_budget", totalShutdownBudget,
		"http_budget", httpDrainBudget,
		"ws_budget", wsDrainBudget,
		"worker_budget", workerDrainBudget,
	)

	overallCtx, overallCancel := context.WithTimeout(context.Background(), totalShutdownBudget)
	defer overallCancel()

	drainHTTP(overallCtx, srv)
	drainWS(overallCtx, deps.WSHub)
	drainWorkers(overallCtx, deps)

	slog.Info("server stopped")
}

// drainHTTP runs phase 1 of the 3-step shutdown: srv.Shutdown stops
// accepting new connections and waits for in-flight requests to
// complete. Bounded by httpDrainBudget — anything still in flight
// after that is forced closed.
func drainHTTP(parent context.Context, srv *http.Server) {
	ctx, cancel := context.WithTimeout(parent, httpDrainBudget)
	defer cancel()

	start := time.Now()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("http server shutdown error", "error", err, "elapsed", time.Since(start))
		return
	}
	slog.Info("http server drained", "elapsed", time.Since(start))
}

// drainWS runs phase 2: WSHub.GracefulShutdown closes every active
// WebSocket connection with a 1001 "Going Away" frame so clients can
// reconnect to the next instance. nil hub is a no-op (covered when a
// deployment runs with the WS path disabled).
func drainWS(parent context.Context, hub *ws.Hub) {
	if hub == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, wsDrainBudget)
	defer cancel()

	start := time.Now()
	if err := hub.GracefulShutdown(ctx); err != nil {
		slog.Warn("ws graceful shutdown timed out", "error", err, "elapsed", time.Since(start))
		return
	}
	slog.Info("ws connections drained", "elapsed", time.Since(start))
}

// drainWorkers runs phase 3: every worker / scheduler context is
// cancelled, the upload goroutines are stopped, OTel spans are
// flushed. The phase is bounded by workerDrainBudget but each
// individual sub-step is best-effort — a slow flush should not
// prevent the rest of the cleanup from running.
func drainWorkers(parent context.Context, deps serveDeps) {
	ctx, cancel := context.WithTimeout(parent, workerDrainBudget)
	defer cancel()

	start := time.Now()

	// Trip every worker context so the loops observe ctx.Done() and
	// exit at the next tick. Cancels are idempotent so calling them
	// again from the deferred chains in main.go is safe.
	for _, c := range deps.WorkerCancels {
		if c != nil {
			c()
		}
	}

	// Drain the upload goroutines (BUG-17 fix). UploadCancel signals
	// them; Stop blocks until they exit cleanly OR the ctx expires.
	if deps.UploadCancel != nil {
		deps.UploadCancel()
	}
	if deps.UploadHandler != nil {
		if err := deps.UploadHandler.Stop(ctx); err != nil {
			slog.Warn("upload handler shutdown timed out", "error", err)
		}
	}

	// Final OTel flush so spans recorded during shutdown make it
	// onto the wire. observability.Init returns a no-op closure when
	// OTel is disabled so this is always safe to call. The flush
	// must be the last step — anything after it would not be
	// captured.
	if deps.OtelShutdown != nil {
		flushCtx, flushCancel := context.WithTimeout(ctx, 2*time.Second)
		if err := deps.OtelShutdown(flushCtx); err != nil {
			slog.Warn("otel flush failed", "error", err)
		}
		flushCancel()
	}

	slog.Info("workers drained", "elapsed", time.Since(start))
}
