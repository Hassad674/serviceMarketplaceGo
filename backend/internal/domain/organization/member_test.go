package organization

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMember_Valid(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	m, err := NewMember(orgID, userID, RoleMember, "Lead Designer")
	require.NoError(t, err)
	assert.Equal(t, orgID, m.OrganizationID)
	assert.Equal(t, userID, m.UserID)
	assert.Equal(t, RoleMember, m.Role)
	assert.Equal(t, "Lead Designer", m.Title)
	assert.NotEqual(t, uuid.Nil, m.ID)
}

func TestNewMember_Invalid(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name    string
		orgID   uuid.UUID
		userID  uuid.UUID
		role    Role
		title   string
		wantErr error
	}{
		{"nil org", uuid.Nil, userID, RoleMember, "", ErrMemberNotFound},
		{"nil user", orgID, uuid.Nil, RoleMember, "", ErrMemberNotFound},
		{"invalid role", orgID, userID, Role("king"), "", ErrInvalidRole},
		{"title too long", orgID, userID, RoleMember, strings.Repeat("x", maxTitleLength+1), ErrTitleTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMember(tt.orgID, tt.userID, tt.role, tt.title)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, m)
		})
	}
}

func TestMember_ChangeRole(t *testing.T) {
	m, err := NewMember(uuid.New(), uuid.New(), RoleMember, "")
	require.NoError(t, err)
	originalUpdate := m.UpdatedAt

	// Small delay so UpdatedAt actually moves forward
	time.Sleep(2 * time.Millisecond)

	err = m.ChangeRole(RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, m.Role)
	assert.True(t, m.UpdatedAt.After(originalUpdate))

	err = m.ChangeRole(Role("bogus"))
	assert.ErrorIs(t, err, ErrInvalidRole)
	assert.Equal(t, RoleAdmin, m.Role, "invalid change should not mutate state")
}

func TestMember_UpdateTitle(t *testing.T) {
	m, _ := NewMember(uuid.New(), uuid.New(), RoleMember, "")
	require.NoError(t, m.UpdateTitle("Senior Consultant"))
	assert.Equal(t, "Senior Consultant", m.Title)

	err := m.UpdateTitle(strings.Repeat("a", maxTitleLength+1))
	assert.ErrorIs(t, err, ErrTitleTooLong)
}

func TestMember_HasPermission(t *testing.T) {
	owner, _ := NewMember(uuid.New(), uuid.New(), RoleOwner, "")
	assert.True(t, owner.HasPermission(PermWalletWithdraw))

	viewer, _ := NewMember(uuid.New(), uuid.New(), RoleViewer, "")
	assert.False(t, viewer.HasPermission(PermWalletWithdraw))
	assert.True(t, viewer.HasPermission(PermJobsView))
}

func TestMember_IsOwner(t *testing.T) {
	owner, _ := NewMember(uuid.New(), uuid.New(), RoleOwner, "")
	assert.True(t, owner.IsOwner())

	admin, _ := NewMember(uuid.New(), uuid.New(), RoleAdmin, "")
	assert.False(t, admin.IsOwner())
}

func TestMember_CanManageMember(t *testing.T) {
	orgID := uuid.New()
	owner, _ := NewMember(orgID, uuid.New(), RoleOwner, "")
	admin, _ := NewMember(orgID, uuid.New(), RoleAdmin, "")
	member, _ := NewMember(orgID, uuid.New(), RoleMember, "")
	viewer, _ := NewMember(orgID, uuid.New(), RoleViewer, "")

	// Owner can manage Admin, Member, Viewer — but NOT another Owner
	assert.True(t, owner.CanManageMember(admin))
	assert.True(t, owner.CanManageMember(member))
	assert.True(t, owner.CanManageMember(viewer))
	owner2, _ := NewMember(orgID, uuid.New(), RoleOwner, "")
	assert.False(t, owner.CanManageMember(owner2),
		"Owners cannot manage other Owners in V1 — transfer flow only")

	// Admin can manage Admin (peer), Member, Viewer — but not Owner
	assert.True(t, admin.CanManageMember(member))
	assert.True(t, admin.CanManageMember(viewer))
	admin2, _ := NewMember(orgID, uuid.New(), RoleAdmin, "")
	assert.True(t, admin.CanManageMember(admin2))
	assert.False(t, admin.CanManageMember(owner))

	// Member and Viewer cannot manage anyone
	assert.False(t, member.CanManageMember(viewer))
	assert.False(t, viewer.CanManageMember(member))

	// Cross-org is always denied
	otherOrgMember, _ := NewMember(uuid.New(), uuid.New(), RoleMember, "")
	assert.False(t, owner.CanManageMember(otherOrgMember))

	// Nil target is denied
	assert.False(t, owner.CanManageMember(nil))
}
