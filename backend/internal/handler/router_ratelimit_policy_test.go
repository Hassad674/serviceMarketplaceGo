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
// preserves the documented production cap (600 req/min/IP after the
// RATE-LIMIT-PROD bump that absorbs CGNAT-shared mobile users).
//
// Regression guard: a future refactor that drifts the default in
// either direction would silently change the throttle behaviour —
// downward causes 429 storms on legitimate CGNAT traffic, upward
// relaxes SEC-11. The assertion pins both the constant value AND
// the fall-through path through middleware.DefaultGlobalPolicy.
func TestGlobalRateLimitPolicy_Default(t *testing.T) {
	t.Parallel()
	policy := GlobalRateLimitPolicy(nil)
	assert.Equal(t, middleware.DefaultGlobalPolicy.Limit, policy.Limit,
		"nil config must fall back to middleware.DefaultGlobalPolicy")
	assert.Equal(t, 600, policy.Limit,
		"RATE-LIMIT-PROD production cap is 600 req/min/IP")

	policy = GlobalRateLimitPolicy(&config.Config{})
	assert.Equal(t, 600, policy.Limit,
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
// mutation-class throttle. Default bumped to 120/min by
// RATE-LIMIT-PROD to give a SPA polling several queries per minute
// headroom on top of an active user typing in the app.
func TestMutationRateLimitPolicy(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 120, MutationRateLimitPolicy(nil).Limit,
		"nil config preserves middleware.DefaultMutationPolicy (120 req/min)")
	assert.Equal(t, 120, MutationRateLimitPolicy(&config.Config{}).Limit,
		"zero RateLimitMutationPerMinute preserves the default")
	assert.Equal(t, 200, MutationRateLimitPolicy(
		&config.Config{RateLimitMutationPerMinute: 200},
	).Limit, "positive RateLimitMutationPerMinute must replace the default")
}

// TestUploadRateLimitPolicy mirrors the global / mutation tests for
// the upload-class throttle. Default bumped to 30/min by
// RATE-LIMIT-PROD so a multi-image portfolio upload sequence does
// not trip the cap on a single user iterating quickly.
func TestUploadRateLimitPolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 30, UploadRateLimitPolicy(nil).Limit,
		"nil config preserves middleware.DefaultUploadPolicy (30 req/min)")
	assert.Equal(t, 30, UploadRateLimitPolicy(&config.Config{}).Limit,
		"zero RateLimitUploadPerMinute preserves the default")
	assert.Equal(t, 60, UploadRateLimitPolicy(
		&config.Config{RateLimitUploadPerMinute: 60},
	).Limit, "positive RateLimitUploadPerMinute must replace the default")
}

// TestAuthLoginRateLimitPolicy pins the per-IP /login class default
// at 10/min and confirms the env override path.
func TestAuthLoginRateLimitPolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 10, AuthLoginRateLimitPolicy(nil).Limit)
	assert.Equal(t, 10, AuthLoginRateLimitPolicy(&config.Config{}).Limit)
	assert.Equal(t, 25, AuthLoginRateLimitPolicy(
		&config.Config{RateLimitAuthLoginPerMinute: 25},
	).Limit)
}

// TestAuth2FAVerifyRateLimitPolicy pins the per-IP /verify-2fa class
// default at 10/min and confirms the env override path.
func TestAuth2FAVerifyRateLimitPolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 10, Auth2FAVerifyRateLimitPolicy(nil).Limit)
	assert.Equal(t, 10, Auth2FAVerifyRateLimitPolicy(&config.Config{}).Limit)
	assert.Equal(t, 20, Auth2FAVerifyRateLimitPolicy(
		&config.Config{RateLimitAuth2FAVerifyPerMinute: 20},
	).Limit)
}

// TestAuth2FAEnableRateLimitPolicy pins the per-user /enable class
// default at 5/min — the tightest cap of any class because the
// endpoint sends a fresh confirmation email on every call.
func TestAuth2FAEnableRateLimitPolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 5, Auth2FAEnableRateLimitPolicy(nil).Limit)
	assert.Equal(t, 5, Auth2FAEnableRateLimitPolicy(&config.Config{}).Limit)
	assert.Equal(t, 10, Auth2FAEnableRateLimitPolicy(
		&config.Config{RateLimitAuth2FAEnablePerMinute: 10},
	).Limit)
}

// TestPasswordResetRateLimitPolicy pins the per-email /forgot-password
// class default at 3/min — the tightest cap because the abuse vector
// (inbox flooding, spam complaints) has zero legitimate user demand
// beyond a single retry-or-two per minute.
func TestPasswordResetRateLimitPolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 3, PasswordResetRateLimitPolicy(nil).Limit)
	assert.Equal(t, 3, PasswordResetRateLimitPolicy(&config.Config{}).Limit)
	assert.Equal(t, 6, PasswordResetRateLimitPolicy(
		&config.Config{RateLimitPasswordResetPerMinute: 6},
	).Limit)
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
