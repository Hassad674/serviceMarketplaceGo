package repository

import (
	"context"
	"time"

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

	// FindByUserID returns the organization the given user currently
	// belongs to (via users.organization_id). Used by flows that carry
	// a user_id and need the full org — e.g. KYC enforcement middleware
	// or payment flows that check the merchant-of-record's state.
	FindByUserID(ctx context.Context, userID uuid.UUID) (*organization.Organization, error)

	// Update persists changes to the organization row. Uses optimistic
	// update on the primary key. Callers mutate the domain entity via its
	// methods and then call Update to flush.
	Update(ctx context.Context, org *organization.Organization) error

	// Delete removes the organization. CASCADE will wipe members and
	// invitations. V1 blocks this from the UI — provided here for admin
	// use and for the /remove-feature tooling.
	Delete(ctx context.Context, id uuid.UUID) error

	// SaveRoleOverrides persists just the JSONB role_overrides column
	// of the given organization, leaving every other field untouched.
	// Used by the role-permissions editor so a permission save does
	// not have to write through the full row. Returns ErrOrgNotFound
	// when the org does not exist.
	SaveRoleOverrides(ctx context.Context, orgID uuid.UUID, overrides organization.RoleOverrides) error

	// CountAll returns the total number of organizations on the
	// platform. Used by the admin dashboard to surface a team-aware
	// "Organisations" tile. O(n) scan — acceptable for V1 volumes
	// and refreshed at most once per dashboard page load.
	CountAll(ctx context.Context) (int, error)

	// FindByStripeAccountID returns the org that owns the given
	// Stripe Connect account. Used by webhooks to route Stripe
	// events back to the merchant org.
	FindByStripeAccountID(ctx context.Context, accountID string) (*organization.Organization, error)

	// ListKYCPending returns orgs that have earned at least once and
	// have not yet completed KYC. Used by the enforcement scheduler
	// to decide when to send a reminder or block the team's wallet.
	ListKYCPending(ctx context.Context) ([]*organization.Organization, error)

	// Stripe Connect account operations (moved from users in phase R5).
	// These all operate on the given org's row.
	GetStripeAccount(ctx context.Context, orgID uuid.UUID) (accountID, country string, err error)
	// GetStripeAccountByUserID resolves the Stripe account of whichever
	// org the given user currently belongs to — a single JOIN so payment
	// flows that carry a user_id (e.g. a proposal's provider_id) don't
	// need an extra round-trip.
	GetStripeAccountByUserID(ctx context.Context, userID uuid.UUID) (accountID, country string, err error)
	SetStripeAccount(ctx context.Context, orgID uuid.UUID, accountID, country string) error
	ClearStripeAccount(ctx context.Context, orgID uuid.UUID) error
	GetStripeLastState(ctx context.Context, orgID uuid.UUID) ([]byte, error)
	SaveStripeLastState(ctx context.Context, orgID uuid.UUID, state []byte) error

	// KYC enforcement — mirrors the old user-level API, now org-keyed.
	SetKYCFirstEarning(ctx context.Context, orgID uuid.UUID, at time.Time) error
	SaveKYCNotificationState(ctx context.Context, orgID uuid.UUID, state map[string]time.Time) error
}
