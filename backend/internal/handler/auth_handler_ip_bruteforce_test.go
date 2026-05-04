package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// N4: per-IP brute-force gate tests (handler-level).
//
// These tests exercise the integration of the per-IP guard with the
// auth handler. The Redis adapter's behaviour is exercised separately
// in adapter/redis/bruteforce_test.go; here we focus on the handler
// flow: which methods are called when, and what HTTP shape the client
// sees in each scenario.

// newAuthHandlerWithIPGate builds an AuthHandler with both gates wired
// (per-email + per-IP) plus a fixed IP extractor. The fixed extractor
// makes tests deterministic — we don't depend on the rate limiter's
// trusted-proxy + XFF resolution here.
func newAuthHandlerWithIPGate(
	userRepo *mockUserRepo,
	resetRepo *mockPasswordResetRepo,
	hasher *mockHasher,
	tokens *mockTokenService,
	session *mockSessionService,
	guard service.BruteForceService,
	ipExtractor func(*http.Request) string,
) *AuthHandler {
	authSvc := auth.NewService(userRepo, resetRepo, hasher, tokens, &mockEmailService{}, "https://example.com")
	return NewAuthHandler(authSvc, nil, session, testCookieConfig()).
		WithBruteForce(guard, guard).
		WithIPExtractor(ipExtractor)
}

// fixedIPExtractor returns a constant IP — convenient for tests that
// don't care about the IP extraction logic itself.
func fixedIPExtractor(ip string) func(*http.Request) string {
	return func(_ *http.Request) string { return ip }
}

// xffFirstHopExtractor returns the leftmost IP from X-Forwarded-For,
// or RemoteAddr's host on absence. Used by the proxy-aware test.
func xffFirstHopExtractor(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Mimic ClientIP: leftmost candidate.
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	return r.RemoteAddr
}

func TestBruteforce_IPGate_LocksAfter20Failures(t *testing.T) {
	// N4 main scenario. 1-19 failures from one IP must propagate the
	// underlying 401, not 429. The 20th failure must STILL surface as
	// 401 (the gate is checked at the start of the next request, not
	// the one that pushes the counter over). The 21st request must be
	// 429.
	tests := []struct {
		name          string
		failuresFirst int
		expectedCode  int
	}{
		{
			name:          "1 failure passes through as 401",
			failuresFirst: 1,
			expectedCode:  http.StatusUnauthorized,
		},
		{
			name:          "19 failures still pass through as 401",
			failuresFirst: 19,
			expectedCode:  http.StatusUnauthorized,
		},
		{
			name:          "20th failure still 401 — gate fires on next request",
			failuresFirst: 20,
			expectedCode:  http.StatusUnauthorized,
		},
		{
			name:          "21st request from same IP locked (429)",
			failuresFirst: 21,
			expectedCode:  http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &mockUserRepo{
				getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
					return nil, user.ErrUserNotFound
				},
			}
			// IP threshold 20, email threshold high enough not to trip
			// in this test (we want the IP gate to be the only signal).
			guard := newMockBruteForceWithIP(99, 20)

			h := newAuthHandlerWithIPGate(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
				&mockTokenService{}, &mockSessionService{},
				guard, fixedIPExtractor("203.0.113.50"))

			doLogin := func(email string) *httptest.ResponseRecorder {
				body, _ := json.Marshal(map[string]string{"email": email, "password": "WrongPass1!"})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Auth-Mode", "token")
				rec := httptest.NewRecorder()
				h.Login(rec, req)
				return rec
			}

			for i := 0; i < tt.failuresFirst-1; i++ {
				rec := doLogin("victim" + strconv.Itoa(i) + "@example.com")
				require.NotEqual(t, http.StatusTooManyRequests, rec.Code,
					"failure %d must not lock yet", i+1)
			}

			rec := doLogin("victim-final@example.com")
			assert.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode == http.StatusTooManyRequests {
				retry := rec.Header().Get("Retry-After")
				assert.NotEmpty(t, retry, "429 must include Retry-After")
			}
		})
	}
}

