package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// ---------- in-memory invoice repo ----------

type invFakeRepo struct {
	byID  map[uuid.UUID]*domain.Invoice
	byOrg map[uuid.UUID][]*domain.Invoice

	// current-month data — simple: always returned in full.
	released []repository.ReleasedPaymentRecord
}

func newInvRepo() *invFakeRepo {
	return &invFakeRepo{
		byID:  map[uuid.UUID]*domain.Invoice{},
		byOrg: map[uuid.UUID][]*domain.Invoice{},
	}
}

func (r *invFakeRepo) CreateInvoice(_ context.Context, _ *domain.Invoice) error    { return nil }
func (r *invFakeRepo) CreateCreditNote(_ context.Context, _ *domain.CreditNote) error { return nil }
func (r *invFakeRepo) ReserveNumber(_ context.Context, _ domain.CounterScope) (int64, error) {
	return 0, nil
}
func (r *invFakeRepo) FindInvoiceByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if inv, ok := r.byID[id]; ok {
		return inv, nil
	}
	return nil, domain.ErrNotFound
}
func (r *invFakeRepo) FindInvoiceByStripeEventID(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, domain.ErrNotFound
}
func (r *invFakeRepo) FindCreditNoteByStripeEventID(_ context.Context, _ string) (*domain.CreditNote, error) {
	return nil, domain.ErrNotFound
}
func (r *invFakeRepo) FindInvoiceByStripePaymentIntentID(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, domain.ErrNotFound
}
func (r *invFakeRepo) MarkInvoiceCredited(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (r *invFakeRepo) ListInvoicesByOrganization(_ context.Context, organizationID uuid.UUID, cursor string, limit int) ([]*domain.Invoice, string, error) {
	all := r.byOrg[organizationID]
	// naive pagination: cursor is the index as a string for tests.
	start := 0
	if cursor != "" {
		// integer cursor for simplicity in tests
		for i, inv := range all {
			if inv.ID.String() == cursor {
				start = i + 1
				break
			}
		}
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	page := all[start:end]
	next := ""
	if end < len(all) {
		next = page[len(page)-1].ID.String()
	}
	return page, next, nil
}
func (r *invFakeRepo) HasInvoiceItemForPaymentRecord(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (r *invFakeRepo) ListReleasedPaymentRecordsForOrg(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
	return r.released, nil
}
func (r *invFakeRepo) ListInvoicesAdmin(_ context.Context, _ repository.AdminInvoiceFilters, _ string, _ int) ([]*repository.AdminInvoiceRow, string, error) {
	return nil, "", nil
}
func (r *invFakeRepo) FindCreditNoteByID(_ context.Context, _ uuid.UUID) (*domain.CreditNote, error) {
	return nil, domain.ErrNotFound
}

// ---------- harness ----------

type invHarness struct {
	handler  *handler.InvoiceHandler
	invoices *invFakeRepo
	userID   uuid.UUID
	orgID    uuid.UUID
}

func newInvHarness(t *testing.T) *invHarness {
	t.Helper()
	invoices := newInvRepo()
	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    invoices,
		Profiles:    newBPRepo(),
		PDF:         bpFakePDF{},
		Storage:     bpFakeStorage{},
		Deliverer:   bpFakeDeliverer{},
		Issuer:      domain.IssuerInfo{Country: "FR", LegalName: "Issuer SAS"},
		Idempotency: bpFakeIdempotency{},
	})
	return &invHarness{
		handler:  handler.NewInvoiceHandler(svc),
		invoices: invoices,
		userID:   uuid.New(),
		orgID:    uuid.New(),
	}
}

func (h *invHarness) reqAuth(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, h.userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, h.orgID)
	req = req.WithContext(ctx)
	return req
}

// withChiURLParam adds chi URL params to a request so {id} resolves
// during direct handler invocations (no real chi router in the test).
func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func makeInvoice(t *testing.T, orgID uuid.UUID, number string, issued time.Time) *domain.Invoice {
	t.Helper()
	return &domain.Invoice{
		ID:                      uuid.New(),
		Number:                  number,
		RecipientOrganizationID: orgID,
		IssuedAt:                issued,
		Currency:                "EUR",
		AmountInclTaxCents:      1990,
		SourceType:              domain.SourceSubscription,
		Status:                  domain.StatusIssued,
		PDFR2Key:                "invoices/" + orgID.String() + "/" + number + ".pdf",
	}
}

