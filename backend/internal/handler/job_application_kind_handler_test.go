package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jobapp "marketplace-backend/internal/app/job"
	jobdomain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// kindApplicationRepo is a focused mock of the JobApplicationRepository
// surface this test file exercises (Create + ListByJob). Unrelated
// methods are no-ops because the kind+filter flow doesn't touch them.
//
// The file lives under handler/ rather than reusing the app/job/ test
// mocks because Go test packages are not transitive — handler/ tests
// need their own implementation of the port interface.
type kindApplicationRepo struct {
	createdApps []*jobdomain.JobApplication
	createErr   error

	listResult     []*jobdomain.JobApplication
	listLastKind   jobdomain.ApplicantKind
	listLastJobID  uuid.UUID
	listNextCursor string
	listErr        error
}

func (m *kindApplicationRepo) Create(_ context.Context, app *jobdomain.JobApplication) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdApps = append(m.createdApps, app)
	return nil
}

func (m *kindApplicationRepo) GetByID(_ context.Context, _ uuid.UUID) (*jobdomain.JobApplication, error) {
	return nil, jobdomain.ErrApplicationNotFound
}

func (m *kindApplicationRepo) GetByJobAndApplicant(_ context.Context, _, _ uuid.UUID) (*jobdomain.JobApplication, error) {
	return nil, jobdomain.ErrApplicationNotFound
}

func (m *kindApplicationRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *kindApplicationRepo) ListByJob(_ context.Context, jobID uuid.UUID, _ string, _ int, kindFilter jobdomain.ApplicantKind) ([]*jobdomain.JobApplication, string, error) {
	m.listLastJobID = jobID
	m.listLastKind = kindFilter
	if m.listErr != nil {
		return nil, "", m.listErr
	}
	return m.listResult, m.listNextCursor, nil
}

func (m *kindApplicationRepo) ListByApplicantOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*jobdomain.JobApplication, string, error) {
	return nil, "", nil
}

func (m *kindApplicationRepo) CountByJob(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *kindApplicationRepo) ListAdmin(_ context.Context, _ repository.AdminApplicationFilters) ([]repository.AdminJobApplication, string, error) {
	return nil, "", nil
}

func (m *kindApplicationRepo) CountAdmin(_ context.Context, _ repository.AdminApplicationFilters) (int, error) {
	return 0, nil
}

var _ repository.JobApplicationRepository = (*kindApplicationRepo)(nil)

func newApplyHandlerWithRepos(jobRepo *mockJobRepo, userRepo *mockUserRepo, appRepo repository.JobApplicationRepository) *JobApplicationHandler {
	svc := jobapp.NewService(jobapp.ServiceDeps{
		Jobs:         jobRepo,
		Users:        userRepo,
		Applications: appRepo,
	})
	return NewJobApplicationHandler(svc)
}

func openTestJob(creatorID uuid.UUID) *jobdomain.Job {
	j, _ := jobdomain.NewJob(jobdomain.NewJobInput{
		CreatorID:     creatorID,
		Title:         "Backend job",
		Description:   "We need a backend dev",
		ApplicantType: jobdomain.ApplicantAll,
		BudgetType:    jobdomain.BudgetOneShot,
		MinBudget:     1000,
		MaxBudget:     2000,
	})
	return j
}

func chiRouteCtx(jobID uuid.UUID) *chi.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	return rctx
}

func TestApplyHandler_AcceptsApplicantKind_Referrer(t *testing.T) {
	jobRepo := &mockJobRepo{}
	userRepo := &mockUserRepo{}
	appRepo := &kindApplicationRepo{}
	h := newApplyHandlerWithRepos(jobRepo, userRepo, appRepo)

	creatorID := uuid.New()
	applicantID := uuid.New()
	orgID := uuid.New()
	j := openTestJob(creatorID)

	jobRepo.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) { return j, nil }
	userRepo.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &orgID, ReferrerEnabled: true}, nil
	}

	body := map[string]any{"message": "I bring a great freelance", "applicant_kind": "referrer"}
	buf, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/jobs/"+j.ID.String()+"/apply", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, applicantID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chiRouteCtx(j.ID))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ApplyToJob(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, appRepo.createdApps, 1)
	assert.Equal(t, jobdomain.ApplicantKindReferrer, appRepo.createdApps[0].ApplicantKind)
}

