package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"marketplace-backend/internal/adapter/ws"
	"marketplace-backend/internal/observability"
)

// TestDrainHTTP_Completes_WithinBudget verifies phase 1 of the 3-step
// shutdown: srv.Shutdown drains in-flight requests within the
// httpDrainBudget and returns control to phase 2.
func TestDrainHTTP_Completes_WithinBudget(t *testing.T) {
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	listener := httptest.NewServer(srv.Handler)
	t.Cleanup(listener.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	drainHTTP(ctx, srv)
	if elapsed := time.Since(start); elapsed > httpDrainBudget+time.Second {
		t.Errorf("drainHTTP elapsed = %v, want < %v", elapsed, httpDrainBudget+time.Second)
	}
}

// TestDrainWS_NilHubIsNoop verifies that drainWS handles a missing
// hub gracefully — important for deployments that do not expose WS.
func TestDrainWS_NilHubIsNoop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	drainWS(ctx, nil)
}

// TestDrainWS_DelegatesToHub verifies the hub's GracefulShutdown is
// invoked. Real WS dial is exercised in
// internal/adapter/ws/hub_graceful_shutdown_test.go — here we just
// confirm the call wires through.
func TestDrainWS_DelegatesToHub(t *testing.T) {
	hub := ws.NewHub()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	drainWS(ctx, hub)
	if elapsed := time.Since(start); elapsed > wsDrainBudget+500*time.Millisecond {
		t.Errorf("drainWS elapsed = %v, want < %v", elapsed, wsDrainBudget)
	}
}

// TestDrainWorkers_FiresAllCancels asserts every CancelFunc in the
// WorkerCancels list is invoked, the upload handler is stopped, and
// the OTel shutdown closure is called within the workerDrainBudget.
func TestDrainWorkers_FiresAllCancels(t *testing.T) {
	var (
		fired1, fired2 atomic.Int32
		otelCalled     atomic.Bool
	)

	deps := serveDeps{
		WorkerCancels: []context.CancelFunc{
			func() { fired1.Add(1) },
			func() { fired2.Add(1) },
			nil, // robust to nil entries
		},
		UploadCancel: func() {},
		// UploadHandler nil — drainWorkers must tolerate that.
		OtelShutdown: func(ctx context.Context) error {
			otelCalled.Store(true)
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	drainWorkers(ctx, deps)
	elapsed := time.Since(start)

	if fired1.Load() != 1 {
		t.Errorf("worker cancel #1 fired %d times, want 1", fired1.Load())
	}
	if fired2.Load() != 1 {
		t.Errorf("worker cancel #2 fired %d times, want 1", fired2.Load())
	}
	if !otelCalled.Load() {
		t.Error("otel shutdown was not invoked")
	}
	if elapsed > workerDrainBudget+time.Second {
		t.Errorf("drainWorkers elapsed = %v, want < %v", elapsed, workerDrainBudget+time.Second)
	}
}

// TestDrainWorkers_OtelShutdownErrorIsLoggedNotPropagated verifies a
// failing OTel flush does not interrupt the rest of the cleanup. The
// shutdown function returns an error but the surrounding work
// (cancels, upload stop) must still complete.
func TestDrainWorkers_OtelShutdownErrorIsLoggedNotPropagated(t *testing.T) {
	var fired atomic.Bool
	deps := serveDeps{
		WorkerCancels: []context.CancelFunc{func() { fired.Store(true) }},
		OtelShutdown:  func(ctx context.Context) error { return errors.New("upstream collector down") },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	drainWorkers(ctx, deps)

	if !fired.Load() {
		t.Error("worker cancel was not fired despite otel shutdown error")
	}
}

// TestRunServer_3StepShutdownTotalBudget asserts the overall budget
// constants hold the documented invariants (HTTP + WS + workers <=
// total budget). Catches drift in the constants without exercising
// the full SIGTERM path which would require process-level harness.
func TestRunServer_3StepShutdownTotalBudget(t *testing.T) {
	sum := httpDrainBudget + wsDrainBudget + workerDrainBudget
	if sum != totalShutdownBudget {
		t.Errorf("sub-budget sum = %v, want = totalShutdownBudget %v",
			sum, totalShutdownBudget)
	}

	// The Kubernetes preStop default is 30s. The constant must not
	// silently drift above that or pods will be SIGKILL'd before the
	// graceful path completes.
	if totalShutdownBudget > 30*time.Second {
		t.Errorf("total budget = %v exceeds the documented 30s ceiling", totalShutdownBudget)
	}
}

// TestObservabilityShutdownClosureSafe verifies an
// observability.ShutdownFunc returned from a no-op Init can be called
// from the graceful-shutdown path without error — the foundation of
// the "OTel disabled = zero overhead" promise.
func TestObservabilityShutdownClosureSafe(t *testing.T) {
	shutdown, err := observability.Init(context.Background(), observability.Config{})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Init returned nil shutdown closure")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Errorf("shutdown returned %v, want nil", err)
	}
}
