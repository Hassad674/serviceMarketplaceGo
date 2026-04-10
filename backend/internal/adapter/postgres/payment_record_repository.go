package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type PaymentRecordRepository struct {
	db *sql.DB
}

func NewPaymentRecordRepository(db *sql.DB) *PaymentRecordRepository {
	return &PaymentRecordRepository{db: db}
}

func (r *PaymentRecordRepository) Create(ctx context.Context, rec *payment.PaymentRecord) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// organization_id is resolved from organization_members keyed on the
	// client — Agencies/Enterprises get their org denormalized onto the
	// record, Providers stay NULL. Used by the dashboard wallet view for
	// operators in later phases.
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payment_records (
			id, proposal_id, client_id, provider_id,
			stripe_payment_intent_id, stripe_transfer_id,
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at,
			organization_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			(SELECT organization_id FROM organization_members WHERE user_id = $3 LIMIT 1)
		)`,
		rec.ID, rec.ProposalID, rec.ClientID, rec.ProviderID,
		ptrString(rec.StripePaymentIntentID), ptrString(rec.StripeTransferID),
		rec.ProposalAmount, rec.StripeFeeAmount, rec.PlatformFeeAmount,
		rec.ClientTotalAmount, rec.ProviderPayout,
		rec.Currency, string(rec.Status), string(rec.TransferStatus),
		rec.PaidAt, rec.TransferredAt, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert payment record: %w", err)
	}
	return nil
}

func (r *PaymentRecordRepository) GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanRecord(r.db.QueryRowContext(ctx, `
		SELECT id, proposal_id, client_id, provider_id,
			COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at
		FROM payment_records WHERE proposal_id = $1`, proposalID))
}

func (r *PaymentRecordRepository) GetByPaymentIntentID(ctx context.Context, piID string) (*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanRecord(r.db.QueryRowContext(ctx, `
		SELECT id, proposal_id, client_id, provider_id,
			COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at
		FROM payment_records WHERE stripe_payment_intent_id = $1`, piID))
}

func (r *PaymentRecordRepository) Update(ctx context.Context, rec *payment.PaymentRecord) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		UPDATE payment_records SET
			stripe_payment_intent_id = $1, stripe_transfer_id = $2,
			status = $3, transfer_status = $4,
			provider_payout = $5,
			paid_at = $6, transferred_at = $7, updated_at = $8
		WHERE id = $9`,
		ptrString(rec.StripePaymentIntentID), ptrString(rec.StripeTransferID),
		string(rec.Status), string(rec.TransferStatus),
		rec.ProviderPayout,
		rec.PaidAt, rec.TransferredAt, rec.UpdatedAt,
		rec.ID,
	)
	if err != nil {
		return fmt.Errorf("update payment record: %w", err)
	}
	return nil
}

func (r *PaymentRecordRepository) ListByProviderID(ctx context.Context, providerID uuid.UUID) ([]*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, proposal_id, client_id, provider_id,
			COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at
		FROM payment_records WHERE provider_id = $1
		ORDER BY created_at DESC`, providerID)
	if err != nil {
		return nil, fmt.Errorf("list payment records: %w", err)
	}
	defer rows.Close()

	var records []*payment.PaymentRecord
	for rows.Next() {
		var rec payment.PaymentRecord
		var status, transferStatus string
		if err := rows.Scan(
			&rec.ID, &rec.ProposalID, &rec.ClientID, &rec.ProviderID,
			&rec.StripePaymentIntentID, &rec.StripeTransferID,
			&rec.ProposalAmount, &rec.StripeFeeAmount, &rec.PlatformFeeAmount,
			&rec.ClientTotalAmount, &rec.ProviderPayout,
			&rec.Currency, &status, &transferStatus,
			&rec.PaidAt, &rec.TransferredAt, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment record: %w", err)
		}
		rec.Status = payment.PaymentRecordStatus(status)
		rec.TransferStatus = payment.TransferStatus(transferStatus)
		records = append(records, &rec)
	}
	return records, nil
}

func (r *PaymentRecordRepository) scanRecord(row *sql.Row) (*payment.PaymentRecord, error) {
	var rec payment.PaymentRecord
	var status, transferStatus string

	err := row.Scan(
		&rec.ID, &rec.ProposalID, &rec.ClientID, &rec.ProviderID,
		&rec.StripePaymentIntentID, &rec.StripeTransferID,
		&rec.ProposalAmount, &rec.StripeFeeAmount, &rec.PlatformFeeAmount,
		&rec.ClientTotalAmount, &rec.ProviderPayout,
		&rec.Currency, &status, &transferStatus,
		&rec.PaidAt, &rec.TransferredAt, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, payment.ErrPaymentRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan payment record: %w", err)
	}

	rec.Status = payment.PaymentRecordStatus(status)
	rec.TransferStatus = payment.TransferStatus(transferStatus)
	return &rec, nil
}

func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
