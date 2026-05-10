package posthog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	portservice "marketplace-backend/internal/port/service"
)

// captureSink is a tiny httptest.Server that intercepts the PostHog
// SDK's HTTP traffic so the adapter test asserts on what would have
// gone over the wire to https://eu.posthog.com without needing a
// real network. Mirrors the pattern adapter/openai uses.
type captureSink struct {
	mu    sync.Mutex
	calls []map[string]any
	srv   *httptest.Server
}

func newCaptureSink(t *testing.T) *captureSink {
	t.Helper()
	sink := &captureSink{}
	sink.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		sink.mu.Lock()
		sink.calls = append(sink.calls, payload)
		sink.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(sink.srv.Close)
	return sink
}

func (s *captureSink) snapshot() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, len(s.calls))
	copy(out, s.calls)
	return out
}

func newTestService(t *testing.T) (*AnalyticsService, *captureSink) {
	t.Helper()
	sink := newCaptureSink(t)
	svc, err := NewAnalyticsService(Config{
		ProjectKey: "phc_test_key",
		Endpoint:   sink.srv.URL,
		Verbose:    false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })
	return svc, sink
}

func TestNewAnalyticsService_RejectsEmptyKey(t *testing.T) {
	_, err := NewAnalyticsService(Config{ProjectKey: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project key")
}

func TestAnalyticsService_Capture_ShipsEvent(t *testing.T) {
	svc, sink := newTestService(t)

	svc.Capture(context.Background(), portservice.AnalyticsEvent{
		DistinctID: "user-123",
		EventName:  "smoke_test.backend",
		Properties: map[string]any{"source": "unit-test"},
		GroupKey:   "org-42",
	})

	require.NoError(t, svc.Close())

	calls := sink.snapshot()
	require.NotEmpty(t, calls, "PostHog SDK must have shipped at least one batch")
	// The SDK posts to /batch with a `batch` array of events.
	batch, ok := calls[0]["batch"].([]any)
	require.True(t, ok, "expected batch payload, got %T", calls[0]["batch"])
	require.NotEmpty(t, batch)
	first, ok := batch[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "smoke_test.backend", first["event"])
	assert.Equal(t, "user-123", first["distinct_id"])
	props, ok := first["properties"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "unit-test", props["source"])
	groups, ok := props["$groups"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "org-42", groups["organization"])
}

func TestAnalyticsService_Capture_DropsBlank(t *testing.T) {
	svc, sink := newTestService(t)

	// Missing distinct id — should silently drop.
	svc.Capture(context.Background(), portservice.AnalyticsEvent{
		EventName: "should.not.ship",
	})
	// Missing event name — also dropped.
	svc.Capture(context.Background(), portservice.AnalyticsEvent{
		DistinctID: "user-1",
	})

	require.NoError(t, svc.Close())

	calls := sink.snapshot()
	for _, c := range calls {
		batch, ok := c["batch"].([]any)
		if !ok {
			continue
		}
		for _, raw := range batch {
			evt, _ := raw.(map[string]any)
			assert.NotEqual(t, "should.not.ship", evt["event"], "blank events must not ship")
		}
	}
}

func TestAnalyticsService_Identify_ShipsProfileUpdate(t *testing.T) {
	svc, sink := newTestService(t)

	svc.Identify(context.Background(), "user-7", map[string]any{
		"email": "alice@example.com",
		"role":  "agency",
	})
	require.NoError(t, svc.Close())

	calls := sink.snapshot()
	found := false
	for _, c := range calls {
		batch, _ := c["batch"].([]any)
		for _, raw := range batch {
			evt, _ := raw.(map[string]any)
			if evt["event"] == "$identify" && evt["distinct_id"] == "user-7" {
				found = true
				props, _ := evt["$set"].(map[string]any)
				if props == nil {
					props, _ = evt["properties"].(map[string]any)
					if set, ok := props["$set"].(map[string]any); ok {
						props = set
					}
				}
				assert.Equal(t, "alice@example.com", props["email"])
			}
		}
	}
	assert.True(t, found, "expected an $identify event for user-7")
}

func TestAnalyticsService_GroupIdentify_ShipsGroupUpdate(t *testing.T) {
	svc, sink := newTestService(t)

	svc.GroupIdentify(context.Background(), "organization", "org-42", map[string]any{
		"plan": "premium",
		"type": "agency",
	})
	require.NoError(t, svc.Close())

	calls := sink.snapshot()
	found := false
	for _, c := range calls {
		batch, _ := c["batch"].([]any)
		for _, raw := range batch {
			evt, _ := raw.(map[string]any)
			if evt["event"] == "$groupidentify" {
				found = true
			}
		}
	}
	assert.True(t, found, "expected a $groupidentify event")
}

func TestAnalyticsService_Capture_FailOpen_OnNetworkError(t *testing.T) {
	// Point at a closed endpoint so the SDK's HTTP calls fail.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()

	svc, err := NewAnalyticsService(Config{
		ProjectKey: "phc_test_key",
		Endpoint:   srv.URL,
	})
	require.NoError(t, err)
	defer func() { _ = svc.Close() }()

	// Should NOT panic, NOT block, NOT return error — fail-open contract.
	done := make(chan struct{})
	go func() {
		svc.Capture(context.Background(), portservice.AnalyticsEvent{
			DistinctID: "user-1",
			EventName:  "smoke.network_failure",
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Capture blocked the caller — analytics must be fully async")
	}
}
