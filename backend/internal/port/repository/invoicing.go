package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
)

// InvoiceRepository persists invoices and credit notes alongside the
// atomic numbering counter. The interface is deliberately narrow: each
// method maps to a single database operation the app layer needs.
//
// The persistence guarantees the atomic counter / insert pair runs in
// a single transaction so a number is never burned on a row that did
// not commit. Callers must receive a finalized Invoice (number +
// pdf_r2_key set) in return — drafts are never persisted.
type InvoiceRepository interface {
	// CreateInvoice persists an invoice that is already finalized.
	// The implementation reserves the next number atomically (SELECT
	// FOR UPDATE on the counter row, increments, INSERTs invoice +
	// items, all in one transaction). The number returned by
	// reservation MUST match the invoice's Number field — callers
	// pass it through after building the row from the counter.
	CreateInvoice(ctx context.Context, inv *invoicing.Invoice) error

	// CreateCreditNote is the avoir equivalent of CreateInvoice, with
	// its own counter scope.
	CreateCreditNote(ctx context.Context, cn *invoicing.CreditNote) error

	// ReserveNumber draws the next number for the given scope inside a
	// caller-controlled transaction. Used when the caller wants to
	// build the PDF before INSERTing (the PDF needs the number on it).
	ReserveNumber(ctx context.Context, scope invoicing.CounterScope) (int64, error)

	// FindInvoiceByID returns a single invoice with all its items.
	FindInvoiceByID(ctx context.Context, id uuid.UUID) (*invoicing.Invoice, error)

	// FindInvoiceByStripeEventID is the lookup path used by webhook
	// handlers when checking dedup at the persistence level (defense
	// in depth on top of the Redis idempotency claim).
	FindInvoiceByStripeEventID(ctx context.Context, eventID string) (*invoicing.Invoice, error)

	// FindCreditNoteByStripeEventID is the avoir analogue.
	FindCreditNoteByStripeEventID(ctx context.Context, eventID string) (*invoicing.CreditNote, error)

	// FindInvoiceByStripePaymentIntentID is the lookup the refund webhook
	// uses to bridge a charge.refunded event back to the invoice we
	// originally issued for that subscription payment. Subscription
	// invoices store the PI on `invoice.stripe_payment_intent_id` (set at
	// finalization time when the parent invoice carries the default
	// payment).
	FindInvoiceByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*invoicing.Invoice, error)

	// MarkInvoiceCredited flips the invoice status to 'credited'. Called
	// after a credit note has been issued for the FULL outstanding
	// amount; partial refunds leave the original invoice status alone.
	MarkInvoiceCredited(ctx context.Context, invoiceID uuid.UUID) error

	// ListInvoicesByOrganization returns the org's invoices in the
	// "Mes factures" page. Cursor-based pagination per project rule.
	ListInvoicesByOrganization(ctx context.Context, organizationID uuid.UUID, cursor string, limit int) ([]*invoicing.Invoice, string, error)

	// HasInvoiceItemForPaymentRecord is the idempotency probe of the
	// monthly batch — every payment_record may be invoiced at most
	// once. Returns true when an invoice item already references the
	// given payment record.
	HasInvoiceItemForPaymentRecord(ctx context.Context, paymentRecordID uuid.UUID) (bool, error)

	// ListReleasedPaymentRecordsForOrg returns the released
	// payment_records belonging to the org during the given window
	// that have NOT yet been invoiced. Used both by the live current-
	// month view and by the monthly consolidation batch.
	ListReleasedPaymentRecordsForOrg(ctx context.Context, organizationID uuid.UUID, periodStart, periodEnd time.Time) ([]ReleasedPaymentRecord, error)

	// ListInvoicesAdmin returns a page of invoices+credit-notes across
	// every org. Filters that are zero-value act as "no filter".
	// Sorted by issued_at DESC, id DESC. Cursor opaque, limit capped
	// at 100. Used exclusively by the admin "Toutes les factures
	// emises" page — never expose this method to end users.
	ListInvoicesAdmin(ctx context.Context, filters AdminInvoiceFilters, cursor string, limit int) ([]*AdminInvoiceRow, string, error)

	// FindCreditNoteByID returns the credit-note row for the admin
	// PDF redirect. Returns invoicing.ErrNotFound when no row matches.
	FindCreditNoteByID(ctx context.Context, id uuid.UUID) (*invoicing.CreditNote, error)
}

// AdminInvoiceFilters are the optional filter axes the admin listing
// page supports. Every field is nil-safe / zero-value-safe — leaving a
// field empty means "do not filter on this axis".
type AdminInvoiceFilters struct {
	// RecipientOrgID, when non-nil, restricts results to invoices and
	// credit notes addressed to a single recipient organization.
	RecipientOrgID *uuid.UUID

	// Status accepts one of: "subscription", "monthly_commission",
	// "credit_note", or empty for no restriction. The first two map to
	// the invoice.source_type column; "credit_note" filters out
	// invoices entirely (only credit_note rows survive).
	Status string

	// DateFrom / DateTo are inclusive bounds on issued_at. Either may
	// be nil to leave that side of the range unbounded.
	DateFrom *time.Time
	DateTo   *time.Time

	// MinAmountCents / MaxAmountCents are inclusive bounds on
	// amount_incl_tax_cents. Either may be nil to skip that bound.
	MinAmountCents *int64
	MaxAmountCents *int64

	// Search runs a case-insensitive ILIKE against the invoice/credit-note
	// number AND the recipient_snapshot legal_name. Empty disables the
	// filter entirely.
	Search string
}

// AdminInvoiceRow is the slim projection used by the admin listing
// page. It deliberately collapses invoices and credit notes onto a
// single shape so the UI can render a unified table without juggling
// two row types.
type AdminInvoiceRow struct {
	ID                 uuid.UUID
	Number             string
	IsCreditNote       bool
	RecipientOrgID     uuid.UUID
	RecipientLegalName string
	IssuedAt           time.Time
	AmountInclTaxCents int64
	Currency           string
	TaxRegime          string
	// Status carries the invoice.status column for invoice rows
	// ("issued", "credited", "draft") and the literal "credit_note"
	// for credit-note rows so the UI can render a single badge column.
	Status string
	PDFR2Key string
	// OriginalInvoiceID is set on credit-note rows only — points back
	// to the invoice the avoir corrects. nil for invoice rows.
	OriginalInvoiceID *uuid.UUID
	// SourceType carries the invoice.source_type column for invoice
	// rows ("subscription", "monthly_commission"). Empty on credit
	// notes.
	SourceType string
}

// ReleasedPaymentRecord is the slim projection of payment_records the
// invoicing layer cares about. Stored alongside the repo so consumers
// don't have to depend on the payment domain's types.
type ReleasedPaymentRecord struct {
	ID                  uuid.UUID
	MilestoneID         uuid.UUID
	ProposalID          uuid.UUID
	ProposalAmountCents int64
	PlatformFeeCents    int64
	Currency            string
	TransferredAt       time.Time
}
