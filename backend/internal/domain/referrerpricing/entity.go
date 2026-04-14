// Package referrerpricing owns the domain model for the pricing row
// attached to a referrer profile (migration 100). It is the persona-
// specific sibling of freelancepricing: where freelance pricing
// covers daily / hourly / project_from / project_range, referrer
// pricing covers the two commission-based offerings that an
// apporteur d'affaires may declare — commission_pct (percentage
// bracket) and commission_flat (flat fee per deal).
//
// Design note — zero cross-feature imports. This package imports
// nothing from other feature domains. commission_pct uses the
// literal currency "pct" with min/max expressed as basis points
// (0..10000 = 0..100%), while commission_flat uses a conventional
// ISO 4217 currency with cents. The correlation is enforced in
// NewPricing.
package referrerpricing

import (
	"time"

	"github.com/google/uuid"
)

// PricingType is the frozen enum of valid pricing shapes for the
// referrer persona.
type PricingType string

const (
	// TypeCommissionPct — percentage bracket. min_amount and
	// max_amount are basis points (10000 = 100.00%). Currency is
	// the literal string "pct".
	TypeCommissionPct PricingType = "commission_pct"

	// TypeCommissionFlat — flat fee per deal. min_amount is cents,
	// max_amount nil. Currency is an ISO 4217 code.
	TypeCommissionFlat PricingType = "commission_flat"
)

// IsValid reports whether t is one of the two frozen values. The
// zero value ("") returns false.
func (t PricingType) IsValid() bool {
	switch t {
	case TypeCommissionPct, TypeCommissionFlat:
		return true
	}
	return false
}

// Pricing is one row of the referrer_pricing table. Exactly one
// row per referrer profile (PK on profile_id).
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
// constructor stays under the 4-parameter cap.
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
// inputs. Validation order:
//
//  1. Type must be a known enum.
//  2. min_amount must be >= 0.
//  3. commission_pct is a range type — max_amount is required and
//     must be >= min_amount. commission_flat is a scalar — max_amount
//     must be nil.
//  4. commission_pct MUST use the literal currency "pct".
//     commission_flat MUST NOT use "pct".
func NewPricing(in NewPricingInput) (*Pricing, error) {
	if !in.Type.IsValid() {
		return nil, ErrInvalidType
	}
	if err := validateAmounts(in.Type, in.MinAmount, in.MaxAmount); err != nil {
		return nil, err
	}
	if err := validateCurrency(in.Type, in.Currency); err != nil {
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
// vs-scalar rule. commission_pct is a range, commission_flat is a
// scalar — the rules are identical to the freelance side except for
// which type carries which shape.
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
		// commission_pct max is capped at 10000 basis points (100%).
		// Beyond that is nonsensical and would surface as a broken
		// rendering on the public profile.
		if t == TypeCommissionPct && *maxAmount > 10000 {
			return ErrCommissionPctOutOfRange
		}
		return nil
	}
	if typeRequiresRange(t) {
		return ErrRangeRequiredForType
	}
	// commission_pct scalar check: when max_amount is nil we still
	// reject it because this type is range-only. Falls through the
	// guard above.
	return nil
}

// validateCurrency enforces the "pct ⇄ commission_pct" correlation.
// commission_flat requires a non-empty, non-"pct" currency so the
// handler/service layer can layer on an ISO 4217 allow-list later
// without touching this function.
func validateCurrency(t PricingType, currency string) error {
	if currency == "" {
		return ErrInvalidCurrency
	}
	if t == TypeCommissionPct && currency != "pct" {
		return ErrInvalidCurrencyForType
	}
	if t != TypeCommissionPct && currency == "pct" {
		return ErrInvalidCurrencyForType
	}
	return nil
}

// typeAcceptsRange reports whether max_amount may be non-nil.
// commission_pct is always a range (min..max percentage);
// commission_flat is always a scalar (one flat fee).
func typeAcceptsRange(t PricingType) bool {
	return t == TypeCommissionPct
}

// typeRequiresRange reports whether max_amount MUST be non-nil.
// Same set as typeAcceptsRange on the referrer side — commission_pct
// is meaningless without an upper bound.
func typeRequiresRange(t PricingType) bool {
	return typeAcceptsRange(t)
}
