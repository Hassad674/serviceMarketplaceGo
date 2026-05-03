package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/referral"
)

func (r *ReferralRepository) CreateCommission(ctx context.Context, c *referral.Commission) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertCommission,
		c.ID, c.AttributionID, c.MilestoneID,
		c.GrossAmountCents, c.CommissionCents, c.Currency,
		string(c.Status), c.StripeTransferID, c.StripeReversalID, c.FailureReason,
		c.PaidAt, c.ClawedBackAt, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		// UNIQUE(attribution_id, milestone_id) — the distributor uses this
		// to detect that another retry already created the row and skip.
		if pqErr := (*pq.Error)(nil); errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return referral.ErrCommissionAlreadyExists
		}
		return fmt.Errorf("insert commission: %w", err)
	}
	return nil
}

func (r *ReferralRepository) UpdateCommission(ctx context.Context, c *referral.Commission) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	res, err := r.db.ExecContext(ctx, queryUpdateCommission,
		c.ID, string(c.Status), c.StripeTransferID, c.StripeReversalID, c.FailureReason, c.PaidAt, c.ClawedBackAt,
	)
	if err != nil {
		return fmt.Errorf("update commission: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update commission rows affected: %w", err)
	}
	if rows == 0 {
		return referral.ErrCommissionNotFound
	}
	return nil
}

func (r *ReferralRepository) FindCommissionByMilestone(ctx context.Context, milestoneID uuid.UUID) (*referral.Commission, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	c, err := scanCommission(r.db.QueryRowContext(ctx, queryFindCommissionByMilestone, milestoneID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, referral.ErrCommissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find commission by milestone: %w", err)
	}
	return c, nil
}

func (r *ReferralRepository) ListCommissionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Commission, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListCommissionsByReferral, referralID)
	if err != nil {
		return nil, fmt.Errorf("list commissions by referral: %w", err)
	}
	defer rows.Close()
	return scanCommissionRows(rows)
}

func (r *ReferralRepository) ListRecentCommissionsByReferrer(ctx context.Context, referrerID uuid.UUID, limit int) ([]*referral.Commission, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, queryListRecentCommissionsByReferrer, referrerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent commissions by referrer: %w", err)
	}
	defer rows.Close()
	return scanCommissionRows(rows)
}

func (r *ReferralRepository) ListPendingKYCByReferrer(ctx context.Context, referrerID uuid.UUID) ([]*referral.Commission, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListPendingKYCByReferrer, referrerID)
	if err != nil {
		return nil, fmt.Errorf("list pending_kyc commissions: %w", err)
	}
	defer rows.Close()
	return scanCommissionRows(rows)
}

// ─── Cron support ──────────────────────────────────────────────────────────

func (r *ReferralRepository) ListExpiringIntros(ctx context.Context, cutoff time.Time, limit int) ([]*referral.Referral, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, queryListExpiringIntros, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list expiring intros: %w", err)
	}
	defer rows.Close()
	return scanReferralRows(rows)
}

func (r *ReferralRepository) ListExpiringActives(ctx context.Context, now time.Time, limit int) ([]*referral.Referral, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, queryListExpiringActives, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list expiring actives: %w", err)
	}
	defer rows.Close()
	return scanReferralRows(rows)
}

// ─── Aggregations ──────────────────────────────────────────────────────────

func (r *ReferralRepository) CountByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.Status]int, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryCountByReferrer, referrerID)
	if err != nil {
		return nil, fmt.Errorf("count referrals by status: %w", err)
	}
	defer rows.Close()

	out := make(map[referral.Status]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan count row: %w", err)
		}
		out[referral.Status(status)] = count
	}
	return out, rows.Err()
}

