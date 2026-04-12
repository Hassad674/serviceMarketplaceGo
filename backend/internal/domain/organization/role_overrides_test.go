package organization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsOverridable verifies that the explicit allowlist of
// non-overridable permissions is respected.
func TestIsOverridable(t *testing.T) {
	cases := []struct {
		perm     Permission
		expected bool
	}{
		// Dangerous / legal — always locked
		{PermOrgDelete, false},
		{PermTeamTransferOwner, false},
		{PermWalletWithdraw, false},
		{PermKYCManage, false},
		{PermTeamManageRolePermissions, false},

		// Operational — freely customizable
		{PermJobsView, true},
		{PermJobsCreate, true},
		{PermJobsDelete, true},
		{PermMessagingSend, true},
		{PermProposalsRespond, true},
		{PermBillingView, true},
		{PermBillingManage, true},
	}
	for _, tc := range cases {
		t.Run(string(tc.perm), func(t *testing.T) {
			assert.Equal(t, tc.expected, IsOverridable(tc.perm))
		})
	}
}

// TestEffectivePermissionsFor_NoOverrides verifies that with no
// overrides the result matches PermissionsFor exactly.
func TestEffectivePermissionsFor_NoOverrides(t *testing.T) {
	for _, role := range AllRoles() {
		t.Run(string(role), func(t *testing.T) {
			defaultPerms := PermissionsFor(role)
			effective := EffectivePermissionsFor(role, nil)
			assert.ElementsMatch(t, defaultPerms, effective)
		})
	}
}

// TestEffectivePermissionsFor_GrantsNewPermission verifies that an
// override granting a permission the role did not have by default
// actually grants it.
func TestEffectivePermissionsFor_GrantsNewPermission(t *testing.T) {
	overrides := RoleOverrides{
		RoleMember: {PermJobsDelete: true},
	}
	perms := EffectivePermissionsFor(RoleMember, overrides)
	assert.Contains(t, perms, PermJobsDelete)
}

// TestEffectivePermissionsFor_RevokesDefault verifies that an override
// revoking a permission the role had by default removes it.
func TestEffectivePermissionsFor_RevokesDefault(t *testing.T) {
	overrides := RoleOverrides{
		RoleMember: {PermMessagingSend: false},
	}
	perms := EffectivePermissionsFor(RoleMember, overrides)
	assert.NotContains(t, perms, PermMessagingSend)
}

// TestEffectivePermissionsFor_IgnoresNonOverridable verifies that
// attempts to toggle a locked permission are silently ignored in the
// resolution pipeline (defense in depth on top of validation).
func TestEffectivePermissionsFor_IgnoresNonOverridable(t *testing.T) {
	overrides := RoleOverrides{
		RoleAdmin: {PermWalletWithdraw: true}, // illegal, must be ignored
	}
	perms := EffectivePermissionsFor(RoleAdmin, overrides)
	assert.NotContains(t, perms, PermWalletWithdraw)
}

// TestEffectivePermissionsFor_UnknownRole_ReturnsNil verifies that
// unknown roles fall through safely.
func TestEffectivePermissionsFor_UnknownRole_ReturnsNil(t *testing.T) {
	perms := EffectivePermissionsFor(Role("ghost"), nil)
	assert.Nil(t, perms)
}

// TestHasEffectivePermission_OverrideRespected verifies the point-check
// variant honors overrides.
func TestHasEffectivePermission_OverrideRespected(t *testing.T) {
	overrides := RoleOverrides{
		RoleViewer: {PermMessagingSend: true},
	}
	assert.True(t, HasEffectivePermission(RoleViewer, PermMessagingSend, overrides))
	assert.False(t, HasEffectivePermission(RoleViewer, PermMessagingSend, nil))
}

