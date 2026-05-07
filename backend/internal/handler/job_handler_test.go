package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newTestJobHandler(jobRepo *mockJobRepo, userRepo *mockUserRepo) *JobHandler {
	svc := jobapp.NewService(jobapp.ServiceDeps{Jobs: jobRepo, Users: userRepo})
	return NewJobHandler(svc)
}

// newTestJobApplicationHandler wires the JobApplicationHandler with the
// JobView mock so /jobs/open tests can exercise the social-proof
// counts path end-to-end (handler → service → batch helper).
func newTestJobApplicationHandler(jobRepo *mockJobRepo, viewRepo *mockJobViewRepo) *JobApplicationHandler {
	svc := jobapp.NewService(jobapp.ServiceDeps{
		Jobs:     jobRepo,
		Users:    &mockUserRepo{},
		JobViews: viewRepo,
	})
	return NewJobApplicationHandler(svc)
}

func TestJobHandler_CreateJob(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		body       map[string]any
		setupMocks func(*mockUserRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:   "success",
			userID: &uid,
			body: map[string]any{
				"title": "Go Developer", "description": "Build APIs",
				"applicant_type": "all", "budget_type": "one_shot",
				"min_budget": 1000, "max_budget": 5000, "description_type": "text",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return testUser(uid, user.RoleEnterprise), nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			body:       map[string]any{"title": "X"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "empty title",
			userID: &uid,
			body: map[string]any{
				"title": "", "description": "X",
				"applicant_type": "all", "budget_type": "one_shot",
				"min_budget": 100, "max_budget": 500, "description_type": "text",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return testUser(uid, user.RoleEnterprise), nil
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "empty_title",
		},
		{
			name:   "invalid budget negative",
			userID: &uid,
			body: map[string]any{
				"title": "Job", "description": "Desc",
				"applicant_type": "all", "budget_type": "one_shot",
				"min_budget": -1, "max_budget": 500, "description_type": "text",
			},
			setupMocks: func(ur *mockUserRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return testUser(uid, user.RoleEnterprise), nil
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_budget",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jobRepo := &mockJobRepo{}
			userRepo := &mockUserRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(userRepo)
			}
			h := newTestJobHandler(jobRepo, userRepo)

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.CreateJob(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestJobHandler_GetJob(t *testing.T) {
	jobID := uuid.New()
	creatorID := uuid.New()

	tests := []struct {
		name       string
		urlParam   string
		setupMock  func(*mockJobRepo)
		wantStatus int
	}{
		{
			name:     "success",
			urlParam: jobID.String(),
			setupMock: func(r *mockJobRepo) {
				r.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) {
					j := testJob(creatorID)
					j.ID = jobID
					return j, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			urlParam:   "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			urlParam:   jobID.String(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jobRepo := &mockJobRepo{}
			if tc.setupMock != nil {
				tc.setupMock(jobRepo)
			}
			h := newTestJobHandler(jobRepo, &mockUserRepo{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+tc.urlParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.urlParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()

			h.GetJob(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestJobHandler_ListMyJobs(t *testing.T) {
	uid := uuid.New()
	oid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		orgID      *uuid.UUID
		setupMock  func(*mockJobRepo)
		wantStatus int
	}{
		{
			name:   "success with results",
			userID: &uid,
			orgID:  &oid,
			setupMock: func(r *mockJobRepo) {
				r.listByOrgFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*jobdomain.Job, string, error) {
					return []*jobdomain.Job{testJob(uid)}, "next_123", nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "success empty",
			userID:     &uid,
			orgID:      &oid,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing org",
			userID:     &uid,
			orgID:      nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jobRepo := &mockJobRepo{}
			if tc.setupMock != nil {
				tc.setupMock(jobRepo)
			}
			h := newTestJobHandler(jobRepo, &mockUserRepo{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
			ctx := req.Context()
			if tc.userID != nil {
				ctx = context.WithValue(ctx, middleware.ContextKeyUserID, *tc.userID)
			}
			if tc.orgID != nil {
				ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, *tc.orgID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.ListMyJobs(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestJobHandler_CloseJob(t *testing.T) {
	uid := uuid.New()
	otherUID := uuid.New()
	jobID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMock  func(*mockJobRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success",
			userID:   &uid,
			urlParam: jobID.String(),
			setupMock: func(r *mockJobRepo) {
				r.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) {
					j := testJob(uid)
					j.ID = jobID
					return j, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "not owner",
			userID:   &otherUID,
			urlParam: jobID.String(),
			setupMock: func(r *mockJobRepo) {
				r.getByIDFn = func(_ context.Context, _ uuid.UUID) (*jobdomain.Job, error) {
					j := testJob(uid) // owned by uid, not otherUID
					j.ID = jobID
					return j, nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "not_owner",
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			urlParam:   jobID.String(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid uuid",
			userID:     &uid,
			urlParam:   "bad",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jobRepo := &mockJobRepo{}
			if tc.setupMock != nil {
				tc.setupMock(jobRepo)
			}
			h := newTestJobHandler(jobRepo, &mockUserRepo{})

			req := httptest.NewRequest(http.MethodPut, "/api/v1/jobs/"+tc.urlParam+"/close", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.urlParam)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			if tc.userID != nil {
				ctx = context.WithValue(ctx, middleware.ContextKeyUserID, *tc.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.CloseJob(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

// --- ListOpenJobs (public marketplace feed with social-proof counts) ---

// jobOpenItem mirrors the JobResponse subset the handler surfaces on
// /jobs/open. Defined inline (not imported from dto/response) so the
// test asserts the wire-format JSON, not the Go struct shape.
type jobOpenItem struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	TotalApplicants *int   `json:"total_applicants,omitempty"`
	NewApplicants   *int   `json:"new_applicants,omitempty"` // must NEVER appear
}

type jobOpenListResponse struct {
	Data       []jobOpenItem `json:"data"`
	NextCursor string        `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

func TestJobApplicationHandler_ListOpenJobs_ResponseShape(t *testing.T) {
	creatorID := uuid.New()
	j1 := testJob(creatorID)
	j2 := testJob(creatorID)
	j3 := testJob(creatorID) // zero applicants — must serialise as 0

	jobRepo := &mockJobRepo{
		listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*jobdomain.Job, string, error) {
			return []*jobdomain.Job{j1, j2, j3}, "", nil
		},
	}
	viewRepo := &mockJobViewRepo{
		getApplicationCountsBatchFn: func(_ context.Context, ids []uuid.UUID, viewer uuid.UUID) (map[uuid.UUID]repository.ApplicationCounts, error) {
			// Public feed has no per-user "since I last looked" — viewer
			// must be uuid.Nil here. Pin the contract.
			assert.Equal(t, uuid.Nil, viewer, "public feed must NOT leak a per-user viewer id to the count batch")
			assert.Len(t, ids, 3)
			return map[uuid.UUID]repository.ApplicationCounts{
				j1.ID: {Total: 7, NewCount: 2},
				j2.ID: {Total: 0, NewCount: 0},
				// j3 absent — handler must surface 0
			}, nil
		},
	}
	h := newTestJobApplicationHandler(jobRepo, viewRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/open", nil)
	rec := httptest.NewRecorder()
	h.ListOpenJobs(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	rawBody := rec.Body.Bytes()

	var got jobOpenListResponse
	require.NoError(t, json.Unmarshal(rawBody, &got))
	require.Len(t, got.Data, 3)

	// Every item exposes total_applicants (zero is a valid public value).
	for i, item := range got.Data {
		require.NotNil(t, item.TotalApplicants, "item %d must expose total_applicants", i)
	}
	assert.Equal(t, j1.ID.String(), got.Data[0].ID)
	assert.Equal(t, 7, *got.Data[0].TotalApplicants)
	assert.Equal(t, j2.ID.String(), got.Data[1].ID)
	assert.Equal(t, 0, *got.Data[1].TotalApplicants)
	assert.Equal(t, j3.ID.String(), got.Data[2].ID)
	assert.Equal(t, 0, *got.Data[2].TotalApplicants)

	// Critical privacy invariant: the public feed must NEVER expose
	// new_applicants — that semantic is owner-only ("new since I last
	// looked at my own job's candidatures").
	for i, item := range got.Data {
		assert.Nil(t, item.NewApplicants, "item %d must NOT expose new_applicants on the public feed", i)
	}

	// Confirm raw JSON keys: belt-and-braces guard against silent
	// renames or accidental field exposure.
	var rawDecode struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rawBody, &rawDecode))
	for i, item := range rawDecode.Data {
		_, hasTotal := item["total_applicants"]
		assert.True(t, hasTotal, "item %d JSON must contain total_applicants key", i)
		_, hasNew := item["new_applicants"]
		assert.False(t, hasNew, "item %d JSON must NOT contain new_applicants key", i)
	}
}

func TestJobApplicationHandler_ListOpenJobs_EmptyList(t *testing.T) {
	jobRepo := &mockJobRepo{
		listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*jobdomain.Job, string, error) {
			return []*jobdomain.Job{}, "", nil
		},
	}
	h := newTestJobApplicationHandler(jobRepo, &mockJobViewRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/open", nil)
	rec := httptest.NewRecorder()
	h.ListOpenJobs(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Empty data must serialise as `[]`, never `null`.
	body := rec.Body.String()
	assert.Contains(t, body, `"data":[]`, "empty list must serialise as `data:[]`, not `data:null`")
}

func TestJobApplicationHandler_ListOpenJobs_RepoError(t *testing.T) {
	jobRepo := &mockJobRepo{
		listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*jobdomain.Job, string, error) {
			return nil, "", assert.AnError
		},
	}
	h := newTestJobApplicationHandler(jobRepo, &mockJobViewRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/open", nil)
	rec := httptest.NewRecorder()
	h.ListOpenJobs(rec, req)

	// Generic 500 — the handler must NOT leak the internal error string.
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
