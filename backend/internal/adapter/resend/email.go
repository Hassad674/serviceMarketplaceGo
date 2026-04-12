package resend

import (
	"context"
	"fmt"
	"time"

	"github.com/resend/resend-go/v2"

	"marketplace-backend/internal/port/service"
)

type EmailService struct {
	client *resend.Client
	from   string

	// devRedirectEmail, when non-empty, overrides every outgoing
	// recipient address with this value and prefixes the subject with
	// [DEV → originalRecipient]. Used in dev/staging when Resend is in
	// sandbox mode and can only deliver to the account owner's mailbox.
	// Leave empty in production.
	devRedirectEmail string
}

// NewEmailService builds the Resend adapter. The devRedirectEmail
// argument is optional — pass "" in production. In dev, set it via the
// RESEND_DEV_REDIRECT_EMAIL env var (typically the developer's own
// email) so sandbox mode doesn't drop invitation / password-reset
// emails silently.
func NewEmailService(apiKey, devRedirectEmail string) *EmailService {
	client := resend.NewClient(apiKey)
	return &EmailService{
		client:           client,
		from:             "Marketplace Service <onboarding@resend.dev>",
		devRedirectEmail: devRedirectEmail,
	}
}

// applyDevRedirect rewrites the recipient and subject when the adapter
// is in dev-redirect mode, so every outgoing message is still visible
// to the developer and the original "To" stays auditable.
func (s *EmailService) applyDevRedirect(to, subject string) (string, string) {
	if s.devRedirectEmail == "" {
		return to, subject
	}
	return s.devRedirectEmail, fmt.Sprintf("[DEV → %s] %s", to, subject)
}

func (s *EmailService) SendPasswordReset(ctx context.Context, to string, resetURL string) error {
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #F43F5E;">Réinitialisation de mot de passe</h2>
			<p>Vous avez demandé la réinitialisation de votre mot de passe.</p>
			<p>Cliquez sur le bouton ci-dessous pour choisir un nouveau mot de passe :</p>
			<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; margin: 16px 0;">
				Réinitialiser mon mot de passe
			</a>
			<p style="color: #64748B; font-size: 14px;">Ce lien expire dans 1 heure.</p>
			<p style="color: #64748B; font-size: 14px;">Si vous n'avez pas demandé cette réinitialisation, ignorez cet email.</p>
		</div>
	`, resetURL)

	recipient, subject := s.applyDevRedirect(to, "Réinitialisation de votre mot de passe — Marketplace Service")
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{recipient},
		Subject: subject,
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}
	return nil
}

func (s *EmailService) SendNotification(ctx context.Context, to, subject, html string) error {
	recipient, finalSubject := s.applyDevRedirect(to, subject)
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{recipient},
		Subject: finalSubject,
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send notification email: %w", err)
	}
	return nil
}

// SendTeamInvitation sends an invitation email to a new team operator.
// The template is French, uses the marketplace rose color, and lists the
// organization name, the inviter's name, the role being offered, and a
// clear CTA pointing at the acceptance URL.
func (s *EmailService) SendTeamInvitation(ctx context.Context, input service.TeamInvitationEmailInput) error {
	roleLabel := roleLabelFR(input.Role)
	typeLabel := orgTypeLabelFR(input.OrgType)
	inviterDisplay := input.InviterName
	if inviterDisplay == "" {
		inviterDisplay = "Un Owner"
	}

	greeting := input.InviteeFirstName
	if greeting == "" {
		greeting = "bonjour"
	} else {
		greeting = "Bonjour " + greeting
	}

	expiresDisplay := "7 jours"
	if !input.ExpiresAt.IsZero() {
		expiresDisplay = input.ExpiresAt.In(time.Local).Format("02/01/2006 à 15h04")
	}

	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
			<h2 style="color: #F43F5E; margin-bottom: 8px;">Invitation à rejoindre une équipe</h2>
			<p style="color: #64748B; margin-top: 0; font-size: 14px;">Marketplace Service</p>

			<p>%s,</p>
			<p>
				<strong>%s</strong> vous invite à rejoindre l'%s
				<strong>%s</strong> en tant que <strong>%s</strong>.
			</p>

			<p>
				En acceptant, vous créerez votre propre compte pour accéder au tableau
				de bord de l'organisation et collaborer avec l'équipe.
			</p>

			<p style="margin: 24px 0;">
				<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: 600;">
					Accepter l'invitation
				</a>
			</p>

			<p style="color: #64748B; font-size: 14px;">
				Cette invitation expire le <strong>%s</strong>. Passé ce délai, vous devrez demander une nouvelle invitation à l'organisation.
			</p>

			<p style="color: #64748B; font-size: 14px; margin-top: 24px;">
				Si vous ne reconnaissez pas cette invitation, vous pouvez simplement ignorer cet email.
			</p>

			<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
			<p style="color: #94A3B8; font-size: 12px;">
				Lien direct : <a href="%s" style="color: #F43F5E;">%s</a>
			</p>
		</div>
	`,
		greeting,
		inviterDisplay,
		typeLabel,
		input.OrgName,
		roleLabel,
		input.AcceptURL,
		expiresDisplay,
		input.AcceptURL,
		input.AcceptURL,
	)

	subject := fmt.Sprintf("Invitation à rejoindre %s — Marketplace Service", input.OrgName)
	recipient, finalSubject := s.applyDevRedirect(input.To, subject)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{recipient},
		Subject: finalSubject,
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send team invitation email: %w", err)
	}
	return nil
}

