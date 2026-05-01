package gdpr

import (
	"fmt"
	"time"
)

// confirmEmailParams bundles the variables substituted into the
// deletion-confirmation email body. Kept private — the only caller
// is dispatchConfirmationEmail.
type confirmEmailParams struct {
	FirstName  string
	ConfirmURL string
	ExpiresAt  time.Time
}

// renderConfirmationEmail produces the (subject, html) pair for the
// deletion confirmation email. Locale picks the language: "fr" by
// default, "en" when the user's preferred language is English.
//
// Both versions use inline styles only — Resend, Outlook, and
// Apple Mail render <head>-side <style> erratically, so we paint
// every CSS rule on the element directly.
func renderConfirmationEmail(locale string, p confirmEmailParams) (string, string) {
	if locale == "en" {
		return renderConfirmationEmailEN(p)
	}
	return renderConfirmationEmailFR(p)
}

func renderConfirmationEmailFR(p confirmEmailParams) (string, string) {
	greeting := "Bonjour"
	if p.FirstName != "" {
		greeting = "Bonjour " + p.FirstName
	}
	expires := p.ExpiresAt.Format("02/01/2006 à 15h04 UTC")
	subject := "Confirmez la suppression de votre compte — Marketplace Service"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
			<h2 style="color: #F43F5E; margin-bottom: 8px;">Suppression de compte</h2>
			<p style="color: #64748B; margin-top: 0; font-size: 14px;">Marketplace Service</p>
			<p>%s,</p>
			<p>Vous avez demandé la suppression de votre compte Marketplace Service.
			Cliquez sur le bouton ci-dessous pour confirmer cette demande.</p>
			<p>Une fois confirmée, votre compte sera <strong>verrouillé</strong> et
			toutes vos données seront <strong>définitivement supprimées dans 30 jours</strong>.
			Vous pouvez annuler à tout moment pendant ce délai.</p>
			<p style="margin: 24px 0;">
				<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px;">
					Confirmer la suppression
				</a>
			</p>
			<p style="color: #64748B; font-size: 14px;">Ce lien expire le %s.</p>
			<p style="color: #64748B; font-size: 14px;">Si vous n'avez pas demandé cette suppression, ignorez cet email — votre compte reste actif.</p>
		</div>
	`, greeting, p.ConfirmURL, expires)
	return subject, html
}

func renderConfirmationEmailEN(p confirmEmailParams) (string, string) {
	greeting := "Hello"
	if p.FirstName != "" {
		greeting = "Hello " + p.FirstName
	}
	expires := p.ExpiresAt.Format("Jan 2, 2006 at 15:04 UTC")
	subject := "Confirm your account deletion — Marketplace Service"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
			<h2 style="color: #F43F5E; margin-bottom: 8px;">Account deletion</h2>
			<p style="color: #64748B; margin-top: 0; font-size: 14px;">Marketplace Service</p>
			<p>%s,</p>
			<p>You requested deletion of your Marketplace Service account.
			Click the button below to confirm.</p>
			<p>Once confirmed, your account will be <strong>locked</strong> and
			all your data will be <strong>permanently deleted in 30 days</strong>.
			You can cancel at any time during this window.</p>
			<p style="margin: 24px 0;">
				<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px;">
					Confirm deletion
				</a>
			</p>
			<p style="color: #64748B; font-size: 14px;">This link expires on %s.</p>
			<p style="color: #64748B; font-size: 14px;">If you did not request this deletion, please ignore this email — your account remains active.</p>
		</div>
	`, greeting, p.ConfirmURL, expires)
	return subject, html
}

// reminderEmailParams bundles the variables for the T+25j reminder
// + the T+30j final purge notice. The same template renders both
// because they only differ by tone, not by structure.
type reminderEmailParams struct {
	FirstName    string
	HardDeleteAt time.Time
	CancelURL    string
}

// renderReminderEmail returns the T+25j reminder (locale-aware).
func renderReminderEmail(locale string, p reminderEmailParams) (string, string) {
	if locale == "en" {
		return renderReminderEmailEN(p)
	}
	return renderReminderEmailFR(p)
}

func renderReminderEmailFR(p reminderEmailParams) (string, string) {
	greeting := "Bonjour"
	if p.FirstName != "" {
		greeting = "Bonjour " + p.FirstName
	}
	when := p.HardDeleteAt.Format("02/01/2006")
	subject := "Rappel : suppression de votre compte dans 5 jours"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
			<h2 style="color: #F43F5E;">Suppression imminente</h2>
			<p>%s,</p>
			<p>Pour rappel, votre compte Marketplace Service sera <strong>définitivement supprimé le %s</strong>.</p>
			<p>Si vous avez changé d'avis, vous pouvez annuler la suppression :</p>
			<p style="margin: 24px 0;">
				<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px;">
					Annuler la suppression
				</a>
			</p>
		</div>
	`, greeting, when, p.CancelURL)
	return subject, html
}

func renderReminderEmailEN(p reminderEmailParams) (string, string) {
	greeting := "Hello"
	if p.FirstName != "" {
		greeting = "Hello " + p.FirstName
	}
	when := p.HardDeleteAt.Format("Jan 2, 2006")
	subject := "Reminder: your account will be deleted in 5 days"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
			<h2 style="color: #F43F5E;">Upcoming deletion</h2>
			<p>%s,</p>
			<p>This is a reminder that your Marketplace Service account will be <strong>permanently deleted on %s</strong>.</p>
			<p>If you changed your mind, you can cancel the deletion:</p>
			<p style="margin: 24px 0;">
				<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px;">
					Cancel deletion
				</a>
			</p>
		</div>
	`, greeting, when, p.CancelURL)
	return subject, html
}
