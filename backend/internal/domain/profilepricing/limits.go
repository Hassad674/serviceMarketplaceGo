// Package profilepricing owns the domain model for organizations'
// pricing rows exposed on their public profile. It is intentionally
// independent of the main `profile` package because pricing persists
// to its own table (profile_pricing, migration 083) with its own
// edit cadence and cardinality (up to 2 rows per org) — coupling
// it to Profile would force the main profile row to churn on every
// pricing edit.
//
// Design note — zero cross-domain imports: every reference to an
// organization's type is a plain string alias (OrgType below). This
// preserves the hexagonal invariant that domain modules are
// independent and removable — the only shared dependency for the
// pricing feature is the organizations TABLE, referenced by the
// migration via REFERENCES organizations(id) and never the Go
// package.
package profilepricing

// OrgType is a plain string alias matching the wire values used
// throughout the marketplace backend ("agency", "provider_personal",
// "enterprise"). Using a string alias rather than importing the
// organization or profile packages keeps this module fully
// independent at the Go-package level while still matching the
// exact wire values in the database and API.
type OrgType = string

const (
	OrgTypeAgency           OrgType = "agency"
	OrgTypeProviderPersonal OrgType = "provider_personal"
	OrgTypeEnterprise       OrgType = "enterprise"
)

// AllowedTypesForKind returns the pricing types that are valid for
// the given kind. direct covers freelance/agency commercial pricing
// (daily/hourly/project_from/project_range), referral covers
// commission-based apporteur pricing (commission_pct/commission_flat).
//
// Returning nil for an unknown kind makes IsTypeAllowedForKind
// safely return false, preserving the "reject everything unknown"
// invariant without a special case at the caller.
func AllowedTypesForKind(kind PricingKind) []PricingType {
	switch kind {
	case KindDirect:
		return []PricingType{
			TypeDaily,
			TypeHourly,
			TypeProjectFrom,
			TypeProjectRange,
		}
	case KindReferral:
		return []PricingType{
			TypeCommissionPct,
			TypeCommissionFlat,
		}
	}
	return nil
}

// IsTypeAllowedForKind reports whether pricing_type is acceptable
// under the given pricing_kind. O(n) over the small AllowedTypesForKind
// list — callable in validation hot paths without noticeable cost.
func IsTypeAllowedForKind(kind PricingKind, t PricingType) bool {
	for _, allowed := range AllowedTypesForKind(kind) {
		if allowed == t {
			return true
		}
	}
	return false
}

// AllowedTypesForOrg returns the pricing types valid for an
// organization-role, referrer-state, and pricing-kind triplet. It
// is stricter than AllowedTypesForKind because some org roles do
// not unlock every type that is otherwise compatible with the kind.
//
// Current product rules:
//
//   - agency + direct → project_from and project_range only
//     (agencies sell outcomes, not TJM or taux horaire)
//   - provider_personal + direct → all four direct types
//   - *_ + referral → both commission types
//
// Unknown (org, kind) combinations fall through to nil, which makes
// IsTypeAllowedForOrg safely return false.
func AllowedTypesForOrg(orgType OrgType, referrerEnabled bool, kind PricingKind) []PricingType {
	if !IsKindAllowedForOrg(orgType, referrerEnabled, kind) {
		return nil
	}
	if orgType == OrgTypeAgency && kind == KindDirect {
		return []PricingType{TypeProjectFrom, TypeProjectRange}
	}
	return AllowedTypesForKind(kind)
}

// IsTypeAllowedForOrg reports whether a pricing row of the given
// type may be declared by the given (org, referrer, kind) triplet.
// O(n) over the small AllowedTypesForOrg slice.
func IsTypeAllowedForOrg(orgType OrgType, referrerEnabled bool, kind PricingKind, t PricingType) bool {
	for _, allowed := range AllowedTypesForOrg(orgType, referrerEnabled, kind) {
		if allowed == t {
			return true
		}
	}
	return false
}

// IsKindAllowedForOrg reports whether an organization of the given
// type and referrer state may declare a pricing row of the given
// kind.
//
// Rules (mirrors the product spec):
//
//   - enterprise  → NO pricing at all (client-side org, never a provider)
//   - agency      → direct only (agencies cannot declare commissions)
//   - provider_personal without referrer_enabled → direct only
//   - provider_personal with referrer_enabled    → BOTH direct AND
//     referral (the same org may expose a TJM for its own work and
//     a commission rate for the deals it brings in — this is the
//     "double-casquette" use case the feature was built for)
//
// Unknown org types fall through to false as a safe default: adding
// a new provider-like org type in the future is an opt-in code
// change rather than an accidental opening of the feature.
func IsKindAllowedForOrg(orgType OrgType, referrerEnabled bool, kind PricingKind) bool {
	switch orgType {
	case OrgTypeAgency:
		return kind == KindDirect
	case OrgTypeProviderPersonal:
		if kind == KindDirect {
			return true
		}
		return referrerEnabled && kind == KindReferral
	default:
		// enterprise + any unknown future type → no pricing.
		return false
	}
}
