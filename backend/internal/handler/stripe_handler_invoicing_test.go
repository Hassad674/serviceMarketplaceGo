package handler

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// trackingFakes is a tiny dependency kit that lets us instantiate a
// real invoicingapp.Service against in-memory ports and observe what
// the handler triggered. Same-package test (handler internal) so we
// can call h.handleInvoicePaid directly without reverse-engineering
// the signed webhook envelope.
type trackingFakes struct {
	persistCalls int
	pdfCalls     int
	uploadCalls  int
	deliveryCalls int

	persistedInvoice *invoicing.Invoice
}

func newTrackedInvoicingService(t *testing.T, tracking *trackingFakes) *invoicingapp.Service {
	t.Helper()
	invRepo := &handlerFakeInvoiceRepo{tracking: tracking}
	profileRepo := &handlerFakeProfileRepo{}
	pdf := &handlerFakePDF{tracking: tracking}
	storage := &handlerFakeStorage{tracking: tracking}
	deliverer := &handlerFakeDeliverer{tracking: tracking}
	idem := &handlerFakeIdem{}
	return invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    invRepo,
		Profiles:    profileRepo,
		PDF:         pdf,
		Storage:     storage,
		Deliverer:   deliverer,
		Idempotency: idem,
		Issuer: invoicing.IssuerInfo{
			LegalName:    "Test Issuer SAS",
			SIRET:        "00000000000000",
			AddressLine1: "1 rue Issuer",
			PostalCode:   "75002",
			City:         "Paris",
			Country:      "FR",
			Email:        "billing@issuer.example",
		},
	})
}

func TestHandleInvoicePaid_DispatchesToInvoicingService(t *testing.T) {
	tracking := &trackingFakes{}
	invSvc := newTrackedInvoicingService(t, tracking)
	h := (&StripeHandler{}).WithInvoicing(invSvc)

	orgID := uuid.New()
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	event := &portservice.StripeWebhookEvent{
		EventID:                  "evt_dispatch_001",
		Type:                     "invoice.paid",
		InvoicePaid:              true,
		InvoiceID:                "in_dispatch_001",
		InvoicePaymentIntentID:   "pi_dispatch_001",
		InvoiceAmountPaidCents:   4900,
		InvoiceCurrency:          "EUR",
		InvoicePeriodStart:       periodStart,
		InvoicePeriodEnd:         periodStart.AddDate(0, 1, 0),
		InvoiceLineDescription:   "Premium Agence — avril 2026",
		InvoiceSubscriptionID:    "sub_dispatch_001",
		InvoiceSubscriptionOrgID: orgID.String(),
	}
	r := httptest.NewRequest("POST", "/", nil)

	h.handleInvoicePaid(r, event)

	assert.Equal(t, 1, tracking.pdfCalls, "pdf renderer must be called exactly once")
	assert.Equal(t, 1, tracking.uploadCalls, "r2 upload must be called exactly once")
	assert.Equal(t, 1, tracking.persistCalls, "invoice must be persisted exactly once")
	assert.Equal(t, 1, tracking.deliveryCalls, "email must be sent exactly once")
	require.NotNil(t, tracking.persistedInvoice)
	assert.Equal(t, orgID, tracking.persistedInvoice.RecipientOrganizationID)
	assert.Equal(t, "in_dispatch_001", tracking.persistedInvoice.StripeInvoiceID)
	assert.Equal(t, "evt_dispatch_001", tracking.persistedInvoice.StripeEventID)
	assert.Equal(t, int64(4900), tracking.persistedInvoice.AmountInclTaxCents)
	require.Len(t, tracking.persistedInvoice.Items, 1)
	assert.Equal(t, "Premium Agence — avril 2026", tracking.persistedInvoice.Items[0].Description)
}

func TestHandleInvoicePaid_NoOpWhenInvoicingDisabled(t *testing.T) {
	h := &StripeHandler{} // no WithInvoicing call

	event := &portservice.StripeWebhookEvent{
		EventID:                  "evt_disabled",
		Type:                     "invoice.paid",
		InvoicePaid:              true,
		InvoiceSubscriptionID:    "sub_disabled",
		InvoiceSubscriptionOrgID: uuid.NewString(),
	}
	r := httptest.NewRequest("POST", "/", nil)

	// Must not panic, must not produce side effects when feature is
	// off — the only assertion is that the call is a clean no-op.
	h.handleInvoicePaid(r, event)
}

