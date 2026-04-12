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
		PermOrgProfileEdit: true,
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
		PermOrgProfileEdit: true,
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

// PermissionState describes the origin of a role's permission grant in
// the customized view returned by MergePermissionsForUI. Used by the
// role-permissions editor page so the UI can render the correct icon
// (default, granted-override, revoked-override, locked).
type PermissionState string

const (
	// PermissionStateDefaultGranted — the permission is granted by the
	// static defaults and has not been overridden.
	PermissionStateDefaultGranted PermissionState = "default_granted"
	// PermissionStateDefaultRevoked — the permission is NOT granted by
	// the static defaults and has not been overridden. Shown as "off"
	// in the UI without a badge.
	PermissionStateDefaultRevoked PermissionState = "default_revoked"
	// PermissionStateGrantedOverride — the permission is granted by an
	// override on top of a default that was not granted. Shown with
	// a green "Customized" badge.
	PermissionStateGrantedOverride PermissionState = "granted_override"
	// PermissionStateRevokedOverride — the permission is revoked by an
	// override on top of a default that WAS granted. Shown with a red
	// "Customized" badge.
	PermissionStateRevokedOverride PermissionState = "revoked_override"
	// PermissionStateLocked — the permission is non-overridable and
	// its granted value comes entirely from the static defaults. The
	// UI renders a lock icon and disables the toggle.
	PermissionStateLocked PermissionState = "locked"
)

// PermissionView is the fully-resolved UI description of a single
// (role, permission) cell in the role-permissions editor. Built by
// MergePermissionsForUI from the domain defaults + the org overrides.
type PermissionView struct {
	// Key is the permission identifier (e.g. "jobs.create").
	Key Permission
	// Granted is the effective granted value after applying overrides.
	Granted bool
	// State describes WHY Granted has its value — default vs override,
	// or locked (non-overridable). See PermissionState constants.
	State PermissionState
	// Locked is true when the permission is non-overridable. Duplicated
	// from State for convenience — the UI disables the toggle on this
	// flag without having to parse the state string.
	Locked bool
}

// MergePermissionsForUI returns the full permission catalogue for a
// role with every cell resolved into a PermissionView. Used by the
// GET /organizations/{id}/role-permissions endpoint to populate the
// role-permissions editor — one call per role renders the full
// matrix for that role.
//
// The Owner row is handled specially: every permission is reported
// as locked+granted because the Owner always has everything and
// their row is read-only in the UI.
//
// Result is ordered by allPermissionsOrdered so the frontend can
// render without an extra sort step.
func MergePermissionsForUI(role Role, overrides RoleOverrides) []PermissionView {
	views := make([]PermissionView, 0, len(allPermissionsOrdered))

	baseForRole := rolePermissions[role]
	roleOverrides := overrides[role]

	for _, perm := range allPermissionsOrdered {
		locked := nonOverridablePermissions[perm]
		defaultGranted := baseForRole[perm]

		// Owner row: everything granted and everything locked — the UI
		// is read-only for the Owner (they always have every perm).
		if role == RoleOwner {
			views = append(views, PermissionView{
				Key:     perm,
				Granted: true,
				State:   PermissionStateLocked,
				Locked:  true,
			})
			continue
		}

		if locked {
			state := PermissionStateLocked
			views = append(views, PermissionView{
				Key:     perm,
				Granted: defaultGranted,
				State:   state,
				Locked:  true,
			})
			continue
		}

		overrideValue, overridden := roleOverrides[perm]
		if !overridden {
			state := PermissionStateDefaultRevoked
			if defaultGranted {
				state = PermissionStateDefaultGranted
			}
			views = append(views, PermissionView{
				Key:     perm,
				Granted: defaultGranted,
				State:   state,
				Locked:  false,
			})
			continue
		}

		// Overridden cell.
		var state PermissionState
		switch {
		case overrideValue && !defaultGranted:
			state = PermissionStateGrantedOverride
		case !overrideValue && defaultGranted:
			state = PermissionStateRevokedOverride
		case overrideValue && defaultGranted:
			// Redundant override (same as default) — treat as default.
			state = PermissionStateDefaultGranted
		default:
			state = PermissionStateDefaultRevoked
		}
		views = append(views, PermissionView{
			Key:     perm,
			Granted: overrideValue,
			State:   state,
			Locked:  false,
		})
	}

	return views
}

// ValidateRoleOverrides checks that a proposed overrides payload is
// well-formed before it is persisted. Returns the first error found;
// the service layer surfaces this to the handler without wrapping.
//
// Rules enforced:
//   - The Owner role can never appear as a key (Owner is not editable)
//   - Every role key must be a valid Role
//   - Every permission key must be a registered permission
//   - No non-overridable permission can appear in the payload
func ValidateRoleOverrides(overrides RoleOverrides) error {
	for role, perms := range overrides {
		if role == RoleOwner {
			return ErrCannotOverrideOwner
		}
		if !role.IsValid() {
			return ErrInvalidRole
		}
		for perm := range perms {
			if _, known := permissionMetadataByKey[perm]; !known {
				return ErrUnknownPermission
			}
			if nonOverridablePermissions[perm] {
				return ErrPermissionNotOverridable
			}
		}
	}
	return nil
}

