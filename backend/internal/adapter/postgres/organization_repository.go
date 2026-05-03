package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/organization"
)

// OrganizationRepository persists Organization entities.
//
// All queries use a 5-second context timeout (inherited from the shared
// queryTimeout constant) and go through parameterized statements. On
// unique-constraint violations (owner already has an org, duplicate id)
// the repository returns the appropriate domain sentinel.
//
// starterApplicationCredits is the value written into
// organizations.application_credits when a brand-new org row is
// inserted. It is injected at construction time from cmd/api/main.go
// (which sources it from domain/job.WeeklyQuota) so the organization
// package never has to import the job package — that cross-feature
// import would break the modular architecture rule.
type OrganizationRepository struct {
	db                        *sql.DB
	starterApplicationCredits int
}

// NewOrganizationRepository wires the organization repository.
//
// starterApplicationCredits must match the job feature's WeeklyQuota
// (10 at the time of writing). Every new organization is seeded with
// this many application credits at creation time — reproducing the
// pre-team-refactor behavior where a per-user row was auto-created
// with a 10-credit weekly pool on first read. The value is taken as
// a plain int parameter so this package has zero dependency on the
// job domain.
func NewOrganizationRepository(db *sql.DB, starterApplicationCredits int) *OrganizationRepository {
	return &OrganizationRepository{
		db:                        db,
		starterApplicationCredits: starterApplicationCredits,
	}
}

const orgColumns = `
	id, owner_user_id, type, name,
	stripe_account_id, stripe_account_country, stripe_last_state,
	kyc_first_earning_at, kyc_restriction_notified_at,
	pending_transfer_to_user_id, pending_transfer_initiated_at, pending_transfer_expires_at,
	role_overrides,
	auto_payout_enabled_at,
	created_at, updated_at`

// Create inserts a new organization row. Returns ErrOrgAlreadyExists-style
// error (ErrOwnerAlreadyExists) when the owner already has an organization,
// enforced by the UNIQUE(owner_user_id) constraint in migration 053.
func (r *OrganizationRepository) Create(ctx context.Context, org *organization.Organization) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	kycNotified, err := marshalKYCNotified(org.KYCRestrictionNotifiedAt)
	if err != nil {
		return fmt.Errorf("marshal kyc notified: %w", err)
	}

	overridesJSON, err := marshalRoleOverrides(org.RoleOverrides)
	if err != nil {
		return fmt.Errorf("marshal role overrides: %w", err)
	}

	// application_credits is seeded explicitly — the column has a
	// DEFAULT of 0 in the schema (migration 075) which is safe but
	// unusable for new orgs: the job feature needs a starter pool of
	// WeeklyQuota credits on the very first read so a freshly-created
	// team can immediately apply to jobs without waiting for a refill.
	query := `
		INSERT INTO organizations (
			id, owner_user_id, type, name,
			stripe_account_id, stripe_account_country, stripe_last_state,
			kyc_first_earning_at, kyc_restriction_notified_at,
			pending_transfer_to_user_id, pending_transfer_initiated_at, pending_transfer_expires_at,
			role_overrides,
			auto_payout_enabled_at,
			application_credits, credits_last_reset_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`

	_, err = r.db.ExecContext(ctx, query,
		org.ID, org.OwnerUserID, string(org.Type), org.Name,
		org.StripeAccountID, org.StripeAccountCountry, org.StripeLastState,
		org.KYCFirstEarningAt, kycNotified,
		org.PendingTransferToUserID, org.PendingTransferInitiatedAt, org.PendingTransferExpiresAt,
		overridesJSON,
		org.AutoPayoutEnabledAt,
		r.starterApplicationCredits, org.CreatedAt,
		org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return organization.ErrOwnerAlreadyExists
		}
		return fmt.Errorf("insert organization: %w", err)
	}
	return nil
}

// FindByID returns the organization with the given id.
func (r *OrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT ` + orgColumns + ` FROM organizations WHERE id = $1`
	return r.scanOne(r.db.QueryRowContext(ctx, query, id))
}

// FindByOwnerUserID returns the organization owned by the given user, or
// ErrOrgNotFound if no such org exists. Used at JWT issuance to resolve
// a user's org context.
func (r *OrganizationRepository) FindByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*organization.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT ` + orgColumns + ` FROM organizations WHERE owner_user_id = $1`
	return r.scanOne(r.db.QueryRowContext(ctx, query, ownerUserID))
}

