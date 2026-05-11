package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RATE-LIMIT-PROD test suite.
//
// Covers the brief-mandated scenarios:
//   1. New default policies pin to 600 / 120 / 30 (regression guards).
//   2. Each specific class has the correct limit/window/key boundary.
//   3. Authenticated mutation routes are keyed by user_id, not IP.
//   4. CGNAT scenario: 2 distinct user_ids on a shared IP get
//      independent budgets.
//   5. Specific endpoints (login / 2FA verify / 2FA enable / password
//      reset) are keyed correctly.
//   6. 429 responses carry Retry-After + X-RateLimit-* headers.
//   7. The structured slog.Warn metric counter fires on 429.
//   8. Failure mode: Redis blip → middleware fails open (legacy F.5 S7
//      policy pinned).
//   9. EmailKey extracts + hashes the body's email field AND keeps the
//      body re-readable downstream.

// --------------------------------------------------------------------
// 1) Default-policy regression guards.
// --------------------------------------------------------------------

func TestDefaultPolicies_RATELIMITPROD_Limits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		policy    RateLimitPolicy
		wantLimit int
		wantClass RateLimitClass
	}{
		{"global", DefaultGlobalPolicy, 600, RateLimitClassGlobal},
		{"mutation", DefaultMutationPolicy, 120, RateLimitClassMutation},
		{"upload", DefaultUploadPolicy, 30, RateLimitClassUpload},
		{"auth_login", DefaultAuthLoginPolicy, 10, RateLimitClassAuthLogin},
		{"auth_2fa_verify", DefaultAuth2FAVerifyPolicy, 10, RateLimitClassAuth2FAVerify},
		{"auth_2fa_enable", DefaultAuth2FAEnablePolicy, 5, RateLimitClassAuth2FAEnable},
		{"password_reset", DefaultPasswordResetPolicy, 3, RateLimitClassPasswordReset},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantLimit, tc.policy.Limit,
				"%s default limit drifted — regression guard", tc.name)
			assert.Equal(t, tc.wantClass, tc.policy.Class,
				"%s class label drifted — Redis key namespace would shift", tc.name)
			assert.Equal(t, time.Minute, tc.policy.Window,
				"%s window must be 1 minute (per-minute budget convention)", tc.name)
		})
	}
}

// --------------------------------------------------------------------
// 2) CGNAT scenario.
// --------------------------------------------------------------------

// TestRateLimit_CGNAT_TwoUsersSameIP_IndependentBudgets — the core
// reason for the RATE-LIMIT-PROD refactor: a Free Mobile / Orange
// CGNAT user shares an IPv4 /24 with hundreds of neighbours. Keying
// the mutation throttle by IP alone 429'd legitimate users at random.
// The fix is to key by user_id when authenticated; this test pins
// the behaviour so a future refactor cannot regress to IP-only keying.
func TestRateLimit_CGNAT_TwoUsersSameIP_IndependentBudgets(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 2, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	const sharedIP = "203.0.113.50:5555" // CGNAT-shared IPv4
	userA := uuid.New()
	userB := uuid.New()

	makeReq := func(uid uuid.UUID) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", nil)
		r.RemoteAddr = sharedIP
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	// User A exhausts their 2-req budget.
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq(userA))
		require.Equal(t, http.StatusOK, rec.Code,
			"user A request %d/2 must pass (under cap)", i+1)
	}

	// User A's 3rd request: 429 — their own budget is dry.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq(userA))
	require.Equal(t, http.StatusTooManyRequests, rec.Code,
		"user A's 3rd request must be throttled (own budget exhausted)")

	// User B from the SAME IP must still have a full budget — the
	// throttle keys are namespaced by user_id, not by shared IP.
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq(userB))
		assert.Equal(t, http.StatusOK, rec.Code,
			"user B request %d/2 on shared CGNAT IP must pass — CGNAT-safe", i+1)
	}
}

// --------------------------------------------------------------------
// 3) Specific-endpoint class boundaries.
// --------------------------------------------------------------------

// TestAuthLoginClass_IPKeyed_11thRequest429 — the 11th /login attempt
// from a single IP within a minute returns 429.
func TestAuthLoginClass_IPKeyed_11thRequest429(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(DefaultAuthLoginPolicy, rl.IPKey())(newOKHandler())

	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		r.RemoteAddr = "198.51.100.7:5555"
		return r
	}

	for i := 1; i <= 10; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq())
		require.Equal(t, http.StatusOK, rec.Code,
			"login %d/10 must pass (under cap)", i)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"11th /login from same IP must return 429")
	assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

