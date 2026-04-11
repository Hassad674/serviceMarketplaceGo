package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
)

// ---------------------------------------------------------------------------
// Mock specific to social link tests
// ---------------------------------------------------------------------------

type mockSocialLinkRepo struct {
	listByOrgFn func(ctx context.Context, orgID uuid.UUID) ([]*profile.SocialLink, error)
	upsertFn    func(ctx context.Context, link *profile.SocialLink) error
	deleteFn    func(ctx context.Context, orgID uuid.UUID, platform string) error
}

func (m *mockSocialLinkRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*profile.SocialLink, error) {
	if m.listByOrgFn != nil {
		return m.listByOrgFn(ctx, orgID)
	}
	return []*profile.SocialLink{}, nil
}

func (m *mockSocialLinkRepo) Upsert(ctx context.Context, link *profile.SocialLink) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, link)
	}
	return nil
}

func (m *mockSocialLinkRepo) Delete(ctx context.Context, orgID uuid.UUID, platform string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, orgID, platform)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestSocialLinkHandler(repo *mockSocialLinkRepo) *SocialLinkHandler {
	svc := profileapp.NewSocialLinkService(repo)
	return NewSocialLinkHandler(svc)
}

func testSocialLink(orgID uuid.UUID, platform, url string) *profile.SocialLink {
	return &profile.SocialLink{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Platform:       platform,
		URL:            url,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func withOrgCtx(req *http.Request, orgID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	return req.WithContext(ctx)
}

func TestSocialLinkHandler_ListMySocialLinks(t *testing.T) {
	oid := uuid.New()

	tests := []struct {
		name       string
		orgID      *uuid.UUID
		setupMock  func(*mockSocialLinkRepo)
		wantStatus int
		wantLen    int
	}{
		{
			name:  "success with links",
			orgID: &oid,
			setupMock: func(r *mockSocialLinkRepo) {
				r.listByOrgFn = func(_ context.Context, _ uuid.UUID) ([]*profile.SocialLink, error) {
					return []*profile.SocialLink{
						testSocialLink(oid, "github", "https://github.com/user"),
						testSocialLink(oid, "linkedin", "https://linkedin.com/in/user"),
					}, nil
				}
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "success empty",
			orgID:      &oid,
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "unauthenticated",
			orgID:      nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:  "internal error",
			orgID: &oid,
			setupMock: func(r *mockSocialLinkRepo) {
				r.listByOrgFn = func(_ context.Context, _ uuid.UUID) ([]*profile.SocialLink, error) {
					return nil, errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSocialLinkRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestSocialLinkHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/social-links", nil)
			if tc.orgID != nil {
				req = withOrgCtx(req, *tc.orgID)
			}
			rec := httptest.NewRecorder()

			h.ListMySocialLinks(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantStatus == http.StatusOK {
				var resp []json.RawMessage
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Len(t, resp, tc.wantLen)
			}
		})
	}
}

func TestSocialLinkHandler_UpsertSocialLink(t *testing.T) {
	oid := uuid.New()

	tests := []struct {
		name       string
		orgID      *uuid.UUID
		body       map[string]string
		setupMock  func(*mockSocialLinkRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "success",
			orgID:      &oid,
			body:       map[string]string{"platform": "github", "url": "https://github.com/user"},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "success with uppercase platform",
			orgID:      &oid,
			body:       map[string]string{"platform": "GitHub", "url": "https://github.com/user"},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid platform",
			orgID:      &oid,
			body:       map[string]string{"platform": "tiktok", "url": "https://tiktok.com/@user"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_platform",
		},
		{
			name:       "invalid url",
			orgID:      &oid,
			body:       map[string]string{"platform": "github", "url": "not-a-url"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_url",
		},
		{
			name:       "empty url",
			orgID:      &oid,
			body:       map[string]string{"platform": "github", "url": ""},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_url",
		},
		{
			name:       "unauthenticated",
			orgID:      nil,
			body:       map[string]string{"platform": "github", "url": "https://github.com/u"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json",
			orgID:      &oid,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:  "repo error",
			orgID: &oid,
			body:  map[string]string{"platform": "github", "url": "https://github.com/user"},
			setupMock: func(r *mockSocialLinkRepo) {
				r.upsertFn = func(_ context.Context, _ *profile.SocialLink) error {
					return errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSocialLinkRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestSocialLinkHandler(repo)

			var bodyReader *bytes.Reader
			if tc.body != nil {
				b, _ := json.Marshal(tc.body)
				bodyReader = bytes.NewReader(b)
			} else {
				bodyReader = bytes.NewReader([]byte("{bad"))
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/social-links", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			if tc.orgID != nil {
				req = withOrgCtx(req, *tc.orgID)
			}
			rec := httptest.NewRecorder()

			h.UpsertSocialLink(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestSocialLinkHandler_DeleteSocialLink(t *testing.T) {
	oid := uuid.New()

	tests := []struct {
		name       string
		orgID      *uuid.UUID
		platform   string
		setupMock  func(*mockSocialLinkRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "success",
			orgID:      &oid,
			platform:   "github",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid platform",
			orgID:      &oid,
			platform:   "tiktok",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_platform",
		},
		{
			name:       "unauthenticated",
			orgID:      nil,
			platform:   "github",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:     "repo error",
			orgID:    &oid,
			platform: "github",
			setupMock: func(r *mockSocialLinkRepo) {
				r.deleteFn = func(_ context.Context, _ uuid.UUID, _ string) error {
					return errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSocialLinkRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestSocialLinkHandler(repo)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/social-links/"+tc.platform, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("platform", tc.platform)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			if tc.orgID != nil {
				ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, *tc.orgID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.DeleteSocialLink(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestSocialLinkHandler_ListPublicSocialLinks(t *testing.T) {
	targetOrg := uuid.New()

	tests := []struct {
		name       string
		orgParam   string
		setupMock  func(*mockSocialLinkRepo)
		wantStatus int
		wantLen    int
		wantCode   string
	}{
		{
			name:     "success",
			orgParam: targetOrg.String(),
			setupMock: func(r *mockSocialLinkRepo) {
				r.listByOrgFn = func(_ context.Context, _ uuid.UUID) ([]*profile.SocialLink, error) {
					return []*profile.SocialLink{
						testSocialLink(targetOrg, "github", "https://github.com/u"),
					}, nil
				}
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:     "success empty",
			orgParam: targetOrg.String(),
			setupMock: func(r *mockSocialLinkRepo) {
				r.listByOrgFn = func(_ context.Context, _ uuid.UUID) ([]*profile.SocialLink, error) {
					return []*profile.SocialLink{}, nil
				}
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "invalid uuid",
			orgParam:   "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_org_id",
		},
		{
			name:     "internal error",
			orgParam: targetOrg.String(),
			setupMock: func(r *mockSocialLinkRepo) {
				r.listByOrgFn = func(_ context.Context, _ uuid.UUID) ([]*profile.SocialLink, error) {
					return nil, errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSocialLinkRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestSocialLinkHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+tc.orgParam+"/social-links", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orgId", tc.orgParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()

			h.ListPublicSocialLinks(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}

			if tc.wantStatus == http.StatusOK {
				var resp []json.RawMessage
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Len(t, resp, tc.wantLen)
			}
		})
	}
}
