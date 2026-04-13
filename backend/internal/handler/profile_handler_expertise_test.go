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

	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// mockExpertiseRepoHandler is a local stub of repository.ExpertiseRepository
// scoped to the handler-level tests. Kept distinct from the service-layer
// mock so handler tests stay self-contained and don't pollute the package
// scope with test helpers from another file.
type mockExpertiseRepoHandler struct {
	listByOrgFn func(ctx context.Context, orgID uuid.UUID) ([]string, error)
	replaceFn   func(ctx context.Context, orgID uuid.UUID, keys []string) error
}

func (m *mockExpertiseRepoHandler) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	if m.listByOrgFn != nil {
		return m.listByOrgFn(ctx, orgID)
	}
	return []string{}, nil
}

func (m *mockExpertiseRepoHandler) ListByOrganizationIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]string, error) {
	return map[uuid.UUID][]string{}, nil
}

func (m *mockExpertiseRepoHandler) Replace(ctx context.Context, orgID uuid.UUID, keys []string) error {
	if m.replaceFn != nil {
		return m.replaceFn(ctx, orgID, keys)
	}
	return nil
}

// orgRepoForType returns an org repo whose FindByID always resolves
// to an organization of the given type. Used to steer the expertise
// service's per-org-type validation in tests.
func orgRepoForType(t organization.OrgType) repository.OrganizationRepository {
	return &mockOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: t}, nil
		},
	}
}

func newExpertiseHandler(
	profileRepo *mockProfileRepo,
	expRepo *mockExpertiseRepoHandler,
	orgRepo repository.OrganizationRepository,
) *ProfileHandler {
	if profileRepo == nil {
		profileRepo = &mockProfileRepo{}
	}
	if expRepo == nil {
		expRepo = &mockExpertiseRepoHandler{}
	}
	svc := profileapp.NewService(profileRepo)
	expertiseSvc := profileapp.NewExpertiseService(expRepo, orgRepo)
	return NewProfileHandler(svc, expertiseSvc)
}

func putExpertise(orgID *uuid.UUID, body any) *http.Request {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader([]byte("{bad"))
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile/expertise", reader)
	req.Header.Set("Content-Type", "application/json")
	if orgID != nil {
		ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, *orgID)
		req = req.WithContext(ctx)
	}
	return req
}

// --- UpdateMyExpertise ---

func TestProfileHandler_UpdateMyExpertise_SuccessAgency(t *testing.T) {
	orgID := uuid.New()
	var captured []string
	expRepo := &mockExpertiseRepoHandler{
		replaceFn: func(_ context.Context, _ uuid.UUID, keys []string) error {
			captured = keys
			return nil
		},
	}
	h := newExpertiseHandler(nil, expRepo, orgRepoForType(organization.OrgTypeAgency))

	req := putExpertise(&orgID, map[string]any{
		"domains": []string{"development", "design_ui_ux", "marketing_growth"},
	})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"development", "design_ui_ux", "marketing_growth"}, captured)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	got, ok := data["expertise_domains"].([]any)
	require.True(t, ok)
	assert.Len(t, got, 3)
}

