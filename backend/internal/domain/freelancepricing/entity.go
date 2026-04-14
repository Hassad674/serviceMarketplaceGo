// Package freelancepricing owns the domain model for the pricing row
// attached to a freelance profile (migration 099). It is the persona-
// specific sibling of referrerpricing: where referrer pricing covers
// commission_pct / commission_flat, freelance pricing covers the four
// "direct" sale types — daily, hourly, project_from, project_range.
//
// Design note — zero cross-feature imports. This package depends on
// nothing except Go stdlib + the google/uuid library (same rule as
// every domain package). Even the profile pricing legacy domain is
// NOT imported despite having overlapping enums — the split refactor
// takes the opportunity to give each persona its own validated type
// set so future divergence (extra types, different constraints) does
// not require a touch on the other persona.
package freelancepricing

import (
	"time"

	"github.com/google/uuid"
)

// PricingType is the frozen enum of valid pricing shapes for the
// freelance persona. Each value encodes the unit (daily, hourly,
// project) and, where applicable, whether the figure is a scalar
// point or a range.
type PricingType string

const (
	// TypeDaily — a single TJM. min_amount is cents, max_amount nil.
	TypeDaily PricingType = "daily"
	// TypeHourly — single hourly rate. min_amount cents, max nil.
	TypeHourly PricingType = "hourly"
	// TypeProjectFrom — "from X" starting price. min_amount cents, max nil.
	TypeProjectFrom PricingType = "project_from"
	// TypeProjectRange — "X to Y" project bracket. Both min and max set.
	TypeProjectRange PricingType = "project_range"
)

// IsValid reports whether t is one of the four frozen values. The
// zero value ("") returns false.
func (t PricingType) IsValid() bool {
	switch t {
	case TypeDaily, TypeHourly, TypeProjectFrom, TypeProjectRange:
		return true
	}
	return false
}

// Pricing is one row of the freelance_pricing table. Exactly one
// row per freelance profile (PK on profile_id), so the domain model
// never has to think about multi-row atomicity.
type Pricing struct {
	ProfileID  uuid.UUID
	Type       PricingType
	MinAmount  int64
	MaxAmount  *int64
	Currency   string
	Note       string
	Negotiable bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewPricingInput groups the raw inputs to NewPricing so the
// constructor stays under the 4-parameter cap. Every field is
// required — there are no optional defaults at this layer.
type NewPricingInput struct {
	ProfileID  uuid.UUID
	Type       PricingType
	MinAmount  int64
	MaxAmount  *int64
	Currency   string
	Note       string
	Negotiable bool
}

// NewPricing builds a fully-validated Pricing instance from raw
// inputs. Validation order (each step short-circuits on failure):
//
//  1. Type must be a known enum.
//  2. min_amount must be >= 0.
//  3. If max_amount is provided it must be >= min_amount, and the
//     type must accept a range. If the type is a range type,
//     max_amount MUST be provided.
//  4. Currency must be non-empty and must not be the literal "pct"
//     (that value belongs exclusively to commission_pct which lives
//     on the referrer side). Freelance pricing always uses an ISO
//     4217 currency code.
//
// Timestamps are left at their zero value — the repository layer
// fills CreatedAt / UpdatedAt from the database defaults on insert.
func NewPricing(in NewPricingInput) (*Pricing, error) {
	if !in.Type.IsValid() {
		return nil, ErrInvalidType
	}
	if err := validateAmounts(in.Type, in.MinAmount, in.MaxAmount); err != nil {
		return nil, err
	}
	if err := validateCurrency(in.Currency); err != nil {
		return nil, err
	}
	return &Pricing{
		ProfileID:  in.ProfileID,
		Type:       in.Type,
		MinAmount:  in.MinAmount,
		MaxAmount:  in.MaxAmount,
		Currency:   in.Currency,
		Note:       in.Note,
		Negotiable: in.Negotiable,
	}, nil
}

// validateAmounts enforces the min/max relationship and the range-
// vs-scalar rule per pricing type. Extracted so NewPricing reads
// as a validation pipeline and stays under the 50-line cap.
func validateAmounts(t PricingType, minAmount int64, maxAmount *int64) error {
	if minAmount < 0 {
		return ErrNegativeAmount
	}
	if maxAmount != nil {
		if *maxAmount < minAmount {
			return ErrMaxLessThanMin
		}
		if !typeAcceptsRange(t) {
			return ErrRangeNotAllowedForType
		}
		return nil
	}
	if typeRequiresRange(t) {
		return ErrRangeRequiredForType
	}
	return nil
}

// validateCurrency rejects empty strings and the literal "pct"
// which is reserved for the commission_pct type on the referrer
// side. Every other non-empty string is accepted here — the
// handler / service layer may layer on an ISO 4217 allow-list
// later without touching this function.
func validateCurrency(currency string) error {
	if currency == "" {
		return ErrInvalidCurrency
	}
	if currency == "pct" {
		return ErrInvalidCurrencyForType
	}
	return nil
}

// typeAcceptsRange reports whether max_amount may be non-nil for t.
// Only project_range accepts a range on the freelance side.
func typeAcceptsRange(t PricingType) bool {
	return t == TypeProjectRange
}

// typeRequiresRange reports whether max_amount MUST be non-nil for
// t. In the current catalog this is the same set as typeAcceptsRange.
func typeRequiresRange(t PricingType) bool {
	return typeAcceptsRange(t)
}
