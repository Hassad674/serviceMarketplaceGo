// Package search is the application service that converts user-side
// filter inputs into Typesense queries. It wraps the per-persona
// scoped clients defined in internal/search/ and parses the raw
// JSON responses into the typed result struct that handlers expose
// to the frontend.
//
// This package never imports any other feature — it depends only on
// the internal/search package (for the Typesense client primitives)
// and the stdlib. Removing search engine support is a matter of
// deleting this folder + the wiring lines in cmd/api/main.go.
package search

import (
	"fmt"
	"strconv"
	"strings"
)

// FilterInput is the typed payload posted by the frontend (or
// constructed by the server-side proxy handler). Every field maps
// to a single Typesense filter_by clause.
//
// Pointer types are used for optional scalars (PricingMin, GeoLat,
// IsVerified, …) so the zero value (0, false) is distinguishable
// from "not set". Slices are interpreted as "no filter" when empty.
type FilterInput struct {
	// AvailabilityStatus is a list of allowed availability buckets
	// (now / soon / not). Empty = no filter.
	AvailabilityStatus []string

	// Pricing range — both bounds optional. The min bound applies
	// to pricing_min_amount; the max bound applies to
	// pricing_max_amount when present, otherwise to
	// pricing_min_amount. Centimes / smallest-unit semantics.
	PricingMin *int64
	PricingMax *int64

	// City + CountryCode are ANDed together when both set. They
	// search the city/country_code facets exactly (no fuzzy
	// matching — the autocomplete is responsible for canonical
	// values).
	City        string
	CountryCode string

	// GeoLat / GeoLng / GeoRadiusKm activate the geopoint
	// proximity filter. All three must be set; if any is missing
	// the geo filter is dropped silently.
	GeoLat      *float64
	GeoLng      *float64
	GeoRadiusKm *float64

	// Languages is the set of professional languages the actor
	// must speak. Treated as OR (any match).
	Languages []string

	// ExpertiseDomains is the set of expertise keys the actor
	// must declare. Treated as OR.
	ExpertiseDomains []string

	// Skills is the set of skill keys the actor must declare.
	// Treated as OR.
	Skills []string

	// RatingMin filters by `rating_average:>=X`. Nil = no filter.
	RatingMin *float64

	// WorkMode is the list of accepted work modes (remote /
	// on_site / hybrid). Treated as OR.
	WorkMode []string

	// Boolean toggles. Nil = no filter, true = require true,
	// false = require false. The pointer-vs-bool distinction is
	// what lets us tell "user explicitly unchecked" from "user
	// has not interacted".
	IsVerified *bool
	IsTopRated *bool
	Negotiable *bool
}

// BuildFilterBy assembles the Typesense filter_by string from the
// input. Returns an empty string when no filter is set so the
// scoped client's persona clause is the only filter applied.
//
// The function is pure and deterministic — same input always
// produces the same output. Order of clauses is fixed (alphabetic
// by field name) so tests can assert against the exact string.
//
// Implementation note: empty inputs are silently skipped instead of
// raising errors. The handler validates required parameters at the
// HTTP boundary; this layer only formats clauses for whatever made
// it through.
func BuildFilterBy(input FilterInput) string {
	clauses := make([]string, 0, 12)

	if c := buildAvailabilityClause(input.AvailabilityStatus); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildPricingClause(input.PricingMin, input.PricingMax); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildCityClause(input.City); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildCountryClause(input.CountryCode); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildGeoClause(input.GeoLat, input.GeoLng, input.GeoRadiusKm); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildStringSliceClause("languages_professional", input.Languages); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildStringSliceClause("expertise_domains", input.ExpertiseDomains); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildStringSliceClause("skills", input.Skills); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildRatingClause(input.RatingMin); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildStringSliceClause("work_mode", input.WorkMode); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildBoolClause("is_verified", input.IsVerified); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildBoolClause("is_top_rated", input.IsTopRated); c != "" {
		clauses = append(clauses, c)
	}
	if c := buildBoolClause("pricing_negotiable", input.Negotiable); c != "" {
		clauses = append(clauses, c)
	}

	return strings.Join(clauses, " && ")
}

