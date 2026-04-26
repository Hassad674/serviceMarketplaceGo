package service

import "context"

// VIESValidator validates an EU VAT number against the European
// Commission's VIES service. The result is opaque enough that callers
// only need the boolean answer + the raw payload (kept as legal
// proof on the billing_profile row).
//
// The production adapter caches positive results 24h in Redis to
// avoid hitting VIES on every page load. Negative results are NOT
// cached — VAT numbers can be activated retroactively and we want
// the user to be able to retry quickly.
type VIESValidator interface {
	Validate(ctx context.Context, countryCode, vatNumber string) (VIESResult, error)
}

// VIESResult is the slim projection of the VIES response the caller
// stores. RawPayload is the full JSON returned by VIES — kept for
// audit / contestation in case of a tax inspection.
type VIESResult struct {
	Valid           bool
	CountryCode     string // echoed back from VIES, may differ in casing
	VATNumber       string
	RegisteredName  string // Stripe / VIES sometimes returns the company name
	RegisteredAddr  string
	RawPayload      []byte // the full JSON body, opaque to the domain
	CheckedAt       int64  // unix seconds — VIES timestamp, NOT our local clock
}
