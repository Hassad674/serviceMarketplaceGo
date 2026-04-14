package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referrerpricing"
)

// ReferrerPricingRepository is the PostgreSQL-backed implementation
// of repository.ReferrerPricingRepository. Mirrors the freelance
// pricing adapter — same shape, different table and different
// domain types.
type ReferrerPricingRepository struct {
	db *sql.DB
}

// NewReferrerPricingRepository returns a repository ready to talk
// to the given *sql.DB.
func NewReferrerPricingRepository(db *sql.DB) *ReferrerPricingRepository {
	return &ReferrerPricingRepository{db: db}
}

const referrerPricingSelectColumns = `
	profile_id, pricing_type, min_amount, max_amount,
	currency, pricing_note, negotiable, created_at, updated_at`

// Upsert writes or updates the pricing row identified by profile_id.
func (r *ReferrerPricingRepository) Upsert(ctx context.Context, p *referrerpricing.Pricing) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var maxAmount sql.NullInt64
	if p.MaxAmount != nil {
		maxAmount = sql.NullInt64{Int64: *p.MaxAmount, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO referrer_pricing (
			profile_id, pricing_type, min_amount, max_amount,
			currency, pricing_note, negotiable
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (profile_id) DO UPDATE
		SET pricing_type = EXCLUDED.pricing_type,
		    min_amount   = EXCLUDED.min_amount,
		    max_amount   = EXCLUDED.max_amount,
		    currency     = EXCLUDED.currency,
		    pricing_note = EXCLUDED.pricing_note,
		    negotiable   = EXCLUDED.negotiable`,
		p.ProfileID, string(p.Type), p.MinAmount, maxAmount,
		p.Currency, p.Note, p.Negotiable,
	)
	if err != nil {
		return fmt.Errorf("upsert referrer pricing: %w", err)
	}
	return nil
}

// FindByProfileID returns the pricing row for the given profile.
func (r *ReferrerPricingRepository) FindByProfileID(ctx context.Context, profileID uuid.UUID) (*referrerpricing.Pricing, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+referrerPricingSelectColumns+`
		  FROM referrer_pricing
		 WHERE profile_id = $1`,
		profileID,
	)

	p, err := scanReferrerPricingRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, referrerpricing.ErrPricingNotFound
		}
		return nil, fmt.Errorf("find referrer pricing: %w", err)
	}
	return p, nil
}

// DeleteByProfileID removes the pricing row. Idempotent.
func (r *ReferrerPricingRepository) DeleteByProfileID(ctx context.Context, profileID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM referrer_pricing WHERE profile_id = $1`,
		profileID,
	); err != nil {
		return fmt.Errorf("delete referrer pricing: %w", err)
	}
	return nil
}

// scanReferrerPricingRow decodes one SQL row into a Pricing.
func scanReferrerPricingRow(row *sql.Row) (*referrerpricing.Pricing, error) {
	var (
		p       referrerpricing.Pricing
		ptype   string
		maxAmt  sql.NullInt64
	)
	if err := row.Scan(
		&p.ProfileID, &ptype, &p.MinAmount, &maxAmt,
		&p.Currency, &p.Note, &p.Negotiable,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	p.Type = referrerpricing.PricingType(ptype)
	if maxAmt.Valid {
		v := maxAmt.Int64
		p.MaxAmount = &v
	}
	return &p, nil
}
