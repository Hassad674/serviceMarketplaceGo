package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// fakeIdempotencyCache is an in-memory IdempotencyCache used by the
// tests. The set methods emulate SETNX semantics — a second call for
// the same key returns (false, nil) without overwriting. A trace of
// every Get/Set is captured so tests can assert call counts.
type fakeIdempotencyCache struct {
	mu      sync.Mutex
	store   map[string]IdempotentResponse
	expires map[string]time.Time

	getErr error
	setErr error

	getCalls atomic.Int32
	setCalls atomic.Int32
}

func newFakeCache() *fakeIdempotencyCache {
	return &fakeIdempotencyCache{
		store:   make(map[string]IdempotentResponse),
		expires: make(map[string]time.Time),
	}
}

func (c *fakeIdempotencyCache) Get(_ context.Context, key string) (*IdempotentResponse, error) {
	c.getCalls.Add(1)
	if c.getErr != nil {
		return nil, c.getErr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if exp, ok := c.expires[key]; ok && time.Now().After(exp) {
		delete(c.store, key)
		delete(c.expires, key)
		return nil, nil
	}
	v, ok := c.store[key]
	if !ok {
		return nil, nil
	}
	return &v, nil
}

func (c *fakeIdempotencyCache) Set(_ context.Context, key string, resp IdempotentResponse, ttl time.Duration) (bool, error) {
	c.setCalls.Add(1)
	if c.setErr != nil {
		return false, c.setErr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// SETNX semantics: the second concurrent claim must lose.
	if _, exists := c.store[key]; exists {
		return false, nil
	}
	c.store[key] = resp
	if ttl > 0 {
		c.expires[key] = time.Now().Add(ttl)
	}
	return true, nil
}

// successHandler counts invocations and writes a deterministic body
// so the test can assert "handler ran exactly once" in replay scenarios.
func successHandler(invocations *atomic.Int32) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := invocations.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", "/created/42")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         42,
			"invocation": count,
		})
	})
}

// withUserContext stamps a user id on the request so the cache key
// scoping logic can be exercised end-to-end.
func withUserContext(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// 1. No header → no caching, handler always runs.
// ---------------------------------------------------------------------------

func TestIdempotency_NoHeader_BypassesCache(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("iter %d: status %d, want 201", i, rec.Code)
		}
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("handler invocations: got %d, want 3 (no key → no cache)", got)
	}
	if got := cache.getCalls.Load(); got != 0 {
		t.Fatalf("Get should not be called when no header: got %d", got)
	}
	if got := cache.setCalls.Load(); got != 0 {
		t.Fatalf("Set should not be called when no header: got %d", got)
	}
}

// ---------------------------------------------------------------------------
// 2. Header + cache miss → handler runs once, response cached.
// ---------------------------------------------------------------------------

func TestIdempotency_FirstCall_PersistsResponse(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	req := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
	req.Header.Set(IdempotencyHeader, "first-key")
	req = withUserContext(req, uuid.New())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler invocations: got %d, want 1", got)
	}
	if got := cache.setCalls.Load(); got != 1 {
		t.Fatalf("Set calls: got %d, want 1", got)
	}
	if rec.Header().Get(IdempotentReplayedHeader) != "" {
		t.Fatalf("first call must not have Idempotent-Replayed header")
	}
}

// ---------------------------------------------------------------------------
// 3. Header + cache hit → handler skipped, replay served.
// ---------------------------------------------------------------------------

func TestIdempotency_CacheHit_ReplaysWithoutInvokingHandler(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	uid := uuid.New()

	// First call seeds the cache.
	req1 := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
	req1.Header.Set(IdempotencyHeader, "stable-key-123")
	req1 = withUserContext(req1, uid)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	// Second call must hit the cache.
	req2 := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
	req2.Header.Set(IdempotencyHeader, "stable-key-123")
	req2 = withUserContext(req2, uid)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if got := calls.Load(); got != 1 {
		t.Fatalf("handler invocations: got %d, want 1 (replay must not run handler)", got)
	}
	if rec2.Code != http.StatusCreated {
		t.Fatalf("replay status: got %d, want 201", rec2.Code)
	}
	if rec2.Header().Get(IdempotentReplayedHeader) != "true" {
		t.Fatalf("replay must set Idempotent-Replayed: true, got %q",
			rec2.Header().Get(IdempotentReplayedHeader))
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Fatalf("replay body must equal original: %q vs %q",
			rec1.Body.String(), rec2.Body.String())
	}
	// Whitelisted header (Location) must round-trip on replay.
	if rec2.Header().Get("Location") == "" {
		t.Fatalf("Location header must be replayed (got empty)")
	}
}

