package repository

import (
	"context"

	"github.com/google/uuid"
)

// JobCreditRepository manages application credit balances.
//
// R12 — After the team refactor, credits live on the organization row,
// not on individual users. Every operator of the same org debits the
// same pool; refills, top-ups and fraud-bonus credits all credit the
// org. This prevents the "invite 100 operators to multiply your credit
// cap by 100" exploit that existed when credits were per-user.
//
// All methods take an org id. Resolving the org from a user id is the
// caller's responsibility (usually via OrganizationRepository.FindByUserID
// or — for the hot path — via users.organization_id carried in the user
// entity).
type JobCreditRepository interface {
	// GetOrCreate returns the org's current credit balance, creating the
	// row lazily if it does not exist yet. In practice every org gets a
	// row at creation time via migration 075 + the ALTER default, so
	// the "create" branch is only a safety net.
	GetOrCreate(ctx context.Context, orgID uuid.UUID) (credits int, err error)

	// Decrement atomically subtracts one credit from the org's pool.
	// Returns job.ErrNoCreditsLeft when the pool is already zero — this
	// is the authoritative gate against concurrent applies, because the
	// UPDATE is a single statement (`WHERE application_credits > 0`) so
	// two operators racing to apply cannot both get past the check.
	Decrement(ctx context.Context, orgID uuid.UUID) error

	// Refund adds one credit back to the pool. Used by ApplyToJob when
	// the decrement succeeded but the subsequent application INSERT
	// failed, so the caller's balance stays consistent.
	Refund(ctx context.Context, orgID uuid.UUID) error

	// AddBonus adds `amount` credits capped at maxTokens. Used by the
	// proposal service when a mission is paid (bonus credits awarded to
	// the provider's org).
	AddBonus(ctx context.Context, orgID uuid.UUID, amount int, maxTokens int) error

	// ResetForOrg resets a single org's pool to `minCredits` if the
	// current balance is below it. Used by the admin "reset for this
	// team" action.
	ResetForOrg(ctx context.Context, orgID uuid.UUID, minCredits int) error

	// ResetWeekly resets every org whose pool is below minCredits back
	// to minCredits. Intended for the weekly quota cron job.
	ResetWeekly(ctx context.Context, minCredits int) error
}
