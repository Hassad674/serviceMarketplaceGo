package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	repo "marketplace-backend/internal/port/repository"
)

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
			row        repo.AdminInvoiceRow
			origID     uuid.NullUUID
			sourceType sql.NullString
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
