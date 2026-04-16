package referral

import (
	"context"

	"github.com/google/uuid"
)

// OrgMemberResolver returns the list of user ids that should be notified
// together whenever the referral feature needs to reach the "party" a user
// represents. For a freelancer (solo org) the slice is just [userID]. For an
// agency or enterprise, it expands to every member of the user's organization
// so colleagues see the same incoming intro / commission event.
//
// Defined as a port so the referral feature stays decoupled from the team /
// organization feature. The wiring in cmd/api/main.go injects a thin adapter
// that reads from the organization + organization_members repositories.
//
// Implementations MUST always include the input userID in the returned slice
// (even when the user has no org membership). A nil or empty result is
// treated as a soft failure and the caller falls back to a single-recipient
// notification rather than dropping the event entirely.
type OrgMemberResolver interface {
	ResolveMemberUserIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
