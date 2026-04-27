package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
)

// adminInvoiceFakeRepo is a focused stand-in implementing exactly the
// methods AdminListInvoices + AdminGetInvoicePDF call. The shared
// invFakeRepo would also work but is heavier — this keeps each test
// isolated and the assertions explicit.
type adminInvoiceFakeRepo struct {
	rows         []*repository.AdminInvoiceRow
	nextCursor   string
	listErr      error
	listSeen     repository.AdminInvoiceFilters
	listSeenLim  int
	listSeenCur  string
	invoices     map[uuid.UUID]*domain.Invoice
	creditNotes  map[uuid.UUID]*domain.CreditNote
}

func newAdminInvoiceFakeRepo() *adminInvoiceFakeRepo {
	return &adminInvoiceFakeRepo{
		invoices:    map[uuid.UUID]*domain.Invoice{},
		creditNotes: map[uuid.UUID]*domain.CreditNote{},
	}
}

func (r *adminInvoiceFakeRepo) CreateInvoice(_ context.Context, _ *domain.Invoice) error {
	return nil
}
func (r *adminInvoiceFakeRepo) CreateCreditNote(_ context.Context, _ *domain.CreditNote) error {
	return nil
}
func (r *adminInvoiceFakeRepo) ReserveNumber(_ context.Context, _ domain.CounterScope) (int64, error) {
	return 1, nil
}
func (r *adminInvoiceFakeRepo) FindInvoiceByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if inv, ok := r.invoices[id]; ok {
		return inv, nil
	}
	return nil, domain.ErrNotFound
}
func (r *adminInvoiceFakeRepo) FindInvoiceByStripeEventID(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, domain.ErrNotFound
}
func (r *adminInvoiceFakeRepo) FindCreditNoteByStripeEventID(_ context.Context, _ string) (*domain.CreditNote, error) {
	return nil, domain.ErrNotFound
}
func (r *adminInvoiceFakeRepo) FindInvoiceByStripePaymentIntentID(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, domain.ErrNotFound
}
func (r *adminInvoiceFakeRepo) MarkInvoiceCredited(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (r *adminInvoiceFakeRepo) ListInvoicesByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Invoice, string, error) {
	return nil, "", nil
}
func (r *adminInvoiceFakeRepo) HasInvoiceItemForPaymentRecord(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (r *adminInvoiceFakeRepo) ListReleasedPaymentRecordsForOrg(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
	return nil, nil
}
func (r *adminInvoiceFakeRepo) ListInvoicesAdmin(_ context.Context, filters repository.AdminInvoiceFilters, cursor string, limit int) ([]*repository.AdminInvoiceRow, string, error) {
	r.listSeen = filters
	r.listSeenLim = limit
	r.listSeenCur = cursor
	return r.rows, r.nextCursor, r.listErr
}
func (r *adminInvoiceFakeRepo) FindCreditNoteByID(_ context.Context, id uuid.UUID) (*domain.CreditNote, error) {
	if cn, ok := r.creditNotes[id]; ok {
		return cn, nil
	}
	return nil, domain.ErrNotFound
}

// adminInvoiceHarness wires the handler to a fake repo + the same
// trivial PDF/Storage/Deliverer fakes the credit-note tests use.
type adminInvoiceHarness struct {
	handler *handler.AdminInvoiceHandler
	repo    *adminInvoiceFakeRepo
}

func newAdminInvoiceHarness(t *testing.T) *adminInvoiceHarness {
	t.Helper()
	repo := newAdminInvoiceFakeRepo()
	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    repo,
		Profiles:    newBPRepo(),
		PDF:         bpFakePDF{},
		Storage:     bpFakeStorage{},
		Deliverer:   bpFakeDeliverer{},
		Issuer:      domain.IssuerInfo{Country: "FR", LegalName: "Issuer SAS"},
		Idempotency: bpFakeIdempotency{},
	})
	return &adminInvoiceHarness{
		handler: handler.NewAdminInvoiceHandler(svc),
		repo:    repo,
	}
}

// ---- List ----

