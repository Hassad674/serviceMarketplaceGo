package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
)

// OrganizationRepository persists Organization entities.
//
// The org is one of the three tables of the team feature (organizations,
// organization_members, organization_invitations). Each has its own
// repository so a consumer that only needs members doesn't drag in
// invitation logic, and so mocks can be kept focused (interface segregation).
type OrganizationRepository interface {
	// Create inserts a new organization. Fails with ErrAlreadyMember-like
	// semantics if the owner already owns another org (V1 constraint
	// enforced by the UNIQUE constraint on owner_user_id).
	Create(ctx context.Context, org *organization.Organization) error

	// CreateWithOwnerMembership atomically inserts both the organization
	// row and the corresponding Owner membership in a single transaction,
	// so the two never get out of sync at registration time.
	CreateWithOwnerMembership(ctx context.Context, org *organization.Organization, member *organization.Member) error

	// FindByID returns the organization with the given id, or
	// organization.ErrOrgNotFound when no row matches.
	FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error)

	// FindByOwnerUserID returns the organization owned by the given user,
	// or organization.ErrOrgNotFound when the user owns no org.
	// Useful for resolving the "my org" context at JWT issuance.
	FindByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*organization.Organization, error)

	// Update persists changes to the organization row. Uses optimistic
	// update on the primary key. Callers mutate the domain entity via its
	// methods and then call Update to flush.
	Update(ctx context.Context, org *organization.Organization) error

	// Delete removes the organization. CASCADE will wipe members and
	// invitations. V1 blocks this from the UI — provided here for admin
	// use and for the /remove-feature tooling.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountAll returns the total number of organizations on the
	// platform. Used by the admin dashboard to surface a team-aware
	// "Organisations" tile. O(n) scan — acceptable for V1 volumes
	// and refreshed at most once per dashboard page load.
	CountAll(ctx context.Context) (int, error)
}
