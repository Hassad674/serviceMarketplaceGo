package repository

import (
	"context"

	"github.com/google/uuid"
)

// JobCreditRepository manages application credit balances for users.
type JobCreditRepository interface {
	GetOrCreate(ctx context.Context, userID uuid.UUID) (credits int, err error)
	Decrement(ctx context.Context, userID uuid.UUID) error
	AddBonus(ctx context.Context, userID uuid.UUID, amount int, maxTokens int) error
	ResetForUser(ctx context.Context, userID uuid.UUID, minCredits int) error
	ResetWeekly(ctx context.Context, minCredits int) error
}
