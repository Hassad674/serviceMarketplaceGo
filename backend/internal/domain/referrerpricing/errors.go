package referrerpricing

import "errors"

// Sentinel errors surfaced by the referrer_pricing domain. The
// handler layer compares against these with errors.Is to map them
// to a stable HTTP status + error code.
var (
	// ErrInvalidType — the pricing_type field is not one of the two
	// frozen values {commission_pct, commission_flat}. HTTP 400.
	ErrInvalidType = errors.New("invalid referrer pricing type")

	// ErrNegativeAmount — min_amount is strictly negative. HTTP 400.
	ErrNegativeAmount = errors.New("referrer pricing amount cannot be negative")

	// ErrMaxLessThanMin — commission_pct received a max_amount
	// strictly less than min_amount. HTTP 400.
	ErrMaxLessThanMin = errors.New("referrer pricing max amount must be greater than or equal to min amount")

	// ErrRangeNotAllowedForType — commission_flat received a non-nil
	// max_amount. HTTP 400.
	ErrRangeNotAllowedForType = errors.New("referrer pricing type does not support a range")

	// ErrRangeRequiredForType — commission_pct was submitted without
	// a max_amount. HTTP 400.
	ErrRangeRequiredForType = errors.New("referrer pricing type requires a range (max amount)")

	// ErrInvalidCurrency — empty currency. HTTP 400.
	ErrInvalidCurrency = errors.New("invalid referrer pricing currency")

	// ErrInvalidCurrencyForType — commission_pct received a currency
	// other than the literal "pct", OR commission_flat received
	// "pct". HTTP 400.
	ErrInvalidCurrencyForType = errors.New("referrer pricing currency does not match pricing type")

	// ErrCommissionPctOutOfRange — commission_pct max_amount exceeds
	// 10000 basis points (100%). HTTP 400.
	ErrCommissionPctOutOfRange = errors.New("referrer pricing commission_pct max cannot exceed 10000 basis points (100%)")

	// ErrPricingNotFound — the repository layer returns this when a
	// lookup by profile_id misses. HTTP 404.
	ErrPricingNotFound = errors.New("referrer pricing not found")
)
