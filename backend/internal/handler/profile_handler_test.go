package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
)

func newTestProfileHandler(repo *mockProfileRepo) *ProfileHandler {
	svc := profileapp.NewService(repo)
	// Unit tests for the baseline profile endpoints do not exercise
	// the expertise flow — pass nil so the handler returns an empty
	// expertise list in the response without requiring a second mock.
	return NewProfileHandler(svc, nil)
}

func TestProfileHandler_GetMyProfile(t *testing.T) {
	oid := uuid.New()

	tests := []struct {
		name       string
		orgID      *uuid.UUID
		setupMock  func(*mockProfileRepo)
		wantStatus int
	}{
		{
			name:  "success",
			orgID: &oid,
			setupMock: func(r *mockProfileRepo) {
				r.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(oid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			orgID:      nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "profile not found",
			orgID:      &oid,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockProfileRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestProfileHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/me", nil)
			if tc.orgID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, *tc.orgID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.GetMyProfile(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestProfileHandler_UpdateMyProfile(t *testing.T) {
	oid := uuid.New()

	tests := []struct {
		name       string
		orgID      *uuid.UUID
		body       map[string]string
		setupMock  func(*mockProfileRepo)
		wantStatus int
	}{
		{
			name:  "success",
			orgID: &oid,
			body:  map[string]string{"title": "Go Expert", "about": "I build APIs"},
			setupMock: func(r *mockProfileRepo) {
				r.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(oid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			orgID:      nil,
			body:       map[string]string{"title": "X"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json",
			orgID:      &oid,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockProfileRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestProfileHandler(repo)

			var bodyReader *bytes.Reader
			if tc.body != nil {
				b, _ := json.Marshal(tc.body)
				bodyReader = bytes.NewReader(b)
			} else {
				bodyReader = bytes.NewReader([]byte("{bad"))
			}
			req := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/me", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			if tc.orgID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, *tc.orgID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.UpdateMyProfile(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestProfileHandler_GetPublicProfile(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		urlParam   string
		setupMock  func(*mockProfileRepo)
		wantStatus int
	}{
		{
			name:     "success",
			urlParam: uid.String(),
			setupMock: func(r *mockProfileRepo) {
				r.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
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
			urlParam:   uid.String(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockProfileRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestProfileHandler(repo)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/profiles/%s", tc.urlParam), nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orgId", tc.urlParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()

			h.GetPublicProfile(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestProfileHandler_SearchProfiles(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setupMock  func(*mockProfileRepo)
		wantStatus int
		wantLen    int
		wantMore   bool
	}{
		{
			name:  "returns results",
			query: "?type=freelancer",
			setupMock: func(r *mockProfileRepo) {
				r.searchPublicFn = func(_ context.Context, _ string, _ bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
					return []*profile.PublicProfile{{
						OrganizationID: uuid.New(), Name: "Jane", OrgType: "provider_personal",
					}}, "", nil
				}
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "empty results",
			query:      "?type=agency",
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "custom limit",
			query: "?limit=5",
			setupMock: func(r *mockProfileRepo) {
				r.searchPublicFn = func(_ context.Context, _ string, _ bool, _ string, limit int) ([]*profile.PublicProfile, string, error) {
					require.Equal(t, 5, limit)
					return []*profile.PublicProfile{}, "", nil
				}
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockProfileRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestProfileHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/search"+tc.query, nil)
			rec := httptest.NewRecorder()

			h.SearchProfiles(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			var resp map[string]any
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			data, ok := resp["data"].([]any)
			require.True(t, ok, "response should contain data array")
			assert.Len(t, data, tc.wantLen)
			assert.Equal(t, tc.wantMore, resp["has_more"])
		})
	}
}
