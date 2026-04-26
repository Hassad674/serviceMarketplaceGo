package invoicing_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/invoicing"
)

func issuerFR() invoicing.IssuerInfo {
	return invoicing.IssuerInfo{
		LegalName:    "Hassad Smara",
		LegalForm:    "Entrepreneur individuel",
		SIRET:        "12345678900012",
		VATNumber:    "FR26878912963",
		AddressLine1: "12 rue de la République",
		PostalCode:   "75001",
		City:         "Paris",
		Country:      "FR",
		Email:        "contact@example.fr",
		RcsExempt:    true,
	}
}

func recipientFR() invoicing.RecipientInfo {
	return invoicing.RecipientInfo{
		LegalName:    "ACME France",
		TaxID:        "98765432100012",
		Country:      "FR",
		AddressLine1: "1 boulevard Haussmann",
		PostalCode:   "75009",
		City:         "Paris",
	}
}

func recipientDE() invoicing.RecipientInfo {
	return invoicing.RecipientInfo{
		LegalName: "ACME GmbH",
		VATNumber: "DE123456789",
		Country:   "DE",
	}
}

func recipientUS() invoicing.RecipientInfo {
	return invoicing.RecipientInfo{
		LegalName: "ACME Corp",
		Country:   "US",
	}
}

func TestResolveMentions_FRFranchise(t *testing.T) {
	got := invoicing.ResolveMentions(invoicing.RegimeFRFranchiseBase, issuerFR(), recipientFR())

	require := assert.New(t)
	require.Contains(strings.Join(got, "\n"), "TVA non applicable, art. 293 B")
	require.Contains(strings.Join(got, "\n"), "L441-10")
	require.Contains(strings.Join(got, "\n"), "40 €")
	require.Contains(strings.Join(got, "\n"), "Dispensé d'immatriculation au RCS")
	// Reverse charge mention MUST NOT appear domestic.
	require.NotContains(strings.Join(got, "\n"), "Autoliquidation")
	// Out-of-scope mention MUST NOT appear domestic.
	require.NotContains(strings.Join(got, "\n"), "art. 259-1")
}

func TestResolveMentions_EUReverseCharge(t *testing.T) {
	got := invoicing.ResolveMentions(invoicing.RegimeEUReverseCharge, issuerFR(), recipientDE())
	joined := strings.Join(got, "\n")

	require := assert.New(t)
	require.Contains(joined, "Autoliquidation")
	require.Contains(joined, "art. 196 Directive 2006/112/CE")
	// Issuer's franchise still applies.
	require.Contains(joined, "TVA non applicable, art. 293 B")
	// Both VAT numbers echoed for the reverse-charge mention.
	require.Contains(joined, "FR26878912963")
	require.Contains(joined, "DE123456789")
	require.Contains(joined, "L441-10")
	require.Contains(joined, "40 €")
}

func TestResolveMentions_EUReverseCharge_MissingVATSkipsEcho(t *testing.T) {
	// Edge case: regime is reverse charge but recipient VAT is empty
	// (defense-in-depth — completeness gate should have caught this).
	// The mention echoing both numbers must NOT render junk.
	r := recipientDE()
	r.VATNumber = ""
	got := invoicing.ResolveMentions(invoicing.RegimeEUReverseCharge, issuerFR(), r)
	joined := strings.Join(got, "\n")
	assert.Contains(t, joined, "Autoliquidation")
	assert.NotContains(t, joined, "destinataire :")
}

func TestResolveMentions_OutOfScopeEU(t *testing.T) {
	got := invoicing.ResolveMentions(invoicing.RegimeOutOfScopeEU, issuerFR(), recipientUS())
	joined := strings.Join(got, "\n")

	require := assert.New(t)
	require.Contains(joined, "art. 259-1 du CGI")
	require.Contains(joined, "Prestation hors champ")
	require.Contains(joined, "L441-10")
	require.Contains(joined, "40 €")
	// Reverse charge not applicable outside the EU.
	require.NotContains(joined, "Autoliquidation")
}

func TestResolveMentions_RCSExemptOff(t *testing.T) {
	issuer := issuerFR()
	issuer.RcsExempt = false
	got := invoicing.ResolveMentions(invoicing.RegimeFRFranchiseBase, issuer, recipientFR())
	joined := strings.Join(got, "\n")
	assert.NotContains(t, joined, "Dispensé d'immatriculation")
}

func TestResolveMentions_OrderIsStable(t *testing.T) {
	// The first mention is always regime-specific (293 B, autoliquidation,
	// or out-of-scope), then the universal pair, then the optional RCS.
	// Anchoring this in a test prevents accidental reordering during a
	// later refactor — the PDF template depends on the order.
	got := invoicing.ResolveMentions(invoicing.RegimeFRFranchiseBase, issuerFR(), recipientFR())
	require := assert.New(t)
	require.GreaterOrEqual(len(got), 4)
	require.True(strings.HasPrefix(got[0], "TVA non applicable"))
	require.Contains(got[1], "L441-10")
	require.Contains(got[2], "40 €")
	require.Contains(got[3], "RCS")
}