func TestHandleInvoicePaid_NoOpWhenNotSubscriptionInvoice(t *testing.T) {
	tracking := &trackingFakes{}
	invSvc := newTrackedInvoicingService(t, tracking)
	h := (&StripeHandler{}).WithInvoicing(invSvc)

	// invoice.paid with no subscription id → manual one-off invoice,
	// out of scope for the FAC pipeline. The handler must short-
	// circuit before any work.
	event := &portservice.StripeWebhookEvent{
		EventID:     "evt_oneoff",
		Type:        "invoice.paid",
		InvoicePaid: true,
		InvoiceID:   "in_oneoff",
	}
	r := httptest.NewRequest("POST", "/", nil)

	h.handleInvoicePaid(r, event)

	assert.Equal(t, 0, tracking.pdfCalls, "non-subscription invoice must not reach the invoicing service")
	assert.Equal(t, 0, tracking.persistCalls)
	assert.Equal(t, 0, tracking.uploadCalls)
}

// ---------- fakes ----------

type handlerFakeInvoiceRepo struct {
	tracking *trackingFakes
}

func (f *handlerFakeInvoiceRepo) CreateInvoice(_ context.Context, inv *invoicing.Invoice) error {
	f.tracking.persistCalls++
	f.tracking.persistedInvoice = inv
	return nil
}
func (f *handlerFakeInvoiceRepo) CreateCreditNote(_ context.Context, _ *invoicing.CreditNote) error {
	return nil
}
func (f *handlerFakeInvoiceRepo) ReserveNumber(_ context.Context, _ invoicing.CounterScope) (int64, error) {
	return 1, nil
}
func (f *handlerFakeInvoiceRepo) FindInvoiceByID(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
	return nil, invoicing.ErrNotFound
}
func (f *handlerFakeInvoiceRepo) FindInvoiceByStripeEventID(_ context.Context, _ string) (*invoicing.Invoice, error) {
	return nil, invoicing.ErrNotFound
}
func (f *handlerFakeInvoiceRepo) FindCreditNoteByStripeEventID(_ context.Context, _ string) (*invoicing.CreditNote, error) {
	return nil, invoicing.ErrNotFound
}
func (f *handlerFakeInvoiceRepo) ListInvoicesByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*invoicing.Invoice, string, error) {
	return nil, "", nil
}
func (f *handlerFakeInvoiceRepo) HasInvoiceItemForPaymentRecord(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (f *handlerFakeInvoiceRepo) ListReleasedPaymentRecordsForOrg(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
	return nil, nil
}

type handlerFakeProfileRepo struct{}

func (f *handlerFakeProfileRepo) FindByOrganization(_ context.Context, orgID uuid.UUID) (*invoicing.BillingProfile, error) {
	return &invoicing.BillingProfile{
		OrganizationID: orgID,
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "Acme",
		AddressLine1:   "1 rue Test",
		PostalCode:     "75001",
		City:           "Paris",
		Country:        "FR",
		TaxID:          "12345678900012",
		InvoicingEmail: "ap@acme.example",
	}, nil
}
func (f *handlerFakeProfileRepo) Upsert(_ context.Context, _ *invoicing.BillingProfile) error {
	return nil
}

type handlerFakePDF struct{ tracking *trackingFakes }

func (f *handlerFakePDF) RenderInvoice(_ context.Context, _ *invoicing.Invoice, _ string) ([]byte, error) {
	f.tracking.pdfCalls++
	return []byte("%PDF"), nil
}
func (f *handlerFakePDF) RenderCreditNote(_ context.Context, _ *invoicing.CreditNote, _ string) ([]byte, error) {
	return []byte("%PDF"), nil
}

type handlerFakeStorage struct{ tracking *trackingFakes }

func (f *handlerFakeStorage) Upload(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
	f.tracking.uploadCalls++
	return "https://r2.test/" + key, nil
}
func (f *handlerFakeStorage) Delete(_ context.Context, _ string) error           { return nil }
func (f *handlerFakeStorage) GetPublicURL(key string) string                     { return "https://r2.test/" + key }
func (f *handlerFakeStorage) GetPresignedUploadURL(_ context.Context, key string, _ string, _ time.Duration) (string, error) {
	return "https://r2.test/upload/" + key, nil
}
func (f *handlerFakeStorage) Download(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

type handlerFakeDeliverer struct{ tracking *trackingFakes }

func (f *handlerFakeDeliverer) DeliverInvoice(_ context.Context, _ *invoicing.Invoice, _ []byte, _ string) error {
	f.tracking.deliveryCalls++
	return nil
}
func (f *handlerFakeDeliverer) DeliverCreditNote(_ context.Context, _ *invoicing.CreditNote, _ []byte, _ string) error {
	return nil
}

type handlerFakeIdem struct{}

func (f *handlerFakeIdem) TryClaim(_ context.Context, _ string) (bool, error) {
	return true, nil
}
