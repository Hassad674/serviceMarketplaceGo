package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/invoicing"
	repo "marketplace-backend/internal/port/repository"
)

// scanInvoiceFromRowQuery scans an invoice row produced by
// runner.QueryRowContext (where runner is sqlQuerier). It mirrors
// scanInvoice but takes the interface form so both *sql.DB and *sql.Tx
// rows scan through the same shared helper.
func scanInvoiceFromRowQuery(row *sql.Row) (*invoicing.Invoice, error) {
	return scanInvoiceFrom(row)
}

// loadItemsViaRunner is the runner-aware variant of (r *InvoiceRepository).loadItems
// — the closure-passed sqlQuerier may be either *sql.DB or *sql.Tx, so
// the caller can keep the items SELECT inside the same tenant tx as
// the parent SELECT.
func loadItemsViaRunner(ctx context.Context, runner sqlQuerier, invoiceID uuid.UUID) ([]invoicing.InvoiceItem, error) {
	rows, err := runner.QueryContext(ctx, `
		SELECT id, invoice_id, description, quantity, unit_price_cents, amount_cents,
		       milestone_id, payment_record_id, created_at
		FROM invoice_item
		WHERE invoice_id = $1
		ORDER BY created_at ASC, id ASC`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]invoicing.InvoiceItem, 0)
	for rows.Next() {
		var (
			it              invoicing.InvoiceItem
			milestoneID     uuid.NullUUID
			paymentRecordID uuid.NullUUID
		)
		if err := rows.Scan(
			&it.ID, &it.InvoiceID, &it.Description, &it.Quantity,
			&it.UnitPriceCents, &it.AmountCents,
			&milestoneID, &paymentRecordID, &it.CreatedAt,
		); err != nil {
			return nil, err
		}
		if milestoneID.Valid {
			id := milestoneID.UUID
			it.MilestoneID = &id
		}
		if paymentRecordID.Valid {
			id := paymentRecordID.UUID
			it.PaymentRecordID = &id
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// HasInvoiceItemForPaymentRecord is the idempotency probe used by the
// monthly batch — every payment_record may be invoiced at most once.
func (r *InvoiceRepository) HasInvoiceItemForPaymentRecord(ctx context.Context, paymentRecordID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM invoice_item WHERE payment_record_id = $1
		)`, paymentRecordID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has invoice item for payment record: %w", err)
	}
	return exists, nil
}

// ListReleasedPaymentRecordsForOrg returns the released payment_records
// owned by the given organization in [periodStart, periodEnd) that have
// NOT yet been invoiced. The org match flows from
// payment_records.provider_id → users.id → users.organization_id.
func (r *InvoiceRepository) ListReleasedPaymentRecordsForOrg(ctx context.Context, organizationID uuid.UUID, periodStart, periodEnd time.Time) ([]repo.ReleasedPaymentRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT pr.id, pr.milestone_id, pr.proposal_id,
		       pr.proposal_amount, pr.platform_fee_amount,
		       pr.currency, pr.transferred_at
		FROM payment_records pr
		JOIN users u ON u.id = pr.provider_id
		WHERE u.organization_id = $1
		  AND pr.transferred_at IS NOT NULL
		  AND pr.transferred_at >= $2
		  AND pr.transferred_at < $3
		  AND NOT EXISTS (
		      SELECT 1 FROM invoice_item ii WHERE ii.payment_record_id = pr.id
		  )
		ORDER BY pr.transferred_at ASC, pr.id ASC`,
		organizationID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("list released payment records for org: %w", err)
	}
	defer rows.Close()

	out := make([]repo.ReleasedPaymentRecord, 0)
	for rows.Next() {
		var rec repo.ReleasedPaymentRecord
		if err := rows.Scan(
			&rec.ID, &rec.MilestoneID, &rec.ProposalID,
			&rec.ProposalAmountCents, &rec.PlatformFeeCents,
			&rec.Currency, &rec.TransferredAt,
		); err != nil {
			return nil, fmt.Errorf("list released payment records for org: scan: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list released payment records for org: rows: %w", err)
	}
	return out, nil
}

// loadItems pulls all invoice_item rows for the given invoice id.
func (r *InvoiceRepository) loadItems(ctx context.Context, invoiceID uuid.UUID) ([]invoicing.InvoiceItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, invoice_id, description, quantity, unit_price_cents, amount_cents,
		       milestone_id, payment_record_id, created_at
		FROM invoice_item
		WHERE invoice_id = $1
		ORDER BY created_at ASC, id ASC`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]invoicing.InvoiceItem, 0)
	for rows.Next() {
		var (
			it              invoicing.InvoiceItem
			milestoneID     uuid.NullUUID
			paymentRecordID uuid.NullUUID
		)
		if err := rows.Scan(
			&it.ID, &it.InvoiceID, &it.Description, &it.Quantity,
			&it.UnitPriceCents, &it.AmountCents,
			&milestoneID, &paymentRecordID, &it.CreatedAt,
		); err != nil {
			return nil, err
		}
		if milestoneID.Valid {
			id := milestoneID.UUID
			it.MilestoneID = &id
		}
		if paymentRecordID.Valid {
			id := paymentRecordID.UUID
			it.PaymentRecordID = &id
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// invoiceRowScanner abstracts *sql.Row and *sql.Rows so scanInvoice can
// serve both the single-row and the list-iteration paths without
// duplication. Named with the file prefix to avoid collision with
// sibling adapters in this package.
type invoiceRowScanner interface {
	Scan(dest ...any) error
}

func scanInvoice(s *sql.Row) (*invoicing.Invoice, error) {
	return scanInvoiceFrom(s)
}

func scanInvoiceFromRows(s *sql.Rows) (*invoicing.Invoice, error) {
	return scanInvoiceFrom(s)
}

func scanInvoiceFrom(s invoiceRowScanner) (*invoicing.Invoice, error) {
	var (
		inv                   invoicing.Invoice
		recipientJSON         []byte
		issuerJSON            []byte
		taxRegime             string
		sourceType            string
		status                string
		stripeEventID         sql.NullString
		stripePaymentIntentID sql.NullString
		stripeInvoiceID       sql.NullString
		pdfR2Key              sql.NullString
		finalizedAt           sql.NullTime
		mentions              pq.StringArray
	)
	err := s.Scan(
		&inv.ID, &inv.Number, &inv.RecipientOrganizationID, &recipientJSON, &issuerJSON,
		&inv.IssuedAt, &inv.ServicePeriodStart, &inv.ServicePeriodEnd,
		&inv.Currency, &inv.AmountExclTaxCents, &inv.VATRate, &inv.VATAmountCents, &inv.AmountInclTaxCents,
		&taxRegime, &mentions, &sourceType,
		&stripeEventID, &stripePaymentIntentID, &stripeInvoiceID,
		&pdfR2Key, &status, &finalizedAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(recipientJSON, &inv.RecipientSnapshot); err != nil {
		return nil, fmt.Errorf("unmarshal recipient snapshot: %w", err)
	}
	if err := json.Unmarshal(issuerJSON, &inv.IssuerSnapshot); err != nil {
		return nil, fmt.Errorf("unmarshal issuer snapshot: %w", err)
	}
	inv.TaxRegime = invoicing.TaxRegime(taxRegime)
	inv.SourceType = invoicing.SourceType(sourceType)
	inv.Status = invoicing.Status(status)
	inv.MentionsRendered = []string(mentions)
	if stripeEventID.Valid {
		inv.StripeEventID = stripeEventID.String
	}
	if stripePaymentIntentID.Valid {
		inv.StripePaymentIntentID = stripePaymentIntentID.String
	}
	if stripeInvoiceID.Valid {
		inv.StripeInvoiceID = stripeInvoiceID.String
	}
	if pdfR2Key.Valid {
		inv.PDFR2Key = pdfR2Key.String
	}
	if finalizedAt.Valid {
		t := finalizedAt.Time
		inv.FinalizedAt = &t
	}
	return &inv, nil
}

func scanCreditNote(s *sql.Row) (*invoicing.CreditNote, error) {
	var (
		cn             invoicing.CreditNote
		recipientJSON  []byte
		issuerJSON     []byte
		taxRegime      string
		mentions       pq.StringArray
		stripeEventID  sql.NullString
		stripeRefundID sql.NullString
		pdfR2Key       sql.NullString
		finalizedAt    sql.NullTime
	)
	err := s.Scan(
		&cn.ID, &cn.Number, &cn.OriginalInvoiceID, &cn.RecipientOrganizationID,
		&recipientJSON, &issuerJSON, &cn.IssuedAt, &cn.Reason,
		&cn.Currency, &cn.AmountExclTaxCents, &cn.VATRate, &cn.VATAmountCents, &cn.AmountInclTaxCents,
		&taxRegime, &mentions, &stripeEventID, &stripeRefundID,
		&pdfR2Key, &finalizedAt, &cn.CreatedAt, &cn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(recipientJSON, &cn.RecipientSnapshot); err != nil {
		return nil, fmt.Errorf("unmarshal recipient snapshot: %w", err)
	}
	if err := json.Unmarshal(issuerJSON, &cn.IssuerSnapshot); err != nil {
		return nil, fmt.Errorf("unmarshal issuer snapshot: %w", err)
	}
	cn.TaxRegime = invoicing.TaxRegime(taxRegime)
	cn.MentionsRendered = []string(mentions)
	if stripeEventID.Valid {
		cn.StripeEventID = stripeEventID.String
	}
	if stripeRefundID.Valid {
		cn.StripeRefundID = stripeRefundID.String
	}
	if pdfR2Key.Valid {
		cn.PDFR2Key = pdfR2Key.String
	}
	if finalizedAt.Valid {
		t := finalizedAt.Time
		cn.FinalizedAt = &t
	}
	return &cn, nil
}

// FindCreditNoteByID returns a single credit note by primary key.
// Used by the admin PDF redirect path.
func (r *InvoiceRepository) FindCreditNoteByID(ctx context.Context, id uuid.UUID) (*invoicing.CreditNote, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+creditNoteColumns+`
		FROM credit_note
		WHERE id = $1`, id)

	cn, err := scanCreditNote(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invoicing.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find credit note by id: %w", err)
	}
	return cn, nil
}

// invoiceNullableString returns a *string that's NULL when the input
// is empty. Used for columns that have a UNIQUE constraint and must
// accept NULL (stripe_event_id, stripe_payment_intent_id,
// stripe_invoice_id) so monthly-commission invoices without a Stripe
// event id don't collide. Renamed away from the package-level
// nullableString to coexist with the milestone adapter's own helper.
func invoiceNullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
