// Package invoicing models the marketplace's outbound billing — every
// invoice issued from the platform to a provider. Stripe is purely the
// payment processor; mentions, sequential numbering, immutability and
// retention all live here.
//
// The domain has zero infrastructure dependencies: no DB, no Stripe, no
// HTTP. Adapters consume domain values via ports and persist them.
package invoicing

import "errors"

// Sentinel errors. Wrapped with operation context by the app layer,
// matched by handlers via errors.Is — never stringified.
var (
	ErrInvalidOrganization        = errors.New("invoicing: organization id must be non-zero")
	ErrInvalidCurrency            = errors.New("invoicing: currency must be EUR")
	ErrInvalidAmount              = errors.New("invoicing: amount cents must be non-negative and totals must reconcile")
	ErrInvalidPeriod              = errors.New("invoicing: service period end must be on or after start")
	ErrInvalidTaxRegime           = errors.New("invoicing: unknown tax regime")
	ErrInvalidSourceType          = errors.New("invoicing: unknown source type")
	ErrInvalidNumber              = errors.New("invoicing: invoice number must match FAC-NNNNNN or AV-NNNNNN")
	ErrAlreadyFinalized           = errors.New("invoicing: invoice already finalized — emit a credit note to correct it")
	ErrEmptyItems                 = errors.New("invoicing: an invoice must have at least one item")
	ErrItemAmountMismatch         = errors.New("invoicing: item amount_cents must equal quantity * unit_price_cents")
	ErrCountryRequired            = errors.New("invoicing: recipient country is required to determine regime")
	ErrEUVATRequired              = errors.New("invoicing: EU B2B recipients must provide a validated VAT number")
	ErrNotFound                   = errors.New("invoicing: not found")
	ErrCounterScopeUnknown        = errors.New("invoicing: counter scope must be 'invoice' or 'credit_note'")
	ErrCreditNoteOriginalRequired = errors.New("invoicing: credit note must reference an existing invoice")
	ErrPDFKeyRequired             = errors.New("invoicing: cannot finalize without a stored PDF key")
)
