package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
)

// BillingProfileRepository persists the recipient identity used on
// every invoice. One row per organization. Returns ErrNotFound when
// the org has not yet seeded its billing profile.
type BillingProfileRepository interface {
	// FindByOrganization fetches the profile or returns
	// invoicing.ErrNotFound when missing.
	FindByOrganization(ctx context.Context, organizationID uuid.UUID) (*invoicing.BillingProfile, error)

	// Upsert writes the profile, updating an existing row or
	// inserting a new one keyed on organization_id.
	Upsert(ctx context.Context, p *invoicing.BillingProfile) error
}
