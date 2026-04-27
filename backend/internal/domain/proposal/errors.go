package proposal

import "errors"

var (
	ErrProposalNotFound       = errors.New("proposal not found")
	ErrInvalidStatus          = errors.New("invalid proposal status for this operation")
	ErrEmptyTitle             = errors.New("proposal title cannot be empty")
	ErrEmptyDescription       = errors.New("proposal description cannot be empty")
	ErrInvalidAmount          = errors.New("proposal amount must be greater than zero")
	ErrSameUser               = errors.New("cannot create proposal with yourself")
	ErrNotAuthorized          = errors.New("not authorized to perform this action")
	ErrInvalidRoleCombination = errors.New("invalid role combination for proposal")
	ErrCannotModify           = errors.New("only the recipient can modify a pending proposal")
	ErrAlreadyAccepted        = errors.New("proposal is already accepted")
	ErrAlreadyDeclined        = errors.New("proposal is already declined")
	ErrNotProvider            = errors.New("only the provider can perform this action")
	ErrNotClient              = errors.New("only the client can perform this action")
	ErrBelowMinimumAmount     = errors.New("proposal amount must be at least 30 EUR (3000 centimes)")
	// ErrProviderKYCNotReady is returned when the client tries to release
	// a milestone but the provider's organization has no Stripe Connect
	// account, or the connected account does not yet have payouts
	// enabled. The escrowed funds cannot be transferred yet, so we MUST
	// reject the release before flipping any local state — otherwise the
	// client sees a "milestone paid" notification while the money never
	// actually leaves the platform.
	ErrProviderKYCNotReady = errors.New("provider has not completed Stripe onboarding")
)
