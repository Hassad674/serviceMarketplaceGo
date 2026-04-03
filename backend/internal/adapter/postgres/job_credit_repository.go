package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
)

// JobCreditRepository implements repository.JobCreditRepository.
type JobCreditRepository struct {
	db *sql.DB
}

func NewJobCreditRepository(db *sql.DB) *JobCreditRepository {
	return &JobCreditRepository{db: db}
}

// GetOrCreate returns the current credit balance, creating the row if needed.
func (r *JobCreditRepository) GetOrCreate(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO application_credits (user_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("ensure credit row: %w", err)
	}

	var credits int
	err = r.db.QueryRowContext(ctx,
		`SELECT credits FROM application_credits WHERE user_id = $1`,
		userID,
	).Scan(&credits)
	if err != nil {
		return 0, fmt.Errorf("get credits: %w", err)
	}
	return credits, nil
}

// Decrement subtracts one credit. Returns ErrNoCreditsLeft if already zero.
func (r *JobCreditRepository) Decrement(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx,
		`UPDATE application_credits
		 SET credits = credits - 1, updated_at = now()
		 WHERE user_id = $1 AND credits > 0`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("decrement credits: %w", err)
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

// AddBonus adds credits capped at maxTokens. Used when a mission is signed.
func (r *JobCreditRepository) AddBonus(ctx context.Context, userID uuid.UUID, amount int, maxTokens int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO application_credits (user_id, credits)
		 VALUES ($1, LEAST($2, $3))
		 ON CONFLICT (user_id) DO UPDATE
		 SET credits = LEAST(application_credits.credits + $2, $3),
		     updated_at = now()`,
		userID, amount, maxTokens,
	)
	if err != nil {
		return fmt.Errorf("add bonus credits: %w", err)
	}
	return nil
}

// ResetForUser resets a single user's credits if below minCredits. Used by admin for testing.
func (r *JobCreditRepository) ResetForUser(ctx context.Context, userID uuid.UUID, minCredits int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE application_credits
		 SET credits = $2, last_reset_at = now(), updated_at = now()
		 WHERE user_id = $1 AND credits < $2`,
		userID, minCredits,
	)
	if err != nil {
		return fmt.Errorf("reset credits for user: %w", err)
	}
	return nil
}

// ResetWeekly resets all users below minCredits to minCredits. Run by cron.
func (r *JobCreditRepository) ResetWeekly(ctx context.Context, minCredits int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE application_credits
		 SET credits = $1, last_reset_at = now(), updated_at = now()
		 WHERE credits < $1`,
		minCredits,
	)
	if err != nil {
		return fmt.Errorf("reset weekly credits: %w", err)
	}
	return nil
}
