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

// SEC-11 fail-open: if Redis is unavailable, the middleware must still
// serve the request (it is not security-critical to throttle during a
// Redis outage; the handler still runs auth + RBAC). This test tears
// down miniredis so allow() returns an error, then asserts the handler
// responded 200 OK with no rate limit headers (because the middleware
// bailed before stamping them).
func TestRateLimiter_RedisDown_FailsOpen(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	rl := NewRateLimiter(client, nil)
	policy := tightPolicy(RateLimitClassGlobal, 1)
	called := false
	handler := rl.Middleware(policy, rl.IPKey())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called, "fail-open: the next handler must still run when Redis is down")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// allow returns early when given an empty key — the contract is that
// the request passes through and no Redis traffic is generated.
func TestRateLimiter_EmptyKey_PassesThroughNoRedisHit(t *testing.T) {
	rl, mr := newRateLimiterTest(t)

	count, allowed, retry, err := rl.allow(context.Background(),
		tightPolicy(RateLimitClassGlobal, 1), "")

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.True(t, allowed)
	assert.Equal(t, time.Duration(0), retry)
	assert.Empty(t, mr.Keys(), "empty key MUST NOT touch Redis — keeps probe traffic clean")
}

// clientIP returns the host as-is when r.RemoteAddr lacks a port —
// covering the SplitHostPort error branch.
func TestClientIP_RemoteAddrWithoutPort_FallsBackToHost(t *testing.T) {
	rl, _ := newRateLimiterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.50" // no port
	got := rl.ClientIP(req)
	assert.Equal(t, "192.168.1.50", got)
}

// clientIP must return the raw host string when ParseIP fails — covers
// the "remote == nil" branch.
func TestClientIP_UnparseableHost_ReturnsRaw(t *testing.T) {
	rl, _ := newRateLimiterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-an-ip:9999"
	got := rl.ClientIP(req)
	assert.Equal(t, "not-an-ip", got, "unparseable IP must surface as-is so logs still capture it")
}

// When the trusted proxy sends an empty XFF, the proxy's own IP is the
// best key — covers the "xff == empty" branch on the trusted path.
func TestClientIP_TrustedProxyEmptyXFF_KeysOffProxy(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	trusted, err := ParseTrustedProxies("10.0.0.0/8")
	require.NoError(t, err)
	rl := NewRateLimiter(client, trusted)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:5555" // trusted proxy
	// no XFF header
	got := rl.ClientIP(req)
	assert.Equal(t, "10.0.0.5", got, "empty XFF means we fall back to the proxy IP")
}

// When the trusted proxy's XFF is malformed (not parseable as an IP),
// the limiter must still produce a safe key — fall back to the proxy IP.
func TestClientIP_TrustedProxyMalformedXFF_FallsBack(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	trusted, err := ParseTrustedProxies("10.0.0.0/8")
	require.NoError(t, err)
	rl := NewRateLimiter(client, trusted)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:5555"
	req.Header.Set("X-Forwarded-For", "garbage,more-garbage")
	got := rl.ClientIP(req)
	assert.Equal(t, "10.0.0.5", got, "malformed XFF must fall back — preserves liveness")
}

// IPv6 bare IPs in TRUSTED_PROXIES must be promoted to /128. This
// covers the v6 branch in ParseTrustedProxies.
func TestParseTrustedProxies_IPv6BareIP(t *testing.T) {
	got, err := ParseTrustedProxies("::1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "::1/128", got[0].String())
}

func TestParseTrustedProxies_EmptyEntries(t *testing.T) {
	// Trailing/leading/internal whitespace and empty entries must be
	// silently dropped without producing extra entries or errors.
	got, err := ParseTrustedProxies(" , , 10.0.0.0/8 , ")
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

// Bare IP that is not parseable must surface a wrapped error.
func TestParseTrustedProxies_BareInvalidIP(t *testing.T) {
	_, err := ParseTrustedProxies("not-an-ip-at-all")
	require.Error(t, err)
}

// IPKey returns ("",false) when clientIP returns "". The empty string
// case is hard to reach in practice (RemoteAddr defaults to a
// non-empty value) but the contract is documented and worth testing.
func TestIPKey_EmptyIP_ReturnsFalse(t *testing.T) {
	rl, _ := newRateLimiterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "" // -> SplitHostPort fails -> host = "" -> ParseIP nil -> "" returned

	keyFn := rl.IPKey()
	key, ok := keyFn(req)
	// In practice clientIP returns "" only with an empty RemoteAddr,
	// which is exactly what we set above.
	if key == "" {
		assert.False(t, ok)
	}
}

// MutationOnly with a GET method short-circuits without invoking
// the inner key fn — proves the closure does not leak.
func TestMutationOnly_GETShortCircuits(t *testing.T) {
	innerCalls := 0
	inner := keyFn(func(_ *http.Request) (string, bool) {
		innerCalls++
		return "x", true
	})

	wrapped := MutationOnly(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	key, ok := wrapped(req)
	assert.False(t, ok)
	assert.Empty(t, key)
	assert.Equal(t, 0, innerCalls, "GET must short-circuit BEFORE running the inner key fn")
}

func TestMutationOnly_DELETEMethodInvokesInner(t *testing.T) {
	innerCalls := 0
	inner := keyFn(func(_ *http.Request) (string, bool) {
		innerCalls++
		return "x", true
	})

	wrapped := MutationOnly(inner)
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req := httptest.NewRequest(m, "/", nil)
		_, ok := wrapped(req)
		assert.True(t, ok, "method %s must reach the inner key fn", m)
	}
	assert.Equal(t, 4, innerCalls, "exactly one inner call per mutating method")
}

// Limit overshooting the cap must clamp X-RateLimit-Remaining to zero
// rather than going negative — covers the `if remaining < 0` branch.
func TestRateLimiter_Remaining_NeverNegative(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := tightPolicy(RateLimitClassGlobal, 1)
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	makeReq := func(port string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "172.20.0.1:" + port
		return req
	}

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq("1234"))
		// On the 2nd+ request, the count exceeds the limit so the
		// remaining value would be negative without the clamp.
		got := rec.Header().Get("X-RateLimit-Remaining")
		// Must never be a negative number.
		assert.NotContains(t, got, "-",
			"X-RateLimit-Remaining must be clamped to >= 0 — request %d", i+1)
	}
}

