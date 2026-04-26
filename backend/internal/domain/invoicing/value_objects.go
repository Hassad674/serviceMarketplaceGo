package invoicing

import "strings"

// IssuerInfo is the snapshot of the marketplace's legal identity at
// the moment an invoice is issued. Comes from INVOICE_ISSUER_* env
// vars at boot. Stored verbatim in `invoice.issuer_snapshot` JSONB so
// past invoices retain the issuer state of their issuance day even
// after the operator's address or SIRET changes.
type IssuerInfo struct {
	LegalName    string `json:"legal_name"`
	LegalForm    string `json:"legal_form"`
	SIRET        string `json:"siret"`
	APECode      string `json:"ape_code"`
	VATNumber    string `json:"vat_number"`     // FR26878912963 in our case — empty allowed if franchise alone
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	PostalCode   string `json:"postal_code"`
	City         string `json:"city"`
	Country      string `json:"country"` // ISO alpha-2
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	IBAN         string `json:"iban"`
	RcsExempt    bool   `json:"rcs_exempt"`
}

// RecipientInfo is the snapshot of the receiving organization at the
// moment of issuance — pulled from billing_profile and frozen.
type RecipientInfo struct {
	OrganizationID string `json:"organization_id"`
	ProfileType    string `json:"profile_type"` // individual / business
	LegalName      string `json:"legal_name"`
	TradingName    string `json:"trading_name"`
	LegalForm      string `json:"legal_form"`
	TaxID          string `json:"tax_id"`    // SIRET for FR, tax id otherwise
	VATNumber      string `json:"vat_number"`
	AddressLine1   string `json:"address_line1"`
	AddressLine2   string `json:"address_line2"`
	PostalCode     string `json:"postal_code"`
	City           string `json:"city"`
	Country        string `json:"country"` // ISO alpha-2
	Email          string `json:"email"`
}

// SourceType discriminates between the two invoice origins.
type SourceType string

const (
	SourceSubscription       SourceType = "subscription"
	SourceMonthlyCommission  SourceType = "monthly_commission"
)

// IsValid reports whether the value matches the DB CHECK constraint.
func (s SourceType) IsValid() bool {
	return s == SourceSubscription || s == SourceMonthlyCommission
}

// Status mirrors the invoice/credit_note row status column.
type Status string

const (
	StatusDraft    Status = "draft"
	StatusIssued   Status = "issued"
	StatusCredited Status = "credited"
)

// IsValid reports whether the value matches the DB CHECK constraint.
func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusIssued, StatusCredited:
		return true
	}
	return false
}

// HasValidVAT reports whether the recipient carries a non-empty
// VAT number. The actual VIES validation happens at the app layer
// before the recipient snapshot is built — by the time we read this
// flag the value has already been verified.
func (r RecipientInfo) HasValidVAT() bool {
	return strings.TrimSpace(r.VATNumber) != ""
}
