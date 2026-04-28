package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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

// orgRowScanner is satisfied by both *sql.Row and *sql.Rows.
type orgRowScanner interface {
	Scan(dest ...any) error
}

// scanOne turns a sql.Row into a domain Organization, mapping common
// errors to domain sentinels.
func (r *OrganizationRepository) scanOne(row *sql.Row) (*organization.Organization, error) {
	org, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, organization.ErrOrgNotFound
		}
		return nil, fmt.Errorf("scan organization: %w", err)
	}
	return org, nil
}

func (r *OrganizationRepository) scanRow(s orgRowScanner) (*organization.Organization, error) {
	var (
		org           organization.Organization
		orgType       string
		kycNotified   []byte
		roleOverrides []byte
	)
	err := s.Scan(
		&org.ID, &org.OwnerUserID, &orgType, &org.Name,
		&org.StripeAccountID, &org.StripeAccountCountry, &org.StripeLastState,
		&org.KYCFirstEarningAt, &kycNotified,
		&org.PendingTransferToUserID, &org.PendingTransferInitiatedAt, &org.PendingTransferExpiresAt,
		&roleOverrides,
		&org.AutoPayoutEnabledAt,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	org.Type = organization.OrgType(orgType)
	org.KYCRestrictionNotifiedAt = unmarshalKYCNotified(kycNotified)
	org.RoleOverrides = unmarshalRoleOverrides(roleOverrides)
	return &org, nil
}

// marshalRoleOverrides serializes the RoleOverrides map into the
// string-keyed JSON shape stored in the role_overrides JSONB column.
// Nil and empty inputs both produce `{}` so the NOT NULL DEFAULT
// on the column is honored.
func marshalRoleOverrides(overrides organization.RoleOverrides) ([]byte, error) {
	if len(overrides) == 0 {
		return []byte(`{}`), nil
	}
	// Convert typed keys to strings for JSON serialization. We cannot
	// rely on json.Marshal of map[Role]map[Permission]bool because Go
	// does not know these are string-typed aliases of strings at
	// encoding time — explicit conversion keeps the JSON shape stable.
	out := make(map[string]map[string]bool, len(overrides))
	for role, perms := range overrides {
		inner := make(map[string]bool, len(perms))
		for p, v := range perms {
			inner[string(p)] = v
		}
		out[string(role)] = inner
	}
	return json.Marshal(out)
}

// unmarshalRoleOverrides is the inverse of marshalRoleOverrides.
// A nil or empty byte slice returns an empty (non-nil) overrides map
// so downstream code never has to check for nil.
func unmarshalRoleOverrides(data []byte) organization.RoleOverrides {
	out := organization.RoleOverrides{}
	if len(data) == 0 {
		return out
	}
	raw := map[string]map[string]bool{}
	if err := json.Unmarshal(data, &raw); err != nil {
		// A malformed JSON payload is treated as empty rather than
		// propagating an error — overrides are customization data,
		// not load-bearing state. The defaults kick in automatically.
		return out
	}
	for roleKey, perms := range raw {
		role := organization.Role(roleKey)
		inner := make(map[organization.Permission]bool, len(perms))
		for permKey, v := range perms {
			inner[organization.Permission(permKey)] = v
		}
		out[role] = inner
	}
	return out
}

// ---------------------------------------------------------------------------
// Stripe Connect + KYC (org-keyed since phase R5)
// ---------------------------------------------------------------------------

// GetStripeAccount returns the org's Stripe Connect account id and country.
// Empty strings (not an error) when the org has not started KYC yet —
// callers should interpret the empty account id as "needs onboarding".
func (r *OrganizationRepository) GetStripeAccount(ctx context.Context, orgID uuid.UUID) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var accountID, country sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT stripe_account_id, stripe_account_country FROM organizations WHERE id = $1`,
		orgID,
	).Scan(&accountID, &country)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", organization.ErrOrgNotFound
		}
		return "", "", fmt.Errorf("get stripe account: %w", err)
	}
	return accountID.String, country.String, nil
}

// GetStripeAccountByUserID returns the stripe account of the org the
// given user currently belongs to. Used by payment flows that carry a
// user_id (proposal.client_id / provider_id) and need to resolve the
// merchant-of-record in one query.
//
// Resolution path: organization_members (the same source of truth used
// at login/JWT issuance time by ResolveContext) → organizations. The
// older path joined through users.organization_id, which can drift
// out of sync with organization_members on team membership changes
// or partial backfills — leaving the wallet UI happily showing
// "compte Stripe prêt" while a TransferMilestone resolved through
// users.organization_id returned an empty account id and bounced with
// "provider has no Stripe connected account". Aligning on
// organization_members ensures both readers agree.
func (r *OrganizationRepository) GetStripeAccountByUserID(ctx context.Context, userID uuid.UUID) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var accountID, country sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT o.stripe_account_id, o.stripe_account_country
		FROM organizations o
		JOIN organization_members om ON om.organization_id = o.id
		WHERE om.user_id = $1
		ORDER BY om.joined_at DESC
		LIMIT 1`,
		userID,
	).Scan(&accountID, &country)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", organization.ErrOrgNotFound
		}
		return "", "", fmt.Errorf("get stripe account by user id: %w", err)
	}
	return accountID.String, country.String, nil
}

