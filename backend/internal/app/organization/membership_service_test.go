package organization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
)

// ---------------------------------------------------------------------------
// Session bump on role change — promotion vs demotion
// ---------------------------------------------------------------------------

func TestMembershipService_UpdateMemberRole_PromotionDoesNotBumpSession(t *testing.T) {
	tests := []struct {
		name    string
		oldRole organization.Role
		newRole organization.Role
	}{
		{"viewer to member", organization.RoleViewer, organization.RoleMember},
		{"viewer to admin", organization.RoleViewer, organization.RoleAdmin},
		{"member to admin", organization.RoleMember, organization.RoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := buildMembershipHarness(t)

			// Override the target's role to the starting role for this sub-test.
			h.memberMap[h.memberID].Role = tt.oldRole

			_, err := h.svc.UpdateMemberRole(
				context.Background(),
				h.ownerID, h.org.ID, h.memberID,
				tt.newRole,
			)
			require.NoError(t, err)

			users := h.svc.users.(*mockUserRepoForMembership)
			assert.Empty(t, users.bumpSessionCalls,
				"promotion (%s -> %s) must NOT call BumpSessionVersion", tt.oldRole, tt.newRole)
		})
	}
}

func TestMembershipService_UpdateMemberRole_DemotionBumpsSession(t *testing.T) {
	tests := []struct {
		name    string
		oldRole organization.Role
		newRole organization.Role
	}{
		{"admin to member", organization.RoleAdmin, organization.RoleMember},
		{"admin to viewer", organization.RoleAdmin, organization.RoleViewer},
		{"member to viewer", organization.RoleMember, organization.RoleViewer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := buildMembershipHarness(t)

			// Override the target's role to the starting role for this sub-test.
			h.memberMap[h.memberID].Role = tt.oldRole

			_, err := h.svc.UpdateMemberRole(
				context.Background(),
				h.ownerID, h.org.ID, h.memberID,
				tt.newRole,
			)
			require.NoError(t, err)

			users := h.svc.users.(*mockUserRepoForMembership)
			assert.Contains(t, users.bumpSessionCalls, h.memberID,
				"demotion (%s -> %s) must call BumpSessionVersion", tt.oldRole, tt.newRole)
		})
	}
}

func TestMembershipService_UpdateMemberRole_LateralDoesNotBumpSession(t *testing.T) {
	h := buildMembershipHarness(t)

	// Target starts as member; assigning the same role is a no-op lateral.
	h.memberMap[h.memberID].Role = organization.RoleMember

	_, err := h.svc.UpdateMemberRole(
		context.Background(),
		h.ownerID, h.org.ID, h.memberID,
		organization.RoleMember,
	)
	require.NoError(t, err)

	users := h.svc.users.(*mockUserRepoForMembership)
	assert.Empty(t, users.bumpSessionCalls,
		"lateral role change must NOT call BumpSessionVersion")
}
