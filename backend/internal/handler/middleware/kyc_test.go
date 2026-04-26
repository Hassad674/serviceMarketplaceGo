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

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/port/repository"
)

// stubOrgRepo implements repository.OrganizationRepository for KYC
// middleware tests. Only FindByID is meaningful; the rest are no-op
// stubs that satisfy the interface.
type stubOrgRepo struct {
	org *organization.Organization
	err error
}

func (s *stubOrgRepo) FindByID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return s.org, s.err
}

// --- no-op stubs to satisfy the interface ---

func (s *stubOrgRepo) Create(context.Context, *organization.Organization) error { return nil }
func (s *stubOrgRepo) CreateWithOwnerMembership(context.Context, *organization.Organization, *organization.Member) error {
	return nil
}
func (s *stubOrgRepo) FindByOwnerUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (s *stubOrgRepo) FindByUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (s *stubOrgRepo) Update(context.Context, *organization.Organization) error { return nil }
func (s *stubOrgRepo) Delete(context.Context, uuid.UUID) error                  { return nil }
func (s *stubOrgRepo) CountAll(context.Context) (int, error)                    { return 0, nil }
func (s *stubOrgRepo) FindByStripeAccountID(context.Context, string) (*organization.Organization, error) {
	return nil, nil
}
func (s *stubOrgRepo) ListKYCPending(context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (s *stubOrgRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (s *stubOrgRepo) GetStripeAccountByUserID(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (s *stubOrgRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (s *stubOrgRepo) ClearStripeAccount(context.Context, uuid.UUID) error { return nil }
func (s *stubOrgRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (s *stubOrgRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error {
	return nil
}
func (s *stubOrgRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (s *stubOrgRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}
func (s *stubOrgRepo) SaveRoleOverrides(context.Context, uuid.UUID, organization.RoleOverrides) error {
	return nil
}
func (s *stubOrgRepo) ListWithStripeAccount(context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

// Compile-time assertion that stubOrgRepo satisfies the interface.
var _ repository.OrganizationRepository = (*stubOrgRepo)(nil)

func setAuthContext(r *http.Request, userID uuid.UUID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, ContextKeyRole, role)
	ctx = context.WithValue(ctx, ContextKeyOrganizationID, userID) // share id in tests
	return r.WithContext(ctx)
}

func TestRequireKYCCompliant_Enterprise_PassesThrough(t *testing.T) {
	repo := &stubOrgRepo{}
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
	repo := &stubOrgRepo{
		org: &organization.Organization{ID: uuid.New(), Type: organization.OrgTypeProviderPersonal},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.org.ID, "provider")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireKYCCompliant_Provider_KYCDone_PassesThrough(t *testing.T) {
	stripeID := "acct_123"
	past15 := time.Now().Add(-15 * 24 * time.Hour)
	repo := &stubOrgRepo{
		org: &organization.Organization{
			ID:                uuid.New(),
			Type:              organization.OrgTypeProviderPersonal,
			StripeAccountID:   &stripeID,
			KYCFirstEarningAt: &past15,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.org.ID, "provider")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireKYCCompliant_Provider_Blocked_Returns403(t *testing.T) {
	past15 := time.Now().Add(-15 * 24 * time.Hour)
	repo := &stubOrgRepo{
		org: &organization.Organization{
			ID:                uuid.New(),
			Type:              organization.OrgTypeProviderPersonal,
			KYCFirstEarningAt: &past15,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.org.ID, "provider")
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
	repo := &stubOrgRepo{
		org: &organization.Organization{
			ID:                uuid.New(),
			Type:              organization.OrgTypeAgency,
			KYCFirstEarningAt: &past15,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.org.ID, "agency")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireKYCCompliant_Provider_PendingButNotExpired_PassesThrough(t *testing.T) {
	past5 := time.Now().Add(-5 * 24 * time.Hour)
	repo := &stubOrgRepo{
		org: &organization.Organization{
			ID:                uuid.New(),
			Type:              organization.OrgTypeProviderPersonal,
			KYCFirstEarningAt: &past5,
		},
	}
	handler := RequireKYCCompliant(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/proposals", nil)
	req = setAuthContext(req, repo.org.ID, "provider")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
