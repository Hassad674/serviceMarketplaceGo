package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profilepricing"
)

// ProfilePricingRepository persists profile_pricing rows (migration
// 083). The cardinality is capped at 2 rows per organization (one
// per PricingKind: direct + referral) by the composite primary key
// on (organization_id, pricing_kind). Implementations must not leak
// SQL or driver-specific types across this boundary.
//
// The interface is deliberately small: pricing edits are granular
// (one kind at a time), so there is no multi-row atomic replace
// method — Upsert covers both "first write" and "subsequent edit"
// in a single call.
type ProfilePricingRepository interface {
	// Upsert writes or updates the pricing row identified by
	// (OrganizationID, Kind). Primary-key collision updates in place
	// and bumps updated_at via the table trigger. created_at is set
	// on first insert and preserved on subsequent updates.
	Upsert(ctx context.Context, p *profilepricing.Pricing) error

	// FindByOrgID returns every pricing row for the org (0, 1 or 2
	// rows). Callers receive an empty (non-nil) slice when the org
	// has no pricing declared, so they can marshal it directly to an
	// empty JSON array without a nil check. Per-org ordering is
	// stable — direct first, then referral — so UI consumers can
	// render the two sections in a predictable order.
	FindByOrgID(ctx context.Context, orgID uuid.UUID) ([]*profilepricing.Pricing, error)

	// ListByOrgIDs is the batch variant used by listing endpoints
	// (discovery / search) that need to decorate many profile cards
	// with pricing in a single database roundtrip — N+1 prevention
	// is mandatory. The returned map is keyed by organization ID and
	// contains a (non-nil, possibly empty) slice for every ID passed
	// in, so callers can range over the input directly without nil
	// checks.
	ListByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*profilepricing.Pricing, error)

	// DeleteByKind removes the (OrganizationID, Kind) row. No error
	// is returned when the row does not exist — deletion is
	// idempotent so the UI can safely surface a "delete" button
	// without racing against a concurrent edit.
	DeleteByKind(ctx context.Context, orgID uuid.UUID, kind profilepricing.PricingKind) error
}