// TestAuth2FAVerifyClass_IPKeyed — mirrors login class, distinct
// Redis namespace.
func TestAuth2FAVerifyClass_IPKeyed(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(DefaultAuth2FAVerifyPolicy, rl.IPKey())(newOKHandler())

	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login/verify-2fa", nil)
		r.RemoteAddr = "198.51.100.8:5555"
		return r
	}
	for i := 1; i <= 10; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq())
		require.Equal(t, http.StatusOK, rec.Code, "2fa-verify %d/10 must pass", i)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"11th 2fa-verify from same IP must return 429")
}

// TestAuth2FAEnableClass_UserIDKeyed — 6th /enable from same user
// returns 429. Two users with the SAME IP get independent budgets
// (covers the anti-email-bombing rationale).
func TestAuth2FAEnableClass_UserIDKeyed(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(DefaultAuth2FAEnablePolicy, UserKey())(newOKHandler())

	userA := uuid.New()
	userB := uuid.New()
	makeReq := func(uid uuid.UUID) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", nil)
		r.RemoteAddr = "10.0.0.1:5555"
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	for i := 1; i <= 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq(userA))
		require.Equal(t, http.StatusOK, rec.Code, "user A /enable %d/5 must pass", i)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq(userA))
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "user A 6th /enable must 429")

	// User B on the same IP: independent budget.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq(userB))
	assert.Equal(t, http.StatusOK, rec.Code,
		"user B /enable from same IP must have its own user-keyed budget")
}

// TestPasswordResetClass_EmailKeyed_4thRequest429 — keying by email
// prevents an attacker iterating emails from a single IP.
func TestPasswordResetClass_EmailKeyed_4thRequest429(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(DefaultPasswordResetPolicy, EmailKey())(newOKHandler())

	makeReq := func(email string) *http.Request {
		body, _ := json.Marshal(map[string]string{"email": email})
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		r.RemoteAddr = "1.2.3.4:5555"
		return r
	}

	for i := 1; i <= 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq("victim@example.com"))
		require.Equal(t, http.StatusOK, rec.Code,
			"reset %d/3 for victim must pass", i)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq("victim@example.com"))
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"4th reset for the same email must 429")

	// Different email from the SAME IP — independent budget.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq("other@example.com"))
	assert.Equal(t, http.StatusOK, rec.Code,
		"different email from same IP must have its own bucket")
}

// TestPasswordResetClass_EmailCaseAndWhitespaceCollapse — "user@x.com",
// "USER@x.com" and "  user@x.com\t" must all share the same bucket
// so an attacker cannot bypass the cap by toggling case.
func TestPasswordResetClass_EmailCaseAndWhitespaceCollapse(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	handler := rl.Middleware(DefaultPasswordResetPolicy, EmailKey())(newOKHandler())

	makeReq := func(email string) *http.Request {
		body, _ := json.Marshal(map[string]string{"email": email})
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		r.RemoteAddr = "5.6.7.8:5555"
		return r
	}

	variants := []string{"user@example.com", "USER@example.com", "  user@example.com  ", "User@Example.com"}
	for i, email := range variants {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, makeReq(email))
		if i < 3 {
			require.Equal(t, http.StatusOK, rec.Code,
				"variant %d (%q) must pass within 3/min cap", i+1, email)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, rec.Code,
				"4th variant must 429 — case/whitespace collapse into one key")
		}
	}
}

// TestEmailKey_ReadsAndRestoresBody — after EmailKey runs, the
// handler's downstream decoder must still see the original payload.
// Without restoration, the password-reset handler would receive an
// empty body and fail to parse the email.
func TestEmailKey_ReadsAndRestoresBody(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "extra": "data"})
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

	key, ok := EmailKey()(r)
	require.True(t, ok, "well-formed body with email must yield a key")
	assert.True(t, strings.HasPrefix(key, "email:"),
		"EmailKey namespace prefix must be present")

	// Body must still be readable by the downstream handler.
	echoed, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.JSONEq(t, string(body), string(echoed),
		"EmailKey must restore the body byte-for-byte so the handler can decode it")
}