// ---------- tests ----------

func TestInvoice_List_Paginates(t *testing.T) {
	h := newInvHarness(t)
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		inv := makeInvoice(t, h.orgID, "FAC-00000"+string(rune('1'+i)), now.Add(-time.Duration(i)*time.Hour))
		h.invoices.byOrg[h.orgID] = append(h.invoices.byOrg[h.orgID], inv)
		h.invoices.byID[inv.ID] = inv
	}
	req := h.reqAuth(http.MethodGet, "/api/v1/me/invoices?limit=2")
	rec := httptest.NewRecorder()
	h.handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data       []map[string]any `json:"data"`
		NextCursor string           `json:"next_cursor"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Len(t, body.Data, 2)
	assert.NotEmpty(t, body.NextCursor)
	// PDF URL must be empty in the list (clients call /pdf to fetch a
	// fresh signed URL; we never expose a long-lived URL via list).
	assert.Equal(t, "", body.Data[0]["pdf_url"])
}

func TestInvoice_GetPDF_RedirectsToPresignedURL(t *testing.T) {
	h := newInvHarness(t)
	inv := makeInvoice(t, h.orgID, "FAC-000001", time.Now().UTC())
	h.invoices.byID[inv.ID] = inv

	req := h.reqAuth(http.MethodGet, "/api/v1/me/invoices/"+inv.ID.String()+"/pdf")
	req = withChiURLParam(req, "id", inv.ID.String())
	rec := httptest.NewRecorder()
	h.handler.GetPDF(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	loc := rec.Result().Header.Get("Location")
	assert.Contains(t, loc, "download/", "must redirect to a presigned download URL")
	// Drain body.
	_, _ = io.ReadAll(rec.Body)
}

func TestInvoice_GetPDF_OtherOrg_403(t *testing.T) {
	h := newInvHarness(t)
	otherOrg := uuid.New()
	inv := makeInvoice(t, otherOrg, "FAC-000001", time.Now().UTC())
	h.invoices.byID[inv.ID] = inv

	req := h.reqAuth(http.MethodGet, "/api/v1/me/invoices/"+inv.ID.String()+"/pdf")
	req = withChiURLParam(req, "id", inv.ID.String())
	rec := httptest.NewRecorder()
	h.handler.GetPDF(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestInvoice_CurrentMonth_HappyPath(t *testing.T) {
	h := newInvHarness(t)
	now := time.Now().UTC()
	h.invoices.released = []repository.ReleasedPaymentRecord{
		{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 100000,
			PlatformFeeCents:    5000,
			Currency:            "EUR",
			TransferredAt:       now,
		},
		{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 50000,
			PlatformFeeCents:    2500,
			Currency:            "EUR",
			TransferredAt:       now,
		},
	}
	req := h.reqAuth(http.MethodGet, "/api/v1/me/invoicing/current-month")
	rec := httptest.NewRecorder()
	h.handler.CurrentMonth(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		MilestoneCount int   `json:"milestone_count"`
		TotalFeeCents  int64 `json:"total_fee_cents"`
		Lines          []any `json:"lines"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, 2, body.MilestoneCount)
	assert.Equal(t, int64(7500), body.TotalFeeCents)
	assert.Len(t, body.Lines, 2)
}

func TestInvoice_CurrentMonth_EmptyState(t *testing.T) {
	h := newInvHarness(t)
	// No released records.
	req := h.reqAuth(http.MethodGet, "/api/v1/me/invoicing/current-month")
	rec := httptest.NewRecorder()
	h.handler.CurrentMonth(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		MilestoneCount int   `json:"milestone_count"`
		TotalFeeCents  int64 `json:"total_fee_cents"`
		Lines          []any `json:"lines"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, 0, body.MilestoneCount)
	assert.Equal(t, int64(0), body.TotalFeeCents)
	assert.Empty(t, body.Lines)
}
