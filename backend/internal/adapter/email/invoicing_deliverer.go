// Package email implements service.InvoiceDeliverer on top of the
// generic service.EmailService — V1 delivery is "send the recipient
// a link to the PDF stored in object storage". V2 will branch on
// regime (FR domestic B2B → PDP / Factur-X) but the contract stays
// the same: we accept the finalized invoice, the rendered bytes, and
// a download URL.
package email

import (
	"context"
	"fmt"
	"html"
	"strings"

	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/service"
)

// Deliverer is the V1 link-based invoice deliverer. It wraps
// service.EmailService (Resend in production) and renders a sober
// FR/EN body that links to the rendered PDF stored in R2.
type Deliverer struct {
	email service.EmailService
}

// NewDeliverer wires the deliverer onto an existing EmailService.
// Same rule as every adapter: ONE construction site (cmd/api/main.go),
// no global state, no package-level singletons.
func NewDeliverer(emailSvc service.EmailService) *Deliverer {
	return &Deliverer{email: emailSvc}
}

// DeliverInvoice sends a download link to the recipient's invoicing
// email. The pdfBytes parameter is accepted to satisfy the port
// contract but UNUSED in V1 — link-only delivery is the policy until
// Factur-X attachment-based delivery ships in V2.
func (d *Deliverer) DeliverInvoice(ctx context.Context, inv *invoicing.Invoice, pdfBytes []byte, downloadURL string) error {
	if inv == nil {
		return fmt.Errorf("email deliverer: invoice is nil")
	}
	to := strings.TrimSpace(inv.RecipientSnapshot.Email)
	if to == "" {
		return fmt.Errorf("email deliverer: recipient has no invoicing email (org=%s)", inv.RecipientOrganizationID)
	}
	_ = pdfBytes // V1 link-only — reserved for V2 attachment-based delivery.

	lang := pickLanguage(inv.RecipientSnapshot.Country)
	subject, body := renderInvoiceEmail(inv, downloadURL, lang)

	return d.email.SendNotification(ctx, to, subject, body)
}

// DeliverCreditNote is the avoir counterpart of DeliverInvoice — same
// link-only V1 strategy, different subject + intro paragraph.
func (d *Deliverer) DeliverCreditNote(ctx context.Context, cn *invoicing.CreditNote, pdfBytes []byte, downloadURL string) error {
	if cn == nil {
		return fmt.Errorf("email deliverer: credit note is nil")
	}
	to := strings.TrimSpace(cn.RecipientSnapshot.Email)
	if to == "" {
		return fmt.Errorf("email deliverer: recipient has no invoicing email (org=%s)", cn.RecipientOrganizationID)
	}
	_ = pdfBytes // V1 link-only.

	lang := pickLanguage(cn.RecipientSnapshot.Country)
	subject, body := renderCreditNoteEmail(cn, downloadURL, lang)

	return d.email.SendNotification(ctx, to, subject, body)
}

// pickLanguage chooses FR vs EN. Default is FR — the issuer is a
// French operator and the vast majority of recipients are domestic.
// Anything outside the francophone ISO codes falls through to EN.
func pickLanguage(countryCode string) string {
	switch strings.ToUpper(strings.TrimSpace(countryCode)) {
	case "FR", "BE", "LU", "MC", "CH":
		return "fr"
	case "":
		return "fr" // safe default
	default:
		return "en"
	}
}

// formatEUR renders a cents amount as "1 234,56 €" (FR) or
// "EUR 1,234.56" (EN). Kept inline to avoid a tiny package.
func formatEUR(cents int64, lang string) string {
	euros := cents / 100
	cs := cents % 100
	if cs < 0 {
		cs = -cs
	}
	if lang == "fr" {
		return fmt.Sprintf("%s,%02d €", groupThousands(euros, ' '), cs)
	}
	return fmt.Sprintf("EUR %s.%02d", groupThousands(euros, ','), cs)
}

