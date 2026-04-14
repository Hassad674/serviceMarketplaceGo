package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referrerpricing"
)

// ReferrerPricingRepository persists referrer_pricing rows. Mirrors
// FreelancePricingRepository shape — one row per referrer profile,
// idempotent upserts and deletes.
type ReferrerPricingRepository interface {
	// Upsert writes or updates the pricing row identified by the
	// profile_id.
	Upsert(ctx context.Context, p *referrerpricing.Pricing) error

	// FindByProfileID returns the pricing row for the given referrer
	// profile. Returns referrerpricing.ErrPricingNotFound when no
	// row exists.
	FindByProfileID(ctx context.Context, profileID uuid.UUID) (*referrerpricing.Pricing, error)

	// DeleteByProfileID removes the pricing row. Idempotent.
	DeleteByProfileID(ctx context.Context, profileID uuid.UUID) error
}
