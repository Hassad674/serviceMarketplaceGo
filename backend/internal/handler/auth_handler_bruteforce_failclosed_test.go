package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/user"
)

// F.5 S7 — brute-force IsLocked fail-closed-in-prod policy.
// `auth_handler.go` previously did `err == nil && locked` — silently
// swallowing a Redis error. An attacker who could trigger a Redis
// blip would bypass the per-email lockout entirely.
//
// New policy:
//   - production : 503 Service Unavailable, login refused.
//   - dev/test    : legacy fail-OPEN preserved (slog.Error breadcrumb).

func newLoginRequest(email, password string) *http.Request {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Mode", "token")
	return req
}

func TestAuthHandler_Login_BruteForceFailClosedInProd(t *testing.T) {
	existingUser := &user.User{
		ID:             uuid.New(),
		Email:          "victim@example.com",
		HashedPassword: "hashed_Password1!",
		Role:           user.RoleProvider,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	guard := newMockBruteForce(5)
	guard.isLockedErr = errors.New("redis: connection refused")

	h := newAuthHandlerWithBruteForce(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, guard, guard).
		WithFailClosed(true) // production

	rec := httptest.NewRecorder()
	h.Login(rec, newLoginRequest("victim@example.com", "WhateverPass1!"))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"production must reject login when the brute-force backend is degraded")
	assert.Contains(t, rec.Body.String(), "auth_unavailable")
	// The handler must NOT have called RecordFailure (it short-circuited
	// before the password check). Otherwise the counter would inflate
	// during a Redis outage and lock out legitimate users on recovery.
	assert.Equal(t, 0, guard.snapshotFailureCount("victim@example.com"),
		"failing closed must short-circuit BEFORE the password check")
}

func TestAuthHandler_Login_BruteForceFailOpenInDev(t *testing.T) {
	existingUser := &user.User{
		ID:             uuid.New(),
		Email:          "dev@example.com",
		HashedPassword: "hashed_Password1!",
		Role:           user.RoleProvider,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	guard := newMockBruteForce(5)
	guard.isLockedErr = errors.New("redis: connection refused")

	h := newAuthHandlerWithBruteForce(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, guard, guard)
	// failClosedInProd left at false → dev/test fail-OPEN.

	rec := httptest.NewRecorder()
	h.Login(rec, newLoginRequest("dev@example.com", "WrongPass1!"))

	// The login proceeds, hits the password mismatch path, and returns
	// 401 — proves the limiter outage did NOT 503 the request.
	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"dev/test must keep legacy fail-OPEN behaviour")
}
