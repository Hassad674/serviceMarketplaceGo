package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/freelancepricing"
)

// FreelancePricingRepository is the PostgreSQL-backed implementation
// of repository.FreelancePricingRepository. Owns the
// freelance_pricing table (migration 099), keyed by profile_id
// (one row per freelance profile).
type FreelancePricingRepository struct {
	db *sql.DB
}

// NewFreelancePricingRepository returns a repository ready to talk
// to the given *sql.DB.
func NewFreelancePricingRepository(db *sql.DB) *FreelancePricingRepository {
	return &FreelancePricingRepository{db: db}
}

const freelancePricingSelectColumns = `
	profile_id, pricing_type, min_amount, max_amount,
	currency, pricing_note, negotiable, created_at, updated_at`

// Upsert writes or updates the pricing row identified by profile_id.
// Primary-key collision updates in place; the trigger bumps
// updated_at.
func (r *FreelancePricingRepository) Upsert(ctx context.Context, p *freelancepricing.Pricing) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var maxAmount sql.NullInt64
	if p.MaxAmount != nil {
		maxAmount = sql.NullInt64{Int64: *p.MaxAmount, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO freelance_pricing (
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
		return fmt.Errorf("upsert freelance pricing: %w", err)
	}
	return nil
}

// FindByProfileID returns the pricing row for the given profile.
func (r *FreelancePricingRepository) FindByProfileID(ctx context.Context, profileID uuid.UUID) (*freelancepricing.Pricing, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+freelancePricingSelectColumns+`
		  FROM freelance_pricing
		 WHERE profile_id = $1`,
		profileID,
	)

	p, err := scanFreelancePricingRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, freelancepricing.ErrPricingNotFound
		}
		return nil, fmt.Errorf("find freelance pricing: %w", err)
	}
	return p, nil
}

// DeleteByProfileID removes the pricing row. Idempotent.
func (r *FreelancePricingRepository) DeleteByProfileID(ctx context.Context, profileID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM freelance_pricing WHERE profile_id = $1`,
		profileID,
	); err != nil {
		return fmt.Errorf("delete freelance pricing: %w", err)
	}
	return nil
}

// scanFreelancePricingRow decodes one SQL row into a Pricing.
func scanFreelancePricingRow(row *sql.Row) (*freelancepricing.Pricing, error) {
	var (
		p       freelancepricing.Pricing
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
	p.Type = freelancepricing.PricingType(ptype)
	if maxAmt.Valid {
		v := maxAmt.Int64
		p.MaxAmount = &v
	}
	return &p, nil
}
