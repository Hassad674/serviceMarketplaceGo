package user

import "errors"

var (
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password must be at least 8 characters with uppercase, lowercase, and digit")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRole        = errors.New("invalid role: must be agency, enterprise, or provider")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrAccountSuspended   = errors.New("account is suspended")
	ErrAccountBanned      = errors.New("account is banned")
	ErrKYCRestricted      = errors.New("account restricted: payment info not configured within 14 days of first earning")

	// ErrDisplayNameInappropriate is returned by the auth service when
	// the moderation pipeline (Phase 2) refuses the registration because
	// the supplied display_name / first_name / last_name combination is
	// flagged as inappropriate for a public-facing identity. The handler
	// maps this to HTTP 422 with code "display_name_inappropriate" so
	// the frontend can show the message in context.
	ErrDisplayNameInappropriate = errors.New("display name inappropriate")
)

// AccountStatusError carries the suspension/ban reason alongside the sentinel.
type AccountStatusError struct {
	Sentinel error
	Reason   string
}

func (e *AccountStatusError) Error() string {
	return e.Sentinel.Error()
}

func (e *AccountStatusError) Unwrap() error {
	return e.Sentinel
}

func NewSuspendedError(reason string) *AccountStatusError {
	return &AccountStatusError{Sentinel: ErrAccountSuspended, Reason: reason}
}

func NewBannedError(reason string) *AccountStatusError {
	return &AccountStatusError{Sentinel: ErrAccountBanned, Reason: reason}
}
