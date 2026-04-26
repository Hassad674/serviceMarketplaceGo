package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
)

// BillingProfileRepository implements repository.BillingProfileRepository
// against Postgres. One row per organization, keyed on organization_id.
type BillingProfileRepository struct {
	db *sql.DB
}

func NewBillingProfileRepository(db *sql.DB) *BillingProfileRepository {
	return &BillingProfileRepository{db: db}
}

// billingProfileColumns lists every column the SELECT scan path reads.
// Kept as a const so the FindByOrganization SELECT and any future helper
// scan in the same order without drift.
const billingProfileColumns = `
	organization_id, profile_type, legal_name, trading_name, legal_form,
	tax_id, vat_number, vat_validated_at, vat_validation_payload,
	address_line1, address_line2, postal_code, city, country,
	invoicing_email, synced_from_kyc_at, created_at, updated_at
`

// FindByOrganization returns the billing profile for the given org or
// invoicing.ErrNotFound when the org has not yet seeded one.
func (r *BillingProfileRepository) FindByOrganization(ctx context.Context, organizationID uuid.UUID) (*invoicing.BillingProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+billingProfileColumns+`
		FROM billing_profile
		WHERE organization_id = $1`, organizationID)

	p, err := scanBillingProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invoicing.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find billing profile by organization: %w", err)
	}
	return p, nil
}

// Upsert writes the profile, creating a new row or updating mutable
// columns of the existing one. created_at is preserved on conflict —
// only updated_at and the user-mutable fields move.
func (r *BillingProfileRepository) Upsert(ctx context.Context, p *invoicing.BillingProfile) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// vat_validation_payload is opaque JSONB. Empty []byte must be sent as
	// NULL, not as the string "" (which would fail JSONB validation).
	var vatPayload interface{}
	if len(p.VATValidationPayload) > 0 {
		vatPayload = []byte(p.VATValidationPayload)
	} else {
		vatPayload = nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO billing_profile (
			organization_id, profile_type, legal_name, trading_name, legal_form,
			tax_id, vat_number, vat_validated_at, vat_validation_payload,
			address_line1, address_line2, postal_code, city, country,
			invoicing_email, synced_from_kyc_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, now(), now()
		)
		ON CONFLICT (organization_id) DO UPDATE SET
			profile_type = EXCLUDED.profile_type,
			legal_name = EXCLUDED.legal_name,
			trading_name = EXCLUDED.trading_name,
			legal_form = EXCLUDED.legal_form,
			tax_id = EXCLUDED.tax_id,
			vat_number = EXCLUDED.vat_number,
			vat_validated_at = EXCLUDED.vat_validated_at,
			vat_validation_payload = EXCLUDED.vat_validation_payload,
			address_line1 = EXCLUDED.address_line1,
			address_line2 = EXCLUDED.address_line2,
			postal_code = EXCLUDED.postal_code,
			city = EXCLUDED.city,
			country = EXCLUDED.country,
			invoicing_email = EXCLUDED.invoicing_email,
			synced_from_kyc_at = EXCLUDED.synced_from_kyc_at,
			updated_at = now()
		`,
		p.OrganizationID, string(p.ProfileType), p.LegalName, p.TradingName, p.LegalForm,
		p.TaxID, p.VATNumber, p.VATValidatedAt, vatPayload,
		p.AddressLine1, p.AddressLine2, p.PostalCode, p.City, p.Country,
		p.InvoicingEmail, p.SyncedFromKYCAt,
	)
	if err != nil {
		return fmt.Errorf("upsert billing profile: %w", err)
	}
	return nil
}

// scanBillingProfile reads a row into the domain type. Nullable columns
// flow through sql.Null* and are normalised to *time.Time / []byte.
func scanBillingProfile(row *sql.Row) (*invoicing.BillingProfile, error) {
	var (
		p                 invoicing.BillingProfile
		profileType       string
		vatValidatedAt    sql.NullTime
		vatPayload        []byte
		syncedFromKYCAt   sql.NullTime
	)
	err := row.Scan(
		&p.OrganizationID, &profileType, &p.LegalName, &p.TradingName, &p.LegalForm,
		&p.TaxID, &p.VATNumber, &vatValidatedAt, &vatPayload,
		&p.AddressLine1, &p.AddressLine2, &p.PostalCode, &p.City, &p.Country,
		&p.InvoicingEmail, &syncedFromKYCAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.ProfileType = invoicing.ProfileType(profileType)
	if vatValidatedAt.Valid {
		t := vatValidatedAt.Time
		p.VATValidatedAt = &t
	}
	if len(vatPayload) > 0 {
		p.VATValidationPayload = vatPayload
	}
	if syncedFromKYCAt.Valid {
		t := syncedFromKYCAt.Time
		p.SyncedFromKYCAt = &t
	}
	return &p, nil
}
