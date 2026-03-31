package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
)

type PaymentInfoRepository struct {
	db *sql.DB
}

func NewPaymentInfoRepository(db *sql.DB) *PaymentInfoRepository {
	return &PaymentInfoRepository{db: db}
}

func (r *PaymentInfoRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*payment.PaymentInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p := &payment.PaymentInfo{}
	var (
		stripeAccID    sql.NullString
		businessType   sql.NullString
		country        sql.NullString
		displayName    sql.NullString
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id,
			stripe_account_id, stripe_verified,
			charges_enabled, payouts_enabled,
			stripe_business_type, stripe_country, stripe_display_name,
			created_at, updated_at
		FROM payment_info
		WHERE user_id = $1`, userID).Scan(
		&p.ID, &p.UserID,
		&stripeAccID, &p.StripeVerified,
		&p.ChargesEnabled, &p.PayoutsEnabled,
		&businessType, &country, &displayName,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, payment.ErrNotFound
		}
		return nil, fmt.Errorf("get payment info: %w", err)
	}

	p.StripeAccountID = stripeAccID.String
	p.StripeBusinessType = businessType.String
	p.StripeCountry = country.String
	p.StripeDisplayName = displayName.String

	return p, nil
}

func (r *PaymentInfoRepository) Upsert(ctx context.Context, info *payment.PaymentInfo) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payment_info (
			id, user_id,
			stripe_account_id, stripe_verified,
			charges_enabled, payouts_enabled,
			stripe_business_type, stripe_country, stripe_display_name,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE SET
			stripe_account_id = EXCLUDED.stripe_account_id,
			stripe_verified = EXCLUDED.stripe_verified,
			charges_enabled = EXCLUDED.charges_enabled,
			payouts_enabled = EXCLUDED.payouts_enabled,
			stripe_business_type = EXCLUDED.stripe_business_type,
			stripe_country = EXCLUDED.stripe_country,
			stripe_display_name = EXCLUDED.stripe_display_name,
			updated_at = NOW()`,
		info.ID, info.UserID,
		nullString(info.StripeAccountID), info.StripeVerified,
		info.ChargesEnabled, info.PayoutsEnabled,
		nullString(info.StripeBusinessType), nullString(info.StripeCountry), nullString(info.StripeDisplayName),
		info.CreatedAt, info.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert payment info: %w", err)
	}
	return nil
}

func (r *PaymentInfoRepository) UpdateStripeFields(ctx context.Context, userID uuid.UUID, stripeAccountID string, stripeVerified bool) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE payment_info SET stripe_account_id = $1, stripe_verified = $2, updated_at = NOW() WHERE user_id = $3`,
		stripeAccountID, stripeVerified, userID)
	if err != nil {
		return fmt.Errorf("update stripe fields: %w", err)
	}
	return nil
}

func (r *PaymentInfoRepository) UpdateStripeSyncFields(ctx context.Context, userID uuid.UUID, input repository.StripeSyncInput) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		UPDATE payment_info SET
			charges_enabled = $1,
			payouts_enabled = $2,
			stripe_verified = $3,
			stripe_business_type = $4,
			stripe_country = $5,
			stripe_display_name = $6,
			updated_at = NOW()
		WHERE user_id = $7`,
		input.ChargesEnabled, input.PayoutsEnabled, input.StripeVerified,
		nullString(input.BusinessType), nullString(input.Country), nullString(input.DisplayName),
		userID)
	if err != nil {
		return fmt.Errorf("update stripe sync fields: %w", err)
	}
	return nil
}

func (r *PaymentInfoRepository) GetByStripeAccountID(ctx context.Context, stripeAccountID string) (*payment.PaymentInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p := &payment.PaymentInfo{}
	var (
		stripeAccID  sql.NullString
		businessType sql.NullString
		country      sql.NullString
		displayName  sql.NullString
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id,
			stripe_account_id, stripe_verified,
			charges_enabled, payouts_enabled,
			stripe_business_type, stripe_country, stripe_display_name,
			created_at, updated_at
		FROM payment_info WHERE stripe_account_id = $1`, stripeAccountID).Scan(
		&p.ID, &p.UserID,
		&stripeAccID, &p.StripeVerified,
		&p.ChargesEnabled, &p.PayoutsEnabled,
		&businessType, &country, &displayName,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, payment.ErrNotFound
		}
		return nil, fmt.Errorf("get payment info by stripe account: %w", err)
	}

	p.StripeAccountID = stripeAccID.String
	p.StripeBusinessType = businessType.String
	p.StripeCountry = country.String
	p.StripeDisplayName = displayName.String

	return p, nil
}

// nullString converts an empty string to a sql.NullString with Valid=false.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
