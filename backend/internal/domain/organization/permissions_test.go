package organization

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		PermOrgProfileEdit, PermOrgClientProfileEdit,
		PermTeamView, PermTeamInvite, PermTeamManage, PermTeamTransferOwner,
		PermTeamManageRolePermissions,
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
		PermWalletWithdraw,            // admins can see balance, not move money
		PermTeamTransferOwner,         // only Owner can initiate a transfer
		PermTeamManageRolePermissions, // editing role permissions is Owner-only
		PermBillingManage,             // Owner-only financial responsibility
		PermOrgDelete,                 // Owner-only destructive op
		PermKYCManage,                 // compliance is the Owner's responsibility
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
		PermJobsDelete,                // Members can't delete jobs, only Admin/Owner
		PermWalletWithdraw,            // no money out for Members
		PermTeamInvite,                // can't invite people
		PermTeamManage,                // can't change others' roles
		PermTeamManageRolePermissions, // Owner-only
		PermOrgProfileEdit,            // can't edit the org's public profile
		PermOrgClientProfileEdit,      // can't edit the org's client profile either
		PermBillingView,               // billing is restricted
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
		PermOrgProfileEdit, PermOrgClientProfileEdit,
		PermTeamInvite, PermTeamManage, PermTeamTransferOwner, PermTeamManageRolePermissions,
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

// TestAllRoles_DisplayOrder asserts the canonical display order of
// roles. The team page renders roles top-to-bottom in this order so
// reordering them here would cascade as a UX regression.
func TestAllRoles_DisplayOrder(t *testing.T) {
	got := AllRoles()
	want := []Role{RoleOwner, RoleAdmin, RoleMember, RoleViewer}
	assert.Equal(t, want, got)
}

// TestAllPermissionMetadata_CoversEveryConstant guards against the
// "added a new permission constant but forgot to register a label"
// failure mode. Every permission referenced by the rolePermissions
// map must have a metadata entry.
func TestAllPermissionMetadata_CoversEveryConstant(t *testing.T) {
	seen := make(map[Permission]bool)
	for _, m := range AllPermissionMetadata() {
		seen[m.Key] = true
		assert.NotEmpty(t, m.Label, "permission %s missing label", m.Key)
		assert.NotEmpty(t, m.Group, "permission %s missing group", m.Key)
		assert.NotEmpty(t, m.Description, "permission %s missing description", m.Key)
	}
	for _, perms := range rolePermissions {
		for p := range perms {
			assert.True(t, seen[p],
				"permission %s referenced in rolePermissions but missing metadata entry", p)
		}
	}
}

// TestMetadataForRole_AllRolesHaveDescription ensures every V1 role
// has a registered display label and description so the team page
// can render it without falling back to the raw key.
func TestMetadataForRole_AllRolesHaveDescription(t *testing.T) {
	for _, r := range AllRoles() {
		t.Run(string(r), func(t *testing.T) {
			meta := MetadataForRole(r)
			assert.Equal(t, r, meta.Key)
			assert.NotEmpty(t, meta.Label)
			assert.NotEmpty(t, meta.Description)
		})
	}
}

// TestMetadataForRole_UnknownRoleSafeFallback verifies the fallback
// path returns a non-empty key so a typo in the role name does not
// crash the response builder.
func TestMetadataForRole_UnknownRoleSafeFallback(t *testing.T) {
	meta := MetadataForRole(Role("ghost"))
	assert.Equal(t, Role("ghost"), meta.Key)
	assert.Equal(t, "ghost", meta.Label)
}

// TestMetadataForPermission_UnknownPermissionSafeFallback ensures the
// fallback yields a stable structure (key + group=other) so the
// frontend never receives a malformed row.
func TestMetadataForPermission_UnknownPermissionSafeFallback(t *testing.T) {
	meta := MetadataForPermission(Permission("future.feature"))
	assert.Equal(t, Permission("future.feature"), meta.Key)
	assert.Equal(t, "other", meta.Group)
	assert.Equal(t, "future.feature", meta.Label)
}

// TestAllPermissionMetadata_ClientProfileRegistered guards the
// client-profile permission catalog entry: ordered list includes it
// right after PermOrgProfileEdit, metadata label/description are the
// two agreed-upon strings, and the provider-facing label was renamed
// to "Edit provider profile" to disambiguate from the client one.
func TestAllPermissionMetadata_ClientProfileRegistered(t *testing.T) {
	byKey := map[Permission]PermissionMetadata{}
	for _, m := range AllPermissionMetadata() {
		byKey[m.Key] = m
	}
	clientMeta, ok := byKey[PermOrgClientProfileEdit]
	require.True(t, ok, "PermOrgClientProfileEdit must have metadata registered")
	assert.Equal(t, "org_profile", clientMeta.Group)
	assert.Equal(t, "Edit client profile", clientMeta.Label)
	assert.Contains(t, clientMeta.Description, "client-facing")

	providerMeta, ok := byKey[PermOrgProfileEdit]
	require.True(t, ok)
	assert.Equal(t, "Edit provider profile", providerMeta.Label,
		"legacy provider-profile label must be renamed to disambiguate from the new client-profile permission")

	// Ordered list invariant — the client-profile entry must sit
	// directly after the provider-profile entry so the team-page UI
	// renders the two next to each other.
	providerIdx := -1
	clientIdx := -1
	for i, p := range allPermissionsOrdered {
		if p == PermOrgProfileEdit {
			providerIdx = i
		}
		if p == PermOrgClientProfileEdit {
			clientIdx = i
		}
	}
	require.GreaterOrEqual(t, providerIdx, 0)
	require.GreaterOrEqual(t, clientIdx, 0)
	assert.Equal(t, providerIdx+1, clientIdx, "client profile permission must immediately follow provider profile")
}