func (r *ReferralRepository) SumCommissionsByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.CommissionStatus]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, querySumCommissionsByReferrer, referrerID)
	if err != nil {
		return nil, fmt.Errorf("sum commissions by status: %w", err)
	}
	defer rows.Close()

	out := make(map[referral.CommissionStatus]int64)
	for rows.Next() {
		var status string
		var sum int64
		if err := rows.Scan(&status, &sum); err != nil {
			return nil, fmt.Errorf("scan sum row: %w", err)
		}
		out[referral.CommissionStatus(status)] = sum
	}
	return out, rows.Err()
}

// ─── Scan helpers ──────────────────────────────────────────────────────────

func scanReferral(row *sql.Row) (*referral.Referral, error) {
	ref := &referral.Referral{}
	var snapshotRaw []byte
	var status string
	if err := row.Scan(
		&ref.ID, &ref.ReferrerID, &ref.ProviderID, &ref.ClientID,
		&ref.RatePct, &ref.DurationMonths,
		&snapshotRaw, &ref.IntroSnapshotVersion,
		&ref.IntroMessageProvider, &ref.IntroMessageClient,
		&status, &ref.Version,
		&ref.ActivatedAt, &ref.ExpiresAt, &ref.LastActionAt,
		&ref.RejectionReason, &ref.RejectedBy,
		&ref.CreatedAt, &ref.UpdatedAt,
	); err != nil {
		return nil, err
	}
	ref.Status = referral.Status(status)
	snapshot, err := referral.UnmarshalSnapshot(snapshotRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	ref.IntroSnapshot = snapshot
	return ref, nil
}

func scanReferralRows(rows *sql.Rows) ([]*referral.Referral, error) {
	var out []*referral.Referral
	for rows.Next() {
		ref := &referral.Referral{}
		var snapshotRaw []byte
		var status string
		if err := rows.Scan(
			&ref.ID, &ref.ReferrerID, &ref.ProviderID, &ref.ClientID,
			&ref.RatePct, &ref.DurationMonths,
			&snapshotRaw, &ref.IntroSnapshotVersion,
			&ref.IntroMessageProvider, &ref.IntroMessageClient,
			&status, &ref.Version,
			&ref.ActivatedAt, &ref.ExpiresAt, &ref.LastActionAt,
			&ref.RejectionReason, &ref.RejectedBy,
			&ref.CreatedAt, &ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan referral row: %w", err)
		}
		ref.Status = referral.Status(status)
		snapshot, err := referral.UnmarshalSnapshot(snapshotRaw)
		if err != nil {
			return nil, fmt.Errorf("unmarshal snapshot in row: %w", err)
		}
		ref.IntroSnapshot = snapshot
		out = append(out, ref)
	}
	return out, rows.Err()
}

func scanCommission(row *sql.Row) (*referral.Commission, error) {
	c := &referral.Commission{}
	var status string
	if err := row.Scan(
		&c.ID, &c.AttributionID, &c.MilestoneID,
		&c.GrossAmountCents, &c.CommissionCents, &c.Currency,
		&status, &c.StripeTransferID, &c.StripeReversalID, &c.FailureReason,
		&c.PaidAt, &c.ClawedBackAt, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, err
	}
	c.Status = referral.CommissionStatus(status)
	return c, nil
}

func scanCommissionRows(rows *sql.Rows) ([]*referral.Commission, error) {
	var out []*referral.Commission
	for rows.Next() {
		c := &referral.Commission{}
		var status string
		if err := rows.Scan(
			&c.ID, &c.AttributionID, &c.MilestoneID,
			&c.GrossAmountCents, &c.CommissionCents, &c.Currency,
			&status, &c.StripeTransferID, &c.StripeReversalID, &c.FailureReason,
			&c.PaidAt, &c.ClawedBackAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan commission row: %w", err)
		}
		c.Status = referral.CommissionStatus(status)
		out = append(out, c)
	}
	return out, rows.Err()
}

func statusesToStrings(in []referral.Status) pq.StringArray {
	out := make(pq.StringArray, 0, len(in))
	for _, s := range in {
		out = append(out, string(s))
	}
	return out
}
