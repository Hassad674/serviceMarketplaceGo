// Package profilepricing is the application service layer for the
// profile pricing feature. It orchestrates the pricing repository
// port and a local OrgInfoResolver to enforce role-based rules
// (agency direct-only, provider_personal direct + optional
// referral, enterprise forbidden) on every write.
//
// Like the skill app service, profilepricing defines its
// collaboration contract locally (OrgInfoResolver) so the feature
// never imports the organization or user packages — preserving the
// hexagonal feature-independence invariant. The wiring layer
// (cmd/api/main.go) supplies a thin shim that bridges this
// interface to the real organization + user repositories.
package profilepricing

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domainpricing "marketplace-backend/internal/domain/profilepricing"
	"marketplace-backend/internal/port/repository"
)

// OrgInfoResolver is a thin read-only dependency the pricing
// service needs to look up an organization's type AND its owner's
// referrer_enabled flag — both inputs are required to answer the
// "may this org declare this pricing kind?" question in
// IsKindAllowedForOrg.
//
// Defined locally in this package rather than in port/repository
// because it is a one-method collaboration scoped to this feature.
// Keeping it here preserves the invariant that profilepricing does
// not import the organization or user packages even at the
// interface level.
type OrgInfoResolver interface {
	GetOrgInfo(ctx context.Context, orgID uuid.UUID) (orgType string, referrerEnabled bool, err error)
}

// Service orchestrates the pricing use cases: upsert with role
// validation, per-org read, batch read, and delete by kind.
type Service struct {
	pricing repository.ProfilePricingRepository
	orgs    OrgInfoResolver
}

// NewService wires the pricing service with its dependencies. Both
// parameters are required — the service has no optional
// collaborators and no sane default for either of them.
func NewService(
	pricing repository.ProfilePricingRepository,
	orgs OrgInfoResolver,
) *Service {
	return &Service{pricing: pricing, orgs: orgs}
}

// UpsertInput is the payload for Upsert. Grouping the seven raw
// inputs in a struct keeps the Upsert signature under the 4-param
// cap and gives the handler layer a stable point of entry.
type UpsertInput struct {
	OrganizationID uuid.UUID
	Kind           domainpricing.PricingKind
	Type           domainpricing.PricingType
	MinAmount      int64
	MaxAmount      *int64
	Currency       string
	Note           string
}

// Upsert writes or updates a pricing row after a three-stage
// validation:
//
//  1. Kind / type / amount / currency invariants via NewPricing
//     (domain-level).
//  2. Kind must be legal for the org's role and referrer state
//     (service-level, requires OrgInfoResolver lookup).
//  3. Kind + type must be compatible (already covered by step 1
//     via IsTypeAllowedForKind inside NewPricing).
//
// Step 2 runs BEFORE step 1's NewPricing so an enterprise org
// declaring pricing fails fast with ErrKindNotAllowedForRole
// rather than first being told its min_amount shape is wrong.
//
// On success the persisted row is returned — useful for the
// handler to echo back the canonical result including any DB
// defaults (created_at/updated_at).
func (s *Service) Upsert(ctx context.Context, input UpsertInput) (*domainpricing.Pricing, error) {
	orgType, referrerEnabled, err := s.orgs.GetOrgInfo(ctx, input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("profile pricing upsert: resolve org: %w", err)
	}
	if !domainpricing.IsKindAllowedForOrg(orgType, referrerEnabled, input.Kind) {
		return nil, domainpricing.ErrKindNotAllowedForRole
	}

	p, err := domainpricing.NewPricing(
		input.OrganizationID,
		input.Kind,
		input.Type,
		input.MinAmount,
		input.MaxAmount,
		input.Currency,
		input.Note,
	)
	if err != nil {
		return nil, fmt.Errorf("profile pricing upsert: validate: %w", err)
	}
	if err := s.pricing.Upsert(ctx, p); err != nil {
		return nil, fmt.Errorf("profile pricing upsert: persist: %w", err)
	}
	return p, nil
}

// GetForOrg returns every pricing row for the org (0, 1 or 2
// rows). The result is a guaranteed non-nil slice so callers can
// marshal it directly to `[]` in JSON without a nil check.
func (s *Service) GetForOrg(ctx context.Context, orgID uuid.UUID) ([]*domainpricing.Pricing, error) {
	out, err := s.pricing.FindByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("profile pricing get: %w", err)
	}
	return out, nil
}

// GetForOrgsBatch is the listing-endpoint helper. Every input org
// ID is present in the returned map (possibly with an empty
// slice) so callers can range over the input directly without
// nil-checks.
func (s *Service) GetForOrgsBatch(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*domainpricing.Pricing, error) {
	out, err := s.pricing.ListByOrgIDs(ctx, orgIDs)
	if err != nil {
		return nil, fmt.Errorf("profile pricing batch get: %w", err)
	}
	return out, nil
}

// DeleteKind removes a single (org, kind) row. Validates the
// kind enum at the domain level before touching the repository
// so malformed inputs never reach the DB.
func (s *Service) DeleteKind(ctx context.Context, orgID uuid.UUID, kind domainpricing.PricingKind) error {
	if !kind.IsValid() {
		return domainpricing.ErrInvalidKind
	}
	if err := s.pricing.DeleteByKind(ctx, orgID, kind); err != nil {
		return fmt.Errorf("profile pricing delete: %w", err)
	}
	return nil
}
