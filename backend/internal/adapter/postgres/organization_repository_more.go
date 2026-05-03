package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
)

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

// WithStripeAccountLock serialises concurrent Stripe account
// check-and-create flows for the given org via a PostgreSQL
// transaction-scoped advisory lock. Closes BUG-04: two concurrent
// CreateAccountSession requests from the same org could each see
// "no account" → each create one → orphan Stripe-side, only the
// last persisted in DB.
//
// The advisory lock key is hashtext(org_id) to fit Postgres' bigint
// (advisory locks take int8 keys, not UUIDs). Hash collisions across
// different orgs are extremely unlikely (32-bit hashtext into 64-bit
// space) and would only cause minor over-serialisation, never
// incorrect behaviour. Different orgs lock independently.
//
// The lock is held for the lifetime of fn and released exactly when
// the transaction commits (ROLLBACK / COMMIT — both release advisory
// locks taken via pg_advisory_xact_lock). fn's error is propagated
// as-is; the lock scope shields the caller from worrying about
// partial commits leaking the lock.
//
// NOTE: fn receives a context that is the same as the caller's
// (no per-call sub-context with the tx attached). The tx itself is
// held privately inside this method; fn must use the standard
// repository methods which open their own short transactions. That
// is intentional: this helper provides MUTEX semantics, not "give
// me a tx handle". Mixing the two would force every caller to know
// about transactions, which leaks the abstraction.
func (r *OrganizationRepository) WithStripeAccountLock(
	ctx context.Context,
	orgID uuid.UUID,
	fn func(ctx context.Context) error,
) error {
	// queryTimeout caps the lock wait. If contention is high enough
	// that a caller waits >5s for the lock, surface as an error so
	// the request fails loudly instead of silently piling up.
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin advisory lock tx: %w", err)
	}
	// Roll back unless we explicitly commit on success — so the lock
	// is always released, even on panic or fn failure.
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// pg_advisory_xact_lock blocks until the lock is acquired. The
	// lock is automatically released at COMMIT or ROLLBACK — no
	// pg_advisory_unlock call needed. hashtext converts the UUID
	// string into a 32-bit int that fits the advisory lock API.
	if _, err := tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`,
		orgID.String(),
	); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}

	if err := fn(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit advisory lock tx: %w", err)
	}
	committed = true
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
