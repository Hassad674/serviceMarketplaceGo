package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHub_GracefulShutdown_NoClients is the trivial path — no
// connections to close. The function returns nil immediately.
func TestHub_GracefulShutdown_NoClients(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := hub.GracefulShutdown(ctx); err != nil {
		t.Errorf("GracefulShutdown returned %v on empty hub, want nil", err)
	}
}

// TestHub_GracefulShutdown_SkipsConnsWithoutSocket verifies that test
// clients fabricated without an underlying *websocket.Conn (the
// established pattern in send_or_drop_test.go and hub_test.go) are
// ignored — GracefulShutdown is robust to nil conn fields.
func TestHub_GracefulShutdown_SkipsConnsWithoutSocket(t *testing.T) {
	hub := NewHub()
	for i := 0; i < 3; i++ {
		client := &Client{
			UserID: uuid.New(),
			Send:   make(chan []byte, 1),
			hub:    hub,
			// conn deliberately nil — fabricated test client
		}
		hub.Register(client)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := hub.GracefulShutdown(ctx); err != nil {
		t.Errorf("GracefulShutdown returned %v, want nil", err)
	}
}

// TestHub_GracefulShutdown_ClosesActiveConns_With1001 verifies the
// real path: two live WS clients connected to a stub handler, the
// hub triggers GracefulShutdown, and both clients receive the 1001
// "Going Away" close frame within the budget.
func TestHub_GracefulShutdown_ClosesActiveConns_With1001(t *testing.T) {
	hub := NewHub()
	go hub.Run(context.Background())

	// Stand-in HTTP handler that upgrades to WS and registers the
	// resulting conn with the hub. We don't need full ServeWS
	// because the test only cares about the close-frame side.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Logf("upgrade failed: %v", err)
			return
		}
		client := &Client{
			UserID: uuid.New(),
			Send:   make(chan []byte, 1),
			hub:    hub,
			conn:   conn,
		}
		hub.Register(client)
	}))
	t.Cleanup(srv.Close)

	wsURL := mustWSURL(t, srv.URL)

	const numClients = 2
	clientConns := make([]*websocket.Conn, 0, numClients)
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
		require.NoError(t, err, "dial #%d", i)
		clientConns = append(clientConns, conn)
	}

	// Wait until all clients have been registered with the hub.
	require.Eventually(t, func() bool {
		return totalRegistered(hub) == numClients
	}, time.Second, 5*time.Millisecond, "all clients should be registered")

	// Each client reads in its own goroutine; we collect the close
	// codes seen on the read path. coder/websocket signals the close
	// status via the error returned from Read.
	var (
		mu         sync.Mutex
		closeCodes []websocket.StatusCode
		readWG     sync.WaitGroup
	)
	for _, c := range clientConns {
		readWG.Add(1)
		go func(conn *websocket.Conn) {
			defer readWG.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for {
				_, _, err := conn.Read(ctx)
				if err != nil {
					mu.Lock()
					closeCodes = append(closeCodes, websocket.CloseStatus(err))
					mu.Unlock()
					return
				}
			}
		}(c)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	require.NoError(t, hub.GracefulShutdown(shutdownCtx))
	elapsed := time.Since(start)

	readWG.Wait()

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, closeCodes, numClients)
	for i, code := range closeCodes {
		assert.Equal(t, websocket.StatusGoingAway, code,
			"client #%d expected 1001 GoingAway, got %d", i, code)
	}
	if elapsed > 5*time.Second {
		t.Errorf("GracefulShutdown took %v, want < 5s", elapsed)
	}
}

// TestHub_GracefulShutdown_RespectsContextTimeout confirms that when
// ctx expires before all conns are closed the function returns
// ctx.Err and the un-touched conns are abandoned. We exercise the
// timeout path by passing an already-cancelled ctx — the loop should
// observe ctx.Err on the very first iteration and return.
func TestHub_GracefulShutdown_RespectsContextTimeout(t *testing.T) {
	hub := NewHub()
	go hub.Run(context.Background())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		client := &Client{
			UserID: uuid.New(),
			Send:   make(chan []byte, 1),
			hub:    hub,
			conn:   conn,
		}
		hub.Register(client)
	}))
	t.Cleanup(srv.Close)

	wsURL := mustWSURL(t, srv.URL)
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	require.NoError(t, err)
	defer conn.CloseNow()

	require.Eventually(t, func() bool { return totalRegistered(hub) == 1 },
		time.Second, 5*time.Millisecond)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel BEFORE call so the very first ctx.Err() check trips

	err = hub.GracefulShutdown(cancelledCtx)
	require.Error(t, err, "expected timeout error from cancelled ctx")
	assert.ErrorIs(t, err, context.Canceled)
}

// totalRegistered counts every connection across all users — used to
// wait for the registrations to settle in async tests.
func totalRegistered(h *Hub) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, set := range h.clients {
		total += len(set)
	}
	return total
}

// mustWSURL turns http://host:port into ws://host:port/ for the
// websocket Dial call.
func mustWSURL(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}
	return strings.TrimRight(u.String(), "/")
}
