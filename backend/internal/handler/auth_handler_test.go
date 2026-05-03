package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

func newTestAuthHandler(
	userRepo *mockUserRepo,
	resetRepo *mockPasswordResetRepo,
	hasher *mockHasher,
	tokens *mockTokenService,
	session *mockSessionService,
	email *mockEmailService,
) *AuthHandler {
	authSvc := auth.NewService(userRepo, resetRepo, hasher, tokens, email, "https://example.com")
	// Handler tests don't exercise the org provisioning path — pass nil
	// as the org service, which makes /me skip org resolution.
	return NewAuthHandler(authSvc, nil, session, testCookieConfig())
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]string
		authMode   string
		setupMocks func(*mockUserRepo, *mockHasher, *mockSessionService)
		wantStatus int
		wantCode   string
	}{
		{
			name: "success with token mode",
			body: map[string]string{
				"email": "new@example.com", "password": "Password1!",
				"first_name": "John", "last_name": "Doe", "role": "provider",
			},
			authMode:   "token",
			wantStatus: http.StatusCreated,
		},
		{
			name: "success with web mode sets cookies",
			body: map[string]string{
				"email": "new@example.com", "password": "Password1!",
				"first_name": "John", "last_name": "Doe", "role": "provider",
			},
			wantStatus: http.StatusCreated,
		},
		{
			// F.5 S5: anti-enumeration. A duplicate email MUST NOT
			// surface a 409 — that would let an attacker probe which
			// addresses are registered. The handler emits a neutral
			// 202 Accepted with a generic message; the legitimate
			// owner receives a security signal email out-of-band.
			name: "email already exists silent (S5)",
			body: map[string]string{
				"email": "exists@example.com", "password": "Password1!",
				"first_name": "John", "last_name": "Doe", "role": "provider",
			},
			authMode: "token",
			setupMocks: func(ur *mockUserRepo, _ *mockHasher, _ *mockSessionService) {
				ur.existsByEmailFn = func(_ context.Context, _ string) (bool, error) {
					return true, nil
				}
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "missing required fields",
			body:       map[string]string{"email": "a@b.com"},
			authMode:   "token",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_error",
		},
		{
			name: "invalid role",
			body: map[string]string{
				"email": "a@b.com", "password": "Password1!",
				"first_name": "J", "last_name": "D", "role": "hacker",
			},
			authMode:   "token",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_role",
		},
		{
			name: "weak password",
			body: map[string]string{
				"email": "new@example.com", "password": "short",
				"first_name": "J", "last_name": "D", "role": "provider",
			},
			authMode:   "token",
			wantStatus: http.StatusBadRequest,
			wantCode:   "weak_password",
		},
		{
			name: "agency requires display_name",
			body: map[string]string{
				"email": "a@b.com", "password": "Password1!", "role": "agency",
			},
			authMode:   "token",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &mockUserRepo{}
			hasher := &mockHasher{}
			sessionSvc := &mockSessionService{}
			if tc.setupMocks != nil {
				tc.setupMocks(userRepo, hasher, sessionSvc)
			}

			h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, hasher,
				&mockTokenService{}, sessionSvc, &mockEmailService{})

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.authMode != "" {
				req.Header.Set("X-Auth-Mode", tc.authMode)
			}
			rec := httptest.NewRecorder()

			h.Register(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	uid := uuid.New()
	existingUser := &user.User{
		ID: uid, Email: "test@example.com", HashedPassword: "hashed_Password1!",
		FirstName: "Test", LastName: "User", DisplayName: "Test User",
		Role: user.RoleProvider, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	tests := []struct {
		name       string
		body       map[string]string
		authMode   string
		setupMocks func(*mockUserRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success",
			body:     map[string]string{"email": "test@example.com", "password": "Password1!"},
			authMode: "token",
			setupMocks: func(ur *mockUserRepo) {
				ur.getByEmailFn = func(_ context.Context, _ string) (*user.User, error) {
					return existingUser, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong password",
			body:       map[string]string{"email": "test@example.com", "password": "WrongPass1!"},
			authMode:   "token",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "invalid_credentials",
		},
		{
			name:       "missing fields",
			body:       map[string]string{"email": "test@example.com"},
			authMode:   "token",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_error",
		},
		{
			name:       "empty body",
			body:       map[string]string{},
			authMode:   "token",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &mockUserRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(userRepo)
			}

			h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
				&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.authMode != "" {
				req.Header.Set("X-Auth-Mode", tc.authMode)
			}
			rec := httptest.NewRecorder()

			h.Login(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

// TestAuthHandler_Login_WebMode_SessionVersion verifies that the session
// created during a web-mode login carries the user's current session_version
// from the database. This is critical: after a role change (e.g. ownership
// transfer) the backend bumps session_version, invalidating all old sessions.
// If the new session is created with version=0 instead of the current value,
// the auth middleware rejects it immediately — creating an infinite login loop.
func TestAuthHandler_Login_WebMode_SessionVersion(t *testing.T) {
	uid := uuid.New()
	bumpedUser := &user.User{
		ID: uid, Email: "owner@example.com", HashedPassword: "hashed_Password1!",
		FirstName: "Owner", LastName: "User", DisplayName: "Owner User",
		Role: user.RoleAgency, SessionVersion: 3,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return bumpedUser, nil
		},
	}

	var capturedInput service.CreateSessionInput
	sessionSvc := &mockSessionService{
		createFn: func(_ context.Context, input service.CreateSessionInput) (*service.Session, error) {
			capturedInput = input
			return &service.Session{
				ID:             "sess_new",
				UserID:         input.UserID,
				Role:           input.Role,
				SessionVersion: input.SessionVersion,
			}, nil
		},
	}

	h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, sessionSvc, &mockEmailService{})

	body, _ := json.Marshal(map[string]string{
		"email": "owner@example.com", "password": "Password1!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Auth-Mode header → web mode → session cookie path
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 3, capturedInput.SessionVersion,
		"session must carry the user's current session_version so the auth middleware does not reject it")
	assert.Equal(t, uid, capturedInput.UserID)
}

func TestAuthHandler_Logout(t *testing.T) {
	h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess_abc"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Verify session cookie cleared
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" && c.MaxAge == -1 {
			found = true
		}
	}
	assert.True(t, found, "session cookie should be cleared")
}

func TestAuthHandler_Refresh(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		body       map[string]string
		setupMocks func(*mockTokenService, *mockUserRepo)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]string{"refresh_token": "valid_refresh"},
			setupMocks: func(ts *mockTokenService, ur *mockUserRepo) {
				ts.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
					return &service.TokenClaims{UserID: uid, Role: "provider"}, nil
				}
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return testUser(uid, user.RoleProvider), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing refresh token",
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid refresh token",
			body:       map[string]string{"refresh_token": "invalid"},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens := &mockTokenService{}
			userRepo := &mockUserRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(tokens, userRepo)
			}

			h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
				tokens, &mockSessionService{}, &mockEmailService{})

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.Refresh(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestAuthHandler_Me(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMocks func(*mockUserRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:   "success",
			userID: &uid,
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return testUser(uid, user.RoleProvider), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no user in context",
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			// R16: when /auth/me is called with a still-valid JWT but the
			// backing user row has been deleted (e.g. the caller is an
			// operator who just left their org), we must respond with 401
			// session_invalid — NOT 404 — so the client interprets it as
			// "log me out" instead of "retry later".
			name:   "user deleted returns 401 session_invalid",
			userID: &uid,
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return nil, user.ErrUserNotFound
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "session_invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &mockUserRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(userRepo)
			}

			h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
				&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.Me(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var body map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tc.wantCode, body["error"])
			}
		})
	}
}

