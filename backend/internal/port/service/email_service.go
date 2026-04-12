package service

import (
	"context"
	"time"
)

// TeamInvitationEmailInput groups the fields needed to render a team
// invitation email. Using a struct keeps the SendTeamInvitation method
// signature manageable as we add optional fields like a custom message.
type TeamInvitationEmailInput struct {
	To               string    // recipient email address
	OrgName          string    // organization display name (e.g. "Acme Corp")
	OrgType          string    // "agency" | "enterprise"
	InviterName      string    // full name of the Owner/Admin who sent the invitation
	InviteeFirstName string    // invited user's first name, used in the greeting
	Role             string    // assigned role: admin | member | viewer
	AcceptURL        string    // {FRONTEND_URL}/invitation/{token}
	ExpiresAt        time.Time // when the invitation ceases to be valid
}

// RolePermissionsChangedEmailInput bundles the fields rendered in the
// "your organization's role permissions were just edited" notice sent
// to the Owner after a successful save. The email is anti-tampering:
// it reaches the real Owner's inbox so a session takeover attempt
// surfaces even when the attacker suppresses UI-side warnings.
type RolePermissionsChangedEmailInput struct {
	To              string    // Owner's email address
	OwnerFirstName  string    // for the greeting; optional
	OrgName         string    // the organization whose matrix was edited
	Role            string    // "admin" | "member" | "viewer"
	GrantedLabels   []string  // permissions freshly granted (human labels)
	RevokedLabels   []string  // permissions freshly revoked (human labels)
	AffectedMembers int       // how many team members are affected by this role's edit
	ChangedAt       time.Time // when the save landed on the server
}

type EmailService interface {
	SendPasswordReset(ctx context.Context, to string, resetURL string) error
	SendNotification(ctx context.Context, to, subject, html string) error
	SendTeamInvitation(ctx context.Context, input TeamInvitationEmailInput) error

	// SendRolePermissionsChanged notifies the Owner that the role
	// permissions matrix of their organization was just edited. The
	// email lists the role that changed, the granted / revoked
	// permissions, and the number of affected members. Sent
	// best-effort after every successful save — delivery failures
	// MUST NOT break the main API call.
	SendRolePermissionsChanged(ctx context.Context, input RolePermissionsChangedEmailInput) error
}
