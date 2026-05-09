package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/profilecompletion"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

// stubCompletionService implements ProfileCompletionService inline so
// each test wires whatever Compute behaviour it needs.
type stubCompletionService struct {
	report *profilecompletion.Report
	err    error

	gotUserID  uuid.UUID
	gotOrgID   uuid.UUID
	gotPersona profilecompletion.Persona
}

func (s *stubCompletionService) ComputeWithPersona(
	_ context.Context,
	userID, orgID uuid.UUID,
	override profilecompletion.Persona,
) (*profilecompletion.Report, error) {
	s.gotUserID = userID
	s.gotOrgID = orgID
	s.gotPersona = override
	return s.report, s.err
}

func TestProfileCompletionHandler_GetMyCompletion_Success(t *testing.T) {
	report := &profilecompletion.Report{
		Role:           "provider",
		Persona:        "freelance",
		Percent:        42,
		TotalSections:  13,
		FilledSections: 5,
		Sections: []profilecompletion.Section{
			{Key: profilecompletion.SectionTitle, Filled: true,
				LabelKey:       "profile.completion.section.title",
				CompletionPath: "/dashboard/profile/edit"},
		},
	}
	stub := &stubCompletionService{report: report}
	h := handler.NewProfileCompletionHandler(stub)

	uid := uuid.New()
	oid := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile/completion", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, oid)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.GetMyCompletion(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "private, max-age=30", rr.Header().Get("Cache-Control"))
	assert.Equal(t, uid, stub.gotUserID)
	assert.Equal(t, oid, stub.gotOrgID)
	assert.Equal(t, profilecompletion.Persona(""), stub.gotPersona,
		"no query param means empty override")

	var got profilecompletion.Report
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, 42, got.Percent)
	assert.Equal(t, 13, got.TotalSections)
	assert.Equal(t, "freelance", got.Persona)
	require.Len(t, got.Sections, 1)
	assert.Equal(t, profilecompletion.SectionTitle, got.Sections[0].Key)
}

func TestProfileCompletionHandler_GetMyCompletion_PersonaOverride_PassedThrough(t *testing.T) {
	report := &profilecompletion.Report{
		Role:           "provider",
		Persona:        "referrer",
		Percent:        25,
		TotalSections:  8,
		FilledSections: 2,
	}
	stub := &stubCompletionService{report: report}
	h := handler.NewProfileCompletionHandler(stub)

	uid := uuid.New()
	oid := uuid.New()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/me/profile/completion?persona=referrer", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, oid)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.GetMyCompletion(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, profilecompletion.PersonaReferrer, stub.gotPersona,
		"the handler must forward the persona query param verbatim")
}

func TestProfileCompletionHandler_GetMyCompletion_MissingUser_Returns401(t *testing.T) {
	h := handler.NewProfileCompletionHandler(&stubCompletionService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile/completion", nil)
	rr := httptest.NewRecorder()
	h.GetMyCompletion(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestProfileCompletionHandler_GetMyCompletion_MissingOrg_Returns401(t *testing.T) {
	h := handler.NewProfileCompletionHandler(&stubCompletionService{})

	uid := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile/completion", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.GetMyCompletion(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestProfileCompletionHandler_GetMyCompletion_ServiceError_Returns500(t *testing.T) {
	stub := &stubCompletionService{err: errors.New("downstream blew up")}
	h := handler.NewProfileCompletionHandler(stub)

	uid := uuid.New()
	oid := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile/completion", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, oid)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.GetMyCompletion(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
