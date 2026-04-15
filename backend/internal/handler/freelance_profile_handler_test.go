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

	freelanceprofileapp "marketplace-backend/internal/app/freelanceprofile"
	domainfreelance "marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// mockFreelanceProfileRepo is a minimal handler-layer mock for the
// freelance profile repository interface.
type mockFreelanceProfileRepo struct {
	getFn         func(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error)
	updateCoreFn  func(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error
	updateAvailFn func(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error
	updateExpFn   func(ctx context.Context, orgID uuid.UUID, domains []string) error
	updateVideoFn func(ctx context.Context, orgID uuid.UUID, videoURL string) error
	getVideoFn    func(ctx context.Context, orgID uuid.UUID) (string, error)
}

func (m *mockFreelanceProfileRepo) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	if m.getFn != nil {
		return m.getFn(ctx, orgID)
	}
	return nil, domainfreelance.ErrProfileNotFound
}
func (m *mockFreelanceProfileRepo) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	// Mirror GetByOrgID for tests that don't wire a dedicated
	// lazy-create behaviour. The service's owner path now calls
	// GetOrCreateByOrgID internally.
	return m.GetByOrgID(ctx, orgID)
}
func (m *mockFreelanceProfileRepo) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	if m.updateCoreFn != nil {
		return m.updateCoreFn(ctx, orgID, title, about, videoURL)
	}
	return nil
}
func (m *mockFreelanceProfileRepo) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	if m.updateAvailFn != nil {
		return m.updateAvailFn(ctx, orgID, status)
	}
	return nil
}
func (m *mockFreelanceProfileRepo) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	if m.updateExpFn != nil {
		return m.updateExpFn(ctx, orgID, domains)
	}
	return nil
}
func (m *mockFreelanceProfileRepo) UpdateVideo(ctx context.Context, orgID uuid.UUID, videoURL string) error {
	if m.updateVideoFn != nil {
		return m.updateVideoFn(ctx, orgID, videoURL)
	}
	return nil
}
func (m *mockFreelanceProfileRepo) GetVideoURL(ctx context.Context, orgID uuid.UUID) (string, error) {
	if m.getVideoFn != nil {
		return m.getVideoFn(ctx, orgID)
	}
	return "", nil
}

func newFreelanceHandler(repo *mockFreelanceProfileRepo) *FreelanceProfileHandler {
	return NewFreelanceProfileHandler(freelanceprofileapp.NewService(repo))
}

func newFreelanceStubView(orgID uuid.UUID) *repository.FreelanceProfileView {
	return &repository.FreelanceProfileView{
		Profile: &domainfreelance.Profile{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			Title:              "Senior Go Engineer",
			About:              "Builds marketplaces.",
			AvailabilityStatus: profile.AvailabilityNow,
			ExpertiseDomains:   []string{"development"},
		},
		Shared: repository.OrganizationSharedProfile{
			PhotoURL:                "https://example.com/p.png",
			City:                    "Paris",
			CountryCode:             "FR",
			WorkMode:                []string{"remote"},
			LanguagesProfessional:   []string{"fr", "en"},
			LanguagesConversational: []string{},
		},
	}
}

func TestFreelanceProfileHandler_GetMy(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name       string
		setupRepo  func(*mockFreelanceProfileRepo)
		orgCtx     *uuid.UUID
		wantStatus int
	}{
		{
			name: "success",
			setupRepo: func(r *mockFreelanceProfileRepo) {
				r.getFn = func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
					return newFreelanceStubView(id), nil
				}
			},
			orgCtx:     &orgID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			orgCtx:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "not found",
			setupRepo: func(r *mockFreelanceProfileRepo) {
				r.getFn = func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
					return nil, domainfreelance.ErrProfileNotFound
				}
			},
			orgCtx:     &orgID,
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockFreelanceProfileRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			h := newFreelanceHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/freelance-profile", nil)
			if tc.orgCtx != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, *tc.orgCtx)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.GetMy(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestFreelanceProfileHandler_UpdateMy_DecodesAndWrites(t *testing.T) {
	orgID := uuid.New()
	var gotTitle, gotAbout, gotVideo string
	repo := &mockFreelanceProfileRepo{
		updateCoreFn: func(ctx context.Context, id uuid.UUID, title, about, videoURL string) error {
			gotTitle = title
			gotAbout = about
			gotVideo = videoURL
			return nil
		},
		getFn: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newFreelanceStubView(id), nil
		},
	}
	h := newFreelanceHandler(repo)

	body := `{"title":"Senior Go Engineer","about":"Builds marketplaces.","video_url":"https://example.com/v.mp4"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/freelance-profile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateMy(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Senior Go Engineer", gotTitle)
	assert.Equal(t, "Builds marketplaces.", gotAbout)
	assert.Equal(t, "https://example.com/v.mp4", gotVideo)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, orgID.String(), payload["organization_id"])
	assert.Equal(t, "Paris", payload["city"])
}

func TestFreelanceProfileHandler_UpdateMyAvailability_RejectsInvalidValue(t *testing.T) {
	orgID := uuid.New()
	repo := &mockFreelanceProfileRepo{
		getFn: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newFreelanceStubView(id), nil
		},
	}
	h := newFreelanceHandler(repo)

	body := `{"availability_status":"definitely_not_valid"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/freelance-profile/availability", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateMyAvailability(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestFreelanceProfileHandler_UpdateMyExpertise_RejectsOverMaxPayload(t *testing.T) {
	orgID := uuid.New()
	h := newFreelanceHandler(&mockFreelanceProfileRepo{})

	// Build an array of 21 items to exceed the max of 20.
	items := make([]string, 21)
	for i := range items {
		items[i] = "x"
	}
	body, _ := json.Marshal(map[string]any{"domains": items})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/freelance-profile/expertise", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateMyExpertise(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
