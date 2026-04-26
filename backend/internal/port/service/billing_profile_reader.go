package service

import (
	"context"

	"github.com/google/uuid"
)

// BillingProfileSnapshotReader is the narrow port the subscription app
// uses to enrich the Stripe Customer with the org's billing address +
// legal name BEFORE creating an Embedded Checkout session. Kept separate
// from the broader invoicing service so consumers depend only on the
// single read method they actually need (Interface Segregation).
//
// Production adapter is the invoicing app service. When the invoicing
// module is disabled the dependency is left nil and the subscription
// service falls back to creating a session without enrichment — Stripe
// then either uses what it already has on the Customer or shows an empty
// address form (the user has nothing to fill though, since billing
// collection is disabled at session level).
type BillingProfileSnapshotReader interface {
	// GetBillingProfileSnapshotForStripe returns the slim projection of
	// the org's billing profile that maps onto Stripe Customer fields.
	// Returns a zero-value snapshot (all empty strings) when the org has
	// no profile yet — the caller decides whether to skip enrichment in
	// that case.
	GetBillingProfileSnapshotForStripe(ctx context.Context, organizationID uuid.UUID) (BillingProfileStripeSnapshot, error)
}

// BillingProfileStripeSnapshot is the subset of the org's billing profile
// the subscription service pushes onto the Stripe Customer record. Kept
// transport-agnostic so the subscription package never imports the
// invoicing domain.
type BillingProfileStripeSnapshot struct {
	LegalName      string
	AddressLine1   string
	AddressLine2   string
	PostalCode     string
	City           string
	Country        string // ISO alpha-2
	InvoicingEmail string
	// VATNumber is the intracom VAT id (e.g. FR12345678901). Stored on
	// the Stripe customer's metadata for traceability — Stripe Customer
	// API doesn't accept tax_id_data on Update, so we don't try.
	VATNumber string
}

// IsEmpty reports whether the snapshot carries any usable data. Callers
// use this to skip Stripe Customer.Update when there is nothing to push
// (e.g. provider with no billing profile yet — the form will be saved
// in step 1 of the embedded modal anyway).
func (s BillingProfileStripeSnapshot) IsEmpty() bool {
	return s.LegalName == "" &&
		s.AddressLine1 == "" &&
		s.PostalCode == "" &&
		s.City == "" &&
		s.Country == "" &&
		s.InvoicingEmail == ""
}
