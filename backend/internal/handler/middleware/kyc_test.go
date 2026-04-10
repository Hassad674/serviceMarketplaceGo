package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// stubUserRepo implements repository.UserRepository for KYC middleware tests.
type stubUserRepo struct {
	user *user.User
	err  error
}

func (s *stubUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	return s.user, s.err
}

// Unused interface methods — stubs to satisfy the interface.
func (s *stubUserRepo) Create(context.Context, *user.User) error { return nil }
func (s *stubUserRepo) GetByEmail(context.Context, string) (*user.User, error) {
	return nil, nil
}
func (s *stubUserRepo) Update(context.Context, *user.User) error { return nil }
func (s *stubUserRepo) Delete(context.Context, uuid.UUID) error  { return nil }
func (s *stubUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}
func (s *stubUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (s *stubUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (s *stubUserRepo) CountByRole(context.Context) (map[string]int, error)      { return nil, nil }
func (s *stubUserRepo) CountByStatus(context.Context) (map[string]int, error)    { return nil, nil }
func (s *stubUserRepo) RecentSignups(context.Context, int) ([]*user.User, error) { return nil, nil }
func (s *stubUserRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (s *stubUserRepo) FindUserIDByStripeAccount(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *stubUserRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (s *stubUserRepo) ClearStripeAccount(context.Context, uuid.UUID) error { return nil }
func (s *stubUserRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (s *stubUserRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error { return nil }
func (s *stubUserRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (s *stubUserRepo) GetKYCPendingUsers(context.Context) ([]*user.User, error) { return nil, nil }
func (s *stubUserRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}

func setAuthContext(r *http.Request, userID uuid.UUID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, ContextKeyRole, role)
	return r.WithContext(ctx)
}

func TestRequireKYCCompliant_Enterprise_PassesThrough(t *testing.T) {
	repo := &stubUserRepo{}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, uuid.New(), "enterprise")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireKYCCompliant_Provider_NoEarnings_PassesThrough(t *testing.T) {
	repo := &stubUserRepo{
		user: &user.User{ID: uuid.New(), Role: user.RoleProvider},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	uid := repo.user.ID
	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, uid, "provider")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireKYCCompliant_Provider_KYCDone_PassesThrough(t *testing.T) {
	stripeID := "acct_123"
	past15 := time.Now().Add(-15 * 24 * time.Hour)
	repo := &stubUserRepo{
		user: &user.User{
			ID:                uuid.New(),
			Role:              user.RoleProvider,
			StripeAccountID:   &stripeID,
			KYCFirstEarningAt: &past15,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.user.ID, "provider")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireKYCCompliant_Provider_Blocked_Returns403(t *testing.T) {
	past15 := time.Now().Add(-15 * 24 * time.Hour)
	repo := &stubUserRepo{
		user: &user.User{
			ID:                uuid.New(),
			Role:              user.RoleProvider,
			KYCFirstEarningAt: &past15,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.user.ID, "provider")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "kyc_restricted", errObj["code"])
}

func TestRequireKYCCompliant_Agency_Blocked_Returns403(t *testing.T) {
	past15 := time.Now().Add(-15 * 24 * time.Hour)
	repo := &stubUserRepo{
		user: &user.User{
			ID:                uuid.New(),
			Role:              user.RoleAgency,
			KYCFirstEarningAt: &past15,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.user.ID, "agency")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireKYCCompliant_Provider_PendingButNotExpired_PassesThrough(t *testing.T) {
	past5 := time.Now().Add(-5 * 24 * time.Hour)
	repo := &stubUserRepo{
		user: &user.User{
			ID:                uuid.New(),
			Role:              user.RoleProvider,
			KYCFirstEarningAt: &past5,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.user.ID, "provider")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Session version stubs (migration 056, Phase 3) ---
func (s *stubUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