// ---------------------------------------------------------------------------
// 4. Concurrent first-execute race → SETNX winner, only one persisted.
// ---------------------------------------------------------------------------

func TestIdempotency_ConcurrentRequests_OnePersists(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	uid := uuid.New()
	const N = 8
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/v1/jobs", strings.NewReader(`{}`))
			req.Header.Set(IdempotencyHeader, "race-key")
			req = withUserContext(req, uid)
			h.ServeHTTP(httptest.NewRecorder(), req)
		}()
	}
	wg.Wait()

	// At least one Set must have succeeded; no replays were possible
	// because all Get calls run before the first Set lands. The
	// invariant we enforce is: Set was called AT LEAST once but
	// only one writer wins thanks to SETNX in the cache fake.
	if got := cache.setCalls.Load(); got < 1 {
		t.Fatalf("expected at least 1 Set call, got %d", got)
	}
	// Sanity: only one cache entry for this key.
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if got := len(cache.store); got != 1 {
		t.Fatalf("cache must hold exactly 1 entry after race, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// 5. TTL expiry → cache miss after window passes.
// ---------------------------------------------------------------------------

func TestIdempotency_TTLExpiry_RecomputesAfterWindow(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := IdempotencyWithTTL(cache, 10*time.Millisecond)
	h := mw(successHandler(&calls))

	uid := uuid.New()
	req1 := httptest.NewRequest("POST", "/api/v1/disputes", strings.NewReader(`{}`))
	req1.Header.Set(IdempotencyHeader, "ttl-key")
	req1 = withUserContext(req1, uid)
	h.ServeHTTP(httptest.NewRecorder(), req1)

	// Sleep just past the TTL.
	time.Sleep(20 * time.Millisecond)

	req2 := httptest.NewRequest("POST", "/api/v1/disputes", strings.NewReader(`{}`))
	req2.Header.Set(IdempotencyHeader, "ttl-key")
	req2 = withUserContext(req2, uid)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if got := calls.Load(); got != 2 {
		t.Fatalf("handler must run twice after TTL expiry, got %d", got)
	}
	if rec2.Header().Get(IdempotentReplayedHeader) != "" {
		t.Fatalf("post-expiry response must not be marked Idempotent-Replayed")
	}
}

// ---------------------------------------------------------------------------
// 6. Error response → cache NOT populated (don't poison retries).
// ---------------------------------------------------------------------------

func TestIdempotency_ErrorResponse_NotCached(t *testing.T) {
	cache := newFakeCache()
	mw := Idempotency(cache)

	calls := atomic.Int32{}
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	})
	h := mw(errorHandler)

	uid := uuid.New()
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
		req.Header.Set(IdempotencyHeader, "error-key")
		req = withUserContext(req, uid)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	// Both calls must execute the handler — a 5xx must not poison
	// the cache and must remain retryable.
	if got := calls.Load(); got != 2 {
		t.Fatalf("error responses must NOT be cached: got %d invocations, want 2", got)
	}
	if got := cache.setCalls.Load(); got != 0 {
		t.Fatalf("Set must not be called on 5xx: got %d calls", got)
	}
}

// ---------------------------------------------------------------------------
// 7. Cache transport failure → handler still runs, request not blocked.
// ---------------------------------------------------------------------------

