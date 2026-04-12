package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// TestTeamHandler_RoleDefinitions_Unauthorized verifies the endpoint
// rejects callers without an authenticated user id in context.
func TestTeamHandler_RoleDefinitions_Unauthorized(t *testing.T) {
	h := NewTeamHandler(TeamHandlerDeps{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/role-definitions", nil)
	rec := httptest.NewRecorder()

	h.RoleDefinitions(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestTeamHandler_RoleDefinitions_AuthenticatedReturnsCatalogue checks
// that an authenticated request receives the full role + permission
// catalogue, with all four V1 roles and at least the team.* permissions.
func TestTeamHandler_RoleDefinitions_AuthenticatedReturnsCatalogue(t *testing.T) {
	h := NewTeamHandler(TeamHandlerDeps{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/role-definitions", nil)
	// Inject a fake user id into the request context to satisfy the
	// auth gate. The actual middleware does this from the JWT.
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uuid.New())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.RoleDefinitions(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body response.RoleDefinitionsPayload
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	// Four V1 roles, in display order.
	assert.Len(t, body.Roles, 4)
	assert.Equal(t, "owner", body.Roles[0].Key)
	assert.Equal(t, "admin", body.Roles[1].Key)
	assert.Equal(t, "member", body.Roles[2].Key)
	assert.Equal(t, "viewer", body.Roles[3].Key)

	// Every role has a label + description filled in.
	for _, r := range body.Roles {
		assert.NotEmpty(t, r.Label, "role %s missing label", r.Key)
		assert.NotEmpty(t, r.Description, "role %s missing description", r.Key)
	}

	// Owner has the most permissions of any role.
	for _, other := range body.Roles[1:] {
		assert.GreaterOrEqual(t, len(body.Roles[0].Permissions), len(other.Permissions))
	}

	// Permission catalogue contains the team.* family.
	keys := make(map[string]bool)
	for _, p := range body.Permissions {
		keys[p.Key] = true
		assert.NotEmpty(t, p.Label, "permission %s missing label", p.Key)
		assert.NotEmpty(t, p.Group, "permission %s missing group", p.Key)
	}
	for _, expected := range []string{
		"team.view", "team.invite", "team.manage", "team.transfer_ownership",
	} {
		assert.True(t, keys[expected], "missing permission key %s", expected)
	}
}

// TestNewMemberListResponseWithUsers_HydratesIdentity verifies the
// response builder attaches the user identity block when a matching
// user is found, and leaves it nil when the user is missing (e.g.
// deleted user race condition).
func TestNewMemberListResponseWithUsers_HydratesIdentity(t *testing.T) {
	orgID := uuid.New()
	aliceID := uuid.New()
	bobID := uuid.New()

	members := []*organization.Member{
		{
			ID:             uuid.New(),
			OrganizationID: orgID,
			UserID:         aliceID,
			Role:           organization.RoleOwner,
			Title:          "",
			JoinedAt:       time.Now(),
		},
		{
			ID:             uuid.New(),
			OrganizationID: orgID,
			UserID:         bobID,
			Role:           organization.RoleMember,
			Title:          "Designer",
			JoinedAt:       time.Now(),
		},
	}

	// Only Alice is in the lookup map — Bob's identity is missing
	// (e.g. user was deleted between the member list query and the
	// batch user fetch).
	alice := &user.User{
		ID:          aliceID,
		Email:       "alice@example.com",
		FirstName:   "Alice",
		LastName:    "Anderson",
		DisplayName: "Alice",
	}
	usersByID := map[string]*user.User{
		aliceID.String(): alice,
	}

	resp := response.NewMemberListResponseWithUsers(members, usersByID, "")
	require.Len(t, resp.Data, 2)

	// Alice's row carries the identity block.
	require.NotNil(t, resp.Data[0].User)
	assert.Equal(t, "alice@example.com", resp.Data[0].User.Email)
	assert.Equal(t, "Alice", resp.Data[0].User.DisplayName)

	// Bob's row falls back to a missing block — the frontend renders
	// the generic label.
	assert.Nil(t, resp.Data[1].User)
}

// ---------------------------------------------------------------------------
// AcceptTransfer — inline session refresh
// ---------------------------------------------------------------------------

// TestTeamHandler_AcceptTransfer_WebMode_RefreshesSession verifies that
// the AcceptTransfer endpoint (web mode):
//   - deletes the old session
//   - creates a new session with the correct SessionVersion and OrgRole
//   - sets the new cookie in the response
//   - returns a /me-style response with updated org context
func TestTeamHandler_AcceptTransfer_WebMode_RefreshesSession(t *testing.T) {
	ownerID := uuid.New()
	accepterID := uuid.New()
	orgID := uuid.New()

	// Build the org in "transfer pending" state targeting the accepter.
	org := &organization.Organization{
		ID:          orgID,
		OwnerUserID: ownerID,
		Type:        organization.OrgTypeAgency,
	}
	org.InitiateTransfer(accepterID, 72*time.Hour)

	// Mock org repo: FindByID returns the transfer-pending org; Update succeeds.
	orgRepo := &mockOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			if id == orgID {
				return org, nil
			}
			return nil, organization.ErrOrgNotFound
		},
	}

	// Mock member repo: returns both the old owner and the accepter members.
	ownerMember := &organization.Member{
		ID: uuid.New(), OrganizationID: orgID, UserID: ownerID,
		Role: organization.RoleOwner, JoinedAt: time.Now(),
	}
	accepterMember := &organization.Member{
		ID: uuid.New(), OrganizationID: orgID, UserID: accepterID,
		Role: organization.RoleAdmin, JoinedAt: time.Now(),
	}
	memberRepo := &mockOrgMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _ uuid.UUID, uid uuid.UUID) (*organization.Member, error) {
			switch uid {
			case ownerID:
				return ownerMember, nil
			case accepterID:
				return accepterMember, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		findUserPrimaryOrgFn: func(_ context.Context, uid uuid.UUID) (*organization.Member, error) {
			if uid == accepterID {
				// After transfer, the accepter is now Owner
				return &organization.Member{
					ID: accepterMember.ID, OrganizationID: orgID, UserID: accepterID,
					Role: organization.RoleOwner, JoinedAt: accepterMember.JoinedAt,
				}, nil
			}
			return nil, organization.ErrMemberNotFound
		},
	}

	// Mock user repo: GetByID returns a fresh user with bumped SessionVersion.
	freshUser := &user.User{
		ID: accepterID, Email: "accepter@example.com",
		FirstName: "Jane", LastName: "Doe", DisplayName: "Jane Doe",
		Role: user.RoleAgency, SessionVersion: 4,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == accepterID {
				return freshUser, nil
			}
			// Return stub for owner id (used by AcceptTransferOwnership notifier)
			return &user.User{ID: id, DisplayName: "Owner"}, nil
		},
	}

	// Track whether old session was deleted and new one was created.
	var deletedSessionID string
	var capturedInput service.CreateSessionInput
	sessionSvc := &mockSessionService{
		deleteFn: func(_ context.Context, sid string) error {
			deletedSessionID = sid
			return nil
		},
		createFn: func(_ context.Context, input service.CreateSessionInput) (*service.Session, error) {
			capturedInput = input
			return &service.Session{
				ID:             "sess_fresh",
				UserID:         input.UserID,
				Role:           input.Role,
				SessionVersion: input.SessionVersion,
				OrgRole:        input.OrgRole,
				OrganizationID: input.OrganizationID,
			}, nil
		},
	}

	// Build real MembershipService with mock repos.
	membershipSvc := orgapp.NewMembershipService(orgapp.MembershipServiceDeps{
		Orgs:    orgRepo,
		Members: memberRepo,
		Users:   userRepo,
	})

	// Build real OrgService (for ResolveContext) with the same mock repos.
	orgService := orgapp.NewService(orgRepo, memberRepo, nil)

	// Build TeamHandler with all deps.
	h := NewTeamHandler(TeamHandlerDeps{
		Membership:     membershipSvc,
		OrgService:     orgService,
		UserBatch:      nil,
		SessionService: sessionSvc,
		Cookie:         testCookieConfig(),
		Users:          userRepo,
	})

	// Build the request: web mode (no X-Auth-Mode header), with existing session cookie.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgID.String()+"/transfer/accept", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "old_session_abc"})

	// Inject auth context.
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, accepterID)
	// Inject chi URL param for orgID.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orgID", orgID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.AcceptTransfer(rec, req)

	// Assertions
	require.Equal(t, http.StatusOK, rec.Code, "expected 200 OK, got %d: %s", rec.Code, rec.Body.String())

	// (a) Old session was deleted
	assert.Equal(t, "old_session_abc", deletedSessionID, "old session should be deleted")

	// (b) New session was created with correct values
	assert.Equal(t, accepterID, capturedInput.UserID)
	assert.Equal(t, "agency", capturedInput.Role)
	assert.Equal(t, 4, capturedInput.SessionVersion, "session must carry the bumped version")
	assert.Equal(t, "owner", capturedInput.OrgRole, "accepter is now the org owner")
	require.NotNil(t, capturedInput.OrganizationID)
	assert.Equal(t, orgID, *capturedInput.OrganizationID)

	// (c) Cookie is set in response
	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session_id" && c.Value == "sess_fresh" {
			found = true
			break
		}
	}
	assert.True(t, found, "new session cookie must be set")

	// (d) Response body is /me-style with user + org
	var meResp response.MeResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&meResp))
	assert.Equal(t, accepterID.String(), meResp.User.ID)
	assert.Equal(t, "agency", meResp.User.Role)
	require.NotNil(t, meResp.Organization)
	assert.Equal(t, orgID.String(), meResp.Organization.ID)
	assert.Equal(t, "owner", meResp.Organization.MemberRole)
}