// NonOverridablePermissions returns a copy of the set of permissions
// that cannot be customized per organization. Used by the role
// definitions endpoint so the frontend can render lock icons without
// hard-coding the list on the client side.
func NonOverridablePermissions() []Permission {
	out := make([]Permission, 0, len(nonOverridablePermissions))
	for p := range nonOverridablePermissions {
		out = append(out, p)
	}
	return out
}

// PermissionsFor returns the set of permissions granted to a role.
// Used to populate the user's effective permissions in the /me response
// so the frontend can enable/disable UI elements without round-tripping
// a permission check for every button.
//
// Returned slice is a fresh allocation owned by the caller.
func PermissionsFor(role Role) []Permission {
	perms, ok := rolePermissions[role]
	if !ok {
		return nil
	}
	result := make([]Permission, 0, len(perms))
	for p, allowed := range perms {
		if allowed {
			result = append(result, p)
		}
	}
	return result
}

// AllRoles returns the four V1 roles in the canonical display order
// (Owner > Admin > Member > Viewer). Order matters because the team
// page renders the role cards top-to-bottom in this sequence.
func AllRoles() []Role {
	return []Role{RoleOwner, RoleAdmin, RoleMember, RoleViewer}
}

// PermissionMetadata is the static description of a single permission
// constant — its grouping (resource family), an English label, and a
// short English description. The frontend uses these as fallbacks
// when its own i18n catalogue does not yet have a translation for
// a newly-introduced permission.
type PermissionMetadata struct {
	Key         Permission
	Group       string
	Label       string
	Description string
}

// permissionMetadataByKey is the static catalogue of human-readable
// metadata for every permission constant. New permissions added to
// the rolePermissions map MUST also be registered here so the
// role-definitions endpoint can describe them — the test in
// permissions_test.go enforces this.
var permissionMetadataByKey = map[Permission]PermissionMetadata{
	PermJobsView:          {Key: PermJobsView, Group: "jobs", Label: "View jobs", Description: "Can browse and read jobs published by the organization."},
	PermJobsCreate:        {Key: PermJobsCreate, Group: "jobs", Label: "Create jobs", Description: "Can publish new jobs on behalf of the organization."},
	PermJobsEdit:          {Key: PermJobsEdit, Group: "jobs", Label: "Edit jobs", Description: "Can update jobs already published by the organization."},
	PermJobsDelete:        {Key: PermJobsDelete, Group: "jobs", Label: "Delete jobs", Description: "Can take down jobs published by the organization."},
	PermProposalsView:     {Key: PermProposalsView, Group: "proposals", Label: "View proposals", Description: "Can read proposals sent or received by the organization."},
	PermProposalsCreate:   {Key: PermProposalsCreate, Group: "proposals", Label: "Send proposals", Description: "Can draft and send new proposals to clients."},
	PermProposalsRespond:  {Key: PermProposalsRespond, Group: "proposals", Label: "Respond to proposals", Description: "Can accept, decline, or counter incoming proposals."},
	PermMessagingView:     {Key: PermMessagingView, Group: "messaging", Label: "Read conversations", Description: "Can open and read messaging threads the organization is part of."},
	PermMessagingSend:     {Key: PermMessagingSend, Group: "messaging", Label: "Send messages", Description: "Can write and send messages from the organization account."},
	PermWalletView:        {Key: PermWalletView, Group: "wallet", Label: "View wallet", Description: "Can see the organization's balance and transaction history."},
	PermWalletWithdraw:    {Key: PermWalletWithdraw, Group: "wallet", Label: "Request payouts", Description: "Can move money out of the organization wallet to the connected Stripe account."},
	PermOrgProfileEdit:    {Key: PermOrgProfileEdit, Group: "org_profile", Label: "Edit public profile", Description: "Can update the organization's public-facing marketplace profile (logo, about, video)."},
	PermTeamView:          {Key: PermTeamView, Group: "team", Label: "View team", Description: "Can see the list of members and pending invitations."},
	PermTeamInvite:        {Key: PermTeamInvite, Group: "team", Label: "Invite members", Description: "Can send email invitations to join the organization."},
	PermTeamManage:        {Key: PermTeamManage, Group: "team", Label: "Manage team", Description: "Can change member roles and titles, and remove members from the organization."},
	PermTeamTransferOwner:         {Key: PermTeamTransferOwner, Group: "team", Label: "Transfer ownership", Description: "Can initiate the ownership transfer flow to hand the organization to another admin."},
	PermTeamManageRolePermissions: {Key: PermTeamManageRolePermissions, Group: "team", Label: "Customize role permissions", Description: "Can edit which permissions each role (Admin, Member, Viewer) has in this organization. Owner-only and non-delegable."},
	PermBillingView:       {Key: PermBillingView, Group: "billing", Label: "View billing", Description: "Can see invoices and payment history for the organization."},
	PermBillingManage:     {Key: PermBillingManage, Group: "billing", Label: "Manage billing", Description: "Can change payment methods and update billing settings."},
	PermOrgDelete:         {Key: PermOrgDelete, Group: "danger", Label: "Delete organization", Description: "Can permanently delete the organization. This action is irreversible."},
	PermKYCManage:         {Key: PermKYCManage, Group: "kyc", Label: "Manage KYC", Description: "Can complete and update the organization's Stripe Connect KYC verification."},
	PermReviewsRespond:    {Key: PermReviewsRespond, Group: "reviews", Label: "Respond to reviews", Description: "Can publicly reply to reviews left on the organization."},
}

