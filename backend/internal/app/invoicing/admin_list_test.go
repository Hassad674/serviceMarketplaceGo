package invoicing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
)

// newAdminListService builds a Service with the same mock harness used
// by the rest of the package's tests, but exposes the listInvoicesAdmin
// hook so each test can install its own canned response.
func newAdminListService(t *testing.T, repo *mockInvoiceRepo) *invoicingapp.Service {
	t.Helper()
	storage := &mockStorage{}
	return invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    repo,
		Profiles:    &mockProfileRepo{},
		PDF:         &mockPDF{},
		Storage:     storage,
		Deliverer:   &mockDeliverer{},
		Issuer:      defaultIssuer(),
		Idempotency: &mockIdempotency{},
	})
}

func TestAdminListInvoices_PassesFiltersThrough(t *testing.T) {
	orgID := uuid.New()
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)
	min := int64(1000)
	max := int64(50000)

	type call struct {
		filters repository.AdminInvoiceFilters
		cursor  string
		limit   int
	}
	var seen call

	repo := &mockInvoiceRepo{
		listAdminFn: func(_ context.Context, f repository.AdminInvoiceFilters, c string, l int) ([]*repository.AdminInvoiceRow, string, error) {
			seen = call{filters: f, cursor: c, limit: l}
			return []*repository.AdminInvoiceRow{
				{ID: uuid.New(), Number: "FAC-000001", IssuedAt: time.Now()},
			}, "next-cursor", nil
		},
	}
	svc := newAdminListService(t, repo)

	filters := repository.AdminInvoiceFilters{
		RecipientOrgID: &orgID,
		Status:         "subscription",
		DateFrom:       &from,
		DateTo:         &to,
		MinAmountCents: &min,
		MaxAmountCents: &max,
		Search:         "Acme",
	}
	rows, next, err := svc.AdminListInvoices(context.Background(), filters, "incoming-cursor", 50)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "next-cursor", next)
	assert.Equal(t, "incoming-cursor", seen.cursor)
	assert.Equal(t, 50, seen.limit)
	assert.Equal(t, &orgID, seen.filters.RecipientOrgID)
	assert.Equal(t, "subscription", seen.filters.Status)
	assert.Equal(t, &from, seen.filters.DateFrom)
	assert.Equal(t, &to, seen.filters.DateTo)
	assert.Equal(t, &min, seen.filters.MinAmountCents)
	assert.Equal(t, &max, seen.filters.MaxAmountCents)
	assert.Equal(t, "Acme", seen.filters.Search)
}

func TestAdminListInvoices_EmptyFiltersAreOK(t *testing.T) {
	repo := &mockInvoiceRepo{
		listAdminFn: func(_ context.Context, f repository.AdminInvoiceFilters, _ string, _ int) ([]*repository.AdminInvoiceRow, string, error) {
			// Every nilable field must remain nil.
			assert.Nil(t, f.RecipientOrgID)
			assert.Nil(t, f.DateFrom)
			assert.Nil(t, f.DateTo)
			assert.Nil(t, f.MinAmountCents)
			assert.Nil(t, f.MaxAmountCents)
			assert.Equal(t, "", f.Status)
			assert.Equal(t, "", f.Search)
			return nil, "", nil
		},
	}
	svc := newAdminListService(t, repo)
	rows, next, err := svc.AdminListInvoices(context.Background(), repository.AdminInvoiceFilters{}, "", 0)
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, "", next)
}

func TestAdminListInvoices_RepositoryErrorWraps(t *testing.T) {
	sentinel := errors.New("boom")
	repo := &mockInvoiceRepo{
		listAdminFn: func(_ context.Context, _ repository.AdminInvoiceFilters, _ string, _ int) ([]*repository.AdminInvoiceRow, string, error) {
			return nil, "", sentinel
		},
	}
	svc := newAdminListService(t, repo)
	_, _, err := svc.AdminListInvoices(context.Background(), repository.AdminInvoiceFilters{}, "", 20)
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
}

func TestAdminGetInvoicePDF_InvoiceBranch(t *testing.T) {
	id := uuid.New()
	repo := &mockInvoiceRepo{
		findByIDFn: func(_ context.Context, queryID uuid.UUID) (*invoicing.Invoice, error) {
			assert.Equal(t, id, queryID)
			return &invoicing.Invoice{ID: id, PDFR2Key: "invoices/x/FAC-1.pdf"}, nil
		},
	}
	svc := newAdminListService(t, repo)
	url, err := svc.AdminGetInvoicePDF(context.Background(), id, false, 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "https://r2.test/download/invoices/x/FAC-1.pdf", url)
}

func TestAdminGetInvoicePDF_CreditNoteBranch(t *testing.T) {
	id := uuid.New()
	repo := &mockInvoiceRepo{
		findCnByIDFn: func(_ context.Context, queryID uuid.UUID) (*invoicing.CreditNote, error) {
			assert.Equal(t, id, queryID)
			return &invoicing.CreditNote{ID: id, PDFR2Key: "credit-notes/x/AV-1.pdf"}, nil
		},
	}
	svc := newAdminListService(t, repo)
	url, err := svc.AdminGetInvoicePDF(context.Background(), id, true, 0)
	require.NoError(t, err)
	assert.Equal(t, "https://r2.test/download/credit-notes/x/AV-1.pdf", url)
}

func TestAdminGetInvoicePDF_NotFoundIsTransparent(t *testing.T) {
	repo := &mockInvoiceRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
			return nil, invoicing.ErrNotFound
		},
	}
	svc := newAdminListService(t, repo)
	_, err := svc.AdminGetInvoicePDF(context.Background(), uuid.New(), false, time.Minute)
	assert.ErrorIs(t, err, invoicing.ErrNotFound)
}

func TestAdminGetInvoicePDF_RowWithoutPDFKey(t *testing.T) {
	id := uuid.New()
	repo := &mockInvoiceRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
			return &invoicing.Invoice{ID: id, PDFR2Key: ""}, nil
		},
	}
	svc := newAdminListService(t, repo)
	_, err := svc.AdminGetInvoicePDF(context.Background(), id, false, time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no stored PDF key")
}

func TestAdminGetInvoicePDF_NilIDRejected(t *testing.T) {
	svc := newAdminListService(t, &mockInvoiceRepo{})
	_, err := svc.AdminGetInvoicePDF(context.Background(), uuid.Nil, false, time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id required")
}
