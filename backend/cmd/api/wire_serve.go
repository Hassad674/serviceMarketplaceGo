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

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
)

// serveDeps captures the resources the HTTP server lifecycle hooks
// reach into: the *config.Config (read at boot for the port + env
// metadata), the wired router, and the upload handler whose Stop
// drain shares the 30s shutdown budget.
type serveDeps struct {
	Cfg           *config.Config
	Router        chi.Router
	UploadCancel  context.CancelFunc
	UploadHandler *handler.UploadHandler
}

// runServer brings up the HTTP server and waits for SIGINT/SIGTERM
// to drive a 30s graceful shutdown. The upload handler's drain runs
// inside the same budget so in-flight RecordUpload goroutines wind
// down their downstream Rekognition / S3 work cleanly.
//
// Behaviour and timeouts are byte-identical to the legacy inline
// block in main.go: ReadTimeout 15s, WriteTimeout 0 (long-lived
// WS), IdleTimeout 60s, Shutdown ctx 30s.
func runServer(deps serveDeps) {
	// Create HTTP server
	// WriteTimeout is 0 to allow long-lived WebSocket connections.
	// Handler-level timeouts protect regular HTTP endpoints instead.
	srv := &http.Server{
		Addr:         ":" + deps.Cfg.Port,
		Handler:      deps.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server starting", "port", deps.Cfg.Port, "env", deps.Cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	// BUG-17: drain in-flight upload goroutines (max 30s budget shared
	// with the HTTP shutdown above). uploadCancel above triggers the
	// individual goroutine's WithCancel so they observe the shutdown
	// signal; Stop() then waits for them to exit cleanly.
	deps.UploadCancel()
	if err := deps.UploadHandler.Stop(ctx); err != nil {
		slog.Warn("upload handler shutdown timed out", "error", err)
	}

	slog.Info("server stopped")
}
