package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/domain/profile"
	domainreferrer "marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

type mockReferrerProfileRepo struct {
	getOrCreateFn func(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error)
	updateCoreFn  func(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error
	updateAvailFn func(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error
	updateExpFn   func(ctx context.Context, orgID uuid.UUID, domains []string) error
}

func (m *mockReferrerProfileRepo) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, orgID)
	}
	return newReferrerStubView(orgID), nil
}
func (m *mockReferrerProfileRepo) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	if m.updateCoreFn != nil {
		return m.updateCoreFn(ctx, orgID, title, about, videoURL)
	}
	return nil
}
func (m *mockReferrerProfileRepo) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	if m.updateAvailFn != nil {
		return m.updateAvailFn(ctx, orgID, status)
	}
	return nil
}
func (m *mockReferrerProfileRepo) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	if m.updateExpFn != nil {
		return m.updateExpFn(ctx, orgID, domains)
	}
	return nil
}

func newReferrerHandler(repo *mockReferrerProfileRepo) *ReferrerProfileHandler {
	return NewReferrerProfileHandler(referrerprofileapp.NewService(repo))
}

func newReferrerStubView(orgID uuid.UUID) *repository.ReferrerProfileView {
	return &repository.ReferrerProfileView{
		Profile: &domainreferrer.Profile{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			Title:              "Top Apporteur",
			About:              "Finds deals.",
			AvailabilityStatus: profile.AvailabilityNow,
			ExpertiseDomains:   []string{},
		},
		Shared: repository.OrganizationSharedProfile{
			PhotoURL:                "https://example.com/r.png",
			City:                    "Lyon",
			CountryCode:             "FR",
			WorkMode:                []string{"remote"},
			LanguagesProfessional:   []string{},
			LanguagesConversational: []string{},
		},
	}
}

func TestReferrerProfileHandler_GetMy_AutoCreatesOnFirstRead(t *testing.T) {
	orgID := uuid.New()
	calls := 0
	repo := &mockReferrerProfileRepo{
		getOrCreateFn: func(ctx context.Context, id uuid.UUID) (*repository.ReferrerProfileView, error) {
			calls++
			return newReferrerStubView(id), nil
		},
	}
	h := newReferrerHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrer-profile", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetMy(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, calls, "GetOrCreate must be called exactly once per GetMy")
}

func TestReferrerProfileHandler_UpdateMyAvailability_AcceptsValidValue(t *testing.T) {
	orgID := uuid.New()
	captured := profile.AvailabilityStatus("")
	repo := &mockReferrerProfileRepo{
		updateAvailFn: func(ctx context.Context, id uuid.UUID, status profile.AvailabilityStatus) error {
			captured = status
			return nil
		},
	}
	h := newReferrerHandler(repo)

	body := `{"availability_status":"available_soon"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/referrer-profile/availability", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateMyAvailability(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, profile.AvailabilitySoon, captured)
}

func TestReferrerProfileHandler_GetMy_Unauthenticated(t *testing.T) {
	h := newReferrerHandler(&mockReferrerProfileRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrer-profile", nil)
	rec := httptest.NewRecorder()
	h.GetMy(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