// SendRolePermissionsChanged sends the anti-tampering notice to the
// Owner after a role permissions save. Template lists the role that
// changed, the granted/revoked permission labels, and the number of
// affected team members so the Owner can immediately spot an
// unauthorized edit.
func (s *EmailService) SendRolePermissionsChanged(ctx context.Context, input service.RolePermissionsChangedEmailInput) error {
	greeting := input.OwnerFirstName
	if greeting == "" {
		greeting = "Bonjour"
	} else {
		greeting = "Bonjour " + greeting
	}

	changedAt := input.ChangedAt
	if changedAt.IsZero() {
		changedAt = time.Now()
	}
	when := changedAt.In(time.Local).Format("02/01/2006 à 15h04")

	grantedHTML := renderPermList(input.GrantedLabels, "#10B981", "Aucune")
	revokedHTML := renderPermList(input.RevokedLabels, "#EF4444", "Aucune")

	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
			<h2 style="color: #F43F5E; margin-bottom: 8px;">Permissions de rôle mises à jour</h2>
			<p style="color: #64748B; margin-top: 0; font-size: 14px;">Marketplace Service — %s</p>

			<p>%s,</p>
			<p>
				Les permissions du rôle <strong>%s</strong> de votre organisation
				<strong>%s</strong> viennent d'être modifiées.
			</p>

			<div style="margin: 16px 0; padding: 16px; background: #F8FAFC; border-radius: 8px; border: 1px solid #E2E8F0;">
				<p style="margin: 0 0 8px 0; font-weight: 600;">Permissions accordées</p>
				%s
				<p style="margin: 16px 0 8px 0; font-weight: 600;">Permissions révoquées</p>
				%s
			</div>

			<p style="color: #64748B; font-size: 14px;">
				<strong>%d</strong> membre(s) sont concerné(s). Leur session a été
				invalidée : ils seront déconnectés et devront se reconnecter pour
				que les nouvelles permissions prennent effet.
			</p>

			<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
			<p style="color: #94A3B8; font-size: 12px;">
				Si vous n'êtes PAS à l'origine de cette modification, votre compte
				est peut-être compromis. Changez votre mot de passe immédiatement
				et contactez le support.
			</p>
		</div>
	`,
		when,
		greeting,
		roleLabelFR(input.Role),
		input.OrgName,
		grantedHTML,
		revokedHTML,
		input.AffectedMembers,
	)

	subject := fmt.Sprintf("Permissions du rôle %s modifiées — %s", roleLabelFR(input.Role), input.OrgName)
	recipient, finalSubject := s.applyDevRedirect(input.To, subject)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{recipient},
		Subject: finalSubject,
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send role permissions changed email: %w", err)
	}
	return nil
}

// renderPermList renders a permission list as an HTML <ul>, or a
// fallback line when empty. Used by the role-permissions email.
func renderPermList(labels []string, color, emptyLabel string) string {
	if len(labels) == 0 {
		return fmt.Sprintf(`<p style="margin: 0; color: #94A3B8; font-size: 14px;">%s</p>`, emptyLabel)
	}
	items := ""
	for _, label := range labels {
		items += fmt.Sprintf(`<li style="color: %s; margin: 4px 0;">%s</li>`, color, label)
	}
	return fmt.Sprintf(`<ul style="margin: 0; padding-left: 20px;">%s</ul>`, items)
}

// roleLabelFR returns the French human label for an organization role.
func roleLabelFR(role string) string {
	switch role {
	case "admin":
		return "Admin"
	case "member":
		return "Membre"
	case "viewer":
		return "Viewer (lecture seule)"
	default:
		return role
	}
}

// orgTypeLabelFR returns the French article + label for an org type.
func orgTypeLabelFR(orgType string) string {
	switch orgType {
	case "agency":
		return "agence"
	case "enterprise":
		return "entreprise"
	default:
		return orgType
	}
}