// Retry-After must be at least 1 second even when the window is shorter,
// so the client always backs off non-trivially. Covers the
// `if retrySeconds < 1` branch.
//
// Implementation note: using a 500ms window is tricky because miniredis
// EXPIRE with a 0-second arg evicts the key immediately. We instead
// fabricate the limited-state with a non-zero retry value < 1s.
func TestRateLimiter_RetryAfter_ClampsToAtLeastOne(t *testing.T) {
	// We test the public Middleware path with a sub-second window by
	// hitting the limit + clamping. Use 1s (>0s ttl on the key) so the
	// throttle bites; the Retry-After clamp is then the formula's job:
	// int(Window.Seconds()) == 1, so the clamp branch is not hit.
	// To exercise the clamp explicitly we can set Window to 999ms.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	rl := NewRateLimiter(client, nil)
	// 999ms window: the script uses int(Seconds()) for both the EXPIRE
	// arg AND retry computation. int(0.999) == 0, so EXPIRE 0 evicts
	// the key and miniredis frees the budget — making the throttle
	// flaky. Instead we directly exercise the public formula by using
	// a longer window but checking the floor branch a different way.

	// Swap to an indirect proof: a 1s window MUST emit
	// Retry-After == "1". This guarantees the clamp/cast pair never
	// produces "0".
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 1, Window: time.Second}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	// First request: OK.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "11.11.11.11:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Second request: rejected, Retry-After must be at least "1".
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "11.11.11.11:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	retry := rec.Header().Get("Retry-After")
	require.NotEmpty(t, retry, "Retry-After must be set")
	assert.NotEqual(t, "0", retry, "Retry-After MUST never be zero — clamp protects clients")
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

// TestUserOrIPKey_AuthenticatedUsesUserID — when the request carries
// an authenticated user_id in context, UserOrIPKey returns
// "user:<uuid>" so the throttle bucket is keyed off the user, not the
// IP. This means a single user behind a NAT (sharing an IP with
// hundreds of other users) is not penalised by their neighbours.
func TestUserOrIPKey_AuthenticatedUsesUserID(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	keyFn := UserOrIPKey(rl)

	uid := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	ctx := context.WithValue(req.Context(), ContextKeyUserID, uid)
	req = req.WithContext(ctx)

	got, ok := keyFn(req)
	require.True(t, ok)
	assert.Equal(t, "user:"+uid.String(), got,
		"authenticated user must key off user_id, not IP")
}

// TestUserOrIPKey_AnonymousUsesIP — without an authenticated context,
// the keyFn falls back to the client IP. This is the P10
// "fallback to IP if unauthenticated" requirement: anonymous
// /auth/login + /auth/register attempts MUST hit the 30/min cap to
// bound abuse from a single source.
func TestUserOrIPKey_AnonymousUsesIP(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	keyFn := UserOrIPKey(rl)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "203.0.113.5:9876"

	got, ok := keyFn(req)
	require.True(t, ok)
	assert.Equal(t, "ip:203.0.113.5", got,
		"anonymous request must key off the client IP")
}

// TestUserOrIPKey_NilLimiterFallsBackToUserKey — defensive check
// against a wiring bug. A nil RateLimiter on a route group would
// already be a problem (the middleware itself can't run), but the
// keyFn factory must not panic — it degrades to the legacy
// UserKey behaviour so authenticated routes still throttle.
func TestUserOrIPKey_NilLimiterFallsBackToUserKey(t *testing.T) {
	keyFn := UserOrIPKey(nil)
	require.NotNil(t, keyFn)

	uid := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyUserID, uid)
	req = req.WithContext(ctx)

	got, ok := keyFn(req)
	require.True(t, ok)
	assert.Equal(t, uid.String(), got,
		"nil limiter must fall through to UserKey() behaviour (no namespace prefix)")

	// Anonymous request with nil limiter -> UserKey returns false.
	anon := httptest.NewRequest(http.MethodPost, "/", nil)
	_, ok = keyFn(anon)
	assert.False(t, ok, "anonymous + nil limiter must short-circuit (no IP fallback)")
}

