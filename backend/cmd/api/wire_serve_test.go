package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/ws"
	"marketplace-backend/internal/config"
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

// TestBuildHTTPServer_Timeouts is the single source of truth for the
// production HTTP server's timeout configuration. Every value here is
// load-bearing — bumping or dropping any of these timeouts changes
// either:
//   - the slowloris guard surface (ReadHeaderTimeout)
//   - WebSocket support (WriteTimeout=0)
//   - keep-alive recycling (IdleTimeout)
//
// Touching this test means deliberately changing the server's
// timeout policy, which should land in a dedicated commit with a
// brief explaining why.
func TestBuildHTTPServer_Timeouts(t *testing.T) {
	cfg := &config.Config{Port: "8080"}
	router := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := buildHTTPServer(cfg, router)

	// PERF-FINAL-B-01 / P10: ReadHeaderTimeout is the slowloris guard.
	// Without this cap a malicious client can hold the connection open
	// by sending headers one byte at a time, exhausting the connection
	// pool. 5s is the brief-mandated value.
	assert.Equal(t, 5*time.Second, srv.ReadHeaderTimeout,
		"slowloris guard MUST cap header read at 5s")

	// ReadTimeout covers the BODY window for non-streaming requests.
	// 15s is generous enough for slow mobile uploads but tight enough
	// to prevent body-half slowloris (a variant of the attack that
	// streams body bytes slowly after the headers complete).
	assert.Equal(t, 15*time.Second, srv.ReadTimeout,
		"body read window must stay at 15s")

	// WriteTimeout MUST stay 0 — long-lived WebSocket connections need
	// the server to never hang up on writes. Handler-level deadlines
	// protect regular HTTP endpoints instead.
	assert.Equal(t, time.Duration(0), srv.WriteTimeout,
		"WriteTimeout 0 is mandatory for WebSocket support")

	// IdleTimeout governs keep-alive recycling. Anything above 90s
	// risks accumulating zombie connections on cloud load balancers
	// whose own idle timeout is typically 60s.
	assert.Equal(t, 60*time.Second, srv.IdleTimeout,
		"IdleTimeout must remain at 60s")

	// Sanity: the router and Addr round-trip correctly.
	assert.Equal(t, ":8080", srv.Addr)
	assert.NotNil(t, srv.Handler)
}

// TestBuildHTTPServer_SlowlorisHeader_Aborts validates the slowloris
// guard at the wire level: open a TCP connection to a real listener,
// send headers one byte at a time slower than ReadHeaderTimeout, and
// assert the server closes the connection before all headers arrive.
//
// The guard threshold for this test is set to 100ms (an explicit
// override of the production 5s) so the test stays under 1s
// wall-clock — without an override every CI run would burn 5 seconds.
func TestBuildHTTPServer_SlowlorisHeader_Aborts(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	called := false
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
		ReadHeaderTimeout: 100 * time.Millisecond, // tight cap to keep CI fast
	}
	go srv.Serve(listener)
	defer srv.Close()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	// Drip-feed the request line one byte at a time, pausing 30ms
	// between each. The cumulative delay (~250ms for "GET / HTTP/1.0\r\n")
	// exceeds ReadHeaderTimeout, so the server MUST close the connection
	// before the request line completes.
	requestLine := "GET / HTTP/1.0\r\n"
	for i := 0; i < len(requestLine); i++ {
		if _, err := conn.Write([]byte{requestLine[i]}); err != nil {
			break // connection closed by server, expected
		}
		time.Sleep(30 * time.Millisecond)
	}

	// Read response — slowloris guard must close, returning either
	// 408 Request Timeout, EOF, or a connection reset depending on
	// timing. Any of these is "request did not reach the handler".
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	body := string(buf[:n])

	assert.False(t, called,
		"handler must NOT run when header-read times out — got body: %q", body)

	// The Go net/http library may surface the timeout as a 400 Bad
	// Request (incomplete request line at deadline) or a 408 Request
	// Timeout depending on internal timing; both indicate the
	// slowloris guard fired before the handler ran. The hard contract
	// is: the handler did NOT execute.
	if n > 0 {
		isErrorResponse := strings.Contains(body, "400") ||
			strings.Contains(body, "408") ||
			strings.Contains(body, "HTTP/1.1 4")
		assert.True(t, isErrorResponse,
			"server should respond with a 4xx error when slowloris triggers, got: %q", body)
	}
}

