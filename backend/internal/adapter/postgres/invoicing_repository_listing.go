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

	"marketplace-backend/internal/domain/invoicing"
)

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
