package resend

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client *resend.Client
	from   string
}

func NewEmailService(apiKey string) *EmailService {
	client := resend.NewClient(apiKey)
	return &EmailService{
		client: client,
		from:   "Marketplace Service <noreply@marketplace-service.com>",
	}
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

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Réinitialisation de votre mot de passe — Marketplace Service",
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}
	return nil
}

func (s *EmailService) SendNotification(ctx context.Context, to, subject, html string) error {
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send notification email: %w", err)
	}
	return nil
}
