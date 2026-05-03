package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// F.5 S7 — fail-closed-in-prod policy. A Redis blip used to silently
// disable throttling because the limiter unconditionally fell through
// to next.ServeHTTP on any Redis error. The new policy:
//
//   - production : 503 Service Unavailable, request blocked.
//   - dev/test    : legacy fail-OPEN behaviour preserved.
//
// We simulate a Redis outage by closing the miniredis backing store
// before invoking the middleware — the Lua script will fail with a
// connection error.

func newBrokenRateLimiter(t *testing.T, failClosedInProd bool) *RateLimiter {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	mr.Close() // simulate outage
	return NewRateLimiterWithPolicy(client, nil, failClosedInProd)
}

func TestRateLimit_FailClosedInProd_RedisDown_Returns503(t *testing.T) {
	rl := newBrokenRateLimiter(t, true)
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 5, Window: time.Minute}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"production must fail CLOSED on Redis error")
	assert.Contains(t, rec.Body.String(), "rate_limit_unavailable")
}

func TestRateLimit_FailOpenInDev_RedisDown_Allows(t *testing.T) {
	rl := newBrokenRateLimiter(t, false)
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 5, Window: time.Minute}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code,
		"dev/test must keep legacy fail-OPEN behaviour")
}

// TestRateLimit_FailClosedInProd_HealthyRedisStillThrottles is a
// belt-and-braces check: turning failClosedInProd on must NOT change
// the behaviour when Redis is healthy. Otherwise the operator could
// not safely flip the flag in production.
func TestRateLimit_FailClosedInProd_HealthyRedisStillThrottles(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	rl := NewRateLimiterWithPolicy(client, nil, true)
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.8:1234"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