// TestTeamHandler_AcceptTransfer_MobileMode_NoSessionRefresh verifies
// that mobile clients (X-Auth-Mode: token) receive the plain transfer
// response without any session manipulation.
func TestTeamHandler_AcceptTransfer_MobileMode_NoSessionRefresh(t *testing.T) {
	ownerID := uuid.New()
	accepterID := uuid.New()
	orgID := uuid.New()

	org := &organization.Organization{
		ID:          orgID,
		OwnerUserID: ownerID,
		Type:        organization.OrgTypeAgency,
	}
	org.InitiateTransfer(accepterID, 72*time.Hour)

	orgRepo := &mockOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			if id == orgID {
				return org, nil
			}
			return nil, organization.ErrOrgNotFound
		},
	}

	ownerMember := &organization.Member{
		ID: uuid.New(), OrganizationID: orgID, UserID: ownerID,
		Role: organization.RoleOwner, JoinedAt: time.Now(),
	}
	accepterMember := &organization.Member{
		ID: uuid.New(), OrganizationID: orgID, UserID: accepterID,
		Role: organization.RoleAdmin, JoinedAt: time.Now(),
	}
	memberRepo := &mockOrgMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _ uuid.UUID, uid uuid.UUID) (*organization.Member, error) {
			switch uid {
			case ownerID:
				return ownerMember, nil
			case accepterID:
				return accepterMember, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		findUserPrimaryOrgFn: func(_ context.Context, uid uuid.UUID) (*organization.Member, error) {
			if uid == accepterID {
				return &organization.Member{
					ID: accepterMember.ID, OrganizationID: orgID, UserID: accepterID,
					Role: organization.RoleOwner, JoinedAt: accepterMember.JoinedAt,
				}, nil
			}
			return nil, organization.ErrMemberNotFound
		},
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id, DisplayName: "User", SessionVersion: 4}, nil
		},
	}

	sessionCreated := false
	sessionSvc := &mockSessionService{
		createFn: func(_ context.Context, _ service.CreateSessionInput) (*service.Session, error) {
			sessionCreated = true
			return &service.Session{ID: "should_not_happen"}, nil
		},
	}

	membershipSvc := orgapp.NewMembershipService(orgapp.MembershipServiceDeps{
		Orgs:    orgRepo,
		Members: memberRepo,
		Users:   userRepo,
	})
	orgService := orgapp.NewService(orgRepo, memberRepo, nil)

	h := NewTeamHandler(TeamHandlerDeps{
		Membership:     membershipSvc,
		OrgService:     orgService,
		SessionService: sessionSvc,
		Cookie:         testCookieConfig(),
		Users:          userRepo,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgID.String()+"/transfer/accept", nil)
	req.Header.Set("X-Auth-Mode", "token")

	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, accepterID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orgID", orgID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.AcceptTransfer(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, sessionCreated, "mobile mode must not create a session")

	// No session cookies should be set
	for _, c := range rec.Result().Cookies() {
		assert.NotEqual(t, "session_id", c.Name, "mobile mode must not set session cookies")
	}
}