func TestAdminInvoiceList_HappyPath(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	orgID := uuid.New()
	rowID := uuid.New()
	h.repo.rows = []*repository.AdminInvoiceRow{
		{
			ID:                 rowID,
			Number:             "FAC-000123",
			IsCreditNote:       false,
			RecipientOrgID:     orgID,
			RecipientLegalName: "Acme Studio",
			IssuedAt:           time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
			AmountInclTaxCents: 4900,
			Currency:           "EUR",
			TaxRegime:          "fr_franchise_base",
			Status:             "issued",
			SourceType:         "subscription",
		},
	}
	h.repo.nextCursor = "next-page"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices", nil)
	rec := httptest.NewRecorder()
	h.handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Data []struct {
			ID                 string `json:"id"`
			Number             string `json:"number"`
			IsCreditNote       bool   `json:"is_credit_note"`
			RecipientLegalName string `json:"recipient_legal_name"`
			Status             string `json:"status"`
			SourceType         string `json:"source_type"`
		} `json:"data"`
		NextCursor string `json:"next_cursor"`
		HasMore    bool   `json:"has_more"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, rowID.String(), body.Data[0].ID)
	assert.Equal(t, "FAC-000123", body.Data[0].Number)
	assert.False(t, body.Data[0].IsCreditNote)
	assert.Equal(t, "Acme Studio", body.Data[0].RecipientLegalName)
	assert.Equal(t, "issued", body.Data[0].Status)
	assert.Equal(t, "subscription", body.Data[0].SourceType)
	assert.Equal(t, "next-page", body.NextCursor)
	assert.True(t, body.HasMore)
}

func TestAdminInvoiceList_ParsesAllFilters(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	orgID := uuid.New()
	url := "/api/v1/admin/invoices?" +
		"recipient_org_id=" + orgID.String() +
		"&status=monthly_commission" +
		"&date_from=2026-04-01T00:00:00Z" +
		"&date_to=2026-04-30T23:59:59Z" +
		"&min_amount_cents=1000" +
		"&max_amount_cents=50000" +
		"&search=Acme" +
		"&cursor=cur1" +
		"&limit=42"

	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	h.handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.NotNil(t, h.repo.listSeen.RecipientOrgID)
	assert.Equal(t, orgID, *h.repo.listSeen.RecipientOrgID)
	assert.Equal(t, "monthly_commission", h.repo.listSeen.Status)
	require.NotNil(t, h.repo.listSeen.DateFrom)
	require.NotNil(t, h.repo.listSeen.DateTo)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), *h.repo.listSeen.DateFrom)
	assert.Equal(t, time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC), *h.repo.listSeen.DateTo)
	require.NotNil(t, h.repo.listSeen.MinAmountCents)
	require.NotNil(t, h.repo.listSeen.MaxAmountCents)
	assert.Equal(t, int64(1000), *h.repo.listSeen.MinAmountCents)
	assert.Equal(t, int64(50000), *h.repo.listSeen.MaxAmountCents)
	assert.Equal(t, "Acme", h.repo.listSeen.Search)
	assert.Equal(t, "cur1", h.repo.listSeenCur)
	assert.Equal(t, 42, h.repo.listSeenLim)
}

func TestAdminInvoiceList_BadOrgIDReturns400(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices?recipient_org_id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.handler.List(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminInvoiceList_BadDateReturns400(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices?date_from=not-a-date", nil)
	rec := httptest.NewRecorder()
	h.handler.List(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminInvoiceList_DefaultLimit(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices", nil)
	rec := httptest.NewRecorder()
	h.handler.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 20, h.repo.listSeenLim, "default limit must be 20")
}

func TestAdminInvoiceList_LimitClampedAt100(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices?limit=999", nil)
	rec := httptest.NewRecorder()
	h.handler.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	// >100 falls back to default (20) per the handler logic.
	assert.Equal(t, 20, h.repo.listSeenLim)
}

// ---- GetPDF ----

func TestAdminInvoiceGetPDF_InvoiceRedirect(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	id := uuid.New()
	h.repo.invoices[id] = &domain.Invoice{ID: id, PDFR2Key: "invoices/x/FAC-1.pdf"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices/"+id.String()+"/pdf", nil)
	req = withChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()
	h.handler.GetPDF(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	loc := rec.Header().Get("Location")
	assert.Contains(t, loc, "invoices/x/FAC-1.pdf")
}

func TestAdminInvoiceGetPDF_CreditNoteRedirect(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	id := uuid.New()
	h.repo.creditNotes[id] = &domain.CreditNote{ID: id, PDFR2Key: "credit-notes/x/AV-1.pdf"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices/"+id.String()+"/pdf?type=credit_note", nil)
	req = withChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()
	h.handler.GetPDF(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "credit-notes/x/AV-1.pdf")
}

func TestAdminInvoiceGetPDF_BadTypeReturns400(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices/"+id.String()+"/pdf?type=garbage", nil)
	req = withChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()
	h.handler.GetPDF(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminInvoiceGetPDF_BadIDReturns400(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices/not-a-uuid/pdf", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.handler.GetPDF(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminInvoiceGetPDF_NotFoundReturns404(t *testing.T) {
	h := newAdminInvoiceHarness(t)
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices/"+id.String()+"/pdf", nil)
	req = withChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()
	h.handler.GetPDF(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---- Disabled service ----

func TestAdminInvoiceList_NilSvcReturns503(t *testing.T) {
	h := handler.NewAdminInvoiceHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestAdminInvoiceGetPDF_NilSvcReturns503(t *testing.T) {
	h := handler.NewAdminInvoiceHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/invoices/abc/pdf", nil)
	rec := httptest.NewRecorder()
	h.GetPDF(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
