package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

// PaymentRecordRepository is the postgres implementation of the
// payment record port.
//
// BUG-NEW-04 path 7/8: payment_records is RLS-protected by migration
// 125 with the policy
//
//   USING (organization_id = current_setting('app.current_org_id', true)::uuid)
//
// Single-side ownership: the client org owns the record (the org that
// paid). Provider-side reads of money received go through the proposal
// path, which is already tenant-isolated via path 4/8.
type PaymentRecordRepository struct {
	db       *sql.DB
	txRunner *TxRunner
}

func NewPaymentRecordRepository(db *sql.DB) *PaymentRecordRepository {
	return &PaymentRecordRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper.
func (r *PaymentRecordRepository) WithTxRunner(runner *TxRunner) *PaymentRecordRepository {
	r.txRunner = runner
	return r
}

// resolveClientOrg is the defensive lookup that maps a client_id to
// its owning organization. payment_records.organization_id is auto-
// resolved at INSERT time from organization_members, so we mirror
// that resolution here to install the tenant context BEFORE the
// INSERT. Without the matching context, RLS rejects the insert.
func (r *PaymentRecordRepository) resolveClientOrg(ctx context.Context, clientID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.NullUUID
	err := r.db.QueryRowContext(ctx,
		`SELECT organization_id FROM organization_members WHERE user_id = $1 LIMIT 1`,
		clientID,
	).Scan(&orgID)
	if errors.Is(err, sql.ErrNoRows) {
		// Solo provider client — no org membership row. organization_id
		// will be NULL on the inserted row; the policy filters it out
		// from any future SELECT (which is the intended behaviour for
		// solo-provider edge cases — they go through the legacy path).
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve client org: %w", err)
	}
	if orgID.Valid {
		return orgID.UUID, nil
	}
	return uuid.Nil, nil
}

func (r *PaymentRecordRepository) Create(ctx context.Context, rec *payment.PaymentRecord) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// organization_id is resolved from organization_members keyed on the
	// client — Agencies/Enterprises get their org denormalized onto the
	// record, Providers stay NULL. Used by the dashboard wallet view for
	// operators in later phases.
	//
	// provider_organization_id (PERF-B-08, migration 131) mirrors the
	// pattern established for proposals: the provider's users.org_id is
	// captured at INSERT time so the wallet list query no longer needs
	// to JOIN users on provider_id.
	//
	// milestone_id is phase-4: the payment record is scoped to a single
	// milestone, enforced NOT NULL by migration 093. Passing a zero UUID
	// will correctly fail at insert time — rejecting callers that forgot
	// to plumb the milestone through.
	doInsert := func(runner sqlExecutor) error {
		_, err := runner.ExecContext(ctx, `
			INSERT INTO payment_records (
				id, proposal_id, milestone_id, client_id, provider_id,
				stripe_payment_intent_id, stripe_transfer_id,
				proposal_amount, stripe_fee_amount, platform_fee_amount,
				client_total_amount, provider_payout,
				currency, status, transfer_status,
				paid_at, transferred_at, created_at, updated_at,
				organization_id, provider_organization_id
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
				(SELECT organization_id FROM organization_members WHERE user_id = $4 LIMIT 1),
				(SELECT organization_id FROM users WHERE id = $5)
			)`,
			rec.ID, rec.ProposalID, rec.MilestoneID, rec.ClientID, rec.ProviderID,
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

	if r.txRunner != nil {
		// Resolve the client's org so app.current_org_id matches what the
		// auto-resolution sub-select will populate. Without this, RLS
		// rejects the row because the WITH CHECK on the policy compares
		// against the unset/nil current setting.
		orgID, err := r.resolveClientOrg(ctx, rec.ClientID)
		if err != nil {
			return err
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doInsert(tx)
		})
	}

	return doInsert(r.db)
}

// GetByID returns a single record by its primary key.
//
// SYSTEM-ACTOR: same contract as ProposalRepository.GetByID.
// User-facing callers MUST use GetByIDForOrg; the legacy
// signature is preserved only for retry-transfer paths that run
// from the scheduler.
func (r *PaymentRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*payment.PaymentRecord, error) {
	warnIfNotSystemActor(ctx, "PaymentRecordRepository.GetByID")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanRecord(r.db.QueryRowContext(ctx, `
		SELECT id, proposal_id, milestone_id, client_id, provider_id,
			COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at
		FROM payment_records
		WHERE id = $1`, id))
}

// GetByProposalID returns the MOST RECENT payment record for a proposal.
// Phase 4: a proposal can now own multiple records (one per milestone),
// so this lookup is kept for legacy callers but returns the newest row
// by created_at. For strict milestone-level idempotency, use
// GetByMilestoneID instead.
func (r *PaymentRecordRepository) GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanRecord(r.db.QueryRowContext(ctx, `
		SELECT id, proposal_id, milestone_id, client_id, provider_id,
			COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at
		FROM payment_records
		WHERE proposal_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, proposalID))
}

// ListByProposalID returns every payment record for a proposal, ordered
// by created_at ascending (oldest first). Used by the macro-completion
// transfer path which must release ALL pending milestones — picking the
// most recent row (the old GetByProposalID behaviour) missed jalons 1..N-1
// on multi-milestone proposals and left them stuck in escrow.
func (r *PaymentRecordRepository) ListByProposalID(ctx context.Context, proposalID uuid.UUID) ([]*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, proposal_id, milestone_id, client_id, provider_id,
			COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at
		FROM payment_records
		WHERE proposal_id = $1
		ORDER BY created_at ASC`, proposalID)
	if err != nil {
		return nil, fmt.Errorf("list payment records by proposal: %w", err)
	}
	defer rows.Close()

	var records []*payment.PaymentRecord
	for rows.Next() {
		var rec payment.PaymentRecord
		var status, transferStatus string
		if err := rows.Scan(
			&rec.ID, &rec.ProposalID, &rec.MilestoneID, &rec.ClientID, &rec.ProviderID,
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

// GetByMilestoneID returns the single payment record for a milestone.
// Used by CreatePaymentIntent as the idempotency key so a retry on the
// same milestone reuses the existing Stripe PaymentIntent instead of
// creating a duplicate one.
func (r *PaymentRecordRepository) GetByMilestoneID(ctx context.Context, milestoneID uuid.UUID) (*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanRecord(r.db.QueryRowContext(ctx, `
		SELECT id, proposal_id, milestone_id, client_id, provider_id,
			COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			paid_at, transferred_at, created_at, updated_at
		FROM payment_records WHERE milestone_id = $1`, milestoneID))
}

func (r *PaymentRecordRepository) GetByPaymentIntentID(ctx context.Context, piID string) (*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanRecord(r.db.QueryRowContext(ctx, `
		SELECT id, proposal_id, milestone_id, client_id, provider_id,
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

	doUpdate := func(runner sqlExecutor) error {
		_, err := runner.ExecContext(ctx, `
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

	if r.txRunner != nil {
		// Resolve the row's owning org via the legacy db connection
		// (defensive — callers always have the id from a prior read).
		// Then open a tenant tx and run the UPDATE.
		var orgID uuid.NullUUID
		err := r.db.QueryRowContext(ctx,
			`SELECT organization_id FROM payment_records WHERE id = $1`, rec.ID,
		).Scan(&orgID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("update payment record: not found")
		}
		if err != nil {
			return fmt.Errorf("update payment record: lookup org: %w", err)
		}
		ctxOrg := uuid.Nil
		if orgID.Valid {
			ctxOrg = orgID.UUID
		}
		return r.txRunner.RunInTxWithTenant(ctx, ctxOrg, uuid.Nil, func(tx *sql.Tx) error {
			return doUpdate(tx)
		})
	}

	return doUpdate(r.db)
}

// GetByIDForOrg returns the payment record by id under the caller's
// org tenant context. RLS admits the row only when the caller's org
// matches the record's organization_id.
func (r *PaymentRecordRepository) GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rec *payment.PaymentRecord
	doRead := func(runner sqlQuerier) error {
		row := runner.QueryRowContext(ctx, `
			SELECT id, proposal_id, milestone_id, client_id, provider_id,
				COALESCE(stripe_payment_intent_id, ''), COALESCE(stripe_transfer_id, ''),
				proposal_amount, stripe_fee_amount, platform_fee_amount,
				client_total_amount, provider_payout,
				currency, status, transfer_status,
				paid_at, transferred_at, created_at, updated_at
			FROM payment_records
			WHERE id = $1`, id)
		got, err := r.scanRecord(row)
		if err != nil {
			return err
		}
		rec = got
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, callerOrgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		})
		if err != nil {
			return nil, err
		}
		return rec, nil
	}

	if err := doRead(r.db); err != nil {
		return nil, err
	}
	return rec, nil
}

// ListByOrganization returns payment records where the caller's
// organization is either the client or the provider.
//
// PERF-B-08: previously joined users on provider_id which produced a
// BitmapOr + nested-loop plan and added 50–150 ms p50 once
// payment_records crossed ~10k rows for a single org. Migration 131
// adds provider_organization_id to payment_records and the matching
// composite partial index idx_payment_records_provider_org_created.
// The query is now an Index Scan on either side of the OR.
//
// BUG-NEW-04 path 7/8: wraps the SELECT in RunInTxWithTenant with the
// caller's org. The RLS policy keys on organization_id (single-side),
// so the OR predicate against provider_organization_id won't pass the
// policy under non-superuser unless the caller's org matches the
// row's organization_id (client side). Provider-side reads must go
// through the proposal path instead — already RLS-isolated by path 4/8.
func (r *PaymentRecordRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*payment.PaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var records []*payment.PaymentRecord
	doQuery := func(runner sqlQuerier) error {
		rows, err := runner.QueryContext(ctx, `
			SELECT pr.id, pr.proposal_id, pr.milestone_id, pr.client_id, pr.provider_id,
				COALESCE(pr.stripe_payment_intent_id, ''), COALESCE(pr.stripe_transfer_id, ''),
				pr.proposal_amount, pr.stripe_fee_amount, pr.platform_fee_amount,
				pr.client_total_amount, pr.provider_payout,
				pr.currency, pr.status, pr.transfer_status,
				pr.paid_at, pr.transferred_at, pr.created_at, pr.updated_at
			FROM payment_records pr
			WHERE pr.organization_id = $1 OR pr.provider_organization_id = $1
			ORDER BY pr.created_at DESC`, orgID)
		if err != nil {
			return fmt.Errorf("list payment records: %w", err)
		}
		defer rows.Close()

		records = nil
		for rows.Next() {
			var rec payment.PaymentRecord
			var status, transferStatus string
			if err := rows.Scan(
				&rec.ID, &rec.ProposalID, &rec.MilestoneID, &rec.ClientID, &rec.ProviderID,
				&rec.StripePaymentIntentID, &rec.StripeTransferID,
				&rec.ProposalAmount, &rec.StripeFeeAmount, &rec.PlatformFeeAmount,
				&rec.ClientTotalAmount, &rec.ProviderPayout,
				&rec.Currency, &status, &transferStatus,
				&rec.PaidAt, &rec.TransferredAt, &rec.CreatedAt, &rec.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan payment record: %w", err)
			}
			rec.Status = payment.PaymentRecordStatus(status)
			rec.TransferStatus = payment.TransferStatus(transferStatus)
			records = append(records, &rec)
		}
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doQuery(tx)
		})
		if err != nil {
			return nil, err
		}
		return records, nil
	}

	if err := doQuery(r.db); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *PaymentRecordRepository) scanRecord(row *sql.Row) (*payment.PaymentRecord, error) {
	var rec payment.PaymentRecord
	var status, transferStatus string

	err := row.Scan(
		&rec.ID, &rec.ProposalID, &rec.MilestoneID, &rec.ClientID, &rec.ProviderID,
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
