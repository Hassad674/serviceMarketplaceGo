package handler

import (
	"net/http/httptest"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler/middleware"
)

// TestGlobalRateLimitPolicy_Default asserts that an empty config
// preserves the documented production cap (100 req/min/IP).
//
// PERF-FIX-W-IDLE-CPU regression guard: a future refactor that
// changes the default upward would silently relax SEC-11. A future
// refactor that changes it downward would silently aggravate the
// rate-limit storm we just fixed.
func TestGlobalRateLimitPolicy_Default(t *testing.T) {
	t.Parallel()
	policy := GlobalRateLimitPolicy(nil)
	assert.Equal(t, middleware.DefaultGlobalPolicy.Limit, policy.Limit,
		"nil config must fall back to middleware.DefaultGlobalPolicy")
	assert.Equal(t, 100, policy.Limit,
		"documented production cap is 100 req/min")

	policy = GlobalRateLimitPolicy(&config.Config{})
	assert.Equal(t, 100, policy.Limit,
		"zero RateLimitGlobalPerMinute must fall back to the default")
}

// TestGlobalRateLimitPolicy_Override asserts that the env-driven
// override is honored when a positive value is provided. Local dev
// .env can bump the cap without code changes.
func TestGlobalRateLimitPolicy_Override(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{RateLimitGlobalPerMinute: 500}
	policy := GlobalRateLimitPolicy(cfg)
	assert.Equal(t, 500, policy.Limit,
		"positive RateLimitGlobalPerMinute must replace the default")
}

// TestMutationRateLimitPolicy mirrors the global test for the
// mutation-class throttle.
func TestMutationRateLimitPolicy(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30, MutationRateLimitPolicy(nil).Limit,
		"nil config preserves middleware.DefaultMutationPolicy (30 req/min)")
	assert.Equal(t, 30, MutationRateLimitPolicy(&config.Config{}).Limit,
		"zero RateLimitMutationPerMinute preserves the default")
	assert.Equal(t, 200, MutationRateLimitPolicy(
		&config.Config{RateLimitMutationPerMinute: 200},
	).Limit, "positive RateLimitMutationPerMinute must replace the default")
}

// TestExemptHealthIPKey_SkipsHealthAndReady asserts that the keyFn
// returned by ExemptHealthIPKey short-circuits the rate-limiter for
// the liveness / readiness probes. PERF-FIX-W-IDLE-CPU regression
// guard: a future refactor that drops this exemption would once
// again let an exhausted /api/v1/* IP cap return 429 on /health.
func TestExemptHealthIPKey_SkipsHealthAndReady(t *testing.T) {
	t.Parallel()

	// A nil-Redis limiter is fine here — we never reach Redis
	// because the keyFn returns ("", false) on the exempted paths.
	rl := middleware.NewRateLimiter((*goredis.Client)(nil), nil)
	keyFn := ExemptHealthIPKey(rl)

	for _, path := range []string{"/health", "/ready"} {
		req := httptest.NewRequest("GET", path, nil)
		_, ok := keyFn(req)
		assert.False(t, ok, "rate limiter must skip %s (PERF-FIX-W-IDLE-CPU)", path)
	}
}

// TestExemptHealthIPKey_LimitsEverythingElse asserts the inverse:
// every non-health path still flows through the regular IP key.
func TestExemptHealthIPKey_LimitsEverythingElse(t *testing.T) {
	t.Parallel()

	rl := middleware.NewRateLimiter((*goredis.Client)(nil), nil)
	keyFn := ExemptHealthIPKey(rl)

	for _, path := range []string{
		"/api/v1/auth/me",
		"/api/v1/messaging/conversations",
		"/metrics",
		"/api/openapi.json",
	} {
		req := httptest.NewRequest("GET", path, nil)
		req.RemoteAddr = "127.0.0.1:1234"
		key, ok := keyFn(req)
		assert.True(t, ok, "rate limiter must apply to %s", path)
		assert.NotEmpty(t, key, "non-health paths must produce a non-empty rate-limit key")
	}
}