func TestProfileHandler_UpdateMyExpertise_Unauthorized(t *testing.T) {
	h := newExpertiseHandler(nil, nil, orgRepoForType(organization.OrgTypeAgency))

	req := putExpertise(nil, map[string]any{"domains": []string{"development"}})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestProfileHandler_UpdateMyExpertise_InvalidJSON(t *testing.T) {
	orgID := uuid.New()
	h := newExpertiseHandler(nil, nil, orgRepoForType(organization.OrgTypeAgency))

	req := putExpertise(&orgID, nil) // forces malformed body
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProfileHandler_UpdateMyExpertise_EnterpriseForbidden(t *testing.T) {
	orgID := uuid.New()
	h := newExpertiseHandler(nil, nil, orgRepoForType(organization.OrgTypeEnterprise))

	req := putExpertise(&orgID, map[string]any{"domains": []string{"development"}})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestProfileHandler_UpdateMyExpertise_UnknownKey(t *testing.T) {
	orgID := uuid.New()
	h := newExpertiseHandler(nil, nil, orgRepoForType(organization.OrgTypeAgency))

	req := putExpertise(&orgID, map[string]any{"domains": []string{"blockchain"}})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProfileHandler_UpdateMyExpertise_Duplicate(t *testing.T) {
	orgID := uuid.New()
	h := newExpertiseHandler(nil, nil, orgRepoForType(organization.OrgTypeAgency))

	req := putExpertise(&orgID, map[string]any{
		"domains": []string{"development", "development"},
	})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProfileHandler_UpdateMyExpertise_OverMaxAgency(t *testing.T) {
	orgID := uuid.New()
	h := newExpertiseHandler(nil, nil, orgRepoForType(organization.OrgTypeAgency))

	req := putExpertise(&orgID, map[string]any{
		"domains": []string{
			"development", "data_ai_ml", "design_ui_ux", "design_3d_animation",
			"video_motion", "photo_audiovisual", "marketing_growth", "writing_translation",
			"business_dev_sales", // 9 — over the 8 max
		},
	})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProfileHandler_UpdateMyExpertise_OverRequestMax(t *testing.T) {
	orgID := uuid.New()
	h := newExpertiseHandler(nil, nil, orgRepoForType(organization.OrgTypeAgency))

	// The handler's hard request-size cap of 20 must short-circuit
	// before the catalog/duplicate checks even run. Use the valid
	// key "development" repeated so the failure is unambiguously the
	// size cap and nothing else.
	big := make([]string, 21)
	for i := range big {
		big[i] = "development"
	}
	req := putExpertise(&orgID, map[string]any{"domains": big})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProfileHandler_UpdateMyExpertise_EmptyListSucceeds(t *testing.T) {
	orgID := uuid.New()
	var called bool
	expRepo := &mockExpertiseRepoHandler{
		replaceFn: func(_ context.Context, _ uuid.UUID, keys []string) error {
			called = true
			assert.Empty(t, keys)
			return nil
		},
	}
	h := newExpertiseHandler(nil, expRepo, orgRepoForType(organization.OrgTypeAgency))

	req := putExpertise(&orgID, map[string]any{"domains": []string{}})
	rec := httptest.NewRecorder()
	h.UpdateMyExpertise(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
}

// --- GetMyProfile embeds expertise list in response ---

func TestProfileHandler_GetMyProfile_IncludesExpertise(t *testing.T) {
	orgID := uuid.New()

	profileRepo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			return testProfile(id), nil
		},
	}
	expRepo := &mockExpertiseRepoHandler{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return []string{"development", "design_ui_ux"}, nil
		},
	}
	h := newExpertiseHandler(profileRepo, expRepo, orgRepoForType(organization.OrgTypeAgency))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetMyProfile(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	got, ok := resp["expertise_domains"].([]any)
	require.True(t, ok, "expertise_domains must be present in the response")
	require.Len(t, got, 2)
	assert.Equal(t, "development", got[0])
	assert.Equal(t, "design_ui_ux", got[1])
}

func TestProfileHandler_GetMyProfile_EmptyExpertiseIsEmptyArrayNotNull(t *testing.T) {
	orgID := uuid.New()

	profileRepo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			return testProfile(id), nil
		},
	}
	// Empty expertise — the default mock returns [] already, but be
	// explicit so the contract is clear from reading the test.
	expRepo := &mockExpertiseRepoHandler{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return []string{}, nil
		},
	}
	h := newExpertiseHandler(profileRepo, expRepo, orgRepoForType(organization.OrgTypeAgency))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetMyProfile(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Inspect the raw body to make sure the JSON is `[]`, not `null`.
	// This is load-bearing for mobile/web clients that iterate without
	// a nil guard.
	require.Contains(t, rec.Body.String(), `"expertise_domains":[]`)
}
