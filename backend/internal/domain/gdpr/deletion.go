package gdpr

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// PurgeWindow is the cooldown between soft-delete (deleted_at set)
// and hard-purge (cron tx). Decision 3 of the P5 brief.
const PurgeWindow = 30 * 24 * time.Hour

// ConfirmationTokenTTL is the lifetime of the JWT in the deletion
// confirmation email. Decision 5 of the P5 brief: 24 hours.
const ConfirmationTokenTTL = 24 * time.Hour

// ConfirmationTokenPurpose is the value of the JWT `purpose` claim
// for deletion confirmation tokens. The handler MUST verify this
// claim explicitly so a leaked access token from a different flow
// cannot be replayed against /confirm-deletion.
const ConfirmationTokenPurpose = "account_deletion"

// ErrOrgOwnerHasMembers is returned by the deletion request flow
// when the user owns one or more organizations with at least one
// other active member. Decision 6 of the P5 brief: the handler
// maps this to HTTP 409 with a remediation payload listing each
// blocked org + the actions available (transfer / dissolve).
var ErrOrgOwnerHasMembers = errors.New("gdpr: account owns an org with active members")

// ErrAlreadyScheduled is returned when a user who already has
// deleted_at set tries to request another deletion. The handler
// treats this as idempotent (200 with the existing schedule)
// rather than an error, but the service layer surfaces it so the
// orchestrator can short-circuit the email-sending side effect.
var ErrAlreadyScheduled = errors.New("gdpr: deletion already scheduled")

// BlockedOrg describes one organization that prevents a deletion
// request from completing because the requesting user is the
// Owner and there are still active members. The handler renders
// this verbatim into the 409 response body so the frontend can
// show a per-org remediation card.
type BlockedOrg struct {
	OrgID       uuid.UUID         `json:"org_id"`
	OrgName     string            `json:"org_name"`
	MemberCount int               `json:"member_count"`
	Admins      []AvailableAdmin  `json:"available_admins"`
	Actions     []RemediationAction `json:"actions"`
}

// AvailableAdmin is the slim user identity used to suggest a
// transfer target. Only id + email are exposed: this list is
// shown to the requesting Owner who is allowed to see who else
// has admin rights in their own org.
type AvailableAdmin struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

// RemediationAction is the enum of actions the frontend renders
// inline in the 409 response. Adding a new action is a domain
// + frontend change; the backend just echoes whichever ones the
// service layer attaches per blocked org.
type RemediationAction string

const (
	ActionTransferOwnership RemediationAction = "transfer_ownership"
	ActionDissolveOrg       RemediationAction = "dissolve_org"
)

// OwnerBlockedError carries the BlockedOrg list alongside the
// sentinel. The handler unwraps with errors.As to render the 409
// payload. Tests can assert on the slice directly.
type OwnerBlockedError struct {
	Orgs []BlockedOrg
}

func (e *OwnerBlockedError) Error() string {
	return ErrOrgOwnerHasMembers.Error()
}

func (e *OwnerBlockedError) Unwrap() error {
	return ErrOrgOwnerHasMembers
}

// NewOwnerBlockedError builds the deletion-blocked error with the
// provided list of orgs. Used by the service layer when the
// pre-check finds at least one blocking org.
func NewOwnerBlockedError(orgs []BlockedOrg) *OwnerBlockedError {
	return &OwnerBlockedError{Orgs: orgs}
}

// ScheduledHardDeleteAt computes the timestamp at which the cron
// will purge a user whose soft-delete landed at deletedAt. Used
// by the service to fill confirmation/cancel emails and by the
// handler to render the dashboard banner countdown.
func ScheduledHardDeleteAt(deletedAt time.Time) time.Time {
	return deletedAt.Add(PurgeWindow)
}

// IsPurgeable reports whether the cooldown has elapsed for the
// given soft-delete timestamp. The cron purge tx re-checks this
// inside the SELECT FOR UPDATE so a cancel that arrives mid-tick
// cannot be racy.
func IsPurgeable(deletedAt time.Time, now time.Time) bool {
	return !deletedAt.IsZero() && now.Sub(deletedAt) >= PurgeWindow
}