func TestBruteforce_IPGate_DoesNotLockEmail_OnFirst5WrongsFromOneIP(t *testing.T) {
	// N4: confirm per-IP doesn't trigger faster than per-email when a
	// single user mistypes 5 times. The test value is to lock down the
	// behaviour — a busy NAT user must not be locked out before a
	// pure attacker would.
	uid := uuid.New()
	existingUser := &user.User{
		ID: uid, Email: "shared-nat@example.com", HashedPassword: "hashed_Password1!",
		Role: user.RoleProvider, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	guard := newMockBruteForceWithIP(5, 20)

	h := newAuthHandlerWithIPGate(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{},
		guard, fixedIPExtractor("203.0.113.55"))

	doLogin := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"email": "shared-nat@example.com", "password": "WrongPass1!"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Mode", "token")
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		return rec
	}

	// 5 wrongs — email gate locks at 5; we exceed by one to confirm.
	for i := 0; i < 5; i++ {
		rec := doLogin()
		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"failure %d must propagate as 401 before any lockout", i+1)
	}

	// 6th attempt: email gate is now in effect (429 from email side,
	// not IP side). The test value: an attacker would need 20 to lock
	// the IP gate, so a busy NAT user typing wrong 5 times sees the
	// shorter (30min) email lockout, not the longer (60min) IP one.
	rec := doLogin()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	assert.Equal(t, 5, guard.snapshotIPFailureCount("203.0.113.55"),
		"IP counter must reflect the 5 actual failed attempts")
	ipLocked, _ := guard.IsIPLocked(context.Background(), "203.0.113.55")
	assert.False(t, ipLocked, "5 IP failures must not trigger 20-threshold gate")
}

func TestBruteforce_BothGatesIndependent(t *testing.T) {
	// N4: critical invariant. An email-locked IP can still try OTHER
	// emails until the IP threshold; an IP-locked source cannot try
	// ANY email.
	t.Run("email-locked IP can still try other emails until IP threshold", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
				return nil, user.ErrUserNotFound
			},
		}
		guard := newMockBruteForceWithIP(5, 20)
		h := newAuthHandlerWithIPGate(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{},
			guard, fixedIPExtractor("203.0.113.60"))

		doLogin := func(email string) *httptest.ResponseRecorder {
			body, _ := json.Marshal(map[string]string{"email": email, "password": "WrongPass1!"})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Auth-Mode", "token")
			rec := httptest.NewRecorder()
			h.Login(rec, req)
			return rec
		}

		// 5 wrongs against email A — lock email A
		for i := 0; i < 5; i++ {
			rec := doLogin("alpha@example.com")
			require.Equal(t, http.StatusUnauthorized, rec.Code, "alpha attempt %d", i+1)
		}
		// 6th against alpha → 429 from email gate
		rec := doLogin("alpha@example.com")
		assert.Equal(t, http.StatusTooManyRequests, rec.Code,
			"alpha is now email-locked")

		// But the SAME IP can still try beta@example.com — IP threshold (20) > 5
		rec = doLogin("beta@example.com")
		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"beta@example.com from email-locked IP must propagate as 401, not 429")
	})

	t.Run("IP-locked source cannot try any email", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
				return nil, user.ErrUserNotFound
			},
		}
		guard := newMockBruteForceWithIP(99, 20) // email gate effectively disabled
		h := newAuthHandlerWithIPGate(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{},
			guard, fixedIPExtractor("203.0.113.65"))

		doLogin := func(email string) *httptest.ResponseRecorder {
			body, _ := json.Marshal(map[string]string{"email": email, "password": "WrongPass1!"})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Auth-Mode", "token")
			rec := httptest.NewRecorder()
			h.Login(rec, req)
			return rec
		}

		// 20 wrongs against 20 different emails to lock the IP gate
		// without engaging the email gate.
		for i := 0; i < 20; i++ {
			rec := doLogin("victim" + strconv.Itoa(i) + "@example.com")
			require.Equal(t, http.StatusUnauthorized, rec.Code,
				"victim %d must propagate as 401 before IP lock", i)
		}

		// 21st attempt — to ANY email — must be 429 from the IP gate.
		rec := doLogin("fresh-victim@example.com")
		assert.Equal(t, http.StatusTooManyRequests, rec.Code,
			"IP-locked source must be 429 even for a never-seen-before email")
	})
}

