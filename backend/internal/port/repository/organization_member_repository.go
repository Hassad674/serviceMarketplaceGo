package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
)

// ListMembersParams groups the filters for listing members of an org.
// Grouped to stay under the 4-parameter limit on repository methods.
type ListMembersParams struct {
	OrganizationID uuid.UUID
	Cursor         string
	Limit          int
}

// OrganizationMemberRepository persists organization memberships — the
// table that links a user to an org with a specific role and title.
// The Owner is stored as a regular row here with Role=RoleOwner; the
// DB partial unique index guarantees there is at most one such row per org.
type OrganizationMemberRepository interface {
	// Create inserts a new membership row. Fails if:
	//   - the user is already a member (UNIQUE on org_id, user_id)
	//   - the member is Owner and another Owner already exists
	//     (partial UNIQUE on role='owner' per org)
	Create(ctx context.Context, member *organization.Member) error

	// FindByID returns the membership with the given id.
	FindByID(ctx context.Context, id uuid.UUID) (*organization.Member, error)

	// FindByOrgAndUser returns the membership row for the given (org, user)
	// pair, or organization.ErrMemberNotFound if the user isn't a member.
	FindByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*organization.Member, error)

	// FindOwner returns the current Owner membership of the org, or
	// ErrMemberNotFound if (somehow) none exists.
	FindOwner(ctx context.Context, orgID uuid.UUID) (*organization.Member, error)

	// FindUserPrimaryOrg returns the organization the user is currently
	// a member of (V1 restriction: one org per user). Used at auth time
	// to populate the JWT context. Returns ErrMemberNotFound if the user
	// is not a member of any org.
	FindUserPrimaryOrg(ctx context.Context, userID uuid.UUID) (*organization.Member, error)

	// List returns a cursor-paginated list of members for an organization,
	// ordered by joined_at DESC, id DESC.
	List(ctx context.Context, params ListMembersParams) ([]*organization.Member, string, error)

	// CountByRole returns how many members hold each role in the org.
	// Used by the service layer to enforce invariants like "cannot remove
	// the last Owner".
	CountByRole(ctx context.Context, orgID uuid.UUID) (map[organization.Role]int, error)

	// Update persists changes (role, title) on an existing membership.
	Update(ctx context.Context, member *organization.Member) error

	// Delete removes the membership row. Does NOT delete the user — that
	// is the service layer's call when the removed member is an operator
	// whose only purpose was to belong to this org.
	Delete(ctx context.Context, id uuid.UUID) error

	// ListMemberUserIDsByOrgIDs returns the user ids of the active
	// members of each given org, keyed by org id. Used by features
	// that need to project an org-level concept onto the underlying
	// users (e.g. aggregating presence across the team).
	ListMemberUserIDsByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)

	// ListUserIDsByRole returns the user ids of every member of the
	// given org whose membership role matches `role`. Used by the
	// role-permissions editor to bump session_version for every
	// affected user after a permission save, so the new (possibly
	// reduced) permission set takes effect on the next request.
	ListUserIDsByRole(ctx context.Context, orgID uuid.UUID, role organization.Role) ([]uuid.UUID, error)
}
