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
//
// BUG-NEW-04 path 3/8: the invoice table is RLS-protected by migration
// 125 with the policy
//
//   USING (recipient_organization_id = current_setting('app.current_org_id', true)::uuid)
//
// Mutations on rows owned by an org (CreateInvoice / CreateCreditNote /
// MarkInvoiceCredited) wrap the SQL in RunInTxWithTenant with the
// invoice's recipient org as app.current_org_id. The reads
// ListInvoicesByOrganization / FindInvoiceByIDForOrg take the caller's
// org and wrap likewise.
//
// Stripe webhook lookups (FindInvoiceByStripeEventID,
// FindInvoiceByStripePaymentIntentID, FindCreditNoteByStripeEventID)
// run as system-actor and must look up rows BEFORE knowing the org —
// idempotency keys are global. Under prod NOSUPERUSER NOBYPASSRLS
// these reads would return ErrNotFound for every event, breaking
// idempotency. They stay on the legacy direct-db path; the production
// deployment must keep the webhook handler on a privileged DB role
// OR adopt a separate idempotency table that bypasses RLS. This is a
// DEPLOYMENT-LEVEL CONSTRAINT, flagged in the BUG-NEW-04 follow-ups.
type InvoiceRepository struct {
	db       *sql.DB
	txRunner *TxRunner
}

func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper. Wired
// from cmd/api/main.go so every invoice org-scoped read/write fires
// inside RunInTxWithTenant. Returns the same pointer so the wiring
// chain stays terse.
func (r *InvoiceRepository) WithTxRunner(runner *TxRunner) *InvoiceRepository {
	r.txRunner = runner
	return r
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

// creditNoteColumns is the canonical column projection for SELECTs
// against the credit_note table. The string contains only schema
// names; gosec G101 flags it as a potential hardcoded credential
// because of the "secret"-like word `_key` (in `stripe_event_id` and
// `pdf_r2_key`) but those are column identifiers, not credentials.
// #nosec G101 -- SQL column list, not credentials
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
//
// BUG-NEW-04 path 3/8: the parent INSERT into `invoice` is RLS-
// protected. Under prod NOSUPERUSER NOBYPASSRLS the row is rejected
// unless app.current_org_id matches inv.RecipientOrganizationID. The
// txRunner branch wraps the parent + child inserts in
// RunInTxWithTenant so the org context is set before either insert
// fires.
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

	insertAll := func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
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
		); err != nil {
			return fmt.Errorf("create invoice: insert invoice: %w", err)
		}

		for i := range inv.Items {
			it := &inv.Items[i]
			if it.ID == uuid.Nil {
				it.ID = uuid.New()
			}
			it.InvoiceID = inv.ID
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO invoice_item (
					id, invoice_id, description, quantity, unit_price_cents, amount_cents,
					milestone_id, payment_record_id, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				it.ID, it.InvoiceID, it.Description, it.Quantity, it.UnitPriceCents, it.AmountCents,
				it.MilestoneID, it.PaymentRecordID, it.CreatedAt,
			); err != nil {
				return fmt.Errorf("create invoice: insert item: %w", err)
			}
		}
		return nil
	}

	if r.txRunner != nil {
		return r.txRunner.RunInTxWithTenant(ctx, inv.RecipientOrganizationID, uuid.Nil, insertAll)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("create invoice: begin tx: %w", err)
	}
	defer tx.Rollback()
	if err := insertAll(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create invoice: commit: %w", err)
	}
	return nil
}

