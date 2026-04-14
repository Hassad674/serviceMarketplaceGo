package profilepricing

import (
	"time"

	"github.com/google/uuid"
)

// PricingKind distinguishes the two logical buckets an organization
// may declare pricing for. The split matters because a single
// provider_personal with referrer_enabled=true can legitimately hold
// BOTH rows — one for its own freelance work (direct) and one for
// the deals it brings in as apporteur (referral) — and the UI
// surfaces them in two separate sections of the profile editor.
type PricingKind string

const (
	// KindDirect covers the org's own commercial offering:
	// daily / hourly / project_from / project_range.
	KindDirect PricingKind = "direct"

	// KindReferral covers commission-based apporteur pricing:
	// commission_pct / commission_flat.
	KindReferral PricingKind = "referral"
)

// IsValid reports whether k is one of the two frozen PricingKind
// values. The zero value ("") returns false.
func (k PricingKind) IsValid() bool {
	return k == KindDirect || k == KindReferral
}

// PricingType is the frozen enum of actual pricing shapes. Each
// value encodes both the unit (daily vs hourly vs project) and,
// where applicable, whether the figure is a point or a range.
//
// Relation to PricingKind:
//
//	direct   → daily, hourly, project_from, project_range
//	referral → commission_pct, commission_flat
//
// The kind → type map is enforced in limits.go
// (IsTypeAllowedForKind) — do not relax it here.
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
	// TypeCommissionPct — percentage bracket for apporteur. min_amount
	// and max_amount are basis points (10000 = 100.00%), currency is
	// the literal "pct".
	TypeCommissionPct PricingType = "commission_pct"
	// TypeCommissionFlat — flat fee per deal. min_amount cents, max nil.
	TypeCommissionFlat PricingType = "commission_flat"
)

// IsValid reports whether t is one of the six frozen PricingType
// values. The zero value ("") returns false.
func (t PricingType) IsValid() bool {
	switch t {
	case TypeDaily, TypeHourly, TypeProjectFrom, TypeProjectRange,
		TypeCommissionPct, TypeCommissionFlat:
		return true
	}
	return false
}

// Pricing is one row of the profile_pricing table (migration 083).
// The composite primary key is (OrganizationID, Kind), capping the
// cardinality at 2 rows per org — enforced at the DB level by the
// PK and reinforced by the domain validation here.
type Pricing struct {
	OrganizationID uuid.UUID
	Kind           PricingKind
	Type           PricingType

	// MinAmount is the starting / only amount in minor units:
	// - cents (EUR/USD/...) for currency pricings (daily, hourly,
	//   project_from, project_range, commission_flat)
	// - basis points for commission_pct (0..10000)
	MinAmount int64

	// MaxAmount is non-nil only for range types (project_range and
	// commission_pct). For every other type NewPricing rejects a
	// non-nil MaxAmount.
	MaxAmount *int64

	// Currency is an ISO 4217 code (EUR, USD, ...) OR the literal
	// "pct" for commission_pct rows. NewPricing enforces the
	// correlation.
	Currency string

	// Note is an optional free-form clarification surfaced under the
	// price on the public profile ("for day rates inside EU only",
	// "minimum engagement 3 months", ...). Empty by default.
	Note string

	// Negotiable is an explicit yes/no flag surfaced on the public
	// profile card as a "négociable" tag. Distinct from Note: Note
	// describes constraints, Negotiable declares commercial
	// flexibility. Picked by the provider on every save — the
	// request DTO is required to send it.
	Negotiable bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewPricing builds a fully-validated Pricing instance from raw
// inputs. Every branch returns a specific sentinel so the handler
// layer can map it to a stable HTTP status.
//
// Validation order (each step short-circuits on failure):
//
//  1. PricingKind + PricingType must each be a known enum.
//  2. (kind, type) must be a legal pair (direct → daily/hourly/
//     project_*, referral → commission_*).
//  3. min_amount must be >= 0.
//  4. If max_amount is provided it must be >= min_amount.
//  5. If max_amount is provided the type must be a range type.
//     Conversely, if the type is a range type max_amount must be
//     provided.
//  6. Currency must be non-empty. For commission_pct it must be
//     the literal "pct"; for any other type it must NOT be "pct".
//
// Timestamps are left at their zero value — the repository layer
// fills CreatedAt / UpdatedAt from the DB defaults on insert.
// NewPricingInput groups the raw inputs to NewPricing so the
// constructor stays under the 4-parameter cap. Every field is
// required — there are no optional defaults at this layer.
type NewPricingInput struct {
	OrganizationID uuid.UUID
	Kind           PricingKind
	Type           PricingType
	MinAmount      int64
	MaxAmount      *int64
	Currency       string
	Note           string
	Negotiable     bool
}

func NewPricing(in NewPricingInput) (*Pricing, error) {
	if !in.Kind.IsValid() {
		return nil, ErrInvalidKind
	}
	if !in.Type.IsValid() {
		return nil, ErrInvalidType
	}
	if !IsTypeAllowedForKind(in.Kind, in.Type) {
		return nil, ErrTypeNotAllowedForKind
	}
	if err := validateAmounts(in.Type, in.MinAmount, in.MaxAmount); err != nil {
		return nil, err
	}
	if err := validateCurrency(in.Type, in.Currency); err != nil {
		return nil, err
	}
	return &Pricing{
		OrganizationID: in.OrganizationID,
		Kind:           in.Kind,
		Type:           in.Type,
		MinAmount:      in.MinAmount,
		MaxAmount:      in.MaxAmount,
		Currency:       in.Currency,
		Note:           in.Note,
		Negotiable:     in.Negotiable,
	}, nil
}

// validateAmounts enforces the min/max relationship and the
// range-vs-scalar constraint. Extracted so NewPricing stays under
// the 50-line cap and reads as a validation pipeline.
func validateAmounts(t PricingType, min int64, max *int64) error {
	if min < 0 {
		return ErrNegativeAmount
	}
	if max != nil {
		if *max < min {
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

// validateCurrency enforces the "pct ⇄ commission_pct" correlation.
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

// typeAcceptsRange reports whether max_amount may be non-nil for t.
// Only project_range and commission_pct accept a range; everything
// else is a single amount.
func typeAcceptsRange(t PricingType) bool {
	switch t {
	case TypeProjectRange, TypeCommissionPct:
		return true
	}
	return false
}

// typeRequiresRange reports whether max_amount MUST be non-nil for
// t. In the current catalog this is the same set as typeAcceptsRange
// (a range type always needs the upper bound), but the two concepts
// are separated in case a future type accepts an optional range.
func typeRequiresRange(t PricingType) bool {
	return typeAcceptsRange(t)
}
