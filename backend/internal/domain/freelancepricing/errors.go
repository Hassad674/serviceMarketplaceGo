package freelancepricing

import "errors"

// Sentinel errors surfaced by the freelance_pricing domain. The
// handler layer compares against these with errors.Is to map them
// to a stable HTTP status + error code.
var (
	// ErrInvalidType — the pricing_type field is not one of the four
	// frozen values {daily, hourly, project_from, project_range}.
	// HTTP 400.
	ErrInvalidType = errors.New("invalid freelance pricing type")

	// ErrNegativeAmount — min_amount is strictly negative. Zero is
	// accepted (interpretation: "price on request" or "0 base").
	// HTTP 400.
	ErrNegativeAmount = errors.New("freelance pricing amount cannot be negative")

	// ErrMaxLessThanMin — a range type (project_range) received a
	// max_amount strictly less than min_amount. HTTP 400.
	ErrMaxLessThanMin = errors.New("freelance pricing max amount must be greater than or equal to min amount")

	// ErrRangeNotAllowedForType — a non-range type received a non-nil
	// max_amount. HTTP 400.
	ErrRangeNotAllowedForType = errors.New("freelance pricing type does not support a range")

	// ErrRangeRequiredForType — a range type (project_range) was
	// submitted without a max_amount. HTTP 400.
	ErrRangeRequiredForType = errors.New("freelance pricing type requires a range (max amount)")

	// ErrInvalidCurrency — empty currency. HTTP 400.
	ErrInvalidCurrency = errors.New("invalid freelance pricing currency")

	// ErrInvalidCurrencyForType — the currency 'pct' was sent, which
	// is reserved for commission_pct on the referrer side. HTTP 400.
	ErrInvalidCurrencyForType = errors.New("freelance pricing currency 'pct' is reserved for referrer commissions")

	// ErrPricingNotFound — the repository layer returns this when a
	// lookup by profile_id misses. HTTP 404.
	ErrPricingNotFound = errors.New("freelance pricing not found")
)
