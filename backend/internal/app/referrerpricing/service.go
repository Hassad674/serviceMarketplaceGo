// Package referrerpricing is the application service layer for the
// pricing row attached to a referrer profile. Mirrors the freelance
// pricing service shape.
package referrerpricing

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referrerpricing"
	"marketplace-backend/internal/port/repository"
)

// Service orchestrates the referrer pricing use cases.
type Service struct {
	pricing repository.ReferrerPricingRepository
}

// NewService wires the service with its single dependency.
func NewService(pricing repository.ReferrerPricingRepository) *Service {
	return &Service{pricing: pricing}
}

// UpsertInput is the payload for Upsert.
type UpsertInput struct {
	ProfileID  uuid.UUID
	Type       referrerpricing.PricingType
	MinAmount  int64
	MaxAmount  *int64
	Currency   string
	Note       string
	Negotiable bool
}

// Upsert validates via the domain constructor then persists.
func (s *Service) Upsert(ctx context.Context, input UpsertInput) (*referrerpricing.Pricing, error) {
	p, err := referrerpricing.NewPricing(referrerpricing.NewPricingInput{
		ProfileID:  input.ProfileID,
		Type:       input.Type,
		MinAmount:  input.MinAmount,
		MaxAmount:  input.MaxAmount,
		Currency:   input.Currency,
		Note:       input.Note,
		Negotiable: input.Negotiable,
	})
	if err != nil {
		return nil, fmt.Errorf("referrer pricing upsert: validate: %w", err)
	}
	if err := s.pricing.Upsert(ctx, p); err != nil {
		return nil, fmt.Errorf("referrer pricing upsert: persist: %w", err)
	}
	return p, nil
}

// Get returns the pricing row for the referrer profile, or
// referrerpricing.ErrPricingNotFound when none is declared.
func (s *Service) Get(ctx context.Context, profileID uuid.UUID) (*referrerpricing.Pricing, error) {
	p, err := s.pricing.FindByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("referrer pricing get: %w", err)
	}
	return p, nil
}

// Delete removes the pricing row. Idempotent.
func (s *Service) Delete(ctx context.Context, profileID uuid.UUID) error {
	if err := s.pricing.DeleteByProfileID(ctx, profileID); err != nil {
		return fmt.Errorf("referrer pricing delete: %w", err)
	}
	return nil
}