// TestEmailKey_MalformedBody_ReturnsFalse — when the body is not JSON
// or lacks an email field, EmailKey short-circuits so the handler
// (not the limiter) gets to reject it.
func TestEmailKey_MalformedBody_ReturnsFalse(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"not json", "not-json"},
		{"json without email", `{"foo":"bar"}`},
		{"email empty string", `{"email":""}`},
		{"email whitespace only", `{"email":"   "}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			_, ok := EmailKey()(r)
			assert.False(t, ok, "%s must short-circuit the limiter", tc.name)
		})
	}
}

// TestEmailKey_NilBody_ReturnsFalse — defensive: the limiter should
// never panic on a synthetic request without a body.
func TestEmailKey_NilBody_ReturnsFalse(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Body = nil
	_, ok := EmailKey()(r)
	assert.False(t, ok)
}

// TestEmailKey_LargeBody_CappedAt4KiB — pathological inputs (a 1 MB
// body with the email field buried at the end) must not stall the
// middleware. The 4 KiB cap drops the body silently and short-
// circuits the limiter.
func TestEmailKey_LargeBody_CappedAt4KiB(t *testing.T) {
	big := strings.Repeat("a", 5<<10) // 5 KiB of padding before the email
	body := []byte(`{"padding":"` + big + `","email":"buried@example.com"}`)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

	_, ok := EmailKey()(r)
	// The cap drops the tail past 4 KiB so the json.Unmarshal fails ->
	// short-circuits. The behaviour pins the cap so a future refactor
	// that lifts the cap also re-reviews the DoS surface.
	assert.False(t, ok,
		"oversized body must short-circuit (4 KiB cap) — protects against body DoS")
}

// --------------------------------------------------------------------
// 4) 429 response headers + structured log.
// --------------------------------------------------------------------

// Test429Response_AttachesAllSemanticHeaders — every 429 carries:
//   - Retry-After (seconds, >= 1)
//   - X-RateLimit-Limit  (numeric, matches policy.Limit)
//   - X-RateLimit-Remaining = "0"
//   - X-RateLimit-Reset (unix epoch seconds, in the future)
func Test429Response_AttachesAllSemanticHeaders(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassAuthLogin, Limit: 1, Window: 30 * time.Second}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = "192.0.2.1:5555"
		return r
	}

	// Burn the cap.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusOK, rec.Code)

	// Hit the cap — inspect every advertised header.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "1", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.NotEqual(t, "0", rec.Header().Get("Retry-After"),
		"Retry-After must be at least 1s — clients otherwise busy-loop")
}

// Test429Response_StructuredLogIncrements — the metric proxy fires.
// We swap slog.Default with a capture logger, trigger a 429, then
// assert the WARN line lands with the expected attributes.
func Test429Response_StructuredLogIncrements(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, UserKey())(newOKHandler())

	// Capture slog output via a buffer-backed JSON handler.
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	uid := uuid.New()
	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	// Burn cap + trip the 429.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, buf.String(), "200 path must not emit the 429 warn line")

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Decode the JSON warn line and inspect attributes.
	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry),
		"429 must emit exactly one JSON log line — got: %s", buf.String())
	assert.Equal(t, "ratelimit: 429 served", entry["msg"])
	assert.Equal(t, string(RateLimitClassMutation), entry["class"])
	assert.Equal(t, true, entry["user_authenticated"],
		"user_authenticated flag must reflect the request context")
	assert.EqualValues(t, 1, entry["limit"])
	assert.EqualValues(t, 60, entry["window_seconds"])
	// The "key" attribute must be the anonymised fingerprint — never
	// the raw user_id. (UserKey() returns the bare UUID with no
	// namespace prefix; UserOrIPKey would prepend "user:" — both
	// paths run through anonymiseRateLimitKey which guarantees the
	// raw identifier never reaches the log.)
	key, ok := entry["key"].(string)
	require.True(t, ok, "429 log line must carry a non-empty `key` attribute")
	assert.NotContains(t, key, uid.String(),
		"anonymised key MUST NOT leak the raw user_id")
	assert.NotEmpty(t, key, "anonymised key must still be present for log grouping")
}

// Test429Response_StructuredLog_NamespacePrefixedKey — when the
// keyFn produces a namespaced key (UserOrIPKey: "user:<uuid>"), the
// anonymised log fingerprint preserves the prefix so an operator can
// grep by namespace. Pairs with the test above which exercises the
// bare-key path.
func Test429Response_StructuredLog_NamespacePrefixedKey(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, UserOrIPKey(rl))(newOKHandler())

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	uid := uuid.New()
	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = "10.0.0.1:1234"
		ctx := context.WithValue(r.Context(), ContextKeyUserID, uid)
		return r.WithContext(ctx)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusOK, rec.Code)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, makeReq())
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	key, _ := entry["key"].(string)
	assert.True(t, strings.HasPrefix(key, "user:"),
		"UserOrIPKey result must preserve the 'user:' namespace in the anonymised log key, got %q", key)
	assert.NotContains(t, key, uid.String(),
		"raw user_id must never appear in the structured log")
}

// TestAnonymiseRateLimitKey_NamespacePrefix — the helper exposed for
// tests: namespace prefix preserved, value fingerprinted, sha256
// fingerprint stable for the same input.
func TestAnonymiseRateLimitKey_NamespacePrefix(t *testing.T) {
	t.Parallel()
	uid := uuid.New().String()
	got := anonymiseRateLimitKey("user:" + uid)
	assert.True(t, strings.HasPrefix(got, "user:"),
		"namespace prefix preserved")
	assert.NotContains(t, got, uid, "raw UUID must be erased")
	// Stability: same input -> same fingerprint.
	assert.Equal(t, got, anonymiseRateLimitKey("user:"+uid),
		"fingerprint must be deterministic")
}

func TestAnonymiseRateLimitKey_NoNamespace_StillFingerprints(t *testing.T) {
	t.Parallel()
	got := anonymiseRateLimitKey("203.0.113.5")
	assert.NotEqual(t, "203.0.113.5", got, "raw IP must be erased")
	assert.NotContains(t, got, ":")
}

func TestAnonymiseRateLimitKey_EmptyKey_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", anonymiseRateLimitKey(""))
}

// TestHashEmail_SameInputSameHash + length sanity.
func TestHashEmail_Properties(t *testing.T) {
	t.Parallel()
	a := hashEmail("user@example.com")
	b := hashEmail("user@example.com")
	c := hashEmail("other@example.com")
	assert.Equal(t, a, b, "deterministic")
	assert.NotEqual(t, a, c, "distinct emails -> distinct hashes")
	assert.Len(t, a, 16, "16-char truncated sha256 hex digest")

	// Sanity check: the truncation matches the spec (first 16 hex
	// chars of sha256). Re-derive externally.
	want := sha256.Sum256([]byte("user@example.com"))
	assert.Equal(t, hex.EncodeToString(want[:])[:16], a)
}

// --------------------------------------------------------------------
// 5) Fail-open behaviour (legacy F.5 S7 policy is preserved).
// --------------------------------------------------------------------

// TestRedisDown_FailsOpen_LegacyPolicyPreserved — when Redis is dead
// and the limiter was constructed with the legacy NewRateLimiter
// (no failClosedInProd flag), the middleware passes the request
// through. RATE-LIMIT-PROD must not regress this contract — the
// auth+RBAC layers still run, so a missing throttle is acceptable
// for a brief Redis outage.
func TestRedisDown_FailsOpen_LegacyPolicyPreserved(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close() // simulate Redis going down BEFORE the request fires

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	rl := NewRateLimiter(client, nil) // legacy ctor, failClosedInProd=false
	called := false
	handler := rl.Middleware(DefaultAuthLoginPolicy, rl.IPKey())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "10.0.0.99:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called,
		"fail-open: when Redis is dead, the next handler must run")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestRedisDown_FailsClosed_InProd — the F.5 S7 production policy:
// when failClosedInProd=true, a Redis blip returns 503 so a partial
// outage does not silently disable throttling. RATE-LIMIT-PROD must
// preserve this contract too.
func TestRedisDown_FailsClosed_InProd(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	rl := NewRateLimiterWithPolicy(client, nil, true /* failClosedInProd */)
	handler := rl.Middleware(DefaultAuthLoginPolicy, rl.IPKey())(newOKHandler())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.99:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"fail-closed (prod) must return 503 when the throttle backend is down")
}

// --------------------------------------------------------------------
// 6) Class-namespace isolation — every new class gets its own bucket.
// --------------------------------------------------------------------

// TestSpecificClasses_DistinctRedisNamespaces — burning the login
// budget MUST NOT eat into the verify-2fa, enable-2fa, or
// password-reset budgets. Each class is its own Redis key prefix.
func TestSpecificClasses_DistinctRedisNamespaces(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	const sharedIP = "10.0.0.42:5555"
	uid := uuid.New()

	loginHandler := rl.Middleware(
		RateLimitPolicy{Class: RateLimitClassAuthLogin, Limit: 1, Window: time.Minute},
		rl.IPKey(),
	)(newOKHandler())
	verifyHandler := rl.Middleware(
		RateLimitPolicy{Class: RateLimitClassAuth2FAVerify, Limit: 1, Window: time.Minute},
		rl.IPKey(),
	)(newOKHandler())
	enableHandler := rl.Middleware(
		RateLimitPolicy{Class: RateLimitClassAuth2FAEnable, Limit: 1, Window: time.Minute},
		UserKey(),
	)(newOKHandler())
	resetHandler := rl.Middleware(
		RateLimitPolicy{Class: RateLimitClassPasswordReset, Limit: 1, Window: time.Minute},
		EmailKey(),
	)(newOKHandler())

	// Burn the login bucket from sharedIP.
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	loginReq.RemoteAddr = sharedIP
	rec := httptest.NewRecorder()
	loginHandler.ServeHTTP(rec, loginReq)
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify-2fa from the SAME IP — independent bucket -> still passes.
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login/verify-2fa", nil)
	verifyReq.RemoteAddr = sharedIP
	rec = httptest.NewRecorder()
	verifyHandler.ServeHTTP(rec, verifyReq)
	assert.Equal(t, http.StatusOK, rec.Code,
		"auth_2fa_verify class must have its own Redis namespace")

	// Enable-2fa keyed by user_id — independent bucket -> still passes.
	enableReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", nil)
	enableReq.RemoteAddr = sharedIP
	enableReq = enableReq.WithContext(context.WithValue(enableReq.Context(), ContextKeyUserID, uid))
	rec = httptest.NewRecorder()
	enableHandler.ServeHTTP(rec, enableReq)
	assert.Equal(t, http.StatusOK, rec.Code,
		"auth_2fa_enable class must have its own Redis namespace")

	// Password-reset keyed by email — independent bucket.
	resetBody, _ := json.Marshal(map[string]string{"email": "ns-test@example.com"})
	resetReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(resetBody))
	resetReq.RemoteAddr = sharedIP
	rec = httptest.NewRecorder()
	resetHandler.ServeHTTP(rec, resetReq)
	assert.Equal(t, http.StatusOK, rec.Code,
		"password_reset class must have its own Redis namespace")
}

// --------------------------------------------------------------------
// 7) Authentication-required routes are keyed by user_id, NOT IP.
// --------------------------------------------------------------------

// TestAuthenticatedMutation_KeyedByUserID_NotIP — proves that an
// authenticated POST keyed via UserOrIPKey uses "user:<uuid>" not
// "ip:<addr>". The brief mandates this; the test pins it.
func TestAuthenticatedMutation_KeyedByUserID_NotIP(t *testing.T) {
	rl, mr := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	uid := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", nil)
	req.RemoteAddr = "203.0.113.99:5555"
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyUserID, uid))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Inspect the Redis key namespace — must contain "user:" + uid.
	wantKey := "ratelimit:mutation:user:" + uid.String()
	keys := mr.Keys()
	found := false
	for _, k := range keys {
		if k == wantKey {
			found = true
			break
		}
	}
	assert.True(t, found,
		"authenticated mutation must produce a user-keyed Redis entry; got keys: %v", keys)
}

// TestAnonymousMutation_FallsBackToIPKey — without an auth context,
// the same middleware keys by IP. Pinned so the fallback path
// stays intact.
func TestAnonymousMutation_FallsBackToIPKey(t *testing.T) {
	rl, mr := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassMutation, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, MutationOnly(UserOrIPKey(rl)))(newOKHandler())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.77:5555"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	wantKey := "ratelimit:mutation:ip:203.0.113.77"
	keys := mr.Keys()
	found := false
	for _, k := range keys {
		if k == wantKey {
			found = true
			break
		}
	}
	assert.True(t, found,
		"anonymous mutation must produce an ip-keyed Redis entry; got keys: %v", keys)
}