// ---------------------------------------------------------------------------
// Mock: OrganizationMemberRepository (handler test scope)
// ---------------------------------------------------------------------------

type mockOrgMemberRepo struct {
	findByOrgAndUserFn   func(ctx context.Context, orgID, userID uuid.UUID) (*organization.Member, error)
	findUserPrimaryOrgFn func(ctx context.Context, userID uuid.UUID) (*organization.Member, error)
}

func (m *mockOrgMemberRepo) Create(_ context.Context, _ *organization.Member) error { return nil }
func (m *mockOrgMemberRepo) FindByID(_ context.Context, _ uuid.UUID) (*organization.Member, error) {
	return nil, organization.ErrMemberNotFound
}
func (m *mockOrgMemberRepo) FindByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*organization.Member, error) {
	if m.findByOrgAndUserFn != nil {
		return m.findByOrgAndUserFn(ctx, orgID, userID)
	}
	return nil, organization.ErrMemberNotFound
}
func (m *mockOrgMemberRepo) FindOwner(_ context.Context, _ uuid.UUID) (*organization.Member, error) {
	return nil, organization.ErrMemberNotFound
}
func (m *mockOrgMemberRepo) FindUserPrimaryOrg(ctx context.Context, userID uuid.UUID) (*organization.Member, error) {
	if m.findUserPrimaryOrgFn != nil {
		return m.findUserPrimaryOrgFn(ctx, userID)
	}
	return nil, organization.ErrMemberNotFound
}
func (m *mockOrgMemberRepo) List(_ context.Context, _ repository.ListMembersParams) ([]*organization.Member, string, error) {
	return nil, "", nil
}
func (m *mockOrgMemberRepo) CountByRole(_ context.Context, _ uuid.UUID) (map[organization.Role]int, error) {
	return nil, nil
}
func (m *mockOrgMemberRepo) Update(_ context.Context, _ *organization.Member) error { return nil }
func (m *mockOrgMemberRepo) Delete(_ context.Context, _ uuid.UUID) error            { return nil }
func (m *mockOrgMemberRepo) ListMemberUserIDsByOrgIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	return nil, nil
}
func (m *mockOrgMemberRepo) ListUserIDsByRole(_ context.Context, _ uuid.UUID, _ organization.Role) ([]uuid.UUID, error) {
	return nil, nil
}
