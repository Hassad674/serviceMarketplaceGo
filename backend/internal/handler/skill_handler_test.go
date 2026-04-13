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

	appskill "marketplace-backend/internal/app/skill"
	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/handler/middleware"
)

// --- mockSkillService ------------------------------------------------

// mockSkillService is a tiny stub implementing the handler's local
// skillService interface. Every method delegates to a function field
// so individual tests can override exactly the behaviour they need.
type mockSkillService struct {
	getCuratedFn   func(ctx context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error)
	countCuratedFn func(ctx context.Context, key string) (int, error)
	autocompleteFn func(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error)
	getProfileFn   func(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error)
	replaceFn      func(ctx context.Context, in appskill.ReplaceProfileSkillsInput) error
	createFn       func(ctx context.Context, in appskill.CreateUserSkillInput) (*domainskill.CatalogEntry, error)
}

func (m *mockSkillService) GetCuratedForExpertise(ctx context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error) {
	if m.getCuratedFn != nil {
		return m.getCuratedFn(ctx, key, limit)
	}
	return []*domainskill.CatalogEntry{}, nil
}

func (m *mockSkillService) CountCuratedForExpertise(ctx context.Context, key string) (int, error) {
	if m.countCuratedFn != nil {
		return m.countCuratedFn(ctx, key)
	}
	return 0, nil
}

func (m *mockSkillService) Autocomplete(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error) {
	if m.autocompleteFn != nil {
		return m.autocompleteFn(ctx, q, limit)
	}
	return []*domainskill.CatalogEntry{}, nil
}

func (m *mockSkillService) GetProfileSkills(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error) {
	if m.getProfileFn != nil {
		return m.getProfileFn(ctx, orgID)
	}
	return []*domainskill.ProfileSkill{}, nil
}

func (m *mockSkillService) ReplaceProfileSkills(ctx context.Context, in appskill.ReplaceProfileSkillsInput) error {
	if m.replaceFn != nil {
		return m.replaceFn(ctx, in)
	}
	return nil
}

func (m *mockSkillService) CreateUserSkill(ctx context.Context, in appskill.CreateUserSkillInput) (*domainskill.CatalogEntry, error) {
	if m.createFn != nil {
		return m.createFn(ctx, in)
	}
	entry, _ := domainskill.NewCatalogEntry(in.DisplayText, in.DisplayText, in.ExpertiseKeys, false)
	return entry, nil
}

// --- helpers ---------------------------------------------------------

// newSkillRequest builds an *http.Request with the given method, path,
// optional JSON body, and optional organization id in the context.
// Passing a nil orgID skips the org-context injection so tests can
// exercise the 401 branch.
func newSkillRequest(method, path string, body any, orgID *uuid.UUID) *http.Request {
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, path, nil)
	} else if raw, ok := body.([]byte); ok {
		r = httptest.NewRequest(method, path, bytes.NewReader(raw))
	} else {
		encoded, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, bytes.NewReader(encoded))
	}
	r.Header.Set("Content-Type", "application/json")
	if orgID != nil {
		ctx := context.WithValue(r.Context(), middleware.ContextKeyOrganizationID, *orgID)
		r = r.WithContext(ctx)
	}
	return r
}

// sampleCatalogEntry returns a deterministic curated entry used across
// the read-endpoint tests.
func sampleCatalogEntry(text string, curated bool, usage int) *domainskill.CatalogEntry {
	return &domainskill.CatalogEntry{
		SkillText:     text,
		DisplayText:   text,
		ExpertiseKeys: []string{"development"},
		IsCurated:     curated,
		UsageCount:    usage,
	}
}

// --- GET /skills/catalog --------------------------------------------

