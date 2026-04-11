package organization

import "errors"

// Domain sentinel errors.
//
// The app layer wraps these with operation context via fmt.Errorf("%w", err),
// and the handler layer uses errors.Is to map them to HTTP status codes.
// Never wrap sentinels INSIDE the domain package — return them as-is.
var (
	// Input / shape validation
	ErrInvalidOrgType          = errors.New("invalid organization type")
	ErrInvalidRole             = errors.New("invalid organization role")
	ErrInvalidInvitationStatus = errors.New("invalid invitation status")
	ErrInvalidEmail            = errors.New("invalid email format")
	ErrTitleTooLong            = errors.New("title too long")
	ErrNameTooLong             = errors.New("name too long")
	ErrNameRequired            = errors.New("first name and last name are required")

	// Lookup
	ErrOrgNotFound        = errors.New("organization not found")
	ErrMemberNotFound     = errors.New("organization member not found")
	ErrInvitationNotFound = errors.New("invitation not found")

	// Invitation lifecycle
	ErrInvitationExpired     = errors.New("invitation expired")
	ErrInvitationAlreadyUsed = errors.New("invitation already accepted")
	ErrInvitationCancelled   = errors.New("invitation cancelled")
	ErrCannotInviteAsOwner   = errors.New("cannot invite as Owner — use transfer ownership instead")
	ErrAlreadyMember         = errors.New("user is already a member of this organization")
	ErrAlreadyInvited        = errors.New("an invitation is already pending for this email")

	// Ownership transfer
	ErrTransferAlreadyPending  = errors.New("a transfer is already pending")
	ErrNoPendingTransfer       = errors.New("no transfer pending")
	ErrTransferExpired         = errors.New("transfer expired")
	ErrTransferTargetInvalid   = errors.New("transfer target must be an existing Admin of the organization")
	ErrCannotTransferToSelf    = errors.New("cannot transfer ownership to yourself")

	// Membership invariants (V1 single-Owner constraint)
	ErrLastOwnerCannotLeave  = errors.New("the Owner cannot leave the organization without transferring ownership first")
	ErrLastOwnerCannotDemote = errors.New("the Owner cannot be demoted without transferring ownership first")
	ErrOwnerCannotBeDemoted  = errors.New("only the Owner can step down from Owner — other members cannot demote them")
	ErrOwnerCannotBeRemoved  = errors.New("only the Owner can leave the organization — other members cannot remove them")
	ErrOwnerAlreadyExists    = errors.New("organization already has an Owner")

	// Authorization
	ErrForbidden         = errors.New("forbidden")
	ErrPermissionDenied  = errors.New("permission denied for this action")
	ErrNotAMember        = errors.New("not a member of this organization")
	// ErrCannotChangeOwnRole is returned when an actor attempts to
	// PATCH their own membership row to a new role. Self-edits go
	// through the leave / transfer flows instead. Mapped to 403 by
	// the team handler so the frontend can render a clean error.
	ErrCannotChangeOwnRole = errors.New("cannot change your own role — use leave or transfer ownership")
	// ErrCannotRemoveSelf is returned when an actor attempts to DELETE
	// their own membership row instead of using the leave flow.
	ErrCannotRemoveSelf = errors.New("cannot remove yourself — use leave organization")

	// Account type invariants
	ErrProviderCannotOwnOrg = errors.New("providers cannot own an organization")
	ErrOperatorHasNoOrg     = errors.New("operator must be a member of an organization")
)
