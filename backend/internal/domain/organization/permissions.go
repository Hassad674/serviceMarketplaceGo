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
		PermBillingView: true, PermBillingManage: true,
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

// HasPermission reports whether the given role grants the given permission.
// Unknown roles and unknown permissions both return false (safe default:
// deny access when the map does not explicitly grant).
func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
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
	PermTeamTransferOwner: {Key: PermTeamTransferOwner, Group: "team", Label: "Transfer ownership", Description: "Can initiate the ownership transfer flow to hand the organization to another admin."},
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
	PermTeamView, PermTeamInvite, PermTeamManage, PermTeamTransferOwner,
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
