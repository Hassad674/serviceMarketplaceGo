package subscription

import "errors"

// Sentinel errors for the subscription domain. The app layer wraps them
// with operation context; the handler layer maps them to HTTP status codes
// via errors.Is. Never stringify; always compare.
var (
	ErrInvalidOrganization = errors.New("subscription: organization id must be non-zero")
	ErrInvalidPlan         = errors.New("subscription: plan must be freelance or agency")
	ErrInvalidCycle        = errors.New("subscription: billing cycle must be monthly or annual")
	ErrInvalidStatus       = errors.New("subscription: status value unknown")
	ErrInvalidTransition   = errors.New("subscription: status transition not allowed from current state")
	ErrInvalidPeriod       = errors.New("subscription: current_period_end must not be before current_period_start")
	ErrMissingStripeIDs    = errors.New("subscription: stripe customer, subscription and price ids are required")
	ErrSameCycle           = errors.New("subscription: new cycle must differ from the current cycle")
	ErrNoActiveSub         = errors.New("subscription: no active subscription found for organization")
	ErrAlreadySubscribed   = errors.New("subscription: organization already has an open subscription")
	ErrNotFound            = errors.New("subscription: not found")
)
