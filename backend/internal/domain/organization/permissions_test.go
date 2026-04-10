package organization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHasPermission_Owner verifies that the Owner has every defined permission.
// The Owner is the unconditional bearer of all capabilities — if this test
// fails, the rolePermissions map lost a capability the Owner should retain.
func TestHasPermission_Owner(t *testing.T) {
	allPerms := []Permission{
		PermJobsView, PermJobsCreate, PermJobsEdit, PermJobsDelete,
		PermProposalsView, PermProposalsCreate, PermProposalsRespond,
		PermMessagingView, PermMessagingSend,
		PermWalletView, PermWalletWithdraw,
		PermOrgProfileEdit,
		PermTeamView, PermTeamInvite, PermTeamManage, PermTeamTransferOwner,
		PermBillingView, PermBillingManage,
		PermOrgDelete, PermKYCManage,
		PermReviewsRespond,
	}
	for _, perm := range allPerms {
		t.Run(string(perm), func(t *testing.T) {
			assert.True(t, HasPermission(RoleOwner, perm),
				"Owner must have permission %s", perm)
		})
	}
}

// TestHasPermission_AdminRestrictions verifies the Admin's exclusion list.
// Admins can do almost everything, but NOT these sensitive actions.
func TestHasPermission_AdminRestrictions(t *testing.T) {
	forbidden := []Permission{
		PermWalletWithdraw,    // admins can see balance, not move money
		PermTeamTransferOwner, // only Owner can initiate a transfer
		PermBillingManage,     // Owner-only financial responsibility
		PermOrgDelete,         // Owner-only destructive op
		PermKYCManage,         // compliance is the Owner's responsibility
	}
	for _, perm := range forbidden {
		t.Run(string(perm), func(t *testing.T) {
			assert.False(t, HasPermission(RoleAdmin, perm),
				"Admin must NOT have permission %s", perm)
		})
	}

	// Spot-check a few things Admin SHOULD have
	assert.True(t, HasPermission(RoleAdmin, PermTeamInvite))
	assert.True(t, HasPermission(RoleAdmin, PermTeamManage))
	assert.True(t, HasPermission(RoleAdmin, PermWalletView))
	assert.True(t, HasPermission(RoleAdmin, PermJobsCreate))
}

// TestHasPermission_MemberCapabilities verifies the Member's sweet spot:
// can do daily ops, can't touch team/finances beyond view.
func TestHasPermission_MemberCapabilities(t *testing.T) {
	allowed := []Permission{
		PermJobsView, PermJobsCreate, PermJobsEdit,
		PermProposalsView, PermProposalsCreate, PermProposalsRespond,
		PermMessagingView, PermMessagingSend,
		PermWalletView,
		PermTeamView,
		PermReviewsRespond,
	}
	for _, perm := range allowed {
		t.Run("allowed/"+string(perm), func(t *testing.T) {
			assert.True(t, HasPermission(RoleMember, perm))
		})
	}

	forbidden := []Permission{
		PermJobsDelete,     // Members can't delete jobs, only Admin/Owner
		PermWalletWithdraw, // no money out for Members
		PermTeamInvite,     // can't invite people
		PermTeamManage,     // can't change others' roles
		PermOrgProfileEdit, // can't edit the org's public profile
		PermBillingView,    // billing is restricted
		PermOrgDelete,
	}
	for _, perm := range forbidden {
		t.Run("forbidden/"+string(perm), func(t *testing.T) {
			assert.False(t, HasPermission(RoleMember, perm))
		})
	}
}

// TestHasPermission_ViewerIsReadOnly verifies the Viewer's read-only nature.
func TestHasPermission_ViewerIsReadOnly(t *testing.T) {
	allowedViews := []Permission{
		PermJobsView,
		PermProposalsView,
		PermMessagingView,
		PermWalletView,
		PermTeamView,
	}
	for _, perm := range allowedViews {
		t.Run("view/"+string(perm), func(t *testing.T) {
			assert.True(t, HasPermission(RoleViewer, perm))
		})
	}

	// Everything else should be denied
	forbidden := []Permission{
		PermJobsCreate, PermJobsEdit, PermJobsDelete,
		PermProposalsCreate, PermProposalsRespond,
		PermMessagingSend,
		PermWalletWithdraw,
		PermOrgProfileEdit,
		PermTeamInvite, PermTeamManage, PermTeamTransferOwner,
		PermBillingView, PermBillingManage,
		PermOrgDelete, PermKYCManage,
		PermReviewsRespond,
	}
	for _, perm := range forbidden {
		t.Run("denied/"+string(perm), func(t *testing.T) {
			assert.False(t, HasPermission(RoleViewer, perm))
		})
	}
}

func TestHasPermission_UnknownRoleDeniesAll(t *testing.T) {
	assert.False(t, HasPermission(Role("guest"), PermJobsView))
	assert.False(t, HasPermission(Role(""), PermJobsView))
}

func TestPermissionsFor_OwnerHasEverything(t *testing.T) {
	perms := PermissionsFor(RoleOwner)
	// Owner permission count should be at least 20 — if this fails,
	// either we removed a capability or the test list is stale.
	assert.GreaterOrEqual(t, len(perms), 20)

	// All returned perms should indeed be granted (sanity check)
	for _, p := range perms {
		assert.True(t, HasPermission(RoleOwner, p))
	}
}

func TestPermissionsFor_ViewerHasFewPerms(t *testing.T) {
	perms := PermissionsFor(RoleViewer)
	assert.Equal(t, 5, len(perms), "viewer should have exactly 5 read permissions")
}

func TestPermissionsFor_UnknownRoleReturnsNil(t *testing.T) {
	perms := PermissionsFor(Role("guest"))
	assert.Nil(t, perms)
}
