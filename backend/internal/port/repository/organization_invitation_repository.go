package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
)

// ListInvitationsParams groups the filters for listing invitations of an
// org. Grouped to stay under the 4-parameter limit on repository methods.
type ListInvitationsParams struct {
	OrganizationID uuid.UUID
	// StatusFilter — when non-empty, only invitations in that status are
	// returned. The pending/active list in the UI uses "pending".
	StatusFilter organization.InvitationStatus
	Cursor       string
	Limit        int
}

// OrganizationInvitationRepository persists pending invitations for new
// operators to join an organization.
type OrganizationInvitationRepository interface {
	// Create inserts a new invitation. Fails with a unique-constraint
	// violation if another pending invitation already targets the same
	// email in the same org (partial UNIQUE on (org_id, lower(email))
	// WHERE status='pending').
	Create(ctx context.Context, inv *organization.Invitation) error

	// FindByID returns the invitation with the given id.
	FindByID(ctx context.Context, id uuid.UUID) (*organization.Invitation, error)

	// FindByToken looks up an invitation by its secret token. The public
	// /invitation/{token} endpoint routes through this method.
	FindByToken(ctx context.Context, token string) (*organization.Invitation, error)

	// FindPendingByOrgAndEmail returns the pending invitation targeting
	// the given email in the org, if any. Used for the pre-send
	// "already invited" check.
	FindPendingByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*organization.Invitation, error)

	// List returns cursor-paginated invitations for an org, ordered by
	// created_at DESC.
	List(ctx context.Context, params ListInvitationsParams) ([]*organization.Invitation, string, error)

	// Update persists changes to the invitation (status transitions,
	// token regeneration on resend).
	Update(ctx context.Context, inv *organization.Invitation) error

	// Delete removes an invitation. Rarely called — cancellation uses
	// Update with Status=cancelled to preserve the audit trail.
	Delete(ctx context.Context, id uuid.UUID) error

	// ExpireStale marks pending invitations with expires_at < now as
	// expired in a single UPDATE. Called by a background sweeper, returns
	// the number of rows affected for observability.
	ExpireStale(ctx context.Context) (int, error)
}
