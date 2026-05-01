package user

import "errors"

var (
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password must be at least 10 characters with uppercase, lowercase, digit, and special character")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRole        = errors.New("invalid role: must be agency, enterprise, or provider")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrAccountSuspended   = errors.New("account is suspended")
	ErrAccountBanned      = errors.New("account is banned")
	ErrKYCRestricted      = errors.New("account restricted: payment info not configured within 14 days of first earning")

	// ErrAccountScheduledForDeletion is returned by the auth login
	// flow when a user whose GDPR soft-delete flag is set
	// (users.deleted_at IS NOT NULL) tries to authenticate. The
	// handler maps this to HTTP 410 Gone with code
	// account_scheduled_for_deletion so the frontend can guide the
	// user to /account/cancel-deletion if they want to keep the
	// account.
	ErrAccountScheduledForDeletion = errors.New("account is scheduled for deletion")

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

// NewScheduledForDeletionError builds the account-status error returned
// when a soft-deleted user tries to log in. The reason carries the
// scheduled hard-delete date (RFC3339) so the handler can tell the
// frontend exactly when the cron will purge the account if the user
// does not cancel the request.
func NewScheduledForDeletionError(reason string) *AccountStatusError {
	return &AccountStatusError{Sentinel: ErrAccountScheduledForDeletion, Reason: reason}
}
