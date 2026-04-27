package invoicing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
)

// InvoiceListItem is the slim projection used by the /me/invoices list
// endpoint. The detail page (PDF redirect) does its own ownership check
// before serving the file — the list payload itself is safe to render
// without sensitive recipient data.
type InvoiceListItem struct {
	ID                 uuid.UUID
	Number             string
	IssuedAt           time.Time
	SourceType         invoicing.SourceType
	AmountInclTaxCents int64
	Currency           string
	PDFR2Key           string
}

// ListMyInvoicesResult bundles the page + opaque cursor for the next.
type ListMyInvoicesResult struct {
	Items      []InvoiceListItem
	NextCursor string
}

// ListMyInvoices wraps the repository's cursor-paginated list with the
// org scoping check and the slim projection. Note: ownership is implicit
// here because the repo always filters by recipient_organization_id.
func (s *Service) ListMyInvoices(ctx context.Context, organizationID uuid.UUID, cursor string, limit int) (ListMyInvoicesResult, error) {
	if organizationID == uuid.Nil {
		return ListMyInvoicesResult{}, fmt.Errorf("invoicing: organization id required")
	}
	rows, next, err := s.invoices.ListInvoicesByOrganization(ctx, organizationID, cursor, limit)
	if err != nil {
		return ListMyInvoicesResult{}, fmt.Errorf("list my invoices: %w", err)
	}
	out := make([]InvoiceListItem, 0, len(rows))
	for _, inv := range rows {
		out = append(out, InvoiceListItem{
			ID:                 inv.ID,
			Number:             inv.Number,
			IssuedAt:           inv.IssuedAt,
			SourceType:         inv.SourceType,
			AmountInclTaxCents: inv.AmountInclTaxCents,
			Currency:           inv.Currency,
			PDFR2Key:           inv.PDFR2Key,
		})
	}
	return ListMyInvoicesResult{Items: out, NextCursor: next}, nil
}

// ErrCrossOrgInvoiceAccess is returned when a caller asks for an
// invoice that belongs to a different organization. Mapped to 403 by
// the handler. Distinct from ErrNotFound so we never leak existence to
// a non-owner.
var ErrCrossOrgInvoiceAccess = errors.New("invoicing: invoice does not belong to caller's organization")

// GetInvoicePDFURL returns a short-lived presigned download URL for the
// invoice PDF after verifying the invoice belongs to the caller's
// organization. Returns ErrNotFound when the row is missing,
// ErrCrossOrgInvoiceAccess when ownership fails (handler maps to 403).
func (s *Service) GetInvoicePDFURL(ctx context.Context, organizationID, invoiceID uuid.UUID, expiry time.Duration) (string, error) {
	if organizationID == uuid.Nil {
		return "", fmt.Errorf("invoicing: organization id required")
	}
	if invoiceID == uuid.Nil {
		return "", fmt.Errorf("invoicing: invoice id required")
	}
	if expiry <= 0 {
		expiry = 5 * time.Minute
	}
	inv, err := s.invoices.FindInvoiceByID(ctx, invoiceID)
	if err != nil {
		return "", fmt.Errorf("get invoice pdf: %w", err)
	}
	if inv == nil {
		return "", invoicing.ErrNotFound
	}
	if inv.RecipientOrganizationID != organizationID {
		return "", ErrCrossOrgInvoiceAccess
	}
	if inv.PDFR2Key == "" {
		return "", fmt.Errorf("get invoice pdf: invoice has no stored PDF key")
	}
	// Signed URL is generated with Content-Disposition: attachment so
	// the browser saves the file under "<number>.pdf" instead of
	// rendering it inline in a new tab when the user clicks
	// "Télécharger PDF".
	filename := inv.Number + ".pdf"
	url, err := s.storage.GetPresignedDownloadURLAsAttachment(ctx, inv.PDFR2Key, filename, expiry)
	if err != nil {
		return "", fmt.Errorf("get invoice pdf: presign: %w", err)
	}
	return url, nil
}
