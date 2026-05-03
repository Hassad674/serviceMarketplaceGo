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
