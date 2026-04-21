package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProviderMilestoneAmountsRepository implements
// repository.ProviderMilestoneAmountsReader via a tiny query over
// payment_records. Lives in its own file so the subscription feature's
// read path is visible at a glance — and removable with the feature if
// ever needed.
type ProviderMilestoneAmountsRepository struct {
	db *sql.DB
}

func NewProviderMilestoneAmountsRepository(db *sql.DB) *ProviderMilestoneAmountsRepository {
	return &ProviderMilestoneAmountsRepository{db: db}
}

// ListProviderMilestoneAmountsSince returns the proposal_amount of every
// payment_record created for providerID at or after `since`. Used by the
// subscription stats endpoint to compute cumulative fee savings.
//
// Includes records in every status — the fee would have been charged
// regardless of downstream transfer success, so counting them gives the
// user the truest picture of what Premium saved them.
func (r *ProviderMilestoneAmountsRepository) ListProviderMilestoneAmountsSince(
	ctx context.Context, providerID uuid.UUID, since time.Time,
) ([]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT proposal_amount
		FROM payment_records
		WHERE provider_id = $1 AND created_at >= $2`,
		providerID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("list provider milestone amounts: %w", err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var amount int64
		if sErr := rows.Scan(&amount); sErr != nil {
			return nil, fmt.Errorf("scan amount: %w", sErr)
		}
		out = append(out, amount)
	}
	if rErr := rows.Err(); rErr != nil {
		return nil, fmt.Errorf("iterate amounts: %w", rErr)
	}
	return out, nil
}
