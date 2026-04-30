package repository

// Segregated reader / writer / Stripe-store interfaces for the
// organization feature. Carved out of OrganizationRepository (20
// methods).
//
// Three families:
//   - OrganizationReader     — discovery and lookup paths (id, owner,
//     user, Stripe-account, KYC pending list, "any team has a Stripe
//     account?" enumerator, dashboard count).
//   - OrganizationWriter     — life-cycle CRUD plus role-overrides JSONB
//     persistence.
//   - OrganizationStripeStore — every Stripe Connect / KYC field
//     stored on the org row. Owned by the payment + embedded features —
//     pulled into its own port so a wallet handler that only needs the
//     account id does not pull in the full membership API.

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
)

// OrganizationReader exposes read paths over the organizations table.
// All stripe-account fields live in OrganizationStripeStore — this
// interface deals with the org rows themselves only.
type OrganizationReader interface {
	FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
	FindByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*organization.Organization, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*organization.Organization, error)
	FindByStripeAccountID(ctx context.Context, accountID string) (*organization.Organization, error)
	CountAll(ctx context.Context) (int, error)
	ListKYCPending(ctx context.Context) ([]*organization.Organization, error)
	ListWithStripeAccount(ctx context.Context) ([]uuid.UUID, error)
}

// OrganizationWriter exposes mutation paths over the organizations
// table itself (create, update, delete, role-overrides). Stripe state
// changes live in OrganizationStripeStore.
type OrganizationWriter interface {
	Create(ctx context.Context, org *organization.Organization) error
	CreateWithOwnerMembership(ctx context.Context, org *organization.Organization, member *organization.Member) error
	Update(ctx context.Context, org *organization.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveRoleOverrides(ctx context.Context, orgID uuid.UUID, overrides organization.RoleOverrides) error
}

// OrganizationStripeStore exposes the Stripe Connect / KYC bookkeeping
// stored on the organization row. The payment service, embedded
// onboarding service and webhook router each consume this — they never
// need the full OrganizationWriter / Reader API.
type OrganizationStripeStore interface {
	GetStripeAccount(ctx context.Context, orgID uuid.UUID) (accountID, country string, err error)
	GetStripeAccountByUserID(ctx context.Context, userID uuid.UUID) (accountID, country string, err error)
	SetStripeAccount(ctx context.Context, orgID uuid.UUID, accountID, country string) error
	ClearStripeAccount(ctx context.Context, orgID uuid.UUID) error
	GetStripeLastState(ctx context.Context, orgID uuid.UUID) ([]byte, error)
	SaveStripeLastState(ctx context.Context, orgID uuid.UUID, state []byte) error
	SetKYCFirstEarning(ctx context.Context, orgID uuid.UUID, at time.Time) error
	SaveKYCNotificationState(ctx context.Context, orgID uuid.UUID, state map[string]time.Time) error
}

// Compile-time guarantee that the wide OrganizationRepository contract
// is always equivalent to the union of its segregated children.
var _ OrganizationRepository = (interface {
	OrganizationReader
	OrganizationWriter
	OrganizationStripeStore
})(nil)
