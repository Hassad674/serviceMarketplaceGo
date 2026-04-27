package invoicing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
)

// AdminListInvoices returns a page of invoices and credit notes across
// every organization, with the optional filters applied. The caller
// (admin handler) is responsible for the role check — this service
// method assumes RequireAdmin has already gated the route.
//
// Cursor + limit are forwarded to the repository as-is. Limit defaults
// to 20 when zero/negative and is capped at 100 — those bounds are
// enforced inside the repository so we don't duplicate them here.
func (s *Service) AdminListInvoices(ctx context.Context, filters repository.AdminInvoiceFilters, cursor string, limit int) ([]*repository.AdminInvoiceRow, string, error) {
	rows, next, err := s.invoices.ListInvoicesAdmin(ctx, filters, cursor, limit)
	if err != nil {
		return nil, "", fmt.Errorf("admin list invoices: %w", err)
	}
	return rows, next, nil
}

// AdminGetInvoicePDF resolves the row (invoice or credit note) and
// returns a short-lived presigned download URL for its PDF. The
// `isCreditNote` flag tells us which table to look up — the admin UI
// passes ?type=invoice or ?type=credit_note based on the row it just
// rendered.
//
// Returns invoicing.ErrNotFound when no row matches, or a generic error
// when the row exists but has no stored PDF key (should never happen
// for finalized rows; the issuance pipeline always sets pdf_r2_key
// before persisting).
func (s *Service) AdminGetInvoicePDF(ctx context.Context, id uuid.UUID, isCreditNote bool, expiry time.Duration) (string, error) {
	if id == uuid.Nil {
		return "", fmt.Errorf("admin get invoice pdf: id required")
	}
	if expiry <= 0 {
		expiry = 5 * time.Minute
	}

	var pdfKey string
	if isCreditNote {
		cn, err := s.invoices.FindCreditNoteByID(ctx, id)
		if err != nil {
			if errors.Is(err, invoicing.ErrNotFound) {
				return "", invoicing.ErrNotFound
			}
			return "", fmt.Errorf("admin get invoice pdf: load credit note: %w", err)
		}
		pdfKey = cn.PDFR2Key
	} else {
		inv, err := s.invoices.FindInvoiceByID(ctx, id)
		if err != nil {
			if errors.Is(err, invoicing.ErrNotFound) {
				return "", invoicing.ErrNotFound
			}
			return "", fmt.Errorf("admin get invoice pdf: load invoice: %w", err)
		}
		pdfKey = inv.PDFR2Key
	}

	if pdfKey == "" {
		return "", fmt.Errorf("admin get invoice pdf: row has no stored PDF key")
	}

	url, err := s.storage.GetPresignedDownloadURL(ctx, pdfKey, expiry)
	if err != nil {
		return "", fmt.Errorf("admin get invoice pdf: presign: %w", err)
	}
	return url, nil
}
