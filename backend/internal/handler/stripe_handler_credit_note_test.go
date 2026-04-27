package handler

import (
	"context"
	"net/http/httptest"
	"sync"
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

// refundTracking captures the side effects of a charge.refunded event so
// tests can assert "credit note was issued" without re-decoding logs or
// touching real Postgres / R2.
type refundTracking struct {
	mu sync.Mutex

	creditNotePersists int
	persistedCN        *invoicing.CreditNote
	markedCreditedIDs  []uuid.UUID

	// Map of pi_id -> invoice. Empty map → all lookups return ErrNotFound.
	invoicesByPI map[string]*invoicing.Invoice

	// Tracks idempotency claims so we can simulate a replay.
	claimed map[string]bool
}

func newRefundTracking() *refundTracking {
	return &refundTracking{
		invoicesByPI: map[string]*invoicing.Invoice{},
		claimed:      map[string]bool{},
	}
}

// ---- ports backed by the tracking object ----

type refundFakeRepo struct{ t *refundTracking }

func (f *refundFakeRepo) CreateInvoice(_ context.Context, _ *invoicing.Invoice) error { return nil }
func (f *refundFakeRepo) CreateCreditNote(_ context.Context, cn *invoicing.CreditNote) error {
	f.t.mu.Lock()
	defer f.t.mu.Unlock()
	f.t.creditNotePersists++
	f.t.persistedCN = cn
	return nil
}
func (f *refundFakeRepo) ReserveNumber(_ context.Context, _ invoicing.CounterScope) (int64, error) {
	return 1, nil
}
func (f *refundFakeRepo) FindInvoiceByID(_ context.Context, id uuid.UUID) (*invoicing.Invoice, error) {
	f.t.mu.Lock()
	defer f.t.mu.Unlock()
	for _, inv := range f.t.invoicesByPI {
		if inv.ID == id {
			return inv, nil
		}
	}
	return nil, invoicing.ErrNotFound
}
func (f *refundFakeRepo) FindInvoiceByStripeEventID(_ context.Context, _ string) (*invoicing.Invoice, error) {
	return nil, invoicing.ErrNotFound
}
func (f *refundFakeRepo) FindCreditNoteByStripeEventID(_ context.Context, _ string) (*invoicing.CreditNote, error) {
	return nil, invoicing.ErrNotFound
}
func (f *refundFakeRepo) FindInvoiceByStripePaymentIntentID(_ context.Context, pi string) (*invoicing.Invoice, error) {
	f.t.mu.Lock()
	defer f.t.mu.Unlock()
	if inv, ok := f.t.invoicesByPI[pi]; ok {
		return inv, nil
	}
	return nil, invoicing.ErrNotFound
}
func (f *refundFakeRepo) MarkInvoiceCredited(_ context.Context, id uuid.UUID) error {
	f.t.mu.Lock()
	defer f.t.mu.Unlock()
	f.t.markedCreditedIDs = append(f.t.markedCreditedIDs, id)
	return nil
}
func (f *refundFakeRepo) ListInvoicesByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*invoicing.Invoice, string, error) {
	return nil, "", nil
}
func (f *refundFakeRepo) HasInvoiceItemForPaymentRecord(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (f *refundFakeRepo) ListReleasedPaymentRecordsForOrg(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
	return nil, nil
}
func (f *refundFakeRepo) ListInvoicesAdmin(_ context.Context, _ repository.AdminInvoiceFilters, _ string, _ int) ([]*repository.AdminInvoiceRow, string, error) {
	return nil, "", nil
}
func (f *refundFakeRepo) FindCreditNoteByID(_ context.Context, _ uuid.UUID) (*invoicing.CreditNote, error) {
	return nil, invoicing.ErrNotFound
}

type refundFakeIdem struct{ t *refundTracking }

func (f *refundFakeIdem) TryClaim(_ context.Context, eventID string) (bool, error) {
	f.t.mu.Lock()
	defer f.t.mu.Unlock()
	if f.t.claimed[eventID] {
		return false, nil
	}
	f.t.claimed[eventID] = true
	return true, nil
}

// ---- helpers ----

func newRefundHandler(t *testing.T, tracking *refundTracking) *StripeHandler {
	t.Helper()
	repo := &refundFakeRepo{t: tracking}
	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    repo,
		Profiles:    &handlerFakeProfileRepo{},
		PDF:         &handlerFakePDF{tracking: &trackingFakes{}},
		Storage:     &handlerFakeStorage{tracking: &trackingFakes{}},
		Deliverer:   &handlerFakeDeliverer{tracking: &trackingFakes{}},
		Idempotency: &refundFakeIdem{t: tracking},
		Issuer: invoicing.IssuerInfo{
			LegalName: "Test Issuer SAS",
			SIRET:     "00000000000000",
			Country:   "FR",
		},
	})
	return (&StripeHandler{}).WithInvoicing(svc)
}

