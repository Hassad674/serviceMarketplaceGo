package service

import (
	"context"

	"marketplace-backend/internal/domain/payment"
)

// CountrySpecService provides country-specific Stripe field requirements.
type CountrySpecService interface {
	// GetFieldsForCountry returns the field requirements for a specific country and business type.
	GetFieldsForCountry(ctx context.Context, country string) (*payment.CountryFieldSpec, error)

	// WarmCache pre-loads all country specs into the cache.
	WarmCache(ctx context.Context) error
}
