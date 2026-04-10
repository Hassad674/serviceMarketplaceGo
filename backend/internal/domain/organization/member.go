package organization

import (
	"time"

	"github.com/google/uuid"
)

// maxTitleLength caps the free-text job title an operator can hold in the
// org. 100 chars is plenty for "Head of Customer Success, EMEA" and
// similar realistic values while preventing abuse.
const maxTitleLength = 100

// Member represents a user's membership in an organization with a specific
// role and job title. Both Owners and Operators have a row in this table;
// the Owner is a regular member whose role happens to be RoleOwner.
//
// The UNIQUE(organization_id, user_id) constraint enforces that a user
// cannot hold two roles in the same org simultaneously. The partial
// UNIQUE on role='owner' enforces the single-Owner V1 invariant.
type Member struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           Role
	Title          string
	JoinedAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewMember creates a validated org membership row.
//
// The caller must ensure the role assignment is legal given the current
// org state (e.g. don't create a second Owner). The DB partial unique
// index is the last line of defense for the Owner invariant.
func NewMember(orgID, userID uuid.UUID, role Role, title string) (*Member, error) {
	if orgID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrMemberNotFound
	}
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}
	if len(title) > maxTitleLength {
		return nil, ErrTitleTooLong
	}

	now := time.Now()
	return &Member{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		Title:          title,
		JoinedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// ChangeRole updates the member's role and refreshes UpdatedAt.
//
// This method is intentionally permissive on its own — it only validates
// the new role is well-formed. The caller (the app layer) is responsible
// for applying the V1 invariants:
//   - You cannot promote to Owner directly: use the transfer ownership flow
//   - You cannot demote the Owner unless they are demoting themselves as
//     part of an ownership transfer
//   - Admins can freely promote/demote Members and Viewers, and other Admins
//
// Those policies live in the service layer because they require looking
// at the actor vs. the target, their roles, and the org's current state.
func (m *Member) ChangeRole(newRole Role) error {
	if !newRole.IsValid() {
		return ErrInvalidRole
	}
	m.Role = newRole
	m.UpdatedAt = time.Now()
	return nil
}

// UpdateTitle changes the member's free-text job title.
func (m *Member) UpdateTitle(title string) error {
	if len(title) > maxTitleLength {
		return ErrTitleTooLong
	}
	m.Title = title
	m.UpdatedAt = time.Now()
	return nil
}

// HasPermission is a convenience wrapper around the package-level
// HasPermission function, scoped to this member's role.
func (m *Member) HasPermission(perm Permission) bool {
	return HasPermission(m.Role, perm)
}

// IsOwner reports whether this membership holds the Owner role.
func (m *Member) IsOwner() bool {
	return m.Role == RoleOwner
}

// CanManageMember reports whether this member is allowed to change the
// role or remove the target member. It encapsulates the V1 rules:
//
//   - Nobody can touch an Owner except via transfer ownership
//     (Owner protection: returns false if target is Owner)
//   - Owners and Admins can manage any Member or Viewer
//   - Owners and Admins can manage other Admins
//   - Members and Viewers cannot manage anyone
//
// The actor themselves cannot be managed through this check (self-actions
// like leaving the org go through a different path, LeaveOrganization).
func (m *Member) CanManageMember(target *Member) bool {
	if target == nil || m.OrganizationID != target.OrganizationID {
		return false
	}
	if target.IsOwner() {
		return false
	}
	return m.Role == RoleOwner || m.Role == RoleAdmin
}
