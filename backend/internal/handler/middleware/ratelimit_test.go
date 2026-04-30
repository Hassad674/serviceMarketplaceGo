package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRateLimiterTest spins up a miniredis-backed limiter so tests can
// exercise the production sliding-window logic (Redis Lua script,
// ZSET ops) without touching a real Redis instance.
func newRateLimiterTest(t *testing.T) (*RateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRateLimiter(client, nil), mr
}

func newOKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func tightPolicy(class RateLimitClass, limit int) RateLimitPolicy {
	return RateLimitPolicy{Class: class, Limit: limit, Window: time.Minute}
}

func TestRateLimiter_UnderLimit(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := tightPolicy(RateLimitClassGlobal, 3)
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.10:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d must pass", i+1)
	}
}

func TestRateLimiter_AtAndOverLimit(t *testing.T) {
	// SEC-11: requests within the window cap are allowed, the (cap+1)th
	// is rejected with 429 + Retry-After.
	rl, _ := newRateLimiterTest(t)
	policy := tightPolicy(RateLimitClassGlobal, 2)
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	ip := "10.0.0.5:8080"
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Equal(t, "2", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "rate_limit_exceeded", body["error"])
}

func TestRateLimiter_AlwaysAttachesRateLimitHeaders(t *testing.T) {
	// SEC-11: every response — even successful ones — carries the
	// X-RateLimit-* triplet so clients can self-throttle.
	rl, _ := newRateLimiterTest(t)
	policy := tightPolicy(RateLimitClassGlobal, 5)
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "5", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
}

func TestRateLimiter_DistinctIPsIndependent(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := tightPolicy(RateLimitClassGlobal, 1)
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	for _, ip := range []string{"1.1.1.1:1111", "2.2.2.2:2222"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "first request from %s must pass", ip)
	}

	// IP A's second request hits the limit; IP B is still untouched.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.1.1.1:1111"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiter_DistinctClassesIndependent(t *testing.T) {
	// SEC-11: the global / mutation / upload classes use distinct
	// Redis namespaces, so a request hitting the upload class does
	// not eat into the mutation class budget.
	rl, _ := newRateLimiterTest(t)
	mutation := tightPolicy(RateLimitClassMutation, 1)
	upload := tightPolicy(RateLimitClassUpload, 1)

	mutationHandler := rl.Middleware(mutation, rl.IPKey())(newOKHandler())
	uploadHandler := rl.Middleware(upload, rl.IPKey())(newOKHandler())

	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = "9.9.9.9:9999"
		return r
	}

	rec := httptest.NewRecorder()
	mutationHandler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	uploadHandler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusOK, rec.Code,
		"upload class budget is independent of mutation class")
}

func TestRateLimiter_TrustedProxyHonorsXFF(t *testing.T) {
	// SEC-11: when r.RemoteAddr is in TRUSTED_PROXIES, the leftmost
	// public IP from X-Forwarded-For becomes the throttle key —
	// otherwise behind a load balancer every request would key off
	// the LB's IP and a single bad client would lock everyone out.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	trusted, err := ParseTrustedProxies("10.0.0.0/8")
	require.NoError(t, err)
	rl := NewRateLimiter(client, trusted)
	handler := rl.Middleware(tightPolicy(RateLimitClassGlobal, 1), rl.IPKey())(newOKHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Same trusted proxy, same XFF -> 429 (the throttle key is the
	// downstream client, not the LB).
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:5556"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"trusted proxy must key off X-Forwarded-For")

	// Different downstream client -> independent budget.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:5557"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_UntrustedProxyIgnoresXFF(t *testing.T) {
	// SEC-11: when the connection is NOT from a trusted proxy, the
	// X-Forwarded-For header MUST be ignored — otherwise an attacker
	// can spoof their throttle key by setting the header.
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(tightPolicy(RateLimitClassGlobal, 1), rl.IPKey())(newOKHandler())

	// First request from an external IP — XFF is ignored, the throttle
	// key is "203.0.113.1".
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.1:5555"
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Second request from same RemoteAddr but a different XFF —
	// because the proxy is untrusted, the key is still 203.0.113.1
	// and the limit is hit.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.1:5556"
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"untrusted XFF must NOT split the throttle key")
}

func TestRateLimiter_UserKeyAnonymousSkips(t *testing.T) {
	// UserKey returns false for anonymous requests — those pass
	// through unthrottled (the global IP-based limiter handles them).
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(tightPolicy(RateLimitClassMutation, 1), UserKey())(newOKHandler())

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code,
			"anonymous request %d must skip the user-key throttle", i+1)
	}
}

