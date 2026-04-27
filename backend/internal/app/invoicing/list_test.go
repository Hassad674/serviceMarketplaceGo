package invoicing_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
)

// newUserListServiceWithStorage mirrors newAdminListServiceWithStorage
// but is intended for the user-facing GetInvoicePDFURL tests. Both
// helpers use the same mock harness; we keep them separate for
// readability and so each test can grep for the entry point it cares
// about.
func newUserListServiceWithStorage(t *testing.T, repo *mockInvoiceRepo) (*invoicingapp.Service, *mockStorage) {
	t.Helper()
	storage := &mockStorage{}
	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    repo,
		Profiles:    &mockProfileRepo{},
		PDF:         &mockPDF{},
		Storage:     storage,
		Deliverer:   &mockDeliverer{},
		Issuer:      defaultIssuer(),
		Idempotency: &mockIdempotency{},
	})
	return svc, storage
}

// TestGetInvoicePDFURL_ForcesAttachmentDownload verifies that the
// public-facing PDF link the dashboard hands to the user requests an
// attachment-disposition presigned URL. Failing this test means the
// browser will open the PDF inline instead of saving it — the bug we
// just fixed.
func TestGetInvoicePDFURL_ForcesAttachmentDownload(t *testing.T) {
	orgID := uuid.New()
	invoiceID := uuid.New()

	repo := &mockInvoiceRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*invoicing.Invoice, error) {
			assert.Equal(t, invoiceID, id)
			return &invoicing.Invoice{
				ID:                      invoiceID,
				Number:                  "FAC-000789",
				PDFR2Key:                "invoices/org/FAC-000789.pdf",
				RecipientOrganizationID: orgID,
			}, nil
		},
	}
	svc, storage := newUserListServiceWithStorage(t, repo)

	url, err := svc.GetInvoicePDFURL(context.Background(), orgID, invoiceID, 5*time.Minute)
	require.NoError(t, err)

	// Filename is derived from the invoice number, not the opaque
	// R2 key — that is the human-readable name the user will see
	// in the browser save dialog.
	assert.Equal(t, "FAC-000789.pdf", storage.lastAttachmentFilename)
	assert.Equal(t, "invoices/org/FAC-000789.pdf", storage.lastAttachmentKey)
	assert.Contains(t, url, "response-content-disposition=attachment")
	assert.Contains(t, url, "FAC-000789.pdf")
}

// TestGetInvoicePDFURL_RejectsCrossOrgAccess preserves the existing
// security invariant: a user cannot signal a download URL for an
// invoice that belongs to another organization, even after the
// attachment-disposition refactor.
func TestGetInvoicePDFURL_RejectsCrossOrgAccess(t *testing.T) {
	myOrg := uuid.New()
	otherOrg := uuid.New()
	invoiceID := uuid.New()

	repo := &mockInvoiceRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
			return &invoicing.Invoice{
				ID:                      invoiceID,
				Number:                  "FAC-000789",
				PDFR2Key:                "invoices/other/FAC-000789.pdf",
				RecipientOrganizationID: otherOrg,
			}, nil
		},
	}
	svc, storage := newUserListServiceWithStorage(t, repo)

	_, err := svc.GetInvoicePDFURL(context.Background(), myOrg, invoiceID, time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, invoicingapp.ErrCrossOrgInvoiceAccess)
	// Storage must NOT have been called — ownership check happens
	// before any URL is generated.
	assert.Empty(t, storage.lastAttachmentKey)
}

// TestGetInvoicePDFURL_NotFoundIsTransparent ensures we don't leak
// details about an invoice belonging to another org via "not found"
// vs "forbidden" mismatch on the existence side.
func TestGetInvoicePDFURL_NotFoundIsTransparent(t *testing.T) {
	repo := &mockInvoiceRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
			return nil, invoicing.ErrNotFound
		},
	}
	svc, _ := newUserListServiceWithStorage(t, repo)

	_, err := svc.GetInvoicePDFURL(context.Background(), uuid.New(), uuid.New(), time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, invoicing.ErrNotFound)
}