func TestApplyHandler_RejectsReferrerKindWhenFlagOff(t *testing.T) {
	jobRepo := &mockJobRepo{}
	userRepo := &mockUserRepo{}
	appRepo := &kindApplicationRepo{}
	h := newApplyHandlerWithRepos(jobRepo, userRepo, appRepo)

	creatorID := uuid.New()
	applicantID := uuid.New()
	orgID := uuid.New()
	j := openTestJob(creatorID)

	jobRepo.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) { return j, nil }
	userRepo.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &orgID, ReferrerEnabled: false}, nil
	}

	body := map[string]any{"message": "I should not be allowed", "applicant_kind": "referrer"}
	buf, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/jobs/"+j.ID.String()+"/apply", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, applicantID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chiRouteCtx(j.ID))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ApplyToJob(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	assert.Empty(t, appRepo.createdApps, "no application should be created when the kind is rejected")
	assert.Contains(t, rec.Body.String(), "invalid_applicant_kind")
}

func TestApplyHandler_DefaultsToFreelanceForProvider(t *testing.T) {
	jobRepo := &mockJobRepo{}
	userRepo := &mockUserRepo{}
	appRepo := &kindApplicationRepo{}
	h := newApplyHandlerWithRepos(jobRepo, userRepo, appRepo)

	creatorID := uuid.New()
	applicantID := uuid.New()
	orgID := uuid.New()
	j := openTestJob(creatorID)

	jobRepo.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) { return j, nil }
	userRepo.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &orgID}, nil
	}

	// No applicant_kind sent — older clients (mobile pre-update) keep
	// working with the default kind.
	body := map[string]any{"message": "Default kind"}
	buf, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/jobs/"+j.ID.String()+"/apply", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, applicantID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chiRouteCtx(j.ID))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ApplyToJob(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, appRepo.createdApps, 1)
	assert.Equal(t, jobdomain.ApplicantKindFreelance, appRepo.createdApps[0].ApplicantKind)

	// And the response carries the persisted kind so the caller knows
	// what the backend stored.
	assert.Contains(t, rec.Body.String(), `"applicant_kind":"freelance"`)
}

func TestListJobApplicationsHandler_ForwardsKindQuery(t *testing.T) {
	jobRepo := &mockJobRepo{}
	userRepo := &mockUserRepo{}
	appRepo := &kindApplicationRepo{}
	h := newApplyHandlerWithRepos(jobRepo, userRepo, appRepo)

	ownerID := uuid.New()
	j := openTestJob(ownerID)

	jobRepo.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) { return j, nil }

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+j.ID.String()+"/applications?kind=referrer", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, ownerID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chiRouteCtx(j.ID))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ListJobApplications(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, jobdomain.ApplicantKindReferrer, appRepo.listLastKind)
}

func TestListJobApplicationsHandler_NoKindParam_ReturnsAll(t *testing.T) {
	jobRepo := &mockJobRepo{}
	userRepo := &mockUserRepo{}
	appRepo := &kindApplicationRepo{}
	h := newApplyHandlerWithRepos(jobRepo, userRepo, appRepo)

	ownerID := uuid.New()
	j := openTestJob(ownerID)
	jobRepo.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) { return j, nil }

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+j.ID.String()+"/applications", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, ownerID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chiRouteCtx(j.ID))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ListJobApplications(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, jobdomain.ApplicantKind(""), appRepo.listLastKind, "no filter when kind query is absent")
}

func TestListJobApplicationsHandler_RejectsUnknownKind(t *testing.T) {
	jobRepo := &mockJobRepo{}
	userRepo := &mockUserRepo{}
	appRepo := &kindApplicationRepo{}
	h := newApplyHandlerWithRepos(jobRepo, userRepo, appRepo)

	ownerID := uuid.New()
	j := openTestJob(ownerID)
	jobRepo.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) { return j, nil }

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+j.ID.String()+"/applications?kind=hacker", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, ownerID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chiRouteCtx(j.ID))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ListJobApplications(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	assert.Contains(t, strings.ToLower(rec.Body.String()), "invalid_applicant_kind")
}
