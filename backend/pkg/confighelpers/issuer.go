// Package confighelpers groups small startup-time helpers that read
// env vars and produce typed config blobs. Lives under pkg/ rather
// than internal/ because it has no project-private dependencies and
// the parsers are reusable in any future Go entry point.
package confighelpers

import (
	"fmt"
	"os"
	"strings"

	"marketplace-backend/internal/domain/invoicing"
)

// LoadInvoiceIssuer parses the INVOICE_ISSUER_* env vars into the
// immutable issuer snapshot the invoicing app service uses. Validates
// the minimum legally-required fields up front (name, city, country,
// SIRET-shape) so the binary fails fast at boot rather than on the
// first invoice attempt.
//
// The env contract:
//
//	INVOICE_ISSUER_NAME           legal name (required)
//	INVOICE_ISSUER_LEGAL_FORM     SAS / SARL / EI ... (optional)
//	INVOICE_ISSUER_ADDRESS_LINE1  street (required)
//	INVOICE_ISSUER_ADDRESS_LINE2  optional
//	INVOICE_ISSUER_POSTAL_CODE    required
//	INVOICE_ISSUER_CITY           required
//	INVOICE_ISSUER_COUNTRY        ISO alpha-2 (required, e.g. FR)
//	INVOICE_ISSUER_SIRET          14 digits (required)
//	INVOICE_ISSUER_APE_CODE       optional
//	INVOICE_ISSUER_VAT_NUMBER     optional (empty for franchise alone)
//	INVOICE_ISSUER_EMAIL          required
//	INVOICE_ISSUER_PHONE          optional
//	INVOICE_ISSUER_IBAN           optional
//	INVOICE_ISSUER_RCS_EXEMPT     "true"/"false" (defaults to false)
func LoadInvoiceIssuer() (invoicing.IssuerInfo, error) {
	issuer := invoicing.IssuerInfo{
		LegalName:    strings.TrimSpace(os.Getenv("INVOICE_ISSUER_NAME")),
		LegalForm:    strings.TrimSpace(os.Getenv("INVOICE_ISSUER_LEGAL_FORM")),
		AddressLine1: strings.TrimSpace(os.Getenv("INVOICE_ISSUER_ADDRESS_LINE1")),
		AddressLine2: strings.TrimSpace(os.Getenv("INVOICE_ISSUER_ADDRESS_LINE2")),
		PostalCode:   strings.TrimSpace(os.Getenv("INVOICE_ISSUER_POSTAL_CODE")),
		City:         strings.TrimSpace(os.Getenv("INVOICE_ISSUER_CITY")),
		Country:      strings.ToUpper(strings.TrimSpace(os.Getenv("INVOICE_ISSUER_COUNTRY"))),
		SIRET:        stripDigits(os.Getenv("INVOICE_ISSUER_SIRET")),
		APECode:      strings.TrimSpace(os.Getenv("INVOICE_ISSUER_APE_CODE")),
		VATNumber:    strings.TrimSpace(os.Getenv("INVOICE_ISSUER_VAT_NUMBER")),
		Email:        strings.TrimSpace(os.Getenv("INVOICE_ISSUER_EMAIL")),
		Phone:        strings.TrimSpace(os.Getenv("INVOICE_ISSUER_PHONE")),
		IBAN:         strings.TrimSpace(os.Getenv("INVOICE_ISSUER_IBAN")),
		RcsExempt:    strings.EqualFold(strings.TrimSpace(os.Getenv("INVOICE_ISSUER_RCS_EXEMPT")), "true"),
	}

	if err := validateIssuer(issuer); err != nil {
		return invoicing.IssuerInfo{}, err
	}
	return issuer, nil
}

func validateIssuer(i invoicing.IssuerInfo) error {
	if i.LegalName == "" {
		return fmt.Errorf("INVOICE_ISSUER_NAME is required")
	}
	if i.AddressLine1 == "" {
		return fmt.Errorf("INVOICE_ISSUER_ADDRESS_LINE1 is required")
	}
	if i.PostalCode == "" {
		return fmt.Errorf("INVOICE_ISSUER_POSTAL_CODE is required")
	}
	if i.City == "" {
		return fmt.Errorf("INVOICE_ISSUER_CITY is required")
	}
	if len(i.Country) != 2 {
		return fmt.Errorf("INVOICE_ISSUER_COUNTRY must be a 2-letter ISO code (got %q)", i.Country)
	}
	if i.Email == "" {
		return fmt.Errorf("INVOICE_ISSUER_EMAIL is required")
	}
	if len(i.SIRET) != 14 {
		return fmt.Errorf("INVOICE_ISSUER_SIRET must be 14 digits (got %d after stripping non-digits)", len(i.SIRET))
	}
	for _, r := range i.SIRET {
		if r < '0' || r > '9' {
			return fmt.Errorf("INVOICE_ISSUER_SIRET must contain digits only")
		}
	}
	return nil
}

// stripDigits keeps only [0-9] so users can paste a SIRET with spaces
// or dashes ("123 456 789 00012") without breaking validation.
func stripDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
