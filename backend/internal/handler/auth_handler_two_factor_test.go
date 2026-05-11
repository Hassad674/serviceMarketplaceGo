package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/twofactor"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// ----------------------------------------------------------------------------
// Local mocks for the slim 2FA contracts the handler depends on. Each one
// captures call args so individual cases can assert wire-level semantics
// (the right user_id is forwarded, the password verifier is invoked before
// the flag flip, the challenge_id is returned, etc.).
// ----------------------------------------------------------------------------

type mockTwoFactorFlag struct {
	isEnabledFn  func(ctx context.Context, userID uuid.UUID) (bool, error)
	setEnabledFn func(ctx context.Context, userID uuid.UUID, enabled bool) error
	setCalls     []struct {
		UserID  uuid.UUID
		Enabled bool
	}
}

func (m *mockTwoFactorFlag) IsEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	if m.isEnabledFn != nil {
		return m.isEnabledFn(ctx, userID)
	}
	return false, nil
}

func (m *mockTwoFactorFlag) SetEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	m.setCalls = append(m.setCalls, struct {
		UserID  uuid.UUID
		Enabled bool
	}{userID, enabled})
	if m.setEnabledFn != nil {
		return m.setEnabledFn(ctx, userID, enabled)
	}
	return nil
}

type mockTwoFactorChallenger struct {
	requestFn       func(ctx context.Context, in auth.TwoFactorChallengeRequest) (uuid.UUID, error)
	verifyFn        func(ctx context.Context, userID uuid.UUID, code string) error
	lastRequestArgs *auth.TwoFactorChallengeRequest
	lastVerifyCode  string
}

func (m *mockTwoFactorChallenger) RequestChallenge(ctx context.Context, in auth.TwoFactorChallengeRequest) (uuid.UUID, error) {
	m.lastRequestArgs = &in
	if m.requestFn != nil {
		return m.requestFn(ctx, in)
	}
	return uuid.New(), nil
}

func (m *mockTwoFactorChallenger) VerifyChallenge(ctx context.Context, userID uuid.UUID, code string) error {
	m.lastVerifyCode = code
	if m.verifyFn != nil {
		return m.verifyFn(ctx, userID, code)
	}
	return nil
}

type mockTwoFactorPasswordVerifier struct {
	verifyFn       func(ctx context.Context, userID uuid.UUID, password string) error
	lastPassword   string
	lastUserID     uuid.UUID
	verifyCalls    int
}

func (m *mockTwoFactorPasswordVerifier) VerifyPassword(ctx context.Context, userID uuid.UUID, password string) error {
	m.verifyCalls++
	m.lastPassword = password
	m.lastUserID = userID
	if m.verifyFn != nil {
		return m.verifyFn(ctx, userID, password)
	}
	return nil
}

// ----------------------------------------------------------------------------
// EnableTwoFactor
// ----------------------------------------------------------------------------

