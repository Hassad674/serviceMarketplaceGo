package invoicing

import "fmt"

// Common French legal mentions used by all invoices regardless of
// regime. Centralised so the wording is consistent and so a future
// rephrase ripples to every invoice via a single edit.
const (
	mentionLatePenaltiesFR = "En cas de retard de paiement, application des pénalités au taux d'intérêt légal majoré de 10 points (article L441-10 du Code de commerce)."
	mentionRecoveryFeeFR   = "Indemnité forfaitaire pour frais de recouvrement : 40 €."
	mentionFranchiseFR     = "TVA non applicable, art. 293 B du CGI."
	mentionRcsExemptFR     = "Dispensé d'immatriculation au RCS."
	mentionReverseChargeFR = "Autoliquidation – art. 196 Directive 2006/112/CE."
	mentionOutOfScopeFR    = "Prestation hors champ TVA française – art. 259-1 du CGI."
)

// ResolveMentions returns the exact list of legal phrases that must
// appear on the rendered PDF for the given regime. The list is
// stored verbatim in `invoice.mentions_rendered` for audit so a
// later rewording does not change the historical record.
//
// The order matters — the PDF template renders the lines in the
// returned sequence. We surface the most regime-specific mention
// first, then the universal ones (penalties, recovery fee, RCS
// exemption when applicable).
//
// Issuer is required because EU reverse-charge mentions must echo
// both VAT numbers (issuer + recipient) on the document.
func ResolveMentions(regime TaxRegime, issuer IssuerInfo, recipient RecipientInfo) []string {
	out := make([]string, 0, 6)
	switch regime {
	case RegimeFRFranchiseBase:
		out = append(out, mentionFranchiseFR)
	case RegimeEUReverseCharge:
		out = append(out, mentionReverseChargeFR)
		// The issuer is in franchise so they collect no VAT — keep
		// the 293 B mention so the recipient knows the FR side does
		// not invoice VAT either.
		out = append(out, mentionFranchiseFR)
		if issuer.VATNumber != "" && recipient.VATNumber != "" {
			out = append(out, fmt.Sprintf(
				"N° TVA intracommunautaire émetteur : %s — destinataire : %s.",
				issuer.VATNumber, recipient.VATNumber,
			))
		}
	case RegimeOutOfScopeEU:
		out = append(out, mentionOutOfScopeFR)
	}

	// Universal mentions. L441-10 + indemnité 40€ apply to every B2B
	// French invoice; the RCS exemption is conditional on the issuer's
	// situation (auto-entrepreneur services).
	out = append(out, mentionLatePenaltiesFR, mentionRecoveryFeeFR)
	if issuer.RcsExempt {
		out = append(out, mentionRcsExemptFR)
	}
	return out
}
