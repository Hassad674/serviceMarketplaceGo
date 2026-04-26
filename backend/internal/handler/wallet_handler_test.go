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
	"marketplace-backend/internal/handler/middleware"
)

func timeNow() time.Time { return time.Now().UTC() }

// gatedWalletHarness wires a WalletHandler with the invoicing gate
// only. The payment service stays nil — the tests below exercise ONLY
// the gate path, which short-circuits before any payment call. Reaching
// the real payment branch requires deps not available in unit tests
// (KYC, transfers, stripe). Tests for the post-gate behaviour live in
// the payment service's own suite.
func gatedWalletHarness(t *testing.T, profileSeed *domain.BillingProfile) (*handler.WalletHandler, uuid.UUID, uuid.UUID) {
	t.Helper()
	profiles := newBPRepo()
	if profileSeed != nil {
		require.NoError(t, profiles.Upsert(context.Background(), profileSeed))
	}
	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    bpFakeInvoiceRepo{},
		Profiles:    profiles,
		PDF:         bpFakePDF{},
		Storage:     bpFakeStorage{},
		Deliverer:   bpFakeDeliverer{},
		Issuer:      domain.IssuerInfo{Country: "FR", LegalName: "Issuer SAS"},
		Idempotency: bpFakeIdempotency{},
	})
	// payment + proposal services unused on the gate-fail path.
	w := handler.NewWalletHandler(nil, nil).WithInvoicing(svc)
	return w, uuid.New(), uuid.New()
}

func walletAuthReq(method, target string, userID, orgID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	return req.WithContext(ctx)
}

func TestRequestPayout_BillingProfileIncomplete_403(t *testing.T) {
	wh, userID, orgID := gatedWalletHarness(t, nil) // no seed → empty stub → incomplete
	req := walletAuthReq(http.MethodPost, "/api/v1/wallet/payout", userID, orgID)
	rec := httptest.NewRecorder()

	wh.RequestPayout(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "billing_profile_incomplete", errObj["code"])
	missing, _ := body["missing_fields"].([]any)
	assert.NotEmpty(t, missing, "must include missing fields list for the modal")
}

func TestRequestPayout_BillingProfileComplete_PassesGate(t *testing.T) {
	// Complete FR business profile — every domain.CheckCompleteness rule
	// satisfied so the gate lets the request through. We expect a panic
	// (nil paymentSvc) BUT only after the gate, which proves the gate
	// is non-blocking on a complete profile.
	now := timeNow()
	validated := now
	complete := &domain.BillingProfile{
		OrganizationID: uuid.New(),
		ProfileType:    domain.ProfileBusiness,
		LegalName:      "Acme SAS",
		AddressLine1:   "1 rue de la Paix",
		PostalCode:     "75002",
		City:           "Paris",
		Country:        "FR",
		InvoicingEmail: "billing@acme.test",
		TaxID:          "12345678901234",
		VATNumber:      "FR12345678901",
		VATValidatedAt: &validated,
	}
	wh, userID, _ := gatedWalletHarness(t, complete)
	// IMPORTANT: drive the request with the SAME orgID the seeded
	// profile lives under, otherwise the gate sees no row and 403s.
	req := walletAuthReq(http.MethodPost, "/api/v1/wallet/payout", userID, complete.OrganizationID)
	rec := httptest.NewRecorder()

	// We expect a nil-deref panic from the missing payment service —
	// proves the request passed the gate. Recover and assert.
	defer func() {
		r := recover()
		assert.NotNil(t, r, "complete profile must let the request reach the payment service")
	}()
	wh.RequestPayout(rec, req)
}