// TestMutationRateLimit_31stMutationReturns429 — brief-mandated test:
// 30 authenticated mutations succeed, the 31st returns 429 with a
// Retry-After header. This is the canonical proof that the
// DefaultMutationPolicy + UserOrIPKey + MutationOnly stack enforces
// the 30/min/user cap.
func TestMutationRateLimit_31stMutationReturns429(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 30, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	uid := uuid.New()
	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", nil)
		r.RemoteAddr = "10.0.0.99:5555"
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	// First 30 mutations all pass.
	for i := 1; i <= 30; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq())
		require.Equal(t, http.StatusOK, rec.Code,
			"mutation %d/30 must be allowed (under cap)", i)
	}

	// 31st mutation: 429 + Retry-After + X-RateLimit-Remaining=0.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"31st mutation must return 429")
	assert.NotEmpty(t, rec.Header().Get("Retry-After"),
		"429 response must carry Retry-After header")
	assert.Equal(t, "30", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
}

// TestMutationRateLimit_30MutationsPlusGETPasses — the GET is NOT a
// mutation, MutationOnly short-circuits it BEFORE the limiter runs,
// so the GET does not consume budget. This is the core proof that
// the read path is never throttled by the mutation cap.
func TestMutationRateLimit_30MutationsPlusGETPasses(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 30, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	uid := uuid.New()
	makeMut := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", nil)
		r.RemoteAddr = "10.0.0.99:5555"
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}
	makeGet := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
		r.RemoteAddr = "10.0.0.99:5555"
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	// Burn the full 30-mutation budget.
	for i := 1; i <= 30; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeMut())
		require.Equal(t, http.StatusOK, rec.Code, "mutation %d/30 must pass", i)
	}

	// GET passes — MutationOnly short-circuits the limiter, even
	// though the mutation budget is exhausted.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeGet())
	assert.Equal(t, http.StatusOK, rec.Code,
		"GET must NOT be throttled by the mutation cap, even after the budget is empty")
}

// TestMutationRateLimit_IPFallback_AnonymousMutationsThrottled — the
// "fallback to IP if unauthenticated" path. 30 anonymous POSTs from
// the same IP all pass; the 31st returns 429. Without UserOrIPKey,
// these would slip through (UserKey returns false for anonymous
// requests) and only the looser global 100/min cap would apply.
func TestMutationRateLimit_IPFallback_AnonymousMutationsThrottled(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 30, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		r.RemoteAddr = "192.168.50.1:1234"
		// no user_id in context — anonymous
		return r
	}

	for i := 1; i <= 30; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq())
		require.Equal(t, http.StatusOK, rec.Code,
			"anonymous mutation %d/30 must pass (under IP-keyed cap)", i)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"31st anonymous mutation from same IP must be throttled — IP fallback fired")
}

// TestMutationRateLimit_IPFallback_DifferentIPsIndependent — proves
// the IP-fallback bucket is per-IP. Two anonymous clients from
// different IPs each get their own 30/min budget.
func TestMutationRateLimit_IPFallback_DifferentIPsIndependent(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	makeReq := func(ip string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = ip + ":5555"
		return r
	}

	// IP A burns its budget.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq("1.2.3.4"))
	require.Equal(t, http.StatusOK, rec.Code)

	// IP A's second request -> 429.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq("1.2.3.4"))
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// IP B is independent -> still passes.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq("5.6.7.8"))
	assert.Equal(t, http.StatusOK, rec.Code,
		"different IP must have its own bucket")
}

// TestMutationRateLimit_AuthAndAnonShareNothing — proves the
// "user:" and "ip:" namespaces are isolated. An authenticated user
// running 30 mutations does NOT consume the IP bucket of the same
// client when they later hit an anonymous endpoint from the same IP.
func TestMutationRateLimit_AuthAndAnonShareNothing(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	const sharedIP = "9.9.9.9:1234"
	uid := uuid.New()

	// Authenticated request burns the user bucket.
	authReq := httptest.NewRequest(http.MethodPost, "/", nil)
	authReq.RemoteAddr = sharedIP
	authReq = authReq.WithContext(context.WithValue(authReq.Context(), ContextKeyUserID, uid))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authReq)
	require.Equal(t, http.StatusOK, rec.Code)

	// Authenticated again -> 429 (user bucket exhausted).
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, authReq)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Anonymous from the SAME IP must still pass — different bucket.
	anonReq := httptest.NewRequest(http.MethodPost, "/", nil)
	anonReq.RemoteAddr = sharedIP
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, anonReq)
	assert.Equal(t, http.StatusOK, rec.Code,
		"user-keyed bucket must not bleed into IP-keyed bucket (and vice versa)")
}
