package organization

// Permission is a string identifier for a capability within the org.
//
// Permission strings are namespaced by resource ("jobs.create", "wallet.withdraw")
// so new capabilities can be added without name clashes. Checks are always
// against this flat string set, never against role names directly, so the
// permission model can evolve without breaking callers.
type Permission string

const (
	// Jobs
	PermJobsView   Permission = "jobs.view"
	PermJobsCreate Permission = "jobs.create"
	PermJobsEdit   Permission = "jobs.edit"
	PermJobsDelete Permission = "jobs.delete"

	// Proposals
	PermProposalsView    Permission = "proposals.view"
	PermProposalsCreate  Permission = "proposals.create"
	PermProposalsRespond Permission = "proposals.respond"

	// Messaging
	PermMessagingView Permission = "messaging.view"
	PermMessagingSend Permission = "messaging.send"

	// Wallet & finance
	PermWalletView     Permission = "wallet.view"
	PermWalletWithdraw Permission = "wallet.withdraw"

	// Org profile (public-facing)
	PermOrgProfileEdit Permission = "org_profile.edit"

	// Client profile (the client-facing facet of the org's public
	// profile — distinct from the provider-facing profile above).
	// Same default matrix as PermOrgProfileEdit: owner + admin only.
	PermOrgClientProfileEdit Permission = "org_client_profile.edit"

	// Team management (within the org)
	PermTeamView          Permission = "team.view"
	PermTeamInvite        Permission = "team.invite"
	PermTeamManage        Permission = "team.manage"
	PermTeamTransferOwner Permission = "team.transfer_ownership"

	// Billing (V1 scope is "view" only — the actual billing UI comes later)
	PermBillingView   Permission = "billing.view"
	PermBillingManage Permission = "billing.manage"

	// Dangerous
	PermOrgDelete  Permission = "org.delete"
	PermKYCManage  Permission = "kyc.manage"

	// Reviews (respond to reviews left on the org)
	PermReviewsRespond Permission = "reviews.respond"

	// Role permissions management (edit the per-org role overrides).
	//
	// This permission is strictly Owner-only by default AND is listed in
	// nonOverridablePermissions so the Owner cannot delegate it to an
	// Admin — granting the ability to edit permissions to anyone but
	// the Owner would effectively create a second Owner.
	PermTeamManageRolePermissions Permission = "team.manage_role_permissions"
)

// rolePermissions is the single source of truth for what each role can do.
//
// This map is the central authority for V1 permissions. Any code that wants
// to check a permission must call HasPermission(role, perm) — never hard-code
// a role comparison like `if role == RoleOwner`. That rule keeps the model
// swappable when V2 introduces JSONB overrides per member.
//
// Design principles for V1:
//   - Owner:  everything. No capability is locked away from the founder.
//   - Admin:  everything except money movement (withdraw), org deletion,
//             ownership transfer, KYC management and billing management.
//             Admins are trusted operators, not legal/financial representatives.
//   - Member: daily ops (create/edit jobs, send proposals, talk to clients,
//             see the wallet balance) but no invite/remove, no money out.
//             This is the default role for most invited employees.
//   - Viewer: strictly read-only. Used for observers, comptables, trainees.
var rolePermissions = map[Role]map[Permission]bool{
	RoleOwner: {
		PermJobsView: true, PermJobsCreate: true, PermJobsEdit: true, PermJobsDelete: true,
		PermProposalsView: true, PermProposalsCreate: true, PermProposalsRespond: true,
		PermMessagingView: true, PermMessagingSend: true,
		PermWalletView: true, PermWalletWithdraw: true,
		PermOrgProfileEdit: true, PermOrgClientProfileEdit: true,
		PermTeamView: true, PermTeamInvite: true, PermTeamManage: true, PermTeamTransferOwner: true,
		PermTeamManageRolePermissions: true,
		PermBillingView:               true, PermBillingManage: true,
		PermOrgDelete: true, PermKYCManage: true,
		PermReviewsRespond: true,
	},
	RoleAdmin: {
		PermJobsView: true, PermJobsCreate: true, PermJobsEdit: true, PermJobsDelete: true,
		PermProposalsView: true, PermProposalsCreate: true, PermProposalsRespond: true,
		PermMessagingView: true, PermMessagingSend: true,
		PermWalletView: true, // view only — no withdraw for admins
		PermOrgProfileEdit: true, PermOrgClientProfileEdit: true,
		PermTeamView: true, PermTeamInvite: true, PermTeamManage: true,
		PermBillingView: true, // can see invoices but not change payment methods
		PermReviewsRespond: true,
	},
	RoleMember: {
		PermJobsView: true, PermJobsCreate: true, PermJobsEdit: true,
		PermProposalsView: true, PermProposalsCreate: true, PermProposalsRespond: true,
		PermMessagingView: true, PermMessagingSend: true,
		PermWalletView: true,
		PermTeamView: true,
		PermReviewsRespond: true,
	},
	RoleViewer: {
		PermJobsView:      true,
		PermProposalsView: true,
		PermMessagingView: true,
		PermWalletView:    true,
		PermTeamView:      true,
	},
}