func TestAuthHandler_ForgotPassword(t *testing.T) {
	h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name:       "always returns 200 regardless of email existence",
			body:       map[string]string{"email": "whatever@example.com"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json still returns 400",
			body:       nil,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tc.body != nil {
				b, _ := json.Marshal(tc.body)
				bodyReader = bytes.NewReader(b)
			} else {
				bodyReader = bytes.NewReader([]byte("{invalid"))
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.ForgotPassword(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestAuthHandler_ResetPassword(t *testing.T) {
	resetID := uuid.New()
	uid := uuid.New()

	tests := []struct {
		name       string
		body       map[string]string
		setupMocks func(*mockPasswordResetRepo, *mockUserRepo)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]string{"token": "valid_token", "new_password": "NewPassword1!"},
			setupMocks: func(rr *mockPasswordResetRepo, ur *mockUserRepo) {
				rr.getByTokenFn = func(_ context.Context, _ string) (*repository.PasswordReset, error) {
					return &repository.PasswordReset{
						ID: resetID, UserID: uid, Token: "valid_token",
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil
				}
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return testUser(uid, user.RoleProvider), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing fields",
			body:       map[string]string{"token": "abc"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid token",
			body:       map[string]string{"token": "bad", "new_password": "NewPassword1!"},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetRepo := &mockPasswordResetRepo{}
			userRepo := &mockUserRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(resetRepo, userRepo)
			}

			h := newTestAuthHandler(userRepo, resetRepo, &mockHasher{},
				&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.ResetPassword(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestAuthHandler_WSToken(t *testing.T) {
	uid := uuid.New()

	t.Run("success", func(t *testing.T) {
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/ws-token", nil)
		ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.WSToken(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotEmpty(t, resp["token"])
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/ws-token", nil)
		rec := httptest.NewRecorder()

		h.WSToken(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