func TestAuthHandler_EnableTwoFactor(t *testing.T) {
	t.Run("first call without code issues a challenge and returns 202", func(t *testing.T) {
		uid := uuid.New()
		userRepo := &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				u := testUser(uid, user.RoleProvider)
				u.Email = "alice@example.com"
				return u, nil
			},
		}
		flag := &mockTwoFactorFlag{}
		challenger := &mockTwoFactorChallenger{
			requestFn: func(_ context.Context, _ auth.TwoFactorChallengeRequest) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil
			},
		}
		h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(flag, challenger, &mockTwoFactorPasswordVerifier{})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", bytes.NewReader([]byte("{}")))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.EnableTwoFactor(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
		// Challenge was issued for the right user.
		require.NotNil(t, challenger.lastRequestArgs)
		assert.Equal(t, uid, challenger.lastRequestArgs.UserID)
		assert.Equal(t, "alice@example.com", challenger.lastRequestArgs.EmailTo)
		// Flag NOT flipped yet — the code is still pending.
		assert.Empty(t, flag.setCalls)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, true, body["requires_confirmation"])
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", body["challenge_id"])
	})

	t.Run("second call with valid code flips the flag and returns 200", func(t *testing.T) {
		uid := uuid.New()
		flag := &mockTwoFactorFlag{}
		challenger := &mockTwoFactorChallenger{}
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(flag, challenger, &mockTwoFactorPasswordVerifier{})

		body, _ := json.Marshal(map[string]string{"code": "123456"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.EnableTwoFactor(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "123456", challenger.lastVerifyCode)
		require.Len(t, flag.setCalls, 1)
		assert.Equal(t, uid, flag.setCalls[0].UserID)
		assert.True(t, flag.setCalls[0].Enabled)
	})

	t.Run("second call with wrong code returns 400 invalid_code and does not flip", func(t *testing.T) {
		uid := uuid.New()
		flag := &mockTwoFactorFlag{}
		challenger := &mockTwoFactorChallenger{
			verifyFn: func(_ context.Context, _ uuid.UUID, _ string) error {
				return twofactor.ErrCodeMismatch
			},
		}
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(flag, challenger, &mockTwoFactorPasswordVerifier{})

		body, _ := json.Marshal(map[string]string{"code": "000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.EnableTwoFactor(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Empty(t, flag.setCalls)
		assertTwoFactorErrorCode(t, rec, "invalid_code")
	})

	t.Run("expired challenge returns 400 challenge_expired", func(t *testing.T) {
		uid := uuid.New()
		challenger := &mockTwoFactorChallenger{
			verifyFn: func(_ context.Context, _ uuid.UUID, _ string) error {
				return twofactor.ErrChallengeExpired
			},
		}
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(&mockTwoFactorFlag{}, challenger, &mockTwoFactorPasswordVerifier{})

		body, _ := json.Marshal(map[string]string{"code": "111111"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.EnableTwoFactor(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assertTwoFactorErrorCode(t, rec, "challenge_expired")
	})

	t.Run("exhausted attempts returns 429 too_many_attempts", func(t *testing.T) {
		uid := uuid.New()
		challenger := &mockTwoFactorChallenger{
			verifyFn: func(_ context.Context, _ uuid.UUID, _ string) error {
				return twofactor.ErrAttemptsExhausted
			},
		}
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(&mockTwoFactorFlag{}, challenger, &mockTwoFactorPasswordVerifier{})

		body, _ := json.Marshal(map[string]string{"code": "222222"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.EnableTwoFactor(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assertTwoFactorErrorCode(t, rec, "too_many_attempts")
	})

	t.Run("missing 2FA wiring returns 503 feature_unavailable", func(t *testing.T) {
		uid := uuid.New()
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		// Intentionally NOT calling AttachTwoFactor.

		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", bytes.NewReader([]byte("{}")))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.EnableTwoFactor(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assertTwoFactorErrorCode(t, rec, "feature_unavailable")
	})

	t.Run("missing user in context returns 401", func(t *testing.T) {
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(&mockTwoFactorFlag{}, &mockTwoFactorChallenger{}, &mockTwoFactorPasswordVerifier{})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/enable", bytes.NewReader([]byte("{}")))
		rec := httptest.NewRecorder()
		h.EnableTwoFactor(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// ----------------------------------------------------------------------------
// DisableTwoFactor
// ----------------------------------------------------------------------------

func TestAuthHandler_DisableTwoFactor(t *testing.T) {
	t.Run("correct password flips the flag off", func(t *testing.T) {
		uid := uuid.New()
		flag := &mockTwoFactorFlag{}
		verifier := &mockTwoFactorPasswordVerifier{}
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(flag, &mockTwoFactorChallenger{}, verifier)

		body, _ := json.Marshal(map[string]string{"current_password": "Passw0rd!"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/disable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.DisableTwoFactor(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 1, verifier.verifyCalls)
		assert.Equal(t, "Passw0rd!", verifier.lastPassword)
		assert.Equal(t, uid, verifier.lastUserID)
		require.Len(t, flag.setCalls, 1)
		assert.False(t, flag.setCalls[0].Enabled)
	})

	t.Run("wrong password returns 401 invalid_credentials and does NOT flip flag", func(t *testing.T) {
		uid := uuid.New()
		flag := &mockTwoFactorFlag{}
		verifier := &mockTwoFactorPasswordVerifier{
			verifyFn: func(_ context.Context, _ uuid.UUID, _ string) error {
				return user.ErrInvalidCredentials
			},
		}
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(flag, &mockTwoFactorChallenger{}, verifier)

		body, _ := json.Marshal(map[string]string{"current_password": "wrong"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/disable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.DisableTwoFactor(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assertTwoFactorErrorCode(t, rec, "invalid_credentials")
		assert.Empty(t, flag.setCalls)
	})

	t.Run("missing current_password returns 422 validation_error", func(t *testing.T) {
		uid := uuid.New()
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(&mockTwoFactorFlag{}, &mockTwoFactorChallenger{}, &mockTwoFactorPasswordVerifier{})

		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/disable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.DisableTwoFactor(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("missing wiring returns 503", func(t *testing.T) {
		uid := uuid.New()
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

		body, _ := json.Marshal(map[string]string{"current_password": "x"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/two-factor/disable", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.DisableTwoFactor(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})
}

// ----------------------------------------------------------------------------
// VerifyTwoFactor — the LOGIN-completion endpoint
// ----------------------------------------------------------------------------

func TestAuthHandler_VerifyTwoFactor(t *testing.T) {
	t.Run("malformed body returns 400 invalid_request", func(t *testing.T) {
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login/verify-2fa", bytes.NewReader([]byte("not-json")))
		rec := httptest.NewRecorder()
		h.VerifyTwoFactor(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing fields returns 422 validation_error", func(t *testing.T) {
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login/verify-2fa", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		h.VerifyTwoFactor(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("non-uuid user_id returns 400 invalid_request", func(t *testing.T) {
		h := newTestAuthHandler(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})

		body, _ := json.Marshal(map[string]string{"user_id": "not-a-uuid", "code": "123456"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login/verify-2fa", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		h.VerifyTwoFactor(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// ----------------------------------------------------------------------------
// Me — the FIX-2FA regression: two_factor_email_enabled appears on response
// ----------------------------------------------------------------------------

func TestAuthHandler_Me_SurfacesTwoFactorFlag(t *testing.T) {
	t.Run("returns true when the flag store reports enabled", func(t *testing.T) {
		uid := uuid.New()
		userRepo := &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return testUser(uid, user.RoleProvider), nil
			},
		}
		flag := &mockTwoFactorFlag{
			isEnabledFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(flag, &mockTwoFactorChallenger{}, &mockTwoFactorPasswordVerifier{})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.Me(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		// The response is wrapped in res.JSON which does not include
		// an envelope at the helper level — the raw shape is the same
		// as the existing /me tests assume.
		var body struct {
			User struct {
				TwoFactorEmailEnabled bool `json:"two_factor_email_enabled"`
			} `json:"user"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.True(t, body.User.TwoFactorEmailEnabled, "flag must round-trip through /me")
	})

	t.Run("falls back to false when the flag store errors", func(t *testing.T) {
		uid := uuid.New()
		userRepo := &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return testUser(uid, user.RoleProvider), nil
			},
		}
		flag := &mockTwoFactorFlag{
			isEnabledFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return false, errors.New("redis blip")
			},
		}
		h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		h.AttachTwoFactor(flag, &mockTwoFactorChallenger{}, &mockTwoFactorPasswordVerifier{})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.Me(rec, req)

		// Endpoint must NOT 500 when the flag store fails — graceful
		// degradation requirement of the FIX-2FA brief.
		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			User struct {
				TwoFactorEmailEnabled bool `json:"two_factor_email_enabled"`
			} `json:"user"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.False(t, body.User.TwoFactorEmailEnabled)
	})

	t.Run("returns false when 2FA wiring is absent", func(t *testing.T) {
		uid := uuid.New()
		userRepo := &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return testUser(uid, user.RoleProvider), nil
			},
		}
		h := newTestAuthHandler(userRepo, &mockPasswordResetRepo{}, &mockHasher{},
			&mockTokenService{}, &mockSessionService{}, &mockEmailService{})
		// Intentionally NOT wiring 2FA — the field must still be
		// emitted (as false) so the frontend type contract holds.

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, uid))
		rec := httptest.NewRecorder()
		h.Me(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			User struct {
				TwoFactorEmailEnabled bool `json:"two_factor_email_enabled"`
			} `json:"user"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.False(t, body.User.TwoFactorEmailEnabled)
	})
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func assertTwoFactorErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	// Try the envelope shape first; fall back to flat { error: "code" } for
	// the legacy error helper (BadRequest) which writes the code as a string.
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err == nil && body.Error.Code != "" {
		assert.Equal(t, want, body.Error.Code)
		return
	}
	var flat map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &flat); err == nil {
		if got, ok := flat["error"]; ok {
			assert.Equal(t, want, got)
			return
		}
	}
	t.Fatalf("could not extract error code from body: %s", rec.Body.String())
}

// Sanity guard: repository.ErrTwoFactorChallengeNotFound and
// twofactor.ErrChallengeNotFound are aliased / distinct? — the handler
// matches BOTH so a small test confirms the wire mapping is in place.
func TestHandleTwoFactorError_AllSentinelsMapped(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{twofactor.ErrChallengeNotFound, http.StatusBadRequest, "no_challenge"},
		{repository.ErrTwoFactorChallengeNotFound, http.StatusBadRequest, "no_challenge"},
		{twofactor.ErrChallengeExpired, http.StatusBadRequest, "challenge_expired"},
		{twofactor.ErrAttemptsExhausted, http.StatusTooManyRequests, "too_many_attempts"},
		{twofactor.ErrCodeMismatch, http.StatusBadRequest, "invalid_code"},
		{twofactor.ErrUserIDRequired, http.StatusBadRequest, "invalid_request"},
		{user.ErrUserNotFound, http.StatusUnauthorized, "session_invalid"},
	}
	for _, c := range cases {
		t.Run(c.wantCode, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handleTwoFactorError(rec, c.err)
			assert.Equal(t, c.wantStatus, rec.Code)
			assertTwoFactorErrorCode(t, rec, c.wantCode)
		})
	}
}
