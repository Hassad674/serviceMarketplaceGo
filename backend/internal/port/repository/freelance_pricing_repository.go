package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/freelancepricing"
)

// FreelancePricingRepository persists freelance_pricing rows. At
// most one row per freelance profile (PK on profile_id), so the
// interface is intentionally thin — no batch-upsert or multi-row
// replace is needed.
type FreelancePricingRepository interface {
	// Upsert writes or updates the pricing row identified by the
	// profile_id. On conflict the existing row is updated in place
	// (trigger-bumped updated_at, preserved created_at).
	Upsert(ctx context.Context, p *freelancepricing.Pricing) error

	// FindByProfileID returns the pricing row for the given
	// freelance profile. Returns freelancepricing.ErrPricingNotFound
	// when no row exists — callers decide whether to surface the
	// error or render an empty pricing section.
	FindByProfileID(ctx context.Context, profileID uuid.UUID) (*freelancepricing.Pricing, error)

	// DeleteByProfileID removes the pricing row. Idempotent: no
	// error is returned when the row does not exist so the UI can
	// safely surface a "delete" button without racing.
	DeleteByProfileID(ctx context.Context, profileID uuid.UUID) error
}
