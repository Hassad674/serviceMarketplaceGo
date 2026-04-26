package service

import "context"

// StripeKYCSnapshotReader is the narrow port the invoicing app uses to
// pre-fill a billing profile from a Stripe Connect account. Kept
// separate from the broader StripeService interface so consumers (and
// tests) only depend on the single read method they actually need —
// adheres to the Interface Segregation principle.
//
// The production adapter is implemented by the existing Stripe service
// in internal/adapter/stripe; testing wires a struct stub.
type StripeKYCSnapshotReader interface {
	// GetAccountKYCSnapshot fetches the Stripe Account and projects it
	// into the slim, transport-agnostic snapshot the app layer reads
	// when filling the billing profile.
	GetAccountKYCSnapshot(ctx context.Context, accountID string) (*StripeAccountKYCSnapshot, error)
}

// StripeAccountKYCSnapshot is the slim projection of the Stripe Account
// API the invoicing layer reads when pre-filling a billing profile from
// KYC. Every field is optional — a fresh, half-onboarded account may
// expose only a subset, and the caller's merge rule is "fill empty
// fields only, never overwrite". BusinessType drives the initial choice
// of profile_type ("individual" vs "business") on the billing profile.
type StripeAccountKYCSnapshot struct {
	BusinessType string // "individual" | "company" | ""
	LegalName    string // company.name OR individual first+last
	Country      string // 2-letter ISO
	AddressLine1 string
	AddressLine2 string
	City         string
	PostalCode   string
	TaxID        string // SIRET on FR company accounts (best-effort)
	SupportEmail string // business_profile.support_email
}
