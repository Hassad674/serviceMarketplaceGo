package profilepricing

import "errors"

// Sentinel errors surfaced by the profile pricing domain. The
// handler layer compares against these with errors.Is to map to a
// stable HTTP status + error code. Keeping them as package-level
// sentinels (not typed structs) matches the convention in
// domain/skill, domain/expertise, domain/user, and domain/profile.
var (
	// ErrInvalidKind — the pricing_kind field is not one of the two
	// frozen values {direct, referral}. HTTP 400.
	ErrInvalidKind = errors.New("invalid pricing kind")

	// ErrInvalidType — the pricing_type field is not one of the six
	// frozen values. HTTP 400.
	ErrInvalidType = errors.New("invalid pricing type")

	// ErrTypeNotAllowedForKind — the combination (kind, type) is
	// illegal. For instance, direct + commission_pct or referral +
	// daily. HTTP 400.
	ErrTypeNotAllowedForKind = errors.New("pricing type not allowed for this pricing kind")

	// ErrTypeNotAllowedForOrg — the pricing type is legal for the
	// kind but forbidden for this organization role. The canonical
	// example is agency + direct + daily: daily is valid under the
	// direct kind but agencies may only declare project_from /
	// project_range because they sell outcomes, not TJM. HTTP 400.
	ErrTypeNotAllowedForOrg = errors.New("pricing type not allowed for this organization role")

	// ErrKindNotAllowedForRole — the org-role / referrer-enabled
	// combination does not permit a pricing row of the given kind.
	// For instance, an agency trying to declare a referral kind, or
	// any provider_personal without referrer_enabled trying to
	// declare a referral kind. HTTP 403.
	ErrKindNotAllowedForRole = errors.New("pricing kind not allowed for this organization role")

	// ErrNegativeAmount — min_amount is strictly negative. Zero is
	// accepted (interpretation: "price on request" or "0 base").
	// HTTP 400.
	ErrNegativeAmount = errors.New("pricing amount cannot be negative")

	// ErrMaxLessThanMin — a range type (project_range, commission_pct)
	// received a max_amount strictly less than min_amount. HTTP 400.
	ErrMaxLessThanMin = errors.New("pricing max amount must be greater than or equal to min amount")

	// ErrRangeNotAllowedForType — a non-range type (daily, hourly,
	// project_from, commission_flat) received a non-nil max_amount.
	// HTTP 400.
	ErrRangeNotAllowedForType = errors.New("pricing type does not support a range")

	// ErrRangeRequiredForType — a range type (project_range,
	// commission_pct) was submitted without a max_amount. HTTP 400.
	ErrRangeRequiredForType = errors.New("pricing type requires a range (max amount)")

	// ErrInvalidCurrency — empty currency. HTTP 400.
	ErrInvalidCurrency = errors.New("invalid pricing currency")

	// ErrInvalidCurrencyForType — a commission_pct row received a
	// currency other than the literal "pct", OR a non-pct row
	// received currency "pct". HTTP 400.
	ErrInvalidCurrencyForType = errors.New("pricing currency does not match pricing type")

	// ErrPricingNotFound — the repository layer returns this when a
	// lookup by (org_id, kind) misses. HTTP 404.
	ErrPricingNotFound = errors.New("profile pricing not found")
)
