package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
)

// JobCreditRepository persists the application credit pool on the
// organizations row. See repository.JobCreditRepository for the contract.
//
// R12 — The old implementation operated on the dedicated
// `application_credits` table keyed by user_id. That table was dropped
// by migration 075 and replaced by `organizations.application_credits`,
// so every operator of the same team now shares a single pool.
type JobCreditRepository struct {
	db *sql.DB
}

func NewJobCreditRepository(db *sql.DB) *JobCreditRepository {
	return &JobCreditRepository{db: db}
}

// GetOrCreate returns the current credit balance for the org.
//
// Every organization row carries the column with a `DEFAULT 0`, so
// the result is always well-defined. The method name is kept (and not
// "Get") to match the feature's historical shape and keep the port
// contract small.
func (r *JobCreditRepository) GetOrCreate(ctx context.Context, orgID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var credits int
	err := r.db.QueryRowContext(ctx,
		`SELECT application_credits FROM organizations WHERE id = $1`,
		orgID,
	).Scan(&credits)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("get org credits: organization %s not found", orgID)
	}
	if err != nil {
		return 0, fmt.Errorf("get org credits: %w", err)
	}
	return credits, nil
}

// Decrement atomically removes one credit from the org pool.
//
// Race-safety: the UPDATE is a single SQL statement with
// `WHERE application_credits > 0`, so two concurrent applies can never
// both debit the same zero-balance pool. The statement either touches
// one row (success) or zero rows (ErrNoCreditsLeft).
func (r *JobCreditRepository) Decrement(ctx context.Context, orgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx,
		`UPDATE organizations
		 SET    application_credits = application_credits - 1,
		        updated_at = now()
		 WHERE  id = $1 AND application_credits > 0`,
		orgID,
	)
	if err != nil {
		return fmt.Errorf("decrement org credits: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return job.ErrNoCreditsLeft
	}
	return nil
}

// Refund adds one credit back to the org pool. Used when a decrement
// succeeded but the downstream application insert failed, so the
// shared balance stays consistent with what the user actually spent.
func (r *JobCreditRepository) Refund(ctx context.Context, orgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations
		 SET    application_credits = application_credits + 1,
		        updated_at = now()
		 WHERE  id = $1`,
		orgID,
	)
	if err != nil {
		return fmt.Errorf("refund org credits: %w", err)
	}
	return nil
}

// AddBonus adds credits to the org pool, capped at maxTokens. Used by
// the proposal service when a mission is paid and fraud checks pass.
func (r *JobCreditRepository) AddBonus(ctx context.Context, orgID uuid.UUID, amount int, maxTokens int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations
		 SET    application_credits = LEAST(application_credits + $2::int, $3::int),
		        updated_at = now()
		 WHERE  id = $1`,
		orgID, amount, maxTokens,
	)
	if err != nil {
		return fmt.Errorf("add bonus credits: %w", err)
	}
	return nil
}

// ResetForOrg resets a single org's credits to minCredits if its
// current balance is below it. Used by the admin per-org reset button.
func (r *JobCreditRepository) ResetForOrg(ctx context.Context, orgID uuid.UUID, minCredits int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations
		 SET    application_credits = $2,
		        credits_last_reset_at = now(),
		        updated_at = now()
		 WHERE  id = $1 AND application_credits < $2`,
		orgID, minCredits,
	)
	if err != nil {
		return fmt.Errorf("reset credits for org: %w", err)
	}
	return nil
}

// ResetWeekly resets every org below minCredits back to minCredits.
// Run by the weekly quota cron (external — no in-process scheduler).
func (r *JobCreditRepository) ResetWeekly(ctx context.Context, minCredits int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations
		 SET    application_credits = $1,
		        credits_last_reset_at = now(),
		        updated_at = now()
		 WHERE  application_credits < $1`,
		minCredits,
	)
	if err != nil {
		return fmt.Errorf("reset weekly credits: %w", err)
	}
	return nil
}