// AllPermissionMetadata returns the static metadata for every
// permission constant in a stable order. Used by the role-definitions
// endpoint to describe what each role can do.
//
// The returned slice is a fresh allocation owned by the caller.
// Order is by group then alphabetic by key, so the team page can
// render the catalogue without an additional client-side sort.
func AllPermissionMetadata() []PermissionMetadata {
	out := make([]PermissionMetadata, 0, len(permissionMetadataByKey))
	for _, p := range allPermissionsOrdered {
		if meta, ok := permissionMetadataByKey[p]; ok {
			out = append(out, meta)
		}
	}
	return out
}

// MetadataForPermission returns the static metadata for a single
// permission. Returns the zero value with Key set when the permission
// has no registered metadata, so the caller can still emit a row.
func MetadataForPermission(p Permission) PermissionMetadata {
	if meta, ok := permissionMetadataByKey[p]; ok {
		return meta
	}
	return PermissionMetadata{Key: p, Group: "other", Label: string(p), Description: ""}
}

// allPermissionsOrdered fixes the display order of permissions in the
// role-definitions response. Grouped by resource family and otherwise
// stable so the frontend can render predictable lists.
var allPermissionsOrdered = []Permission{
	// Team
	PermTeamView, PermTeamInvite, PermTeamManage, PermTeamTransferOwner, PermTeamManageRolePermissions,
	// Org profile
	PermOrgProfileEdit,
	// Jobs
	PermJobsView, PermJobsCreate, PermJobsEdit, PermJobsDelete,
	// Proposals
	PermProposalsView, PermProposalsCreate, PermProposalsRespond,
	// Messaging
	PermMessagingView, PermMessagingSend,
	// Reviews
	PermReviewsRespond,
	// Wallet
	PermWalletView, PermWalletWithdraw,
	// Billing
	PermBillingView, PermBillingManage,
	// KYC
	PermKYCManage,
	// Danger
	PermOrgDelete,
}

// RoleMetadata is the static description of a single role: its
// stable key (matches Role string), an English display label, and a
// short English description suitable for the team page's "About
// roles" panel.
type RoleMetadata struct {
	Key         Role
	Label       string
	Description string
}

// roleMetadata is the static catalogue of role labels and
// descriptions. Kept in the domain (next to rolePermissions) so the
// "what does this role do" text lives at the same level as the
// permission map it describes — the source of truth for both is one
// file.
var roleMetadata = map[Role]RoleMetadata{
	RoleOwner: {
		Key:         RoleOwner,
		Label:       "Owner",
		Description: "Full control of the organization. The owner is the only role that can transfer ownership, delete the organization, and request payouts. Exactly one owner per organization.",
	},
	RoleAdmin: {
		Key:         RoleAdmin,
		Label:       "Admin",
		Description: "Trusted operator with full operational rights: can manage the team, jobs, proposals, KYC, billing settings, and respond to reviews. Cannot transfer ownership, delete the organization, or move money out of the wallet.",
	},
	RoleMember: {
		Key:         RoleMember,
		Label:       "Member",
		Description: "Daily operator. Can create and edit jobs, send and respond to proposals, message clients, and view the wallet balance. Cannot manage the team or touch finances beyond viewing.",
	},
	RoleViewer: {
		Key:         RoleViewer,
		Label:       "Viewer",
		Description: "Read-only access across all organization resources. Used for observers, external accountants, and trainees who need visibility without the ability to act.",
	},
}

// MetadataForRole returns the static metadata for a single role.
// Returns the zero value with Key set when the role is unknown, so
// the caller can still render a row without crashing.
func MetadataForRole(r Role) RoleMetadata {
	if meta, ok := roleMetadata[r]; ok {
		return meta
	}
	return RoleMetadata{Key: r, Label: string(r), Description: ""}
}