func TestSkillHandler_GetCuratedByExpertise_Success(t *testing.T) {
	svc := &mockSkillService{
		getCuratedFn: func(_ context.Context, key string, _ int) ([]*domainskill.CatalogEntry, error) {
			assert.Equal(t, "development", key)
			return []*domainskill.CatalogEntry{
				sampleCatalogEntry("react", true, 42),
				sampleCatalogEntry("go", true, 31),
			}, nil
		},
		countCuratedFn: func(_ context.Context, _ string) (int, error) { return 2, nil },
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(http.MethodGet, "/api/v1/skills/catalog?expertise=development&limit=50", nil, nil)
	rec := httptest.NewRecorder()
	h.GetCuratedByExpertise(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, float64(2), body["total"])
	skills, ok := body["skills"].([]any)
	require.True(t, ok)
	assert.Len(t, skills, 2)
}

func TestSkillHandler_GetCuratedByExpertise_InvalidKey(t *testing.T) {
	svc := &mockSkillService{
		getCuratedFn: func(_ context.Context, _ string, _ int) ([]*domainskill.CatalogEntry, error) {
			return nil, domainskill.ErrInvalidExpertiseKey
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(http.MethodGet, "/api/v1/skills/catalog?expertise=bogus", nil, nil)
	rec := httptest.NewRecorder()
	h.GetCuratedByExpertise(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "invalid_expertise_key", body["error"])
}

// --- GET /skills/autocomplete ---------------------------------------

func TestSkillHandler_Autocomplete_Success(t *testing.T) {
	svc := &mockSkillService{
		autocompleteFn: func(_ context.Context, q string, _ int) ([]*domainskill.CatalogEntry, error) {
			assert.Equal(t, "re", q)
			return []*domainskill.CatalogEntry{
				sampleCatalogEntry("react", true, 42),
				sampleCatalogEntry("redis", true, 12),
			}, nil
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(http.MethodGet, "/api/v1/skills/autocomplete?q=re", nil, nil)
	rec := httptest.NewRecorder()
	h.Autocomplete(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Len(t, body, 2)
}

func TestSkillHandler_Autocomplete_EmptyQuery_ReturnsEmptyList(t *testing.T) {
	svc := &mockSkillService{
		autocompleteFn: func(_ context.Context, _ string, _ int) ([]*domainskill.CatalogEntry, error) {
			return []*domainskill.CatalogEntry{}, nil
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(http.MethodGet, "/api/v1/skills/autocomplete", nil, nil)
	rec := httptest.NewRecorder()
	h.Autocomplete(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Len(t, body, 0)
}

// --- PUT /profile/skills --------------------------------------------

func TestSkillHandler_PutMyProfileSkills_Success(t *testing.T) {
	orgID := uuid.New()
	var captured []string
	svc := &mockSkillService{
		replaceFn: func(_ context.Context, in appskill.ReplaceProfileSkillsInput) error {
			assert.Equal(t, orgID, in.OrganizationID)
			captured = in.SkillTexts
			return nil
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(
		http.MethodPut,
		"/api/v1/profile/skills",
		map[string]any{"skill_texts": []string{"react", "go"}},
		&orgID,
	)
	rec := httptest.NewRecorder()
	h.PutMyProfileSkills(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"react", "go"}, captured)
}

func TestSkillHandler_PutMyProfileSkills_NoOrgContext_Unauthorized(t *testing.T) {
	h := NewSkillHandler(&mockSkillService{})

	req := newSkillRequest(
		http.MethodPut,
		"/api/v1/profile/skills",
		map[string]any{"skill_texts": []string{"react"}},
		nil,
	)
	rec := httptest.NewRecorder()
	h.PutMyProfileSkills(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSkillHandler_PutMyProfileSkills_TooManySkills(t *testing.T) {
	orgID := uuid.New()
	svc := &mockSkillService{
		replaceFn: func(_ context.Context, _ appskill.ReplaceProfileSkillsInput) error {
			return domainskill.ErrTooManySkills
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(
		http.MethodPut,
		"/api/v1/profile/skills",
		map[string]any{"skill_texts": []string{"a", "b"}},
		&orgID,
	)
	rec := httptest.NewRecorder()
	h.PutMyProfileSkills(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "too_many_skills", body["error"])
}

func TestSkillHandler_PutMyProfileSkills_DisabledForOrgType(t *testing.T) {
	orgID := uuid.New()
	svc := &mockSkillService{
		replaceFn: func(_ context.Context, _ appskill.ReplaceProfileSkillsInput) error {
			return domainskill.ErrSkillsDisabledForOrgType
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(
		http.MethodPut,
		"/api/v1/profile/skills",
		map[string]any{"skill_texts": []string{"a"}},
		&orgID,
	)
	rec := httptest.NewRecorder()
	h.PutMyProfileSkills(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "skills_disabled", body["error"])
}

func TestSkillHandler_PutMyProfileSkills_InvalidBody(t *testing.T) {
	orgID := uuid.New()
	h := NewSkillHandler(&mockSkillService{})

	req := newSkillRequest(
		http.MethodPut,
		"/api/v1/profile/skills",
		[]byte("{bad json"),
		&orgID,
	)
	rec := httptest.NewRecorder()
	h.PutMyProfileSkills(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "invalid_body", body["error"])
}

// --- GET /profile/skills --------------------------------------------

func TestSkillHandler_GetMyProfileSkills_Success(t *testing.T) {
	orgID := uuid.New()
	svc := &mockSkillService{
		getProfileFn: func(_ context.Context, requested uuid.UUID) ([]*domainskill.ProfileSkill, error) {
			assert.Equal(t, orgID, requested)
			return []*domainskill.ProfileSkill{
				{OrganizationID: orgID, SkillText: "react", Position: 0},
				{OrganizationID: orgID, SkillText: "go", Position: 1},
			}, nil
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(http.MethodGet, "/api/v1/profile/skills", nil, &orgID)
	rec := httptest.NewRecorder()
	h.GetMyProfileSkills(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Len(t, body, 2)
	assert.Equal(t, "react", body[0]["skill_text"])
	assert.Equal(t, float64(0), body[0]["position"])
}

func TestSkillHandler_GetMyProfileSkills_Unauthorized(t *testing.T) {
	h := NewSkillHandler(&mockSkillService{})

	req := newSkillRequest(http.MethodGet, "/api/v1/profile/skills", nil, nil)
	rec := httptest.NewRecorder()
	h.GetMyProfileSkills(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- POST /skills ----------------------------------------------------

func TestSkillHandler_CreateUserSkill_Success(t *testing.T) {
	svc := &mockSkillService{
		createFn: func(_ context.Context, in appskill.CreateUserSkillInput) (*domainskill.CatalogEntry, error) {
			assert.Equal(t, "Awesome Skill", in.DisplayText)
			return &domainskill.CatalogEntry{
				SkillText:     "awesome skill",
				DisplayText:   "Awesome Skill",
				ExpertiseKeys: []string{},
				IsCurated:     false,
				UsageCount:    0,
			}, nil
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(
		http.MethodPost,
		"/api/v1/skills",
		map[string]any{"display_text": "Awesome Skill"},
		nil,
	)
	rec := httptest.NewRecorder()
	h.CreateUserSkill(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "awesome skill", body["skill_text"])
	assert.Equal(t, "Awesome Skill", body["display_text"])
}

func TestSkillHandler_CreateUserSkill_InvalidBody(t *testing.T) {
	h := NewSkillHandler(&mockSkillService{})

	req := newSkillRequest(
		http.MethodPost,
		"/api/v1/skills",
		[]byte("{broken"),
		nil,
	)
	rec := httptest.NewRecorder()
	h.CreateUserSkill(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "invalid_body", body["error"])
}

func TestSkillHandler_CreateUserSkill_InvalidDisplayText(t *testing.T) {
	svc := &mockSkillService{
		createFn: func(_ context.Context, _ appskill.CreateUserSkillInput) (*domainskill.CatalogEntry, error) {
			return nil, domainskill.ErrInvalidDisplayText
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(
		http.MethodPost,
		"/api/v1/skills",
		map[string]any{"display_text": ""},
		nil,
	)
	rec := httptest.NewRecorder()
	h.CreateUserSkill(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "invalid_display_text", body["error"])
}

// --- error mapping smoke test ---------------------------------------

func TestSkillHandler_HandleError_UnknownErrorReturns500(t *testing.T) {
	svc := &mockSkillService{
		autocompleteFn: func(_ context.Context, _ string, _ int) ([]*domainskill.CatalogEntry, error) {
			return nil, errors.New("database explosion")
		},
	}
	h := NewSkillHandler(svc)

	req := newSkillRequest(http.MethodGet, "/api/v1/skills/autocomplete?q=re", nil, nil)
	rec := httptest.NewRecorder()
	h.Autocomplete(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
