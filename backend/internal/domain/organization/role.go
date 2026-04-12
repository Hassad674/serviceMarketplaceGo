package organization

// Role identifies a user's level of authority within an organization.
//
// The four roles are hardcoded for V1. Every role has a fixed permission
// set defined in permissions.go — there is no per-member override in V1.
// Opening up to granular permissions is a V2+ concern and will be handled
// by adding a JSONB override column without changing the core roles.
type Role string

const (
	// RoleOwner is the founder / business owner of the organization.
	// Exactly ONE Owner per org is allowed in V1, enforced at the DB level
	// by a partial unique index on organization_members(organization_id)
	// WHERE role = 'owner'. Multi-Owner is a V2 extension — the V1 code
	// already tolerates the concept architecturally; only this constraint
	// needs to be relaxed to open it up.
	RoleOwner Role = "owner"

	// RoleAdmin is a trusted operator with full operational rights except
	// finances (withdraw), ownership transfer, org deletion and KYC.
	RoleAdmin Role = "admin"

	// RoleMember is the default role for invited operators.
	// Can run daily operations (jobs, proposals, messaging) but cannot
	// manage the team or touch finances beyond viewing balances.
	RoleMember Role = "member"

	// RoleViewer is read-only across all org resources. Used for
	// observers, external accountants, trainees.
	RoleViewer Role = "viewer"
)

// IsValid reports whether the role is a known value.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	}
	return false
}

// String implements fmt.Stringer.
func (r Role) String() string {
	return string(r)
}

// CanBeInvitedAs reports whether a new invitation may assign this role.
// Owner cannot be invited — it is only granted via the transfer ownership
// flow. This prevents an Admin from promoting arbitrary emails to Owner
// by crafting an invitation.
func (r Role) CanBeInvitedAs() bool {
	return r == RoleAdmin || r == RoleMember || r == RoleViewer
}

// IsElevated reports whether the role has management privileges beyond
// viewing. Used by permission checks that want to distinguish "can do
// stuff" from "can only look".
func (r Role) IsElevated() bool {
	return r == RoleOwner || r == RoleAdmin
}

// Level returns a numeric authority level for the role, enabling simple
// comparisons between roles: higher level = more permissions.
//
//	Owner=4, Admin=3, Member=2, Viewer=1, unknown=0
func (r Role) Level() int {
	switch r {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleMember:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// IsDemotion reports whether changing from role "from" to role "to"
// constitutes a demotion (reduction in permissions). This is used to
// decide whether a session invalidation is needed — demotions must
// take effect immediately, while promotions do not require forcing a
// re-login.
func IsDemotion(from, to Role) bool {
	return from.Level() > to.Level()
}
