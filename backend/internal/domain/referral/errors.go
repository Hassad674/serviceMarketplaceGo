package referral

import "errors"

// Domain sentinel errors. Wrap them in app/handler layers with errors.Is checks
// to map to HTTP status codes; never wrap them inside the domain package itself.
var (
	// Lookup
	ErrNotFound            = errors.New("referral not found")
	ErrAttributionNotFound = errors.New("referral attribution not found")
	ErrCommissionNotFound  = errors.New("referral commission not found")

	// ErrAttributionAlreadyEnded is returned by the repository when an
	// EndAttribution call targets a row whose ended_at is already set.
	// The app service treats this as the idempotent success path (200
	// no-op) but the repo surfaces it so callers can distinguish "I
	// just ended it" from "it was already ended".
	ErrAttributionAlreadyEnded = errors.New("referral attribution is already ended")

	// Identity / authorization
	ErrSelfReferral      = errors.New("a referrer cannot introduce themselves")
	ErrSameOrganization  = errors.New("provider and client must belong to different organizations")
	ErrNotAuthorized     = errors.New("actor is not authorized to perform this action on this referral")
	ErrReferrerRequired  = errors.New("only a provider with referrer mode enabled can create a referral")
	ErrInvalidProviderRole = errors.New("provider party must be a provider or an agency")
	ErrInvalidClientRole = errors.New("client party must be an enterprise or an agency")

	// Anti-fraud — the provider and client party are already in business
	// relation (an existing 1:1 conversation links them). An apporteur
	// cannot earn a commission for introducing two parties that already
	// know each other.
	ErrPartiesAlreadyInRelation = errors.New("provider and client party are already in relation")

	// Term validation
	ErrRateOutOfRange     = errors.New("commission rate must be between 0 and 50 percent")
	ErrDurationOutOfRange = errors.New("exclusivity duration must be between 1 and 24 months")
	ErrEmptyMessage       = errors.New("intro message cannot be empty")
	ErrMessageTooLong     = errors.New("intro message exceeds maximum length")

	// Lifecycle
	ErrInvalidTransition = errors.New("invalid status transition for this action")
	ErrAlreadyTerminal   = errors.New("referral is already in a terminal state")
	ErrCoupleLocked      = errors.New("an active referral already exists for this provider/client couple")
	ErrConcurrentNegotiation = errors.New("the referral was modified by another actor; please refresh")

	// Snapshot
	ErrSnapshotInvalid = errors.New("intro snapshot is invalid")

	// Commission flow
	ErrCommissionAlreadyExists = errors.New("a commission already exists for this milestone")
	ErrCommissionNotPayable    = errors.New("commission is not in a payable state")
	ErrClawbackNotApplicable   = errors.New("commission cannot be clawed back in its current state")
	ErrInsufficientGrossAmount = errors.New("gross amount must be greater than zero")
)