// FindByUserID returns the organization the given user currently
// belongs to.
//
// Resolution path: organization_members → organizations, mirroring
// GetStripeAccountByUserID and ResolveContext (the JWT issuance
// path). The legacy users.organization_id join can drift out of sync
// with organization_members on team membership changes or partial
// backfills, which previously caused RetryFailedTransfer to
// resolve a stale org and reject a legitimate retry with
// transfer_not_retriable while the wallet UI happily showed the
// payout as ready.
func (r *OrganizationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*organization.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT ` + orgColumns + `
		FROM organizations o
		JOIN organization_members om ON om.organization_id = o.id
		WHERE om.user_id = $1
		ORDER BY om.joined_at DESC
		LIMIT 1`
	return r.scanOne(r.db.QueryRowContext(ctx, query, userID))
}

// FindByStripeAccountID returns the organization that owns the given
// Stripe Connect account. Used by webhooks routing Stripe events back
// to the merchant org.
func (r *OrganizationRepository) FindByStripeAccountID(ctx context.Context, accountID string) (*organization.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT ` + orgColumns + ` FROM organizations WHERE stripe_account_id = $1`
	return r.scanOne(r.db.QueryRowContext(ctx, query, accountID))
}

// Update persists changes to the organization. Covers renames,
// Stripe account onboarding, KYC bookkeeping, and ownership transfers.
func (r *OrganizationRepository) Update(ctx context.Context, org *organization.Organization) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	kycNotified, err := marshalKYCNotified(org.KYCRestrictionNotifiedAt)
	if err != nil {
		return fmt.Errorf("marshal kyc notified: %w", err)
	}

	overridesJSON, err := marshalRoleOverrides(org.RoleOverrides)
	if err != nil {
		return fmt.Errorf("marshal role overrides: %w", err)
	}

	query := `
		UPDATE organizations
		SET owner_user_id                 = $2,
		    type                          = $3,
		    name                          = $4,
		    stripe_account_id             = $5,
		    stripe_account_country        = $6,
		    stripe_last_state             = $7,
		    kyc_first_earning_at          = $8,
		    kyc_restriction_notified_at   = $9,
		    pending_transfer_to_user_id   = $10,
		    pending_transfer_initiated_at = $11,
		    pending_transfer_expires_at   = $12,
		    role_overrides                = $13,
		    auto_payout_enabled_at        = $14,
		    updated_at                    = now()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		org.ID, org.OwnerUserID, string(org.Type), org.Name,
		org.StripeAccountID, org.StripeAccountCountry, org.StripeLastState,
		org.KYCFirstEarningAt, kycNotified,
		org.PendingTransferToUserID, org.PendingTransferInitiatedAt, org.PendingTransferExpiresAt,
		overridesJSON,
		org.AutoPayoutEnabledAt,
	)
	if err != nil {
		return fmt.Errorf("update organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrOrgNotFound
	}
	return nil
}

// SaveRoleOverrides persists just the role_overrides column of an
// organization. Used by the role-permissions editor so a permission
// save does not have to touch every other column (Stripe state, KYC
// bookkeeping, pending transfer, …) just to bump one JSON blob.
//
// The caller is responsible for passing an already-validated
// RoleOverrides map — the domain's SetRoleOverride / ReplaceRoleOverrides
// enforce the non-overridable rules and the app layer applies them
// before reaching this method.
func (r *OrganizationRepository) SaveRoleOverrides(
	ctx context.Context,
	orgID uuid.UUID,
	overrides organization.RoleOverrides,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	payload, err := marshalRoleOverrides(overrides)
	if err != nil {
		return fmt.Errorf("marshal role overrides: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		SET role_overrides = $2, updated_at = now()
		WHERE id = $1`,
		orgID, payload,
	)
	if err != nil {
		return fmt.Errorf("save role overrides: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrOrgNotFound
	}
	return nil
}

// Delete removes an organization row. CASCADE deletes wipe members and
// invitations. V1 blocks this from the UI — exposed only for admin use
// and the /remove-feature tooling.
func (r *OrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, "DELETE FROM organizations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrOrgNotFound
	}
	return nil
}

// CountAll returns the total number of organizations on the platform.
// Used by the admin dashboard to surface an "Organisations" stat tile.
// Single aggregate query — no filters.
func (r *OrganizationRepository) CountAll(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count organizations: %w", err)
	}
	return count, nil
}

