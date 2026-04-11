package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
)

// TestTeamHandler_RoleDefinitions_Unauthorized verifies the endpoint
// rejects callers without an authenticated user id in context.
func TestTeamHandler_RoleDefinitions_Unauthorized(t *testing.T) {
	h := NewTeamHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/role-definitions", nil)
	rec := httptest.NewRecorder()

	h.RoleDefinitions(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestTeamHandler_RoleDefinitions_AuthenticatedReturnsCatalogue checks
// that an authenticated request receives the full role + permission
// catalogue, with all four V1 roles and at least the team.* permissions.
func TestTeamHandler_RoleDefinitions_AuthenticatedReturnsCatalogue(t *testing.T) {
	h := NewTeamHandler(nil, nil, nil)

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
