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

type EmailService interface {
	SendPasswordReset(ctx context.Context, to string, resetURL string) error
	SendNotification(ctx context.Context, to, subject, html string) error
	SendTeamInvitation(ctx context.Context, input TeamInvitationEmailInput) error
}
