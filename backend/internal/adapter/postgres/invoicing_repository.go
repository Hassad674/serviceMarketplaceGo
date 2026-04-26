package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/invoicing"
	repo "marketplace-backend/internal/port/repository"
)

// InvoiceRepository implements repository.InvoiceRepository against
// Postgres. Every public method runs under a 5 second context timeout
// and uses parameterized queries — no string concatenation, ever.
//
// Persistence guarantees:
//   - ReserveNumber atomically increments invoice_number_counter via
//     SELECT ... FOR UPDATE inside its own transaction.
//   - CreateInvoice / CreateCreditNote insert the parent row + items
//     (invoices only) inside a single transaction so a partial write
//     never lands.
type InvoiceRepository struct {
	db *sql.DB
}

func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

// invoiceColumns is the canonical projection used by every SELECT scan
// path so the column order never drifts between callers.
const invoiceColumns = `
	id, number, recipient_organization_id, recipient_snapshot, issuer_snapshot,
	issued_at, service_period_start, service_period_end,
	currency, amount_excl_tax_cents, vat_rate, vat_amount_cents, amount_incl_tax_cents,
	tax_regime, mentions_rendered, source_type,
	stripe_event_id, stripe_payment_intent_id, stripe_invoice_id,
	pdf_r2_key, status, finalized_at, created_at, updated_at
`

const creditNoteColumns = `
	id, number, original_invoice_id, recipient_organization_id,
	recipient_snapshot, issuer_snapshot, issued_at, reason,
	currency, amount_excl_tax_cents, vat_rate, vat_amount_cents, amount_incl_tax_cents,
	tax_regime, mentions_rendered, stripe_event_id, stripe_refund_id,
	pdf_r2_key, finalized_at, created_at, updated_at
`

