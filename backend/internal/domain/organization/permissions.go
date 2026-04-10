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