// HasPermission reports whether the given role grants the given permission
// according to the static defaults. Unknown roles and unknown permissions
// both return false (safe default: deny access when the map does not
// explicitly grant).
//
// NOTE: this function ignores any per-organization overrides. Callers that
// need to respect the customized permissions of a specific organization
// must use EffectivePermissionsFor or HasEffectivePermission with the
// org's RoleOverrides loaded from the database.
func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}

// nonOverridablePermissions lists permissions that cannot be toggled via
// the per-org role overrides, even by the Owner. These are either
// destructive/legal responsibilities that must stay Owner-only, or
// permissions whose delegation would create circular authority (an
// Admin granted PermTeamManageRolePermissions could grant themselves
// any other permission, defeating the whole model).
//
// The API layer rejects any override attempt that touches these keys
// with ErrPermissionNotOverridable. The UI renders them with a lock
// icon and disables the toggle.
//
// This list is append-only by design: removing a permission from here
// would silently expand the attack surface. New dangerous permissions
// must be added here the moment they are introduced.
var nonOverridablePermissions = map[Permission]bool{
	PermOrgDelete:                 true, // irreversible
	PermTeamTransferOwner:         true, // circular authority
	PermWalletWithdraw:            true, // anti-fraud (could drain the wallet)
	PermKYCManage:                 true, // compliance / legal
	PermTeamManageRolePermissions: true, // delegating this defeats the model
}

// IsOverridable reports whether the given permission can be toggled on
// or off by the Owner through the per-org role overrides mechanism.
// Unknown permissions are considered overridable (they did not opt-in
// to protection), so the allowlist here is strict and explicit.
func IsOverridable(perm Permission) bool {
	return !nonOverridablePermissions[perm]
}

// RoleOverrides is the per-organization customization layer on top of
// the static rolePermissions map. It maps a role to the set of
// permissions whose default has been changed:
//
//	overrides[role][perm] = true   → grant a permission NOT in defaults
//	overrides[role][perm] = false  → revoke a permission IN defaults
//
// Permissions not present in the inner map follow the static default.
// The Owner row is never customized — Owner always has every permission.
//
// Persisted as a JSONB column on the organizations table. Empty / nil
// values are valid and represent "no customization".
type RoleOverrides map[Role]map[Permission]bool

// Clone returns a deep copy of the overrides so callers can mutate the
// result without racing the original. Safe to call on a nil receiver —
// returns an empty (but non-nil) RoleOverrides.
func (o RoleOverrides) Clone() RoleOverrides {
	out := make(RoleOverrides, len(o))
	for role, perms := range o {
		inner := make(map[Permission]bool, len(perms))
		for p, v := range perms {
			inner[p] = v
		}
		out[role] = inner
	}
	return out
}

// EffectivePermissionsFor returns the fully-resolved set of permissions
// for a role given a set of per-organization overrides. This is the
// single function every caller must use when computing "what can this
// user do in THIS organization". PermissionsFor (which reads only the
// static map) is retained for the tiny minority of callers that truly
// want the defaults — most code should switch to this function.
//
// Resolution order:
//  1. Start from the static rolePermissions[role] map (the defaults).
//  2. For every override on this role:
//     - Skip the override if the permission is non-overridable — this
//       is defense in depth on top of the service-layer rejection.
//     - Apply the override: true grants, false revokes.
//  3. Collect every permission whose resolved value is true.
//
// The Owner role is intentionally NOT immune to the resolution pipeline
// but overriding the Owner is forbidden at the service layer. This
// function stays simple and symmetric.
//
// The returned slice is a fresh allocation and has no guaranteed order.
// Callers that need a deterministic order should sort by the string
// value or use allPermissionsOrdered as a reference.
func EffectivePermissionsFor(role Role, overrides RoleOverrides) []Permission {
	base, ok := rolePermissions[role]
	if !ok {
		return nil
	}

	resolved := make(map[Permission]bool, len(base))
	for p, granted := range base {
		resolved[p] = granted
	}

	if roleOverrides, hasOverrides := overrides[role]; hasOverrides {
		for p, granted := range roleOverrides {
			if nonOverridablePermissions[p] {
				continue // protection: silently ignore illegal overrides
			}
			resolved[p] = granted
		}
	}

	out := make([]Permission, 0, len(resolved))
	for p, granted := range resolved {
		if granted {
			out = append(out, p)
		}
	}
	return out
}

// HasEffectivePermission is the override-aware counterpart of
// HasPermission. Returns true when the role has the permission after
// applying the organization's overrides. Unknown roles and unknown
// permissions both return false.
func HasEffectivePermission(role Role, perm Permission, overrides RoleOverrides) bool {
	base, ok := rolePermissions[role]
	if !ok {
		return false
	}
	// Check overrides first — but only when the permission is overridable.
	if nonOverridablePermissions[perm] {
		return base[perm]
	}
	if roleOverrides, has := overrides[role]; has {
		if granted, set := roleOverrides[perm]; set {
			return granted
		}
	}
	return base[perm]
}
