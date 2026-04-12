package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
//
// The repository is constructed with both the starter quota
// (WeeklyQuota — the floor the refill tops every org back up to) and
// the refill period (RefillPeriod — the time window between two
// automatic top-ups). These come from the job domain constants at
// wiring time so the SQL never hardcodes them and tests can inject
// shorter windows.
type JobCreditRepository struct {
	db           *sql.DB
	starterQuota int
	refillPeriod time.Duration
}

// NewJobCreditRepository wires the credit repository.
//
// starterQuota is the floor an org's pool is topped back up to when
// the refill period has elapsed; refillPeriod is the duration between
// two automatic top-ups. Both are plain values so the adapter stays
// free of any config knowledge and tests can inject tighter windows.
func NewJobCreditRepository(db *sql.DB, starterQuota int, refillPeriod time.Duration) *JobCreditRepository {
	return &JobCreditRepository{
		db:           db,
		starterQuota: starterQuota,
		refillPeriod: refillPeriod,
	}
}

// GetOrCreate returns the current credit balance for the org, running
// a lazy weekly refill as a side effect.
//
// The method first attempts an atomic conditional UPDATE that tops
// the pool back up to the starter quota and advances the
// `credits_last_reset_at` cursor, but only when the configured refill
// period has elapsed since the last top-up. The UPDATE uses GREATEST
// so bonus credits earned through the proposal fraud flow (up to
// MaxTokens) are preserved untouched — the refill is a floor, never
// destructive. When the UPDATE touches a row we return the refreshed
// balance directly; when the period has not elapsed yet the UPDATE
// affects zero rows and we fall through to a plain SELECT.
//
// Race-safety: the whole refill is a single SQL statement, so two
// concurrent reads on an org that is due can never double-refill.
// PostgreSQL row-level locking turns the second UPDATE into a no-op
// because the first one will have advanced `credits_last_reset_at`
// past the period threshold by the time the second one evaluates its
// WHERE clause.
//
// Self-healing: no cron is required. An org that has not been touched
// for a month gets its pool refreshed on the next read automatically.
// If the backend was down while a refill was due, the first read after
// startup makes the system consistent again.
func (r *JobCreditRepository) GetOrCreate(ctx context.Context, orgID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Attempt the conditional refill. The WHERE clause fires only when
	// the configured period has elapsed; older rows get both their
	// balance floor-bumped and their cursor advanced in one shot.
	//
	// NOTE: the interval is passed as the Go time.Duration's seconds
	// (double precision) and materialized server-side via make_interval
	// so PostgreSQL builds a proper INTERVAL value. Tests inject sub-
	// second periods through this same code path, so the race-safety
	// assertions run the exact production SQL.
	var refreshed int
	refillErr := r.db.QueryRowContext(ctx, `
		UPDATE organizations
		SET    application_credits   = GREATEST(application_credits, $2::int),
		       credits_last_reset_at = now(),
		       updated_at            = now()
		WHERE  id = $1
		  AND  now() - credits_last_reset_at >= make_interval(secs => $3::double precision)
		RETURNING application_credits`,
		orgID, r.starterQuota, r.refillPeriod.Seconds(),
	).Scan(&refreshed)
	if refillErr == nil {
		return refreshed, nil
	}
	if !errors.Is(refillErr, sql.ErrNoRows) {
		return 0, fmt.Errorf("refill org credits: %w", refillErr)
	}

	// Period not elapsed — return the current balance as-is.
	var credits int
	err := r.db.QueryRowContext(ctx,
		`SELECT application_credits FROM organizations WHERE id = $1`,
		orgID,
	).Scan(&credits)
	if errors.Is(err, sql.ErrNoRows) {
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
