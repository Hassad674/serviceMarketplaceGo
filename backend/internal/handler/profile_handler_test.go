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
	return NewProfileHandler(svc)
}

func TestProfileHandler_GetMyProfile(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockProfileRepo)
		wantStatus int
	}{
		{
			name:   "success",
			userID: &uid,
			setupMock: func(r *mockProfileRepo) {
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "profile not found",
			userID:     &uid,
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
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.GetMyProfile(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestProfileHandler_UpdateMyProfile(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		body       map[string]string
		setupMock  func(*mockProfileRepo)
		wantStatus int
	}{
		{
			name:   "success",
			userID: &uid,
			body:   map[string]string{"title": "Go Expert", "about": "I build APIs"},
			setupMock: func(r *mockProfileRepo) {
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			body:       map[string]string{"title": "X"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json",
			userID:     &uid,
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
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
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
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
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
			rctx.URLParams.Add("userId", tc.urlParam)
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
	}{
		{
			name:  "returns results",
			query: "?type=freelancer",
			setupMock: func(r *mockProfileRepo) {
				r.searchPublicFn = func(_ context.Context, _ string, _ bool, _ int) ([]*profile.PublicProfile, error) {
					return []*profile.PublicProfile{{
						UserID: uuid.New(), DisplayName: "Jane", Role: "provider",
					}}, nil
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
				r.searchPublicFn = func(_ context.Context, _ string, _ bool, limit int) ([]*profile.PublicProfile, error) {
					require.Equal(t, 5, limit)
					return []*profile.PublicProfile{}, nil
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

			var resp []any
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Len(t, resp, tc.wantLen)
		})
	}
}
