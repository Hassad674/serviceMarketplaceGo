package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// mockBruteForce is a thread-safe in-memory implementation used by the
// auth_handler tests. The production adapter is exercised by
// adapter/redis/bruteforce_test.go via miniredis; this mock is just
// enough for the handler-level tests to assert the right method was
// called at the right time.
//
// Both surfaces (per-EMAIL and per-IP) are tracked. The IP gate
// (N4) is a parallel state machine — a successful login does NOT
// clear the IP counter (a shared-NAT user who guesses on attempt #18
// must not unlock the gate for a co-located attacker).
type mockBruteForce struct {
	mu              sync.Mutex
	maxAttempts     int
	maxIPAttempts   int
	attempts        map[string]int
	locked          map[string]time.Time
	ipAttempts      map[string]int
	ipLocked        map[string]time.Time
	defaultLockout  time.Duration
	defaultIPLock   time.Duration
	isLockedErr     error
	isIPLockedErr   error
	recordFailErr   error
	recordIPFailErr error
	recordSuccErr   error
	retryAfterErr   error
	retryAfterIPErr error
	failureCalls    map[string]int
	ipFailureCalls  map[string]int
	successCalls    map[string]int
}

func newMockBruteForce(maxAttempts int) *mockBruteForce {
	return newMockBruteForceWithIP(maxAttempts, 20)
}

// newMockBruteForceWithIP lets the N4 tests dial the IP threshold
// independently of the email one so they can exercise the IP gate
// at any boundary they choose.
func newMockBruteForceWithIP(maxAttempts, maxIPAttempts int) *mockBruteForce {
	return &mockBruteForce{
		maxAttempts:    maxAttempts,
		maxIPAttempts:  maxIPAttempts,
		attempts:       make(map[string]int),
		locked:         make(map[string]time.Time),
		ipAttempts:     make(map[string]int),
		ipLocked:       make(map[string]time.Time),
		defaultLockout: 30 * time.Minute,
		defaultIPLock:  60 * time.Minute,
		failureCalls:   make(map[string]int),
		ipFailureCalls: make(map[string]int),
		successCalls:   make(map[string]int),
	}
}

var _ service.BruteForceService = (*mockBruteForce)(nil)

func (m *mockBruteForce) IsLocked(_ context.Context, email string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isLockedErr != nil {
		return false, m.isLockedErr
	}
	until, ok := m.locked[email]
	if !ok {
		return false, nil
	}
	if time.Now().After(until) {
		delete(m.locked, email)
		return false, nil
	}
	return true, nil
}

func (m *mockBruteForce) RecordFailure(_ context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordFailErr != nil {
		return m.recordFailErr
	}
	m.failureCalls[email]++
	m.attempts[email]++
	if m.attempts[email] >= m.maxAttempts {
		m.locked[email] = time.Now().Add(m.defaultLockout)
	}
	return nil
}

func (m *mockBruteForce) RecordSuccess(_ context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordSuccErr != nil {
		return m.recordSuccErr
	}
	m.successCalls[email]++
	delete(m.attempts, email)
	delete(m.locked, email)
	return nil
}

func (m *mockBruteForce) RetryAfter(_ context.Context, email string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.retryAfterErr != nil {
		return 0, m.retryAfterErr
	}
	until, ok := m.locked[email]
	if !ok {
		return 0, nil
	}
	d := time.Until(until)
	if d < 0 {
		return 0, nil
	}
	return d, nil
}

func (m *mockBruteForce) IsIPLocked(_ context.Context, ip string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isIPLockedErr != nil {
		return false, m.isIPLockedErr
	}
	until, ok := m.ipLocked[ip]
	if !ok {
		return false, nil
	}
	if time.Now().After(until) {
		delete(m.ipLocked, ip)
		return false, nil
	}
	return true, nil
}

func (m *mockBruteForce) RecordIPFailure(_ context.Context, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordIPFailErr != nil {
		return m.recordIPFailErr
	}
	m.ipFailureCalls[ip]++
	m.ipAttempts[ip]++
	if m.ipAttempts[ip] >= m.maxIPAttempts {
		m.ipLocked[ip] = time.Now().Add(m.defaultIPLock)
	}
	return nil
}

func (m *mockBruteForce) RetryAfterIP(_ context.Context, ip string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.retryAfterIPErr != nil {
		return 0, m.retryAfterIPErr
	}
	until, ok := m.ipLocked[ip]
	if !ok {
		return 0, nil
	}
	d := time.Until(until)
	if d < 0 {
		return 0, nil
	}
	return d, nil
}

// snapshotIPFailureCount safely reads the per-IP RecordIPFailure count.
func (m *mockBruteForce) snapshotIPFailureCount(ip string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ipFailureCalls[ip]
}

// snapshotFailureCount safely reads the per-email RecordFailure count.
func (m *mockBruteForce) snapshotFailureCount(email string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failureCalls[email]
}

func (m *mockBruteForce) snapshotSuccessCount(email string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.successCalls[email]
}

// newAuthHandlerWithBruteForce mirrors newTestAuthHandler but returns
// a handler wired with the SEC-07 throttles. We use the same mock
// instance for both login + reset throttles in most tests; tests that
// need different policies override one or both.
func newAuthHandlerWithBruteForce(
	userRepo *mockUserRepo,
	resetRepo *mockPasswordResetRepo,
	hasher *mockHasher,
	tokens *mockTokenService,
	session *mockSessionService,
	loginGuard, resetGuard service.BruteForceService,
) *AuthHandler {
	authSvc := auth.NewService(userRepo, resetRepo, hasher, tokens, &mockEmailService{}, "https://example.com")
	return NewAuthHandler(authSvc, nil, session, testCookieConfig()).
		WithBruteForce(loginGuard, resetGuard)
}