func TestIdempotency_CacheGetError_FailsOpen(t *testing.T) {
	cache := newFakeCache()
	cache.getErr = errors.New("redis transport down")
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	req := httptest.NewRequest("POST", "/api/v1/team/invitations", strings.NewReader(`{}`))
	req.Header.Set(IdempotencyHeader, "key-redis-down")
	req = withUserContext(req, uuid.New())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := calls.Load(); got != 1 {
		t.Fatalf("handler must still run when cache.Get fails: got %d", got)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 8. Oversized header → silently ignored, handler runs without caching.
// ---------------------------------------------------------------------------

func TestIdempotency_OversizedKey_IgnoredAndExecuted(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	huge := strings.Repeat("X", MaxIdempotencyKeyLength+1)
	req := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
	req.Header.Set(IdempotencyHeader, huge)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := calls.Load(); got != 1 {
		t.Fatalf("oversized key must not block handler: got %d", got)
	}
	if got := cache.getCalls.Load(); got != 0 {
		t.Fatalf("oversized key must not hit cache: got %d Get calls", got)
	}
}

// ---------------------------------------------------------------------------
// 9. Different users with the same client-side key must NOT collide.
// ---------------------------------------------------------------------------

func TestIdempotency_PerUserScoping(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	uidAlice := uuid.New()
	uidBob := uuid.New()

	for _, uid := range []uuid.UUID{uidAlice, uidBob} {
		req := httptest.NewRequest("POST", "/api/v1/jobs", strings.NewReader(`{}`))
		req.Header.Set(IdempotencyHeader, "shared-uuid-from-clients")
		req = withUserContext(req, uid)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf("alice and bob with same key must not collide: got %d invocations, want 2", got)
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if got := len(cache.store); got != 2 {
		t.Fatalf("cache must hold one entry per user, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// 10. Anonymous flows (e.g. /auth/register) still benefit from idempotency.
// ---------------------------------------------------------------------------

func TestIdempotency_AnonymousScope_Replays(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	mw := Idempotency(cache)
	h := mw(successHandler(&calls))

	req1 := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{}`))
	req1.Header.Set(IdempotencyHeader, "anon-register-1")
	h.ServeHTTP(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{}`))
	req2.Header.Set(IdempotencyHeader, "anon-register-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req2)

	if got := calls.Load(); got != 1 {
		t.Fatalf("anonymous replay must skip handler, got %d invocations", got)
	}
	if rec.Header().Get(IdempotentReplayedHeader) != "true" {
		t.Fatalf("anonymous replay must set Idempotent-Replayed: true")
	}
}

// ---------------------------------------------------------------------------
// 11. Replay must NOT echo Set-Cookie / Authorization headers.
// ---------------------------------------------------------------------------

func TestIdempotency_ReplayDropsUnsafeHeaders(t *testing.T) {
	cache := newFakeCache()
	calls := atomic.Int32{}
	leaky := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "session=secret-token-do-not-leak")
		w.Header().Set("Authorization", "Bearer cached-leak")
		w.Header().Set("Location", "/safe-replay")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mw := Idempotency(cache)
	h := mw(leaky)

	uid := uuid.New()
	req1 := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
	req1.Header.Set(IdempotencyHeader, "leak-test-key")
	req1 = withUserContext(req1, uid)
	h.ServeHTTP(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest("POST", "/api/v1/proposals", strings.NewReader(`{}`))
	req2.Header.Set(IdempotencyHeader, "leak-test-key")
	req2 = withUserContext(req2, uid)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req2)

	if rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("Set-Cookie must NOT be replayed: got %q", rec.Header().Get("Set-Cookie"))
	}
	if rec.Header().Get("Authorization") != "" {
		t.Fatalf("Authorization must NOT be replayed: got %q", rec.Header().Get("Authorization"))
	}
	// Safe header still echoes.
	if rec.Header().Get("Location") != "/safe-replay" {
		t.Fatalf("Location must be replayed: got %q", rec.Header().Get("Location"))
	}
}

// ---------------------------------------------------------------------------
// 12. RedisIdempotencyCache — round-trip via miniredis.
// ---------------------------------------------------------------------------

func newMiniredisCache(t *testing.T) (*RedisIdempotencyCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisIdempotencyCache(client), mr
}

func TestRedisIdempotencyCache_RoundTrip(t *testing.T) {
	cache, _ := newMiniredisCache(t)
	ctx := context.Background()

	key := "idempotency:roundtrip"
	resp := IdempotentResponse{
		Status:      http.StatusCreated,
		ContentType: "application/json",
		Body:        []byte(`{"id":42}`),
		Headers:     map[string]string{"Location": "/x/42"},
	}
	won, err := cache.Set(ctx, key, resp, 30*time.Second)
	if err != nil {
		t.Fatalf("Set err: %v", err)
	}
	if !won {
		t.Fatalf("first Set must succeed (SETNX won)")
	}

	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if got == nil {
		t.Fatal("expected cached response, got nil")
	}
	if got.Status != http.StatusCreated {
		t.Errorf("status: got %d, want 201", got.Status)
	}
	if string(got.Body) != `{"id":42}` {
		t.Errorf("body: got %q, want %q", got.Body, `{"id":42}`)
	}
	if got.Headers["Location"] != "/x/42" {
		t.Errorf("Location header lost in round-trip: %v", got.Headers)
	}
}

func TestRedisIdempotencyCache_Miss(t *testing.T) {
	cache, _ := newMiniredisCache(t)
	got, err := cache.Get(context.Background(), "no-such-key")
	if err != nil {
		t.Fatalf("Miss must not error: %v", err)
	}
	if got != nil {
		t.Fatalf("Miss must return nil, got %+v", got)
	}
}

func TestRedisIdempotencyCache_SetNXLoserGetsFalse(t *testing.T) {
	cache, _ := newMiniredisCache(t)
	ctx := context.Background()
	key := "idempotency:race"
	resp := IdempotentResponse{Status: 200, Body: []byte(`x`)}

	first, err := cache.Set(ctx, key, resp, time.Minute)
	if err != nil || !first {
		t.Fatalf("first Set must win: ok=%v err=%v", first, err)
	}
	second, err := cache.Set(ctx, key, IdempotentResponse{Status: 200, Body: []byte(`y`)}, time.Minute)
	if err != nil {
		t.Fatalf("second Set err: %v", err)
	}
	if second {
		t.Fatalf("second Set must lose SETNX")
	}
	// Cached value must remain the first writer's payload.
	got, _ := cache.Get(ctx, key)
	if got == nil || string(got.Body) != "x" {
		t.Fatalf("loser overwrote winner: %+v", got)
	}
}

func TestRedisIdempotencyCache_CorruptedEntryTreatedAsMiss(t *testing.T) {
	cache, mr := newMiniredisCache(t)
	// Inject malformed bytes directly so the JSON decoder fails.
	mr.Set("idempotency:bad", "{not-json")
	got, err := cache.Get(context.Background(), "idempotency:bad")
	if err != nil {
		t.Fatalf("corrupted entry must not error: %v", err)
	}
	if got != nil {
		t.Fatalf("corrupted entry must be returned as miss, got %+v", got)
	}
}

func TestNewRedisIdempotencyCache_NilClientPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil client")
		}
	}()
	_ = NewRedisIdempotencyCache(nil)
}

// ---------------------------------------------------------------------------
// 13. captureSafeHeaders unit — covers the allow-list directly.
// ---------------------------------------------------------------------------

func TestCaptureSafeHeaders_AllowListOnly(t *testing.T) {
	h := http.Header{}
	h.Set("Location", "/x")
	h.Set("Etag", `W/"abc"`)
	h.Set("X-Request-Id", "req-1")
	h.Set("Cache-Control", "no-store")
	h.Set("Vary", "Accept")
	h.Set("Authorization", "Bearer leak")
	h.Set("Set-Cookie", "session=leak")
	h.Set("X-Custom-Sensitive", "value")

	got := captureSafeHeaders(h)
	if got == nil {
		t.Fatal("expected non-nil header map")
	}
	for _, want := range []string{"Location", "Etag", "X-Request-Id", "Cache-Control", "Vary"} {
		if got[want] == "" {
			t.Errorf("safe header %q dropped: %#v", want, got)
		}
	}
	for _, deny := range []string{"Authorization", "Set-Cookie", "X-Custom-Sensitive"} {
		if _, ok := got[deny]; ok {
			t.Errorf("unsafe header %q must be dropped: %#v", deny, got)
		}
	}
}
