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
