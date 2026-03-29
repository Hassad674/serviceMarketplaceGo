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
)

func newTestJobHandler(jobRepo *mockJobRepo, userRepo *mockUserRepo) *JobHandler {
	svc := jobapp.NewService(jobapp.ServiceDeps{Jobs: jobRepo, Users: userRepo})
	return NewJobHandler(svc)
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

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockJobRepo)
		wantStatus int
	}{
		{
			name:   "success with results",
			userID: &uid,
			setupMock: func(r *mockJobRepo) {
				r.listByCreatorFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*jobdomain.Job, string, error) {
					return []*jobdomain.Job{testJob(uid)}, "next_123", nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "success empty",
			userID:     &uid,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
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
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
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