// ReserveNumber draws the next sequence value for the given scope. The
// implementation opens its own transaction, takes a row-level lock on
// the counter row, increments next_value, and commits. Callers receive
// the value that was *current* at lock time.
func (r *InvoiceRepository) ReserveNumber(ctx context.Context, scope invoicing.CounterScope) (int64, error) {
	if !scope.IsValid() {
		return 0, invoicing.ErrCounterScopeUnknown
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("reserve number: begin tx: %w", err)
	}
	defer tx.Rollback()

	var next int64
	err = tx.QueryRowContext(ctx, `
		SELECT next_value
		FROM invoice_number_counter
		WHERE scope = $1 AND year = 0
		FOR UPDATE`, string(scope)).Scan(&next)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("reserve number: counter row missing for scope %q", scope)
	}
	if err != nil {
		return 0, fmt.Errorf("reserve number: select for update: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE invoice_number_counter
		SET next_value = next_value + 1
		WHERE scope = $1 AND year = 0`, string(scope))
	if err != nil {
		return 0, fmt.Errorf("reserve number: increment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("reserve number: commit: %w", err)
	}
	return next, nil
}

// CreateInvoice persists a finalized invoice and all of its items
// inside a single transaction. The invoice MUST already carry its
// number + pdf_r2_key + finalized_at — drafts are rejected.
func (r *InvoiceRepository) CreateInvoice(ctx context.Context, inv *invoicing.Invoice) error {
	if inv == nil {
		return fmt.Errorf("create invoice: nil invoice")
	}
	if !inv.IsFinalized() {
		return invoicing.ErrAlreadyFinalized
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	recipientJSON, err := json.Marshal(inv.RecipientSnapshot)
	if err != nil {
		return fmt.Errorf("create invoice: marshal recipient: %w", err)
	}
	issuerJSON, err := json.Marshal(inv.IssuerSnapshot)
	if err != nil {
		return fmt.Errorf("create invoice: marshal issuer: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("create invoice: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO invoice (
			id, number, recipient_organization_id, recipient_snapshot, issuer_snapshot,
			issued_at, service_period_start, service_period_end,
			currency, amount_excl_tax_cents, vat_rate, vat_amount_cents, amount_incl_tax_cents,
			tax_regime, mentions_rendered, source_type,
			stripe_event_id, stripe_payment_intent_id, stripe_invoice_id,
			pdf_r2_key, status, finalized_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16,
			$17, $18, $19,
			$20, $21, $22, $23, $24
		)`,
		inv.ID, inv.Number, inv.RecipientOrganizationID, recipientJSON, issuerJSON,
		inv.IssuedAt, inv.ServicePeriodStart, inv.ServicePeriodEnd,
		inv.Currency, inv.AmountExclTaxCents, inv.VATRate, inv.VATAmountCents, inv.AmountInclTaxCents,
		string(inv.TaxRegime), pq.Array(inv.MentionsRendered), string(inv.SourceType),
		invoiceNullableString(inv.StripeEventID), invoiceNullableString(inv.StripePaymentIntentID), invoiceNullableString(inv.StripeInvoiceID),
		invoiceNullableString(inv.PDFR2Key), string(inv.Status), inv.FinalizedAt, inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create invoice: insert invoice: %w", err)
	}

	for i := range inv.Items {
		it := &inv.Items[i]
		if it.ID == uuid.Nil {
			it.ID = uuid.New()
		}
		it.InvoiceID = inv.ID
		_, err = tx.ExecContext(ctx, `
			INSERT INTO invoice_item (
				id, invoice_id, description, quantity, unit_price_cents, amount_cents,
				milestone_id, payment_record_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			it.ID, it.InvoiceID, it.Description, it.Quantity, it.UnitPriceCents, it.AmountCents,
			it.MilestoneID, it.PaymentRecordID, it.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("create invoice: insert item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create invoice: commit: %w", err)
	}
	return nil
}

// CreateCreditNote persists a finalized credit note. There is no items
// table for credit notes — the row carries the totals directly.
func (r *InvoiceRepository) CreateCreditNote(ctx context.Context, cn *invoicing.CreditNote) error {
	if cn == nil {
		return fmt.Errorf("create credit note: nil credit note")
	}
	if !cn.IsFinalized() {
		return invoicing.ErrAlreadyFinalized
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	recipientJSON, err := json.Marshal(cn.RecipientSnapshot)
	if err != nil {
		return fmt.Errorf("create credit note: marshal recipient: %w", err)
	}
	issuerJSON, err := json.Marshal(cn.IssuerSnapshot)
	if err != nil {
		return fmt.Errorf("create credit note: marshal issuer: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO credit_note (
			id, number, original_invoice_id, recipient_organization_id,
			recipient_snapshot, issuer_snapshot, issued_at, reason,
			currency, amount_excl_tax_cents, vat_rate, vat_amount_cents, amount_incl_tax_cents,
			tax_regime, mentions_rendered, stripe_event_id, stripe_refund_id,
			pdf_r2_key, finalized_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19, $20, $21
		)`,
		cn.ID, cn.Number, cn.OriginalInvoiceID, cn.RecipientOrganizationID,
		recipientJSON, issuerJSON, cn.IssuedAt, cn.Reason,
		cn.Currency, cn.AmountExclTaxCents, cn.VATRate, cn.VATAmountCents, cn.AmountInclTaxCents,
		string(cn.TaxRegime), pq.Array(cn.MentionsRendered),
		invoiceNullableString(cn.StripeEventID), invoiceNullableString(cn.StripeRefundID),
		invoiceNullableString(cn.PDFR2Key), cn.FinalizedAt, cn.CreatedAt, cn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create credit note: insert: %w", err)
	}
	return nil
}

// FindInvoiceByID returns the invoice with all of its items. Returns
// invoicing.ErrNotFound when no row matches.
func (r *InvoiceRepository) FindInvoiceByID(ctx context.Context, id uuid.UUID) (*invoicing.Invoice, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+invoiceColumns+`
		FROM invoice
		WHERE id = $1`, id)

	inv, err := scanInvoice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invoicing.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invoice by id: %w", err)
	}

	items, err := r.loadItems(ctx, inv.ID)
	if err != nil {
		return nil, fmt.Errorf("find invoice by id: load items: %w", err)
	}
	inv.Items = items
	return inv, nil
}

// FindInvoiceByStripeEventID is the lookup the webhook handler uses to
// dedupe at the persistence layer (defense-in-depth on top of Redis).
func (r *InvoiceRepository) FindInvoiceByStripeEventID(ctx context.Context, eventID string) (*invoicing.Invoice, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+invoiceColumns+`
		FROM invoice
		WHERE stripe_event_id = $1`, eventID)

	inv, err := scanInvoice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invoicing.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invoice by stripe event id: %w", err)
	}

	items, err := r.loadItems(ctx, inv.ID)
	if err != nil {
		return nil, fmt.Errorf("find invoice by stripe event id: load items: %w", err)
	}
	inv.Items = items
	return inv, nil
}