// CreateCreditNote persists a finalized credit note. There is no items
// table for credit notes — the row carries the totals directly.
//
// credit_note is NOT directly RLS-protected by migration 125, so this
// runs on the legacy direct-db path even when a TxRunner is wired.
// However, callers typically combine credit-note creation with
// MarkInvoiceCredited (which IS RLS-protected) in the same flow, so
// the org context is established at the caller's level.
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
//
// BUG-NEW-04 path 3/8: the UPDATE on `invoice` is RLS-protected. To
// install app.current_org_id the runner-aware path first reads the
// row's org via a privileged SELECT (uuid.Nil context — admin-style
// read), then opens the tenant tx with that org and runs the UPDATE.
// In production this works because MarkInvoiceCredited is called by
// the credit-note flow which already holds the original invoice in
// memory; the lookup here is a defensive fallback for callers that
// only have the id. When the row is not visible (e.g. the webhook
// path under non-superuser without privilege), we surface ErrNotFound
// — same observable behaviour as the legacy path on a missing id.
func (r *InvoiceRepository) MarkInvoiceCredited(ctx context.Context, invoiceID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	doUpdate := func(runner sqlExecutor) error {
		res, err := runner.ExecContext(ctx, `
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

	if r.txRunner != nil {
		// Two-step under tenant tx: first SELECT the org from the row
		// (works because invoice.id is the PK and the row exists), then
		// open a NEW tenant tx with that org and run the UPDATE. The
		// SELECT runs on the legacy db connection because we don't yet
		// know the org; this is OK because under prod the webhook role
		// is privileged for the lookup. If the row doesn't exist in the
		// caller's accessible scope, ErrNotFound is the right answer.
		var orgID uuid.UUID
		err := r.db.QueryRowContext(ctx, `
			SELECT recipient_organization_id FROM invoice WHERE id = $1`, invoiceID,
		).Scan(&orgID)
		if errors.Is(err, sql.ErrNoRows) {
			return invoicing.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("mark invoice credited: lookup org: %w", err)
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doUpdate(tx)
		})
	}

	return doUpdate(r.db)
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
// callers fetch the detail page via FindInvoiceByIDForOrg.
//
// BUG-NEW-04 path 3/8: wraps the SELECT in RunInTxWithTenant with the
// caller's org so the rows return under prod NOSUPERUSER NOBYPASSRLS.
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

	var out []*invoicing.Invoice
	var nextCursor string

	doQuery := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var qerr error
		if cur == nil {
			rows, qerr = runner.QueryContext(ctx, `
				SELECT `+invoiceColumns+`
				FROM invoice
				WHERE recipient_organization_id = $1
				ORDER BY issued_at DESC, id DESC
				LIMIT $2`, organizationID, limit+1)
		} else {
			rows, qerr = runner.QueryContext(ctx, `
				SELECT `+invoiceColumns+`
				FROM invoice
				WHERE recipient_organization_id = $1
				  AND (issued_at, id) < ($2, $3)
				ORDER BY issued_at DESC, id DESC
				LIMIT $4`, organizationID, cur.IssuedAt, cur.ID, limit+1)
		}
		if qerr != nil {
			return fmt.Errorf("list invoices by organization: %w", qerr)
		}
		defer rows.Close()

		out = make([]*invoicing.Invoice, 0, limit)
		for rows.Next() {
			inv, scanErr := scanInvoiceFromRows(rows)
			if scanErr != nil {
				return fmt.Errorf("list invoices by organization: scan: %w", scanErr)
			}
			out = append(out, inv)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("list invoices by organization: rows: %w", err)
		}

		if len(out) > limit {
			last := out[limit-1]
			nextCursor = encodeInvoiceCursor(last.IssuedAt, last.ID)
			out = out[:limit]
		}
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, organizationID, uuid.Nil, func(tx *sql.Tx) error {
			return doQuery(tx)
		})
		if err != nil {
			return nil, "", err
		}
		return out, nextCursor, nil
	}

	if err := doQuery(r.db); err != nil {
		return nil, "", err
	}
	return out, nextCursor, nil
}

// FindInvoiceByIDForOrg returns the invoice with all of its items,
// fetched under the caller's org tenant context. Use this entry
// point whenever the caller is an authenticated org member —
// FindInvoiceByID stays for admin tooling and webhook flows that
// must bypass tenant isolation.
//
// Wrapped in RunInTxWithTenant(orgID, uuid.Nil, ...) so the SELECT
// passes the policy under prod NOSUPERUSER NOBYPASSRLS. When the
// requested invoice does not belong to the supplied org, the row is
// filtered out by RLS and the function returns ErrNotFound — the
// same observable behaviour as the explicit cross-org check at the
// app layer (ErrCrossOrgInvoiceAccess), but enforced at the database
// level too.
func (r *InvoiceRepository) FindInvoiceByIDForOrg(ctx context.Context, id, orgID uuid.UUID) (*invoicing.Invoice, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var inv *invoicing.Invoice

	doQuery := func(runner sqlQuerier) error {
		row := runner.QueryRowContext(ctx, `
			SELECT `+invoiceColumns+`
			FROM invoice
			WHERE id = $1`, id)
		got, err := scanInvoiceFromRowQuery(row)
		if errors.Is(err, sql.ErrNoRows) {
			return invoicing.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("find invoice by id for org: %w", err)
		}
		// Loading items goes through the same runner so the SELECT runs
		// inside the same tenant tx. invoice_item is not RLS-protected,
		// but using the tx keeps the snapshot consistent with the parent.
		items, err := loadItemsViaRunner(ctx, runner, got.ID)
		if err != nil {
			return fmt.Errorf("find invoice by id for org: load items: %w", err)
		}
		got.Items = items
		inv = got
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doQuery(tx)
		})
		if err != nil {
			return nil, err
		}
		return inv, nil
	}

	if err := doQuery(r.db); err != nil {
		return nil, err
	}
	return inv, nil
}

// scanInvoiceFromRowQuery is the *sql.Row variant returned by
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

// adminInvoiceCursor is the JSON shape we base64-encode for the admin
// listing page cursor. It mirrors invoiceCursor (issued_at + id DESC)
// but lives apart so the two cursor scopes never accidentally cross
// over (admin + per-org list speak the same shape but they walk
// different result sets).
type adminInvoiceCursor struct {
	IssuedAt time.Time `json:"issued_at"`
	ID       uuid.UUID `json:"id"`
}

func encodeAdminInvoiceCursor(t time.Time, id uuid.UUID) string {
	data, _ := json.Marshal(adminInvoiceCursor{IssuedAt: t, ID: id})
	return base64.URLEncoding.EncodeToString(data)
}

func decodeAdminInvoiceCursor(s string) (*adminInvoiceCursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode admin invoice cursor: %w", err)
	}
	var c adminInvoiceCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("decode admin invoice cursor: unmarshal: %w", err)
	}
	return &c, nil
}

// ListInvoicesAdmin walks invoices + credit_notes via UNION ALL with the
// supplied filters, ordered by issued_at DESC, id DESC. The two source
// tables are projected to a common shape so the cursor walk works
// uniformly across both. Filters that are zero-valued on the struct
// behave as "no filter" — see AdminInvoiceFilters.
//
// SourceType column on the invoice side carries the
// 'subscription'/'monthly_commission' enum; the credit_note side has
// no such column, so the projection emits the literal 'credit_note'
// for those rows. The same trick is used for status — invoices keep
// their 'issued'/'credited'/'draft' values, credit notes report
// 'credit_note' so the UI can render a single badge column.
func (r *InvoiceRepository) ListInvoicesAdmin(ctx context.Context, filters repo.AdminInvoiceFilters, cursor string, limit int) ([]*repo.AdminInvoiceRow, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cur, err := decodeAdminInvoiceCursor(cursor)
	if err != nil {
		return nil, "", err
	}

	// We assemble two parallel WHERE clauses (one per source table)
	// instead of filtering after UNION ALL — that lets Postgres push
	// each predicate down to the right index (idx_invoice_recipient_issued
	// + idx_credit_note_recipient_issued).
	//
	// A "credit_note" status filter prunes the invoice branch entirely
	// by injecting a tautologically-false predicate. The "subscription"
	// and "monthly_commission" status filters do the symmetric trick on
	// the credit_note branch.
	args := make([]any, 0, 12)
	idx := 1

	addArg := func(v any) string {
		args = append(args, v)
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}

	invoiceWhere := "TRUE"
	creditNoteWhere := "TRUE"

	if filters.RecipientOrgID != nil {
		ph := addArg(*filters.RecipientOrgID)
		invoiceWhere += " AND i.recipient_organization_id = " + ph
		// Reuse the same placeholder index — Postgres allows reusing
		// a positional arg multiple times in one statement.
		creditNoteWhere += " AND cn.recipient_organization_id = " + ph
	}

	switch filters.Status {
	case "subscription", "monthly_commission":
		ph := addArg(filters.Status)
		invoiceWhere += " AND i.source_type = " + ph
		creditNoteWhere += " AND FALSE"
	case "credit_note":
		invoiceWhere += " AND FALSE"
	case "":
		// no-op — both branches stay open
	default:
		// Unknown status filter: same as "no rows".
		invoiceWhere += " AND FALSE"
		creditNoteWhere += " AND FALSE"
	}

	if filters.DateFrom != nil {
		ph := addArg(*filters.DateFrom)
		invoiceWhere += " AND i.issued_at >= " + ph
		creditNoteWhere += " AND cn.issued_at >= " + ph
	}
	if filters.DateTo != nil {
		ph := addArg(*filters.DateTo)
		invoiceWhere += " AND i.issued_at <= " + ph
		creditNoteWhere += " AND cn.issued_at <= " + ph
	}
	if filters.MinAmountCents != nil {
		ph := addArg(*filters.MinAmountCents)
		invoiceWhere += " AND i.amount_incl_tax_cents >= " + ph
		creditNoteWhere += " AND cn.amount_incl_tax_cents >= " + ph
	}
	if filters.MaxAmountCents != nil {
		ph := addArg(*filters.MaxAmountCents)
		invoiceWhere += " AND i.amount_incl_tax_cents <= " + ph
		creditNoteWhere += " AND cn.amount_incl_tax_cents <= " + ph
	}
	if s := filters.Search; s != "" {
		ph := addArg("%" + s + "%")
		invoiceWhere += " AND (i.number ILIKE " + ph + " OR i.recipient_snapshot->>'legal_name' ILIKE " + ph + ")"
		creditNoteWhere += " AND (cn.number ILIKE " + ph + " OR cn.recipient_snapshot->>'legal_name' ILIKE " + ph + ")"
	}

	cursorPredicate := ""
	if cur != nil {
		issuedPH := addArg(cur.IssuedAt)
		idPH := addArg(cur.ID)
		cursorPredicate = " AND (combined.issued_at, combined.id) < (" + issuedPH + ", " + idPH + ")"
	}

	limitPH := addArg(limit + 1)

	// gosec G202 suppression rationale: the four concatenation sites
	// below splice strings that contain ONLY $N placeholders. They are
	// produced by the closure `addArg(value any) string { args = append…
	// return fmt.Sprintf("$%d", len(args)) }` which never embeds a
	// caller-supplied value into its return — every value lands in
	// `args` and reaches Postgres via parameterised binding. The
	// admin filter sql_injection_test.go drives the function with
	// classic injection payloads (`'; DROP …`, `' OR 1=1 --`, NUL
	// bytes, encoded comments) and verifies the resulting query
	// returns expected rows without executing the malicious clause.
	query := `
		WITH combined AS (
			SELECT
				i.id, i.number, FALSE AS is_credit_note,
				i.recipient_organization_id,
				COALESCE(i.recipient_snapshot->>'legal_name', '') AS recipient_legal_name,
				i.issued_at, i.amount_incl_tax_cents, i.currency,
				i.tax_regime, i.status,
				COALESCE(i.pdf_r2_key, '') AS pdf_r2_key,
				NULL::uuid AS original_invoice_id,
				i.source_type
			FROM invoice i
			WHERE ` + invoiceWhere + ` ` + // #nosec G202 -- placeholder-only concat, tested
		`UNION ALL
			SELECT
				cn.id, cn.number, TRUE AS is_credit_note,
				cn.recipient_organization_id,
				COALESCE(cn.recipient_snapshot->>'legal_name', '') AS recipient_legal_name,
				cn.issued_at, cn.amount_incl_tax_cents, cn.currency,
				cn.tax_regime, 'credit_note' AS status,
				COALESCE(cn.pdf_r2_key, '') AS pdf_r2_key,
				cn.original_invoice_id,
				'' AS source_type
			FROM credit_note cn
			WHERE ` + creditNoteWhere + // #nosec G202 -- placeholder-only concat, tested
		`
		)
		SELECT id, number, is_credit_note, recipient_organization_id,
		       recipient_legal_name, issued_at, amount_incl_tax_cents, currency,
		       tax_regime, status, pdf_r2_key, original_invoice_id, source_type
		FROM combined
		WHERE TRUE` + cursorPredicate + // #nosec G202 -- placeholder-only concat, tested
		`
		ORDER BY issued_at DESC, id DESC
		LIMIT ` + limitPH // #nosec G202 -- placeholder-only concat, tested

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list invoices admin: %w", err)
	}
	defer rows.Close()

	out := make([]*repo.AdminInvoiceRow, 0, limit)
	for rows.Next() {
		var (
			row              repo.AdminInvoiceRow
			origID           uuid.NullUUID
			sourceType       sql.NullString
		)
		if err := rows.Scan(
			&row.ID, &row.Number, &row.IsCreditNote, &row.RecipientOrgID,
			&row.RecipientLegalName, &row.IssuedAt, &row.AmountInclTaxCents, &row.Currency,
			&row.TaxRegime, &row.Status, &row.PDFR2Key, &origID, &sourceType,
		); err != nil {
			return nil, "", fmt.Errorf("list invoices admin: scan: %w", err)
		}
		if origID.Valid {
			id := origID.UUID
			row.OriginalInvoiceID = &id
		}
		if sourceType.Valid {
			row.SourceType = sourceType.String
		}
		out = append(out, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("list invoices admin: rows: %w", err)
	}

	nextCursor := ""
	if len(out) > limit {
		last := out[limit-1]
		nextCursor = encodeAdminInvoiceCursor(last.IssuedAt, last.ID)
		out = out[:limit]
	}
	return out, nextCursor, nil
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