// TestHasEffectivePermission_LockedIgnoresOverride verifies that
// locked permissions never honor an attempted override.
func TestHasEffectivePermission_LockedIgnoresOverride(t *testing.T) {
	overrides := RoleOverrides{
		RoleAdmin: {PermWalletWithdraw: true},
	}
	assert.False(t, HasEffectivePermission(RoleAdmin, PermWalletWithdraw, overrides))
}

// TestValidateRoleOverrides_RejectsOwner verifies that the Owner row
// cannot be customized.
func TestValidateRoleOverrides_RejectsOwner(t *testing.T) {
	err := ValidateRoleOverrides(RoleOverrides{
		RoleOwner: {PermJobsView: false},
	})
	assert.ErrorIs(t, err, ErrCannotOverrideOwner)
}

// TestValidateRoleOverrides_RejectsLockedPermission verifies that
// non-overridable permissions are rejected at validation time.
func TestValidateRoleOverrides_RejectsLockedPermission(t *testing.T) {
	err := ValidateRoleOverrides(RoleOverrides{
		RoleAdmin: {PermWalletWithdraw: true},
	})
	assert.ErrorIs(t, err, ErrPermissionNotOverridable)
}

// TestValidateRoleOverrides_RejectsUnknownPermission verifies that
// unknown permission keys are rejected.
func TestValidateRoleOverrides_RejectsUnknownPermission(t *testing.T) {
	err := ValidateRoleOverrides(RoleOverrides{
		RoleMember: {Permission("future.feature"): true},
	})
	assert.ErrorIs(t, err, ErrUnknownPermission)
}