// ListKYCPending returns all orgs that have earned at least once and
// have not yet completed KYC. The caller (scheduler) sweeps this list
// to decide when to send a reminder or block the team's wallet.
func (r *OrganizationRepository) ListKYCPending(ctx context.Context) ([]*organization.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `SELECT ` + orgColumns + `
		FROM organizations
		WHERE kyc_first_earning_at IS NOT NULL
		  AND stripe_account_id IS NULL
		ORDER BY kyc_first_earning_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list kyc pending orgs: %w", err)
	}
	defer rows.Close()

	var orgs []*organization.Organization
	for rows.Next() {
		org, scanErr := r.scanRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan kyc pending org: %w", scanErr)
		}
		orgs = append(orgs, org)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return orgs, nil
}

// ListWithStripeAccount returns the ids of every organization that
// has completed Stripe Connect onboarding (stripe_account_id IS NOT
// NULL). Used by the invoicing scheduler to enumerate orgs that may
// need a monthly commission invoice. Returns an empty slice when the
// platform has no onboarded merchants — never returns nil to keep
// callers free of nil-checks.
func (r *OrganizationRepository) ListWithStripeAccount(ctx context.Context) ([]uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT id
		FROM organizations
		WHERE stripe_account_id IS NOT NULL
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list orgs with stripe account: %w", err)
	}
	defer rows.Close()

	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan org id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return out, nil
}

// CreateWithOwnerMembership is the atomic convenience method used at
// account registration. It creates the organization row and the
// corresponding Owner membership in a single transaction so both sides
// are always consistent.
//
// This method exists on OrganizationRepository (not on the member repo)
// because the member insert is tightly coupled to the org insert — you
// never create one without the other at registration.
func (r *OrganizationRepository) CreateWithOwnerMembership(
	ctx context.Context,
	org *organization.Organization,
	member *organization.Member,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	kycNotified, err := marshalKYCNotified(org.KYCRestrictionNotifiedAt)
	if err != nil {
		return fmt.Errorf("marshal kyc notified: %w", err)
	}
	overridesJSON, err := marshalRoleOverrides(org.RoleOverrides)
	if err != nil {
		return fmt.Errorf("marshal role overrides: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// application_credits is seeded here at org creation time, which
	// is the only correct place now that credits are org-scoped (R12).
	// Before the team refactor the seeding happened on first read of a
	// per-user row; that pathway no longer exists, so a brand-new org
	// without this explicit seed would be born with 0 credits and the
	// team could not apply to any job until the weekly refill cron ran.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organizations (
			id, owner_user_id, type, name,
			stripe_account_id, stripe_account_country, stripe_last_state,
			kyc_first_earning_at, kyc_restriction_notified_at,
			pending_transfer_to_user_id, pending_transfer_initiated_at, pending_transfer_expires_at,
			role_overrides,
			application_credits, credits_last_reset_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		org.ID, org.OwnerUserID, string(org.Type), org.Name,
		org.StripeAccountID, org.StripeAccountCountry, org.StripeLastState,
		org.KYCFirstEarningAt, kycNotified,
		org.PendingTransferToUserID, org.PendingTransferInitiatedAt, org.PendingTransferExpiresAt,
		overridesJSON,
		r.starterApplicationCredits, org.CreatedAt,
		org.CreatedAt, org.UpdatedAt,
	); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return organization.ErrOwnerAlreadyExists
		}
		return fmt.Errorf("insert organization: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organization_members (
			id, organization_id, user_id, role, title, joined_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		member.ID, member.OrganizationID, member.UserID, string(member.Role),
		member.Title, member.JoinedAt, member.CreatedAt, member.UpdatedAt,
	); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return organization.ErrAlreadyMember
		}
		return fmt.Errorf("insert owner membership: %w", err)
	}

	// Denormalize the org onto users.organization_id so single-row lookups
	// (JWT refresh, /me, resource backfills) can read it without joining
	// organization_members. organization_members remains the source of
	// truth for membership state.
	if _, err := tx.ExecContext(ctx, `
		UPDATE users SET organization_id = $1, updated_at = now() WHERE id = $2`,
		org.ID, member.UserID,
	); err != nil {
		return fmt.Errorf("set user organization_id: %w", err)
	}

	return tx.Commit()
}