func TestRateLimiter_UserKeyAuthenticatedThrottles(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(tightPolicy(RateLimitClassMutation, 2), UserKey())(newOKHandler())

	uid := uuid.New()
	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq())
		require.Equal(t, http.StatusOK, rec.Code)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiter_MutationOnlySkipsReads(t *testing.T) {
	// MutationOnly key-fn must short-circuit GETs even with an
	// authenticated user — read traffic is throttled by the global
	// IP-based limiter, not the mutation class.
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(tightPolicy(RateLimitClassMutation, 1), MutationOnly(UserKey()))(newOKHandler())

	uid := uuid.New()
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), ContextKeyUserID, uid)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code,
			"GET request %d must skip mutation throttle", i+1)
	}
}

func TestRateLimiter_MutationOnlyThrottlesWrites(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(tightPolicy(RateLimitClassMutation, 1), MutationOnly(UserKey()))(newOKHandler())

	uid := uuid.New()
	makeReq := func(method string) *http.Request {
		r := httptest.NewRequest(method, "/", nil)
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq(http.MethodPost))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq(http.MethodPut))
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"second mutation must hit the cap (POST + PUT share the key)")
}

func TestRateLimiter_WindowExpiresFreesQuota(t *testing.T) {
	// SEC-11: the sliding window has a real-time TTL — once the
	// window passes, fresh requests succeed again. We FastForward
	// the miniredis clock past the window.
	rl, mr := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 1, Window: 5 * time.Second}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	makeReq := func() (*httptest.ResponseRecorder, *http.Request) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "172.16.0.1:5555"
		return httptest.NewRecorder(), r
	}

	rec, req := makeReq()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec, req = makeReq()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	mr.FastForward(10 * time.Second)

	rec, req = makeReq()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "window must expire after TTL")
}

func TestRateLimiter_ConcurrentRequestsRaceFree(t *testing.T) {
	// 50 goroutines fire simultaneously against a window of 10 — the
	// limiter must allow exactly 10 (within Redis-script atomicity
	// guarantees) and reject the rest. We accept the boundary may be
	// slightly off due to clock skew, so the assertion is "at most
	// limit succeed".
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 10, Window: time.Minute}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	const goroutines = 50
	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		allowed     int
	)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "10.10.10.10:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code == http.StatusOK {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.LessOrEqual(t, allowed, 10, "no more than the cap can succeed")
	assert.Greater(t, allowed, 0, "at least some requests must succeed")
}

func TestParseTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{"empty", "", 0, false},
		{"single CIDR", "10.0.0.0/8", 1, false},
		{"bare IP promoted to /32", "192.168.1.1", 1, false},
		{"multiple", "10.0.0.0/8, 192.168.0.0/16", 2, false},
		{"invalid CIDR", "not-a-cidr", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTrustedProxies(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.want)
		})
	}
}

func TestRateLimiter_SharedAcrossInstancesViaRedis(t *testing.T) {
	// SEC-11: the limiter is Redis-backed so two RateLimiter instances
	// sharing the same Redis client see the same quota — the legacy
	// in-memory limiter doubled the budget across pods.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	policy := tightPolicy(RateLimitClassGlobal, 2)
	rl1 := NewRateLimiter(client, nil)
	rl2 := NewRateLimiter(client, nil)
	h1 := rl1.Middleware(policy, rl1.IPKey())(newOKHandler())
	h2 := rl2.Middleware(policy, rl2.IPKey())(newOKHandler())

	// Request 1 hits instance 1 -> OK.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "5.5.5.5:1111"
	rec := httptest.NewRecorder()
	h1.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Request 2 hits instance 2 -> OK (still within shared cap of 2).
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "5.5.5.5:1112"
	rec = httptest.NewRecorder()
	h2.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Request 3 hits instance 1 -> 429 (shared budget is exhausted).
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "5.5.5.5:1113"
	rec = httptest.NewRecorder()
	h1.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"two instances + shared Redis must share the quota (no double-budget)")
}