// TestValidateRoleOverrides_RejectsInvalidRole verifies that unknown
// role keys are rejected.
func TestValidateRoleOverrides_RejectsInvalidRole(t *testing.T) {
	err := ValidateRoleOverrides(RoleOverrides{
		Role("super_admin"): {PermJobsCreate: true},
	})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

// TestValidateRoleOverrides_AcceptsValidPayload verifies the happy path.
func TestValidateRoleOverrides_AcceptsValidPayload(t *testing.T) {
	err := ValidateRoleOverrides(RoleOverrides{
		RoleAdmin:  {PermBillingManage: true},
		RoleMember: {PermJobsDelete: true, PermTeamInvite: true},
	})
	assert.NoError(t, err)
}

// TestMergePermissionsForUI_OwnerRowIsLocked verifies that the Owner
// row surfaces every permission as locked+granted so the UI renders
// it as read-only.
func TestMergePermissionsForUI_OwnerRowIsLocked(t *testing.T) {
	views := MergePermissionsForUI(RoleOwner, nil)
	assert.NotEmpty(t, views)
	for _, v := range views {
		assert.True(t, v.Granted, "Owner must have every permission granted")
		assert.True(t, v.Locked, "Owner row must be fully locked")
		assert.Equal(t, PermissionStateLocked, v.State)
	}
}

// TestMergePermissionsForUI_DefaultStates verifies that permissions
// without overrides carry the default_granted / default_revoked state.
func TestMergePermissionsForUI_DefaultStates(t *testing.T) {
	views := MergePermissionsForUI(RoleMember, nil)
	byKey := make(map[Permission]PermissionView, len(views))
	for _, v := range views {
		byKey[v.Key] = v
	}

	assert.Equal(t, PermissionStateDefaultGranted, byKey[PermJobsCreate].State)
	assert.Equal(t, PermissionStateDefaultRevoked, byKey[PermJobsDelete].State)
	assert.False(t, byKey[PermJobsCreate].Locked)
}

// TestMergePermissionsForUI_OverrideStates verifies that overrides
// surface as granted_override / revoked_override.
func TestMergePermissionsForUI_OverrideStates(t *testing.T) {
	overrides := RoleOverrides{
		RoleMember: {
			PermJobsDelete:    true,  // granted override (default was false)
			PermMessagingSend: false, // revoked override (default was true)
		},
	}
	views := MergePermissionsForUI(RoleMember, overrides)
	byKey := make(map[Permission]PermissionView, len(views))
	for _, v := range views {
		byKey[v.Key] = v
	}

	assert.Equal(t, PermissionStateGrantedOverride, byKey[PermJobsDelete].State)
	assert.True(t, byKey[PermJobsDelete].Granted)

	assert.Equal(t, PermissionStateRevokedOverride, byKey[PermMessagingSend].State)
	assert.False(t, byKey[PermMessagingSend].Granted)
}

// TestMergePermissionsForUI_LockedPermissions verifies that locked
// permissions appear with the locked state regardless of override.
func TestMergePermissionsForUI_LockedPermissions(t *testing.T) {
	views := MergePermissionsForUI(RoleAdmin, nil)
	byKey := make(map[Permission]PermissionView, len(views))
	for _, v := range views {
		byKey[v.Key] = v
	}

	// PermWalletWithdraw is locked and not granted to Admin by default.
	assert.Equal(t, PermissionStateLocked, byKey[PermWalletWithdraw].State)
	assert.True(t, byKey[PermWalletWithdraw].Locked)
	assert.False(t, byKey[PermWalletWithdraw].Granted)
}

// TestSetRoleOverride_OwnerRejected verifies that the domain method
// refuses to customize the Owner.
func TestSetRoleOverride_OwnerRejected(t *testing.T) {
	org := &Organization{}
	err := org.SetRoleOverride(RoleOwner, PermJobsView, false)
	assert.ErrorIs(t, err, ErrCannotOverrideOwner)
}

// TestSetRoleOverride_LockedPermissionRejected verifies the non-overridable
// check at the domain level.
func TestSetRoleOverride_LockedPermissionRejected(t *testing.T) {
	org := &Organization{}
	err := org.SetRoleOverride(RoleAdmin, PermWalletWithdraw, true)
	assert.ErrorIs(t, err, ErrPermissionNotOverridable)
}

// TestSetRoleOverride_HappyPath verifies that valid overrides are
// persisted on the entity.
func TestSetRoleOverride_HappyPath(t *testing.T) {
	org := &Organization{}
	err := org.SetRoleOverride(RoleMember, PermJobsDelete, true)
	assert.NoError(t, err)
	assert.True(t, org.RoleOverrides[RoleMember][PermJobsDelete])
}

// TestReplaceRoleOverrides_AtomicReplace verifies that ReplaceRoleOverrides
// wipes previous overrides for the target role.
func TestReplaceRoleOverrides_AtomicReplace(t *testing.T) {
	org := &Organization{
		RoleOverrides: RoleOverrides{
			RoleMember: {PermJobsDelete: true, PermTeamInvite: true},
		},
	}
	err := org.ReplaceRoleOverrides(RoleMember, map[Permission]bool{
		PermJobsDelete: true,
	})
	assert.NoError(t, err)
	// TeamInvite override should be gone now.
	_, has := org.RoleOverrides[RoleMember][PermTeamInvite]
	assert.False(t, has)
	assert.True(t, org.RoleOverrides[RoleMember][PermJobsDelete])
}

// TestReplaceRoleOverrides_EmptyDeletesRole verifies that replacing
// with an empty map removes the role entry entirely.
func TestReplaceRoleOverrides_EmptyDeletesRole(t *testing.T) {
	org := &Organization{
		RoleOverrides: RoleOverrides{
			RoleMember: {PermJobsDelete: true},
		},
	}
	err := org.ReplaceRoleOverrides(RoleMember, map[Permission]bool{})
	assert.NoError(t, err)
	_, has := org.RoleOverrides[RoleMember]
	assert.False(t, has)
}

// TestClone_IsDeepCopy verifies that mutating a cloned override does
// not affect the original.
func TestClone_IsDeepCopy(t *testing.T) {
	original := RoleOverrides{
		RoleMember: {PermJobsDelete: true},
	}
	clone := original.Clone()
	clone[RoleMember][PermJobsDelete] = false
	assert.True(t, original[RoleMember][PermJobsDelete])
}