// FindInvoiceByStripePaymentIntentID looks up the subscription invoice
// that originally captured the given PaymentIntent. Used by the refund
// webhook to bridge a charge.refunded event back to its source invoice.
// Returns invoicing.ErrNotFound when no row matches.
func (r *InvoiceRepository) FindInvoiceByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*invoicing.Invoice, error) {
	if paymentIntentID == "" {
		return nil, invoicing.ErrNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+invoiceColumns+`
		FROM invoice
		WHERE stripe_payment_intent_id = $1
		ORDER BY issued_at DESC
		LIMIT 1`, paymentIntentID)

	inv, err := scanInvoice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invoicing.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invoice by stripe payment intent id: %w", err)
	}

	items, err := r.loadItems(ctx, inv.ID)
	if err != nil {
		return nil, fmt.Errorf("find invoice by stripe payment intent id: load items: %w", err)
	}
	inv.Items = items
	return inv, nil
}

// MarkInvoiceCredited flips the invoice status to 'credited'. Bypasses
// the finalized read-only guard intentionally: status is the ONLY column
// that may transition after Finalize, and only via this single path.
func (r *InvoiceRepository) MarkInvoiceCredited(ctx context.Context, invoiceID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.db.ExecContext(ctx, `
		UPDATE invoice
		SET status = 'credited', updated_at = now()
		WHERE id = $1`, invoiceID)
	if err != nil {
		return fmt.Errorf("mark invoice credited: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark invoice credited: rows affected: %w", err)
	}
	if rows == 0 {
		return invoicing.ErrNotFound
	}
	return nil
}

// FindCreditNoteByStripeEventID is the credit-note analogue used by the
// refund webhook for idempotency.
func (r *InvoiceRepository) FindCreditNoteByStripeEventID(ctx context.Context, eventID string) (*invoicing.CreditNote, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+creditNoteColumns+`
		FROM credit_note
		WHERE stripe_event_id = $1`, eventID)

	cn, err := scanCreditNote(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invoicing.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find credit note by stripe event id: %w", err)
	}
	return cn, nil
}

// invoiceCursor is the JSON shape we base64-encode for cursor pagination
// over (issued_at DESC, id DESC). Kept private to this file — clients
// treat the cursor as opaque.
type invoiceCursor struct {
	IssuedAt time.Time `json:"issued_at"`
	ID       uuid.UUID `json:"id"`
}

func encodeInvoiceCursor(t time.Time, id uuid.UUID) string {
	data, _ := json.Marshal(invoiceCursor{IssuedAt: t, ID: id})
	return base64.URLEncoding.EncodeToString(data)
}

func decodeInvoiceCursor(s string) (*invoiceCursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode invoice cursor: %w", err)
	}
	var c invoiceCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("decode invoice cursor: unmarshal: %w", err)
	}
	return &c, nil
}

// ListInvoicesByOrganization returns the org's invoices in (issued_at,
// id) DESC order with opaque cursor pagination. Items are NOT loaded —
// callers fetch the detail page via FindInvoiceByID.
func (r *InvoiceRepository) ListInvoicesByOrganization(ctx context.Context, organizationID uuid.UUID, cursor string, limit int) ([]*invoicing.Invoice, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cur, err := decodeInvoiceCursor(cursor)
	if err != nil {
		return nil, "", err
	}

	var rows *sql.Rows
	if cur == nil {
		rows, err = r.db.QueryContext(ctx, `
			SELECT `+invoiceColumns+`
			FROM invoice
			WHERE recipient_organization_id = $1
			ORDER BY issued_at DESC, id DESC
			LIMIT $2`, organizationID, limit+1)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT `+invoiceColumns+`
			FROM invoice
			WHERE recipient_organization_id = $1
			  AND (issued_at, id) < ($2, $3)
			ORDER BY issued_at DESC, id DESC
			LIMIT $4`, organizationID, cur.IssuedAt, cur.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list invoices by organization: %w", err)
	}
	defer rows.Close()

	out := make([]*invoicing.Invoice, 0, limit)
	for rows.Next() {
		inv, scanErr := scanInvoiceFromRows(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("list invoices by organization: scan: %w", scanErr)
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("list invoices by organization: rows: %w", err)
	}

	nextCursor := ""
	if len(out) > limit {
		last := out[limit-1]
		nextCursor = encodeInvoiceCursor(last.IssuedAt, last.ID)
		out = out[:limit]
	}
	return out, nextCursor, nil
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
