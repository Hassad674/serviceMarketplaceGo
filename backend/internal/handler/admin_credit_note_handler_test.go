package handler_test

import (
	"bytes"
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
)

// adminCNHarness builds a real Service wired to the same in-memory repo
// the invoice handler test uses, so the admin manual flow exercises the
// full credit-note pipeline end-to-end inside the test process.
type adminCNHarness struct {
	handler  *handler.AdminCreditNoteHandler
	invoices *invFakeRepo
}

func newAdminCNHarness(t *testing.T) *adminCNHarness {
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
	return &adminCNHarness{
		handler:  handler.NewAdminCreditNoteHandler(svc),
		invoices: invoices,
	}
}

func (h *adminCNHarness) postIssue(t *testing.T, invoiceID uuid.UUID, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invoices/"+invoiceID.String()+"/credit-note", bytes.NewReader(raw))
	req = withChiURLParam(req, "id", invoiceID.String())
	rec := httptest.NewRecorder()
	h.handler.Issue(rec, req)
	return rec
}

func makeFinalizedHandlerInvoice(orgID uuid.UUID) *domain.Invoice {
	now := time.Now().UTC()
	finalized := now.Add(-1 * time.Hour)
	return &domain.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-000100",
		RecipientOrganizationID: orgID,
		RecipientSnapshot: domain.RecipientInfo{
			OrganizationID: orgID.String(),
			LegalName:      "Acme Studio SARL",
			Country:        "FR",
			Email:          "billing@acme.example",
		},
		IssuerSnapshot:     domain.IssuerInfo{LegalName: "Issuer SAS", Country: "FR"},
		IssuedAt:           now,
		Currency:           "EUR",
		AmountExclTaxCents: 4900,
		AmountInclTaxCents: 4900,
		TaxRegime:          domain.RegimeFRFranchiseBase,
		Status:             domain.StatusIssued,
		FinalizedAt:        &finalized,
		SourceType:         domain.SourceSubscription,
	}
}

// ---------- tests ----------

func TestAdminCreditNote_HappyPath_201WithViewModel(t *testing.T) {
	h := newAdminCNHarness(t)
	orgID := uuid.New()
	inv := makeFinalizedHandlerInvoice(orgID)
	h.invoices.byID[inv.ID] = inv

	rec := h.postIssue(t, inv.ID, map[string]any{
		"reason":       "Admin correction: wrong plan",
		"amount_cents": 4900,
	})

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	var body struct {
		ID                 string `json:"id"`
		Number             string `json:"number"`
		OriginalInvoiceID  string `json:"original_invoice_id"`
		AmountInclTaxCents int64  `json:"amount_incl_tax_cents"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.NotEmpty(t, body.ID)
	assert.Regexp(t, `^AV-\d{6,}$`, body.Number)
	assert.Equal(t, inv.ID.String(), body.OriginalInvoiceID)
	assert.Equal(t, int64(4900), body.AmountInclTaxCents)
}

func TestAdminCreditNote_InvalidAmount_400(t *testing.T) {
	h := newAdminCNHarness(t)
	orgID := uuid.New()
	inv := makeFinalizedHandlerInvoice(orgID)
	h.invoices.byID[inv.ID] = inv

	rec := h.postIssue(t, inv.ID, map[string]any{
		"reason":       "x",
		"amount_cents": 0,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminCreditNote_NegativeAmount_400(t *testing.T) {
	h := newAdminCNHarness(t)
	orgID := uuid.New()
	inv := makeFinalizedHandlerInvoice(orgID)
	h.invoices.byID[inv.ID] = inv

	rec := h.postIssue(t, inv.ID, map[string]any{
		"reason":       "x",
		"amount_cents": -100,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminCreditNote_MissingReason_400(t *testing.T) {
	h := newAdminCNHarness(t)
	orgID := uuid.New()
	inv := makeFinalizedHandlerInvoice(orgID)
	h.invoices.byID[inv.ID] = inv

	rec := h.postIssue(t, inv.ID, map[string]any{
		"amount_cents": 1000,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminCreditNote_InvoiceNotFound_404(t *testing.T) {
	h := newAdminCNHarness(t)
	rec := h.postIssue(t, uuid.New(), map[string]any{
		"reason":       "Stripe refund",
		"amount_cents": 1000,
	})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminCreditNote_InvalidUUID_400(t *testing.T) {
	h := newAdminCNHarness(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invoices/not-a-uuid/credit-note", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.handler.Issue(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminCreditNote_InvalidJSONBody_400(t *testing.T) {
	h := newAdminCNHarness(t)
	orgID := uuid.New()
	inv := makeFinalizedHandlerInvoice(orgID)
	h.invoices.byID[inv.ID] = inv

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invoices/"+inv.ID.String()+"/credit-note",
		bytes.NewReader([]byte("{not json")))
	req = withChiURLParam(req, "id", inv.ID.String())
	rec := httptest.NewRecorder()
	h.handler.Issue(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminCreditNote_NilService_503(t *testing.T) {
	h := &handler.AdminCreditNoteHandler{}
	// The handler's nil-svc guard kicks in before any work.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invoices/x/credit-note", nil)
	rec := httptest.NewRecorder()
	h.Issue(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