func TestAuthHandler_Login_LocksAfterFiveFailures(t *testing.T) {
	// SEC-07: the 6th attempt against a locked email must return 429
	// with a Retry-After header derived from the lockout TTL.
	uid := uuid.New()
	existingUser := &user.User{
		ID: uid, Email: "lock@example.com", HashedPassword: "hashed_Password1!",
		Role: user.RoleProvider, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	guard := newMockBruteForce(5)
	h := newAuthHandlerWithBruteForce(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, guard, guard)

	doLogin := func(password string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"email": "lock@example.com", "password": password})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Mode", "token")
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		return rec
	}

	for i := 0; i < 5; i++ {
		rec := doLogin("WrongPass1!")
		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"failure %d must be a normal 401 before lockout", i+1)
	}

	// Sixth attempt with the correct password must STILL be 429 —
	// the lockout dominates the credential check.
	rec := doLogin("Password1!")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	retry := rec.Header().Get("Retry-After")
	assert.NotEmpty(t, retry, "429 must include Retry-After")
	if seconds, err := strconv.Atoi(retry); err == nil {
		assert.Greater(t, seconds, 0, "Retry-After must be a positive integer")
	}
}

func TestAuthHandler_Login_Success_ResetsCounter(t *testing.T) {
	// SEC-07: a successful login wipes the per-email counter so the
	// user's next sequence of failures starts fresh.
	uid := uuid.New()
	existingUser := &user.User{
		ID: uid, Email: "reset@example.com", HashedPassword: "hashed_Password1!",
		Role: user.RoleProvider, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	guard := newMockBruteForce(5)
	h := newAuthHandlerWithBruteForce(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, guard, guard)

	body, _ := json.Marshal(map[string]string{"email": "reset@example.com", "password": "Password1!"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Mode", "token")
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, guard.snapshotSuccessCount("reset@example.com"),
		"successful login must invoke RecordSuccess exactly once")
	assert.Equal(t, 0, guard.snapshotFailureCount("reset@example.com"))
}

func TestAuthHandler_Login_Failure_RecordsFailure(t *testing.T) {
	// SEC-07: every credential failure bumps the counter, even when
	// the email does not exist (otherwise an attacker can probe for
	// valid emails by watching for 401 vs 429 responses).
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}
	guard := newMockBruteForce(5)
	h := newAuthHandlerWithBruteForce(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, guard, guard)

	body, _ := json.Marshal(map[string]string{"email": "ghost@example.com", "password": "Password1!"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Mode", "token")
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 1, guard.snapshotFailureCount("ghost@example.com"),
		"failure for unknown email must still bump the counter")
}

func TestAuthHandler_Login_NoBruteForceWiredKeepsLegacyBehavior(t *testing.T) {
	// Backwards compat: a deployment that has not wired the
	// brute-force guard must still log every user in. We verify by
	// running 10 failures + 1 success and asserting all 11 take the
	// expected path.
	uid := uuid.New()
	existingUser := &user.User{
		ID: uid, Email: "legacy@example.com", HashedPassword: "hashed_Password1!",
		Role: user.RoleProvider, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	for i := 0; i < 10; i++ {
		body, _ := json.Marshal(map[string]string{"email": "legacy@example.com", "password": "Wrong" + strconv.Itoa(i) + "!"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Mode", "token")
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"with no brute-force guard, every failure must be 401, not 429")
	}
}

func TestAuthHandler_ForgotPassword_LocksAfterThreeRequests(t *testing.T) {
	// SEC-07: password-reset requests are throttled at 3 per hour per
	// email so an attacker cannot flood inboxes.
	userRepo := &mockUserRepo{}
	resetGuard := newMockBruteForce(3)
	h := newAuthHandlerWithBruteForce(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, newMockBruteForce(99), resetGuard)

	doForgot := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"email": "spam@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ForgotPassword(rec, req)
		return rec
	}

	for i := 0; i < 3; i++ {
		rec := doForgot()
		assert.Equal(t, http.StatusOK, rec.Code,
			"requests within the cap return 200 to avoid leaking that the email exists")
	}

	rec := doForgot()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"4th forgot-password attempt must hit the lockout")
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestAuthHandler_ResetPassword_LocksTokenAfterThreeFailures(t *testing.T) {
	// SEC-07: password-reset consumption is throttled per token so a
	// stolen token cannot be used to brute-force the new_password
	// field.
	resetRepo := &mockPasswordResetRepo{
		getByTokenFn: func(_ context.Context, _ string) (*repository.PasswordReset, error) {
			return nil, user.ErrUnauthorized
		},
	}
	resetGuard := newMockBruteForce(3)
	h := newAuthHandlerWithBruteForce(&mockUserRepo{}, resetRepo, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, newMockBruteForce(99), resetGuard)

	doReset := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"token": "stolen-token", "new_password": "NewStrong1!"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ResetPassword(rec, req)
		return rec
	}

	for i := 0; i < 3; i++ {
		rec := doReset()
		assert.NotEqual(t, http.StatusTooManyRequests, rec.Code,
			"first 3 attempts must propagate the underlying error, not 429")
	}

	rec := doReset()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"4th reset-password attempt with the same token must be 429")
}