func TestBruteforce_BehindProxy_UsesXForwardedFor(t *testing.T) {
	// N4: behind a load balancer, the X-Forwarded-For first IP wins.
	// We use a custom extractor here that mimics what the production
	// rate-limiter's ClientIP method returns (XFF leftmost). The test
	// verifies the integration: extractor output IS the IP gate's
	// key.
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}
	guard := newMockBruteForceWithIP(99, 5) // tight IP gate for the test

	h := newAuthHandlerWithIPGate(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{},
		guard, xffFirstHopExtractor)

	xffIP := "198.51.100.77"
	doLogin := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"email": "victim@example.com", "password": "WrongPass1!"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Mode", "token")
		// XFF chain: client → proxy1 → server. The leftmost is the
		// real client.
		req.Header.Set("X-Forwarded-For", xffIP+", 10.0.0.1")
		req.RemoteAddr = "10.0.0.1:6789"
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		return rec
	}

	for i := 0; i < 5; i++ {
		rec := doLogin()
		require.Equal(t, http.StatusUnauthorized, rec.Code, "attempt %d", i+1)
	}

	// 6th attempt must be 429 keyed by the XFF leftmost, not by
	// the proxy's RemoteAddr.
	rec := doLogin()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"IP gate must key by XFF first hop, not by proxy RemoteAddr")
	assert.Equal(t, 5, guard.snapshotIPFailureCount(xffIP),
		"failure counter must be keyed by the XFF first hop")
}

func TestBruteforce_IPGate_NoExtractorDisablesIPGate(t *testing.T) {
	// Backwards compat: a deployment that has NOT wired
	// WithIPExtractor must not engage the per-IP gate at all. The
	// per-email gate stays in effect.
	uid := uuid.New()
	existingUser := &user.User{
		ID: uid, Email: "no-extractor@example.com", HashedPassword: "hashed_Password1!",
		Role: user.RoleProvider, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	guard := newMockBruteForceWithIP(5, 1) // IP threshold 1 — would lock immediately if engaged

	authSvc := auth.NewService(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockEmailService{}, "https://example.com")
	// Note: NO WithIPExtractor — IP gate disabled.
	h := NewAuthHandler(authSvc, nil, &mockSessionService{}, testCookieConfig()).
		WithBruteForce(guard, guard)

	doLogin := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"email": "no-extractor@example.com", "password": "WrongPass1!"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Mode", "token")
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		return rec
	}

	// Two failures. With IP threshold of 1 the gate would have locked
	// at attempt #1 if engaged. Confirm no 429 from the IP side.
	for i := 0; i < 2; i++ {
		rec := doLogin()
		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"no IP extractor must not trigger IP gate (attempt %d)", i+1)
	}

	// Email counter still ticks
	assert.Equal(t, 2, guard.snapshotFailureCount("no-extractor@example.com"))
	// IP counter must remain zero (no IP key recorded)
	assert.Equal(t, 0, guard.snapshotIPFailureCount("203.0.113.0"),
		"no IP failures recorded when extractor is nil")
}

func TestBruteforce_IPGate_RecordIPFailureCalledOnEveryEmailFailure(t *testing.T) {
	// Symmetry assertion: every email failure increments the IP
	// counter in lockstep. If they ever diverged, the IP gate would
	// be unreliable.
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}
	guard := newMockBruteForceWithIP(99, 99) // both gates loose so we count, not lock

	h := newAuthHandlerWithIPGate(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{},
		guard, fixedIPExtractor("203.0.113.70"))

	doLogin := func(email string) {
		body, _ := json.Marshal(map[string]string{"email": email, "password": "WrongPass1!"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Mode", "token")
		rec := httptest.NewRecorder()
		h.Login(rec, req)
	}

	for i := 0; i < 7; i++ {
		doLogin("u" + strconv.Itoa(i) + "@example.com")
	}

	assert.Equal(t, 7, guard.snapshotIPFailureCount("203.0.113.70"),
		"IP counter must increment on every email failure (lockstep)")
}
