package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
)

// withUserContext attaches the auth middleware's user_id key to a
// request context — every change-* endpoint requires it. Mirrors the
// production middleware.Auth side-effect without standing up the
// whole token validation chain.
func withUserContext(req *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	return req.WithContext(ctx)
}

// --- ChangeEmail handler ---

func TestAuthHandler_ChangeEmail_Success(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "old@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
		Status:         user.StatusActive,
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		updateFn: func(_ context.Context, u *user.User) error {
			existing.Email = u.Email
			return nil
		},
	}

	h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	body, _ := json.Marshal(map[string]string{
		"current_password": "CurrentPass1!",
		"new_email":        "new@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserContext(req, uid)
	rec := httptest.NewRecorder()

	h.ChangeEmail(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "envelope must carry data object")
	assert.Equal(t, "new@example.com", data["email"])
	meta, ok := resp["meta"].(map[string]any)
	require.True(t, ok, "envelope must carry meta object")
	assert.Contains(t, meta, "request_id")
}

func TestAuthHandler_ChangeEmail_TableDriven(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		body       map[string]string
		setupMocks func(*mockUserRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name: "wrong current password",
			body: map[string]string{
				"current_password": "Wrong1Pass!",
				"new_email":        "new@example.com",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return &user.User{
						ID: uid, Email: "old@example.com",
						HashedPassword: "hashed_Right1!", Role: user.RoleProvider,
					}, nil
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "invalid_credentials",
		},
		{
			name: "same email",
			body: map[string]string{
				"current_password": "CurrentPass1!",
				"new_email":        "SAME@example.com",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return &user.User{
						ID: uid, Email: "same@example.com",
						HashedPassword: "hashed_CurrentPass1!", Role: user.RoleProvider,
					}, nil
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "same_email",
		},
		{
			name: "email already taken",
			body: map[string]string{
				"current_password": "CurrentPass1!",
				"new_email":        "taken@example.com",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return &user.User{
						ID: uid, Email: "old@example.com",
						HashedPassword: "hashed_CurrentPass1!", Role: user.RoleProvider,
					}, nil
				}
				ur.existsByEmailFn = func(_ context.Context, _ string) (bool, error) {
					return true, nil
				}
			},
			wantStatus: http.StatusConflict,
			wantCode:   "email_already_exists",
		},
		{
			name: "invalid new email",
			body: map[string]string{
				"current_password": "CurrentPass1!",
				"new_email":        "not-an-email",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_email",
		},
		{
			name: "missing fields",
			body: map[string]string{
				"new_email": "new@example.com",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_error",
		},
		{
			name: "empty body",
			body: map[string]string{},
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-email", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withUserContext(req, uid)
			rec := httptest.NewRecorder()

			h.ChangeEmail(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestAuthHandler_ChangeEmail_NoAuthContext(t *testing.T) {
	h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	body, _ := json.Marshal(map[string]string{
		"current_password": "CurrentPass1!",
		"new_email":        "new@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ChangeEmail(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "unauthorized", resp["error"])
}

// --- ChangePassword handler ---

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "user@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
		Status:         user.StatusActive,
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
		updateFn: func(_ context.Context, u *user.User) error {
			existing.HashedPassword = u.HashedPassword
			return nil
		},
	}

	h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	body, _ := json.Marshal(map[string]string{
		"current_password": "CurrentPass1!",
		"new_password":     "NewStrong1Pass!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserContext(req, uid)
	rec := httptest.NewRecorder()

	h.ChangePassword(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "envelope must carry data object")
	assert.Equal(t, true, data["ok"])
	meta, ok := resp["meta"].(map[string]any)
	require.True(t, ok, "envelope must carry meta object")
	assert.Contains(t, meta, "request_id")
}

func TestAuthHandler_ChangePassword_TableDriven(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		body       map[string]string
		setupMocks func(*mockUserRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name: "wrong current password",
			body: map[string]string{
				"current_password": "Wrong1Pass!",
				"new_password":     "NewStrong1Pass!",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return &user.User{
						ID: uid, Email: "u@example.com",
						HashedPassword: "hashed_Right1!", Role: user.RoleProvider,
					}, nil
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "invalid_credentials",
		},
		{
			name: "same password",
			body: map[string]string{
				"current_password": "CurrentPass1!",
				"new_password":     "CurrentPass1!",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return &user.User{
						ID: uid, Email: "u@example.com",
						HashedPassword: "hashed_CurrentPass1!", Role: user.RoleProvider,
					}, nil
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "same_password",
		},
		{
			name: "weak new password",
			body: map[string]string{
				"current_password": "CurrentPass1!",
				"new_password":     "weak",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "weak_password",
		},
		{
			name: "missing fields",
			body: map[string]string{
				"current_password": "CurrentPass1!",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_error",
		},
		{
			name: "empty body",
			body: map[string]string{},
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withUserContext(req, uid)
			rec := httptest.NewRecorder()

			h.ChangePassword(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestAuthHandler_ChangePassword_NoAuthContext(t *testing.T) {
	h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	body, _ := json.Marshal(map[string]string{
		"current_password": "CurrentPass1!",
		"new_password":     "NewStrong1Pass!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ChangePassword(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestAuthHandler_ChangePassword_RejectsUnknownFields guards the
// project-wide invariant that DecodeJSON disallows unknown fields.
// A typo in the client must be a 400, not a silent drop.
func TestAuthHandler_ChangePassword_RejectsUnknownFields(t *testing.T) {
	uid := uuid.New()
	h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
		&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

	body := []byte(`{"current_password":"CurrentPass1!","new_password":"NewStrong1Pass!","extra_field":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserContext(req, uid)
	rec := httptest.NewRecorder()

	h.ChangePassword(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "invalid_request", resp["error"])
}
