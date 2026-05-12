package invoicing

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProfileType is whether the recipient is a real person or a
// registered legal entity. Mirrors `business_type` on the Stripe
// Account API and the matching CHECK constraint on `billing_profile`.
type ProfileType string

const (
	ProfileIndividual ProfileType = "individual"
	ProfileBusiness   ProfileType = "business"
)

// IsValid reports whether the value matches the DB CHECK constraint.
func (p ProfileType) IsValid() bool {
	return p == ProfileIndividual || p == ProfileBusiness
}

// BillingProfile is the recipient identity an organization keeps for
// invoicing. One row per organization. Pre-filled from Stripe KYC at
// first wallet/withdraw or subscribe touchpoint, then completed by the
// user via the settings page.
type BillingProfile struct {
	OrganizationID       uuid.UUID
	ProfileType          ProfileType
	LegalName            string
	TradingName          string
	LegalForm            string
	TaxID                string
	VATNumber            string
	VATValidatedAt       *time.Time
	VATValidationPayload []byte // raw JSONB from VIES; opaque to the domain
	AddressLine1         string
	AddressLine2         string
	PostalCode           string
	City                 string
	Country              string
	InvoicingEmail       string
	SyncedFromKYCAt      *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// MissingField describes a single completeness gap. The handler returns
// these to the frontend so the modal can render targeted prompts.
type MissingField struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// CheckCompleteness returns the list of fields the operator still has
// to fill before the profile can back an invoice. The ruleset depends
// on the recipient's country:
//
//   - Universal: legal name, address, country.
//   - FR: SIRET (14 digits) is mandatory.
//   - Other EU countries: validated VAT number is mandatory (the only
//     way reverse charge can apply).
//   - Outside the EU: no extra ID — the address + name is enough.
//
// invoicing_email is intentionally NOT required: the app layer defaults
// it to the org owner's account email on every read, so the gate never
// blocks on a field every account already has. The column is kept on
// the row for a future opt-in override.
//
// Returns an empty slice when the profile is good to go. The order of
// returned fields is deterministic so the frontend can render a stable
// list across renders.
func CheckCompleteness(p BillingProfile) []MissingField {
	missing := make([]MissingField, 0, 4)

	if strings.TrimSpace(p.LegalName) == "" {
		missing = append(missing, MissingField{Field: "legal_name", Reason: "legal name (or full name) is required on every invoice"})
	}
	if strings.TrimSpace(p.AddressLine1) == "" {
		missing = append(missing, MissingField{Field: "address_line1", Reason: "billing address is required"})
	}
	if strings.TrimSpace(p.PostalCode) == "" {
		missing = append(missing, MissingField{Field: "postal_code", Reason: "postal code is required"})
	}
	if strings.TrimSpace(p.City) == "" {
		missing = append(missing, MissingField{Field: "city", Reason: "city is required"})
	}
	if strings.TrimSpace(p.Country) == "" {
		missing = append(missing, MissingField{Field: "country", Reason: "country is required"})
	}

	country := strings.ToUpper(strings.TrimSpace(p.Country))
	switch {
	case country == "FR":
		// SIRET is 14 digits; we accept it as plain digits with
		// optional whitespace stripped at write time. Empty is the
		// failure mode we surface here.
		if strings.TrimSpace(p.TaxID) == "" {
			missing = append(missing, MissingField{Field: "tax_id", Reason: "SIRET (14 digits) is required for French entities"})
		}
	case IsEUCountry(country):
		// Reverse charge is only legitimate when VIES has confirmed
		// the recipient's VAT number. We require BOTH a number AND
		// a validated_at timestamp; the app layer is responsible for
		// clearing the timestamp when the VAT field changes.
		if strings.TrimSpace(p.VATNumber) == "" {
			missing = append(missing, MissingField{Field: "vat_number", Reason: "EU VAT number is required for reverse charge"})
		} else if p.VATValidatedAt == nil {
			missing = append(missing, MissingField{Field: "vat_number", Reason: "EU VAT number must be validated against VIES before invoicing"})
		}
	default:
		// Outside the EU: the recipient's address + legal name covers
		// our French invoicing obligation. Their own jurisdiction may
		// require more fields on their side — out of our scope.
	}

	return missing
}

// IsComplete is a convenience wrapper around CheckCompleteness used
// where only the boolean answer is needed.
func (p BillingProfile) IsComplete() bool {
	return len(CheckCompleteness(p)) == 0
}

// HasUniversalFields reports whether the profile carries the five fields
// strictly required to print ANY legal invoice — legal_name, country,
// address_line1, postal_code, city. Country-specific extras (FR SIRET,
// EU validated VAT number) are intentionally NOT checked here.
//
// Why a lighter gate than IsComplete: the per-milestone emission path
// runs at transfer-completion time, right after Stripe Connect has
// approved the provider's bank-payout capability. At that point KYC has
// definitely landed the address-level fields (Stripe collected them as
// part of the onboarding flow). The country-specific tax IDs may still
// be pending the user's manual confirmation on the "Mes infos de
// facturation" page — blocking the invoice on those would defeat the
// whole point of deferring (we want the invoice to fire as soon as the
// legal minimum is in place). The monthly safety-net scheduler is
// already responsible for catching truly-incomplete profiles via its
// own probe.
//
// Used by app/invoicing.IssueFromMilestone as defense-in-depth: even
// after the trigger moves from milestone.approved to transfer.completed,
// an edge case (transfer somehow succeeds but Stripe has not yet
// surfaced the address fields onto our local billing_profile row) will
// SKIP rather than crash the transfer path.
func (p BillingProfile) HasUniversalFields() bool {
	return strings.TrimSpace(p.LegalName) != "" &&
		strings.TrimSpace(p.Country) != "" &&
		strings.TrimSpace(p.AddressLine1) != "" &&
		strings.TrimSpace(p.PostalCode) != "" &&
		strings.TrimSpace(p.City) != ""
}