// groupThousands formats an integer with a thousands separator. Avoids
// pulling in golang.org/x/text just for the locale-aware grouping.
func groupThousands(n int64, sep rune) string {
	if n < 0 {
		return "-" + groupThousands(-n, sep)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	pre := len(s) % 3
	if pre > 0 {
		out = append(out, s[:pre]...)
		if len(s) > pre {
			out = append(out, byte(sep))
		}
	}
	for i := pre; i < len(s); i += 3 {
		out = append(out, s[i:i+3]...)
		if i+3 < len(s) {
			out = append(out, byte(sep))
		}
	}
	return string(out)
}

// renderInvoiceEmail returns (subject, html body) for an invoice
// notification. The HTML is intentionally sober — single CTA, no
// marketing copy — to match the legal nature of the document.
func renderInvoiceEmail(inv *invoicing.Invoice, downloadURL, lang string) (string, string) {
	number := html.EscapeString(inv.Number)
	amount := formatEUR(inv.AmountInclTaxCents, lang)
	url := html.EscapeString(downloadURL)
	siret := html.EscapeString(inv.IssuerSnapshot.SIRET)
	issuer := html.EscapeString(inv.IssuerSnapshot.LegalName)

	if lang == "fr" {
		subject := fmt.Sprintf("Facture %s disponible", inv.Number)
		body := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
	<h2 style="color: #F43F5E; margin-bottom: 8px;">Votre facture est disponible</h2>
	<p>Bonjour,</p>
	<p>Votre facture <strong>%s</strong> d'un montant de <strong>%s</strong> est disponible.</p>
	<p style="margin: 24px 0;">
		<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: 600;">
			Télécharger la facture (PDF)
		</a>
	</p>
	<p style="color: #64748B; font-size: 14px;">Lien direct : <a href="%s" style="color: #F43F5E;">%s</a></p>
	<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
	<p style="color: #94A3B8; font-size: 12px;">Émis par %s — SIRET %s</p>
</div>`, number, amount, url, url, url, issuer, siret)
		return subject, body
	}

	subject := fmt.Sprintf("Invoice %s available", inv.Number)
	body := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
	<h2 style="color: #F43F5E; margin-bottom: 8px;">Your invoice is available</h2>
	<p>Hello,</p>
	<p>Your invoice <strong>%s</strong> for <strong>%s</strong> is now available.</p>
	<p style="margin: 24px 0;">
		<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: 600;">
			Download invoice (PDF)
		</a>
	</p>
	<p style="color: #64748B; font-size: 14px;">Direct link: <a href="%s" style="color: #F43F5E;">%s</a></p>
	<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
	<p style="color: #94A3B8; font-size: 12px;">Issued by %s — SIRET %s</p>
</div>`, number, amount, url, url, url, issuer, siret)
	return subject, body
}

// renderCreditNoteEmail returns (subject, html body) for an avoir.
func renderCreditNoteEmail(cn *invoicing.CreditNote, downloadURL, lang string) (string, string) {
	number := html.EscapeString(cn.Number)
	amount := formatEUR(cn.AmountInclTaxCents, lang)
	url := html.EscapeString(downloadURL)
	siret := html.EscapeString(cn.IssuerSnapshot.SIRET)
	issuer := html.EscapeString(cn.IssuerSnapshot.LegalName)

	if lang == "fr" {
		subject := fmt.Sprintf("Avoir %s disponible", cn.Number)
		body := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
	<h2 style="color: #F43F5E; margin-bottom: 8px;">Votre avoir est disponible</h2>
	<p>Bonjour,</p>
	<p>Votre avoir <strong>%s</strong> d'un montant de <strong>%s</strong> a été émis.</p>
	<p style="margin: 24px 0;">
		<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: 600;">
			Télécharger l'avoir (PDF)
		</a>
	</p>
	<p style="color: #64748B; font-size: 14px;">Lien direct : <a href="%s" style="color: #F43F5E;">%s</a></p>
	<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
	<p style="color: #94A3B8; font-size: 12px;">Émis par %s — SIRET %s</p>
</div>`, number, amount, url, url, url, issuer, siret)
		return subject, body
	}

	subject := fmt.Sprintf("Credit note %s available", cn.Number)
	body := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #0F172A;">
	<h2 style="color: #F43F5E; margin-bottom: 8px;">Your credit note is available</h2>
	<p>Hello,</p>
	<p>Your credit note <strong>%s</strong> for <strong>%s</strong> has been issued.</p>
	<p style="margin: 24px 0;">
		<a href="%s" style="display: inline-block; background-color: #F43F5E; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: 600;">
			Download credit note (PDF)
		</a>
	</p>
	<p style="color: #64748B; font-size: 14px;">Direct link: <a href="%s" style="color: #F43F5E;">%s</a></p>
	<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
	<p style="color: #94A3B8; font-size: 12px;">Issued by %s — SIRET %s</p>
</div>`, number, amount, url, url, url, issuer, siret)
	return subject, body
}
