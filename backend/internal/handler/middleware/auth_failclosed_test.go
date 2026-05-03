package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/port/service"
)

// F.5 S8 — verifySessionVersion fail-closed-in-prod policy. The
// previous behaviour silently fell back to "trust the snapshot" on a
// DB/Redis blip, letting an attacker bypass session_version
// revocation by triggering an upstream incident.
//
// New policy:
//   - production : 503 Service Unavailable. The middleware refuses
//     to authorize on best-effort data when fail-closed is on.
//   - dev/test    : legacy "trust snapshot" behaviour preserved so a
//     contributor's broken local DB does not lock out everyone.

func TestAuth_FailClosedInProdOnLookupError(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:         userID,
				Role:           "provider",
				SessionVersion: 1,
			}, nil
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session — bearer path")
		},
	}
	checker := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, errors.New("postgres: connection refused")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	// failClosedInProd = true → simulates production wiring.
	handler := AuthWithFailClosed(tokenSvc, sessionSvc, checker, nil, true)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"production must return 503 on session_version lookup failure")
	assert.Contains(t, rec.Body.String(), "auth_unavailable")
	assert.False(t, nextCalled,
		"fail-closed must short-circuit BEFORE the protected handler runs")
}

// TestAuth_FailOpenInDevOnLookupError pins the legacy behaviour for
// non-production. A broken local DB still lets a developer use the
// app via the snapshot, with a slog.Error breadcrumb visible in the
// console.
func TestAuth_FailOpenInDevOnLookupError(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:         userID,
				Role:           "provider",
				SessionVersion: 1,
			}, nil
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session — bearer path")
		},
	}
	checker := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, errors.New("postgres: connection refused")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	// failClosedInProd = false → dev/test fall-through.
	handler := AuthWithFailClosed(tokenSvc, sessionSvc, checker, nil, false)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled,
		"dev/test must keep legacy fail-OPEN (snapshot trust) behaviour")
}