// buildAvailabilityClause emits `availability_status:[a,b,c]` for
// the given allowed values. Empty input returns an empty string so
// the caller can skip appending.
func buildAvailabilityClause(values []string) string {
	cleaned := dedupeStrings(values)
	if len(cleaned) == 0 {
		return ""
	}
	return fmt.Sprintf("availability_status:[%s]", strings.Join(cleaned, ","))
}

// buildPricingClause emits a numeric range clause on
// pricing_min_amount. Both bounds are optional; when only one is
// set the corresponding inequality is emitted alone.
func buildPricingClause(minAmt, maxAmt *int64) string {
	if minAmt == nil && maxAmt == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if minAmt != nil {
		parts = append(parts, fmt.Sprintf("pricing_min_amount:>=%d", *minAmt))
	}
	if maxAmt != nil {
		// Use pricing_min_amount for the upper bound too — this
		// matches the user's mental model "I want offers under
		// X". Pricing_max_amount is reserved for ranges (showing
		// "price range" on the card) and rarely lines up cleanly
		// with the upper-bound filter intent.
		parts = append(parts, fmt.Sprintf("pricing_min_amount:<=%d", *maxAmt))
	}
	return strings.Join(parts, " && ")
}

// buildCityClause emits `city:=City Name`. Trimmed to drop
// whitespace-only inputs.
func buildCityClause(city string) string {
	trimmed := strings.TrimSpace(city)
	if trimmed == "" {
		return ""
	}
	// Wrap in backticks because city names commonly contain
	// spaces and Typesense uses backticks as the literal-string
	// delimiter inside filter_by.
	return fmt.Sprintf("city:=`%s`", trimmed)
}

// buildCountryClause emits `country_code:=fr`. ISO codes are
// uppercase by convention but the Typesense field is stored in
// lowercase, so we lowercase here.
func buildCountryClause(code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return ""
	}
	return fmt.Sprintf("country_code:=%s", strings.ToLower(trimmed))
}

// buildGeoClause emits `location:(lat,lng,N km)`. All three
// arguments must be present; otherwise the clause is dropped.
func buildGeoClause(lat, lng, radiusKm *float64) string {
	if lat == nil || lng == nil || radiusKm == nil {
		return ""
	}
	if *radiusKm <= 0 {
		return ""
	}
	return fmt.Sprintf("location:(%s,%s,%s km)",
		formatFloat(*lat), formatFloat(*lng), formatFloat(*radiusKm))
}

// buildStringSliceClause emits `field:[a,b,c]`. Empty input is a
// no-op. Used for languages, expertise_domains, skills, work_mode.
func buildStringSliceClause(field string, values []string) string {
	cleaned := dedupeStrings(values)
	if len(cleaned) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:[%s]", field, strings.Join(cleaned, ","))
}

// buildRatingClause emits `rating_average:>=X` for the given
// minimum. Nil or zero is interpreted as "no filter".
func buildRatingClause(minRating *float64) string {
	if minRating == nil || *minRating <= 0 {
		return ""
	}
	return fmt.Sprintf("rating_average:>=%s", formatFloat(*minRating))
}

// buildBoolClause emits `field:=true` or `field:=false`. Nil is a
// no-op so the caller can leave the toggle unset.
func buildBoolClause(field string, value *bool) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%s:=%t", field, *value)
}

// dedupeStrings trims, drops empty entries, and removes duplicates
// while preserving first-occurrence order. Returned slice is nil
// when the input is empty so the callers can compare against nil
// instead of a zero-length slice.
func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// formatFloat prints a float without trailing zeros — `1` instead
// of `1.000000`. Used by the geo + rating clauses so the wire
// format stays compact.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
