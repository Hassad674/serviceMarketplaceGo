package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marketplace-backend/internal/port/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock types (local to this test file) ---

type mockTokenService struct {
	validateAccessFn func(token string) (*service.TokenClaims, error)
}

func (m *mockTokenService) GenerateAccessToken(_ uuid.UUID, _ string) (string, error) {
	return "", nil
}

func (m *mockTokenService) GenerateRefreshToken(_ uuid.UUID) (string, error) {
	return "", nil
}

func (m *mockTokenService) ValidateAccessToken(token string) (*service.TokenClaims, error) {
	return m.validateAccessFn(token)
}

func (m *mockTokenService) ValidateRefreshToken(_ string) (*service.TokenClaims, error) {
	return nil, nil
}

type mockSessionService struct {
	getFn func(ctx context.Context, sessionID string) (*service.Session, error)
}

func (m *mockSessionService) Create(_ context.Context, _ uuid.UUID, _ string) (*service.Session, error) {
	return nil, nil
}

func (m *mockSessionService) Get(ctx context.Context, sessionID string) (*service.Session, error) {
	return m.getFn(ctx, sessionID)
}

func (m *mockSessionService) Delete(_ context.Context, _ string) error { return nil }

func (m *mockSessionService) CreateWSToken(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}

func (m *mockSessionService) ValidateWSToken(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}

// --- helpers ---

func newAuthRecorder() (*bool, uuid.UUID, string, http.HandlerFunc) {
	called := false
	var gotUID uuid.UUID
	var gotRole string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		gotUID, _ = GetUserID(r.Context())
		gotRole = GetRole(r.Context())
	})
	return &called, gotUID, gotRole, handler
}

// --- tests ---

func TestAuth_ValidSessionCookie(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{ID: "sess-1", UserID: userID, Role: "agency"}, nil
		},
	}
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return nil, errors.New("should not be called")
		},
	}

	var ctxUID uuid.UUID
	var ctxRole string
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxUID, _ = GetUserID(r.Context())
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.True(t, nextCalled, "next handler must be called")
	assert.Equal(t, userID, ctxUID)
	assert.Equal(t, "agency", ctxRole)
}

func TestAuth_ExpiredSessionFallsBackToBearer(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("session expired")
		},
	}
	tokenSvc := &mockTokenService{
		validateAccessFn: func(token string) (*service.TokenClaims, error) {
			if token == "valid-access-token" {
				return &service.TokenClaims{
					UserID:    userID,
					Role:      "enterprise",
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}, nil
			}
			return nil, errors.New("invalid token")
		},
	}

	var ctxUID uuid.UUID
	var ctxRole string
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxUID, _ = GetUserID(r.Context())
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "dead-sess"})
	req.Header.Set("Authorization", "Bearer valid-access-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.True(t, nextCalled, "next handler must be called via bearer fallback")
	assert.Equal(t, userID, ctxUID)
	assert.Equal(t, "enterprise", ctxRole)
}

func TestAuth_ValidBearerToken(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(token string) (*service.TokenClaims, error) {
			if token == "good-token" {
				return &service.TokenClaims{
					UserID: userID,
					Role:   "provider",
				}, nil
			}
			return nil, errors.New("bad")
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}

	var ctxRole string
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.True(t, nextCalled)
	assert.Equal(t, "provider", ctxRole)
}

func TestAuth_InvalidBearerToken(t *testing.T) {
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return nil, errors.New("invalid signature")
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "next must NOT be called with invalid token")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_NoAuthAtAll(t *testing.T) {
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return nil, errors.New("no token")
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_SessionTakesPriorityOverBearer(t *testing.T) {
	sessionUID := uuid.New()
	bearerUID := uuid.New()

	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{
				ID: "sess-priority", UserID: sessionUID, Role: "agency",
			}, nil
		},
	}
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID: bearerUID, Role: "enterprise",
			}, nil
		},
	}

	var ctxUID uuid.UUID
	var ctxRole string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxUID, _ = GetUserID(r.Context())
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-priority"})
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, sessionUID, ctxUID, "session user must take priority")
	assert.Equal(t, "agency", ctxRole, "session role must take priority")
}
