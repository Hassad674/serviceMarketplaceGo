package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProviderMilestoneAmountsReader is the minimal lookup the subscription
// feature needs to compute "fees saved since your subscription started".
// Defining it here keeps subscription independent of the payment feature
// package while letting the postgres adapter satisfy it via a tiny query
// over payment_records.
//
// Returning raw amounts (in cents) instead of pre-computed fees keeps the
// adapter dumb — the subscription service applies the billing schedule
// locally so the business rule stays in one place.
type ProviderMilestoneAmountsReader interface {
	// ListProviderMilestoneAmountsSince returns the proposal_amount (in
	// cents) of every payment_record created for providerID at or after
	// `since`. Order does not matter; callers only sum/iterate. An empty
	// slice (not an error) is returned when no records match.
	ListProviderMilestoneAmountsSince(ctx context.Context, providerID uuid.UUID, since time.Time) ([]int64, error)
}