func newSubscriptionInvoice(orgID uuid.UUID, amount int64) *invoicing.Invoice {
	now := time.Now().UTC()
	finalized := now.Add(-1 * time.Hour)
	return &invoicing.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-000200",
		RecipientOrganizationID: orgID,
		RecipientSnapshot: invoicing.RecipientInfo{
			OrganizationID: orgID.String(),
			LegalName:      "Acme",
			Country:        "FR",
			Email:          "billing@acme.example",
		},
		IssuerSnapshot:     invoicing.IssuerInfo{LegalName: "Issuer", Country: "FR"},
		IssuedAt:           now,
		Currency:           "EUR",
		AmountExclTaxCents: amount,
		AmountInclTaxCents: amount,
		TaxRegime:          invoicing.RegimeFRFranchiseBase,
		Status:             invoicing.StatusIssued,
		FinalizedAt:        &finalized,
		SourceType:         invoicing.SourceSubscription,
	}
}

// ---------- tests ----------

func TestHandleChargeRefunded_DispatchesAndMarksCredited(t *testing.T) {
	tracking := newRefundTracking()
	orgID := uuid.New()
	original := newSubscriptionInvoice(orgID, 4900)
	tracking.invoicesByPI["pi_refunded_001"] = original

	h := newRefundHandler(t, tracking)
	event := &portservice.StripeWebhookEvent{
		EventID:                   "evt_refund_001",
		Type:                      "charge.refunded",
		ChargeRefunded:            true,
		ChargeID:                  "ch_001",
		ChargePaymentIntentID:     "pi_refunded_001",
		ChargeAmountRefundedCents: 4900,
		ChargeRefundID:            "re_001",
	}
	r := httptest.NewRequest("POST", "/", nil)

	h.handleChargeRefunded(r, event)

	assert.Equal(t, 1, tracking.creditNotePersists, "credit note must be persisted exactly once")
	require.NotNil(t, tracking.persistedCN)
	assert.Equal(t, original.ID, tracking.persistedCN.OriginalInvoiceID)
	assert.Equal(t, int64(4900), tracking.persistedCN.AmountInclTaxCents)
	assert.Equal(t, "Stripe refund", tracking.persistedCN.Reason)
	assert.Equal(t, "evt_refund_001", tracking.persistedCN.StripeEventID)
	assert.Equal(t, "re_001", tracking.persistedCN.StripeRefundID)
	require.Len(t, tracking.markedCreditedIDs, 1, "full refund must mark original credited")
	assert.Equal(t, original.ID, tracking.markedCreditedIDs[0])
}

func TestHandleChargeRefunded_NoMatchingInvoice_LogsAndSkips(t *testing.T) {
	tracking := newRefundTracking()
	h := newRefundHandler(t, tracking)
	event := &portservice.StripeWebhookEvent{
		EventID:                   "evt_unmatched",
		Type:                      "charge.refunded",
		ChargeRefunded:            true,
		ChargePaymentIntentID:     "pi_orphan",
		ChargeAmountRefundedCents: 1000,
	}
	r := httptest.NewRequest("POST", "/", nil)

	h.handleChargeRefunded(r, event)

	assert.Equal(t, 0, tracking.creditNotePersists, "no credit note when invoice match fails")
	assert.Empty(t, tracking.markedCreditedIDs)
}

func TestHandleChargeRefunded_IdempotentReplay_NoDuplicate(t *testing.T) {
	tracking := newRefundTracking()
	orgID := uuid.New()
	original := newSubscriptionInvoice(orgID, 4900)
	tracking.invoicesByPI["pi_replay_001"] = original

	h := newRefundHandler(t, tracking)
	event := &portservice.StripeWebhookEvent{
		EventID:                   "evt_replay_001",
		Type:                      "charge.refunded",
		ChargeRefunded:            true,
		ChargePaymentIntentID:     "pi_replay_001",
		ChargeAmountRefundedCents: 4900,
	}
	r := httptest.NewRequest("POST", "/", nil)

	h.handleChargeRefunded(r, event)
	h.handleChargeRefunded(r, event) // replay same event id

	assert.Equal(t, 1, tracking.creditNotePersists, "replay must NOT create a second credit note")
	assert.Len(t, tracking.markedCreditedIDs, 1, "replay must NOT mark credited twice")
}

func TestHandleChargeRefunded_NoOpWhenInvoicingDisabled(t *testing.T) {
	h := &StripeHandler{} // no WithInvoicing call
	event := &portservice.StripeWebhookEvent{
		EventID:               "evt_disabled",
		Type:                  "charge.refunded",
		ChargeRefunded:        true,
		ChargePaymentIntentID: "pi_x",
	}
	r := httptest.NewRequest("POST", "/", nil)

	// Must not panic, must not produce side effects when feature is off.
	h.handleChargeRefunded(r, event)
}

func TestHandleChargeRefunded_NoOpWhenNoPaymentIntent(t *testing.T) {
	tracking := newRefundTracking()
	h := newRefundHandler(t, tracking)
	event := &portservice.StripeWebhookEvent{
		EventID:        "evt_no_pi",
		Type:           "charge.refunded",
		ChargeRefunded: true,
	}
	r := httptest.NewRequest("POST", "/", nil)

	h.handleChargeRefunded(r, event)

	assert.Equal(t, 0, tracking.creditNotePersists)
}
