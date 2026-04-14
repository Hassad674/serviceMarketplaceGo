// Package freelancepricing is the application service layer for
// the pricing row attached to a freelance profile. It is a thin
// orchestrator over FreelancePricingRepository: validation lives
// in the domain (freelancepricing.NewPricing), persistence lives
// in the adapter, this layer wires the two.
//
// Unlike the legacy profilepricing service, this service does NOT
// need an OrgInfoResolver because the target role is implicit in
// the table: you cannot write a freelance_pricing row unless a
// freelance_profiles row already exists for the org, and that only
// happens for provider_personal orgs. The role gating is enforced
// by the FK chain rather than by a separate Go check.
package freelancepricing

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/freelancepricing"
	"marketplace-backend/internal/port/repository"
)

// Service orchestrates the freelance pricing use cases.
type Service struct {
	pricing repository.FreelancePricingRepository
}

// NewService wires the service with its single dependency.
func NewService(pricing repository.FreelancePricingRepository) *Service {
	return &Service{pricing: pricing}
}

// UpsertInput is the payload for Upsert. Grouping the raw inputs
// in a struct keeps the method signature under the 4-parameter cap.
type UpsertInput struct {
	ProfileID  uuid.UUID
	Type       freelancepricing.PricingType
	MinAmount  int64
	MaxAmount  *int64
	Currency   string
	Note       string
	Negotiable bool
}

// Upsert validates via the domain constructor then persists the
// row. Returns the persisted value — useful for the handler to
// echo back the canonical result including defaults.
func (s *Service) Upsert(ctx context.Context, input UpsertInput) (*freelancepricing.Pricing, error) {
	p, err := freelancepricing.NewPricing(freelancepricing.NewPricingInput{
		ProfileID:  input.ProfileID,
		Type:       input.Type,
		MinAmount:  input.MinAmount,
		MaxAmount:  input.MaxAmount,
		Currency:   input.Currency,
		Note:       input.Note,
		Negotiable: input.Negotiable,
	})
	if err != nil {
		return nil, fmt.Errorf("freelance pricing upsert: validate: %w", err)
	}
	if err := s.pricing.Upsert(ctx, p); err != nil {
		return nil, fmt.Errorf("freelance pricing upsert: persist: %w", err)
	}
	return p, nil
}

// Get returns the pricing row for the freelance profile, or
// freelancepricing.ErrPricingNotFound when none is declared.
// Callers decide whether to render an empty state or surface the
// error.
func (s *Service) Get(ctx context.Context, profileID uuid.UUID) (*freelancepricing.Pricing, error) {
	p, err := s.pricing.FindByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("freelance pricing get: %w", err)
	}
	return p, nil
}

// Delete removes the pricing row. Idempotent: deleting a
// non-existent row is a success.
func (s *Service) Delete(ctx context.Context, profileID uuid.UUID) error {
	if err := s.pricing.DeleteByProfileID(ctx, profileID); err != nil {
		return fmt.Errorf("freelance pricing delete: %w", err)
	}
	return nil
}
