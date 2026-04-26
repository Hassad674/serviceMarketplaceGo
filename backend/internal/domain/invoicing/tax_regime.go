package invoicing

import "strings"

// TaxRegime is the deterministic decision applied at invoice issuance.
// It drives both the legal mentions rendered on the PDF and the VAT
// rate (always 0 in V1 because the issuer is in franchise en base TVA,
// but the field exists for future evolution).
type TaxRegime string

const (
	// RegimeFRFranchiseBase — recipient is a French entity. Issuer is
	// FR auto-entrepreneur in franchise: no VAT charged, mention 293 B.
	RegimeFRFranchiseBase TaxRegime = "fr_franchise_base"

	// RegimeEUReverseCharge — recipient is in another EU country and
	// has a VIES-validated VAT number. Reverse charge applies (the
	// recipient self-assesses VAT in their country). The issuer's
	// franchise mention still applies because the issuer collects no
	// VAT regardless.
	RegimeEUReverseCharge TaxRegime = "eu_reverse_charge"

	// RegimeOutOfScopeEU — recipient is outside the EU. The supply is
	// outside the scope of French VAT under art. 259-1 CGI.
	RegimeOutOfScopeEU TaxRegime = "out_of_scope_eu"
)

// IsValid reports whether the value is one of the three allowed
// regimes. The DB CHECK constraint mirrors this enum.
func (r TaxRegime) IsValid() bool {
	switch r {
	case RegimeFRFranchiseBase, RegimeEUReverseCharge, RegimeOutOfScopeEU:
		return true
	}
	return false
}

// euCountries lists the 27 EU member states by ISO alpha-2 code.
// France is included so the comparison "recipient is EU" works against
// any EU recipient including France; the FR-vs-other-EU split happens
// in DetermineRegime.
//
// Kept as a fixed map literal rather than env-config: a country joining
// or leaving the EU is a treaty event that warrants a code change and
// release notes anyway.
var euCountries = map[string]struct{}{
	"AT": {}, "BE": {}, "BG": {}, "CY": {}, "CZ": {},
	"DE": {}, "DK": {}, "EE": {}, "ES": {}, "FI": {},
	"FR": {}, "GR": {}, "HR": {}, "HU": {}, "IE": {},
	"IT": {}, "LT": {}, "LU": {}, "LV": {}, "MT": {},
	"NL": {}, "PL": {}, "PT": {}, "RO": {}, "SE": {},
	"SI": {}, "SK": {},
}

// IsEUCountry reports whether the ISO alpha-2 country code is part of
// the EU. Case-insensitive.
func IsEUCountry(code string) bool {
	_, ok := euCountries[strings.ToUpper(code)]
	return ok
}

// DetermineRegime picks the tax regime for an invoice. The issuer is
// assumed to be FR (this marketplace's legal entity is a single French
// auto-entrepreneur in V1) — the function still takes issuerCountry as
// a parameter so future multi-entity setups don't need a redesign.
//
//   - Recipient FR (and issuer FR) → fr_franchise_base.
//   - Recipient EU non-FR with a validated VAT number → eu_reverse_charge.
//   - Recipient outside EU → out_of_scope_eu.
//
// EU recipient WITHOUT a validated VAT number is rejected upstream by
// CheckCompleteness — if it ever reaches here we conservatively fall
// back to fr_franchise_base, which is the safest mention (no autoliquidation
// claim that we cannot back up). The completeness gate is the actual
// guard; this is defense in depth.
func DetermineRegime(issuerCountry, recipientCountry string, recipientHasValidVAT bool) TaxRegime {
	rc := strings.ToUpper(strings.TrimSpace(recipientCountry))
	ic := strings.ToUpper(strings.TrimSpace(issuerCountry))
	if rc == "" {
		// Caller mistake — should be caught by validation. Default to
		// the strictest regime (treats it as domestic) so we never
		// claim a reverse-charge that wasn't authorised.
		return RegimeFRFranchiseBase
	}
	if rc == ic {
		return RegimeFRFranchiseBase
	}
	if IsEUCountry(rc) {
		if recipientHasValidVAT {
			return RegimeEUReverseCharge
		}
		return RegimeFRFranchiseBase
	}
	return RegimeOutOfScopeEU
}