// SetStripeAccount persists the Stripe Connect account id + country
// after a successful onboarding. Both values are stored together so
// the merchant-of-record info stays consistent.
func (r *OrganizationRepository) SetStripeAccount(ctx context.Context, orgID uuid.UUID, accountID, country string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		SET stripe_account_id = $2, stripe_account_country = $3, updated_at = now()
		WHERE id = $1`,
		orgID, accountID, country,
	)
	if err != nil {
		return fmt.Errorf("set stripe account: %w", err)
	}
	return nil
}

// ClearStripeAccount removes the Stripe account link from the org.
// Used by the dev reset flow.
func (r *OrganizationRepository) ClearStripeAccount(ctx context.Context, orgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		SET stripe_account_id = NULL,
		    stripe_account_country = NULL,
		    stripe_last_state = NULL,
		    updated_at = now()
		WHERE id = $1`,
		orgID,
	)
	if err != nil {
		return fmt.Errorf("clear stripe account: %w", err)
	}
	return nil
}

// GetStripeLastState returns the opaque JSON snapshot of the org's
// last-seen Stripe account state. Used by the embedded Notifier to
// diff incoming webhooks.
func (r *OrganizationRepository) GetStripeLastState(ctx context.Context, orgID uuid.UUID) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var state []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT stripe_last_state FROM organizations WHERE id = $1`,
		orgID,
	).Scan(&state)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, organization.ErrOrgNotFound
		}
		return nil, fmt.Errorf("get stripe last state: %w", err)
	}
	return state, nil
}

// SaveStripeLastState persists the opaque JSON snapshot after a
// successful webhook diff.
func (r *OrganizationRepository) SaveStripeLastState(ctx context.Context, orgID uuid.UUID, state []byte) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations SET stripe_last_state = $2, updated_at = now() WHERE id = $1`,
		orgID, state,
	)
	if err != nil {
		return fmt.Errorf("save stripe last state: %w", err)
	}
	return nil
}

// SetKYCFirstEarning records the first moment the org received
// withdrawable funds. Idempotent: subsequent calls are a no-op because
// only a NULL target is updated.
func (r *OrganizationRepository) SetKYCFirstEarning(ctx context.Context, orgID uuid.UUID, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		SET kyc_first_earning_at = $2, updated_at = now()
		WHERE id = $1 AND kyc_first_earning_at IS NULL`,
		orgID, at,
	)
	if err != nil {
		return fmt.Errorf("set kyc first earning: %w", err)
	}
	return nil
}

// SaveKYCNotificationState persists the tier→timestamp map tracked by
// the KYC scheduler. Stored as JSONB.
func (r *OrganizationRepository) SaveKYCNotificationState(ctx context.Context, orgID uuid.UUID, state map[string]time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal kyc notification state: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE organizations
		SET kyc_restriction_notified_at = $2, updated_at = now()
		WHERE id = $1`,
		orgID, payload,
	)
	if err != nil {
		return fmt.Errorf("save kyc notification state: %w", err)
	}
	return nil
}

func marshalKYCNotified(m map[string]time.Time) ([]byte, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(m)
}

func unmarshalKYCNotified(data []byte) map[string]time.Time {
	out := map[string]time.Time{}
	if len(data) == 0 {
		return out
	}
	_ = json.Unmarshal(data, &out)
	return out
}