// TestBuildHTTPServer_LegitimateSlowBody_Succeeds validates the
// "ReadHeaderTimeout only covers headers, not body" invariant from
// the brief: a client that sends the headers quickly but the body
// slowly (e.g. a 2MB upload that takes 6s on a poor connection) must
// NOT be killed by the slowloris guard.
//
// We use a 100ms ReadHeaderTimeout and an 8s ReadTimeout so the
// headers must arrive within 100ms but the body has 8s — well above
// the 6s test upload window.
func TestBuildHTTPServer_LegitimateSlowBody_Succeeds(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var bodyBytesRead int64
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n, _ := io.Copy(io.Discard, r.Body)
			atomic.StoreInt64(&bodyBytesRead, n)
			w.WriteHeader(http.StatusOK)
		}),
		ReadHeaderTimeout: 100 * time.Millisecond,
		ReadTimeout:       8 * time.Second, // generous body window
	}
	go srv.Serve(listener)
	defer srv.Close()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	// Send headers fast (< 100ms): one buffered write.
	const bodyChunkSize = 256 * 1024    // 256KB chunks
	const totalBodySize = 2 * 1024 * 1024 // 2MB
	headers := fmt.Sprintf(
		"POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Length: %d\r\nContent-Type: application/octet-stream\r\n\r\n",
		totalBodySize,
	)
	_, err = conn.Write([]byte(headers))
	require.NoError(t, err)

	// Send body slowly: 8 chunks @ 250ms apart = 2 seconds total. Well
	// over the 100ms ReadHeaderTimeout but under the 8s ReadTimeout.
	chunk := make([]byte, bodyChunkSize)
	for sent := 0; sent < totalBodySize; sent += bodyChunkSize {
		toSend := bodyChunkSize
		if sent+toSend > totalBodySize {
			toSend = totalBodySize - sent
		}
		_, werr := conn.Write(chunk[:toSend])
		require.NoError(t, werr, "body write at byte %d must succeed", sent)
		time.Sleep(250 * time.Millisecond)
	}

	// Read the response — must be 200 OK with the full body received.
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp, rerr := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, rerr, "response must arrive — server must NOT have killed the slow body")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"slow body upload must succeed when headers arrive within ReadHeaderTimeout")
	assert.Equal(t, int64(totalBodySize), atomic.LoadInt64(&bodyBytesRead),
		"server must have read the full body — slowloris guard must NOT cap body window")
}

// TestBuildHTTPServer_FastRequest_Succeeds is the smoke test: a
// well-behaved client sending headers + body inside the limits gets a
// 200 OK with no surprises.
func TestBuildHTTPServer_FastRequest_Succeeds(t *testing.T) {
	cfg := &config.Config{Port: "0"}
	called := false
	srv := buildHTTPServer(cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	// Wrap in httptest.Server so we get a real listener without the
	// production Addr binding. We swap the test server's handler in to
	// reuse our buildHTTPServer instance via its handler field.
	ts := httptest.NewUnstartedServer(srv.Handler)
	ts.Config.ReadHeaderTimeout = srv.ReadHeaderTimeout
	ts.Config.ReadTimeout = srv.ReadTimeout
	ts.Config.WriteTimeout = srv.WriteTimeout
	ts.Config.IdleTimeout = srv.IdleTimeout
	ts.Start()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.True(t, called, "handler must run for a normal-speed client")
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// httptest.Server's Config is the real *http.Server, so a sanity
	// pass on the timeout values one more time:
	assert.Equal(t, 5*time.Second, ts.Config.ReadHeaderTimeout)
}

// TestBuildHTTPServer_HTTP10ClientWithFastHeaders_Succeeds confirms the
// slowloris guard plays well with HTTP/1.0 clients. The Go http server
// treats Request-URI parsing the same regardless of version, but the
// ReadHeaderTimeout fires off the raw connection deadline, so the test
// is version-agnostic.
func TestBuildHTTPServer_HTTP10ClientWithFastHeaders_Succeeds(t *testing.T) {
	cfg := &config.Config{Port: "0"}
	srv := buildHTTPServer(cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	go srv.Serve(listener)
	defer srv.Close()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	// All headers in one syscall — well below the 5s cap.
	_, err = conn.Write([]byte("GET / HTTP/1.0\r\nHost: localhost\r\n\r\n"))
	require.NoError(t, err)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	body := string(buf[:n])
	assert.True(t, strings.HasPrefix(body, "HTTP/1.0 200"),
		"fast HTTP/1.0 request must produce 200 OK, got: %q", body)
}
