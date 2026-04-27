package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	paymentapp "marketplace-backend/internal/app/payment"
	domain "marketplace-backend/internal/domain/invoicing"
	paymentdomain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

func timeNow() time.Time { return time.Now().UTC() }

// fakeKYCProbe is a tiny stub for the payout readiness gate. The
// wallet handler depends on a 1-method interface so we can drive the
// gate from a test without standing up the entire payment service.
type fakeKYCProbe struct {
	ready bool
	err   error
	calls int
}

func (f *fakeKYCProbe) CanProviderReceivePayouts(_ context.Context, _ uuid.UUID) (bool, error) {
	f.calls++
	return f.ready, f.err
}

// gatedWalletHarnessOpts wires the WalletHandler with a configurable
// KYC probe and an optional billing profile seed. The two gates are
// the only thing exercised here — the post-gate happy path requires
// the real payment service and is covered by the payment service's
// own suite.
type gatedWalletHarnessOpts struct {
	kycProbe   *fakeKYCProbe
	profileFix *domain.BillingProfile
}

func gatedWalletHarnessWith(t *testing.T, opts gatedWalletHarnessOpts) (*handler.WalletHandler, uuid.UUID, uuid.UUID) {
	t.Helper()
	profiles := newBPRepo()
	if opts.profileFix != nil {
		require.NoError(t, profiles.Upsert(context.Background(), opts.profileFix))
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
	wh := handler.NewWalletHandler(nil, nil).WithInvoicing(svc)
	if opts.kycProbe != nil {
		wh = wh.WithPayoutReadinessProbe(opts.kycProbe)
	}
	return wh, uuid.New(), uuid.New()
}

func walletAuthReq(method, target string, userID, orgID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	return req.WithContext(ctx)
}

// completeProfile returns a fully-populated FR business billing
// profile so the gate downstream of KYC has nothing to flag.
func completeProfile(orgID uuid.UUID) *domain.BillingProfile {
	now := timeNow()
	validated := now
	return &domain.BillingProfile{
		OrganizationID: orgID,
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
}

// TestRequestPayout_KYCIncomplete_403 — KYC probe returns not-ready
// → request blocks with `kyc_incomplete` BEFORE the billing-profile
// gate has a chance to run, even when the billing profile is also
// incomplete. The two gates are mutually exclusive: KYC wins.
func TestRequestPayout_KYCIncomplete_403(t *testing.T) {
	probe := &fakeKYCProbe{ready: false}
	wh, userID, orgID := gatedWalletHarnessWith(t, gatedWalletHarnessOpts{
		kycProbe: probe,
	})
	req := walletAuthReq(http.MethodPost, "/api/v1/wallet/payout", userID, orgID)
	rec := httptest.NewRecorder()

	wh.RequestPayout(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "kyc_incomplete", errObj["code"])
	assert.NotEmpty(t, errObj["message"])
	assert.Equal(t, "/payment-info", body["redirect"], "must hint the frontend where to redirect")
	assert.Equal(t, 1, probe.calls, "kyc probe must be called exactly once")
}

// TestRequestPayout_KYCProbeError_BlocksAsKYCIncomplete — when the
// readiness probe errors we treat it as not-ready (fail-closed) so the
// user gets an actionable "finish your KYC" message instead of a 500.
func TestRequestPayout_KYCProbeError_BlocksAsKYCIncomplete(t *testing.T) {
	probe := &fakeKYCProbe{err: errors.New("stripe API down")}
	wh, userID, orgID := gatedWalletHarnessWith(t, gatedWalletHarnessOpts{
		kycProbe: probe,
	})
	req := walletAuthReq(http.MethodPost, "/api/v1/wallet/payout", userID, orgID)
	rec := httptest.NewRecorder()

	wh.RequestPayout(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	errObj, _ := body["error"].(map[string]any)
	assert.Equal(t, "kyc_incomplete", errObj["code"])
}

// TestRequestPayout_KYCOK_BillingIncomplete_403 — once KYC is OK the
// billing-profile gate takes over and surfaces its own 403 with the
// `billing_profile_incomplete` discriminator code so the frontend can
// open the completion modal instead of redirecting to /payment-info.
func TestRequestPayout_KYCOK_BillingIncomplete_403(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	wh, userID, orgID := gatedWalletHarnessWith(t, gatedWalletHarnessOpts{
		kycProbe: probe,
		// no profileFix → billing profile is missing
	})
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
	assert.Equal(t, 1, probe.calls, "kyc probe must run before billing gate")
}

// TestRequestPayout_BothGatesPass_ReachesPaymentService — KYC ready
// AND billing profile complete → request flows past both gates and
// hits the (nil) payment service, panicking. The recovered panic is
// proof both gates let the request through; the post-gate behaviour
// is exercised by the payment service's own suite.
func TestRequestPayout_BothGatesPass_ReachesPaymentService(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	orgID := uuid.New()
	complete := completeProfile(orgID)
	wh, userID, _ := gatedWalletHarnessWith(t, gatedWalletHarnessOpts{
		kycProbe:   probe,
		profileFix: complete,
	})
	// IMPORTANT: drive the request with the SAME orgID the seeded
	// profile lives under, otherwise the gate sees no row and 403s.
	req := walletAuthReq(http.MethodPost, "/api/v1/wallet/payout", userID, complete.OrganizationID)
	rec := httptest.NewRecorder()

	defer func() {
		r := recover()
		assert.NotNil(t, r, "complete KYC + profile must let the request reach the payment service")
		assert.Equal(t, 1, probe.calls)
	}()
	wh.RequestPayout(rec, req)
}

// ---------------------------------------------------------------------------
// RetryFailedTransfer handler — error → HTTP status mapping
// ---------------------------------------------------------------------------

// fakeRetrier drives every RetryFailedTransfer branch without booting
// the real payment service. We assert what the handler does with each
// well-known sentinel + with a generic upstream error (502).
type fakeRetrier struct {
	result *paymentapp.PayoutResult
	err    error
	calls  int
}

func (f *fakeRetrier) RetryFailedTransfer(_ context.Context, _, _, _ uuid.UUID) (*paymentapp.PayoutResult, error) {
	f.calls++
	return f.result, f.err
}

// retryReq builds an authenticated retry request with a chi URL param
// for the record id. Tests can supply a malformed string to hit the
// 400 bad-request branch.
func retryReq(t *testing.T, recordID string, userID, orgID uuid.UUID) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/transfers/"+recordID+"/retry", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("record_id", recordID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestRetryFailedTransfer_HappyPath_200(t *testing.T) {
	retrier := &fakeRetrier{result: &paymentapp.PayoutResult{Status: "transferred", Message: "OK"}}
	wh := handler.NewWalletHandler(nil, nil).WithTransferRetrier(retrier)

	req := retryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryFailedTransfer(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, retrier.calls)
}

// decodeRetryError reads the flat error envelope produced by
// pkg/response.Error: {"error": "code", "message": "..."}.
// Returns the code so individual tests can assert against it.
func decodeRetryError(t *testing.T, rec *httptest.ResponseRecorder) (code, message string) {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	c, _ := body["error"].(string)
	m, _ := body["message"].(string)
	return c, m
}

func TestRetryFailedTransfer_KYCIncomplete_412(t *testing.T) {
	retrier := &fakeRetrier{err: paymentdomain.ErrProviderPayoutsDisabled}
	wh := handler.NewWalletHandler(nil, nil).WithTransferRetrier(retrier)

	req := retryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryFailedTransfer(rec, req)

	assert.Equal(t, http.StatusPreconditionFailed, rec.Code)
	code, msg := decodeRetryError(t, rec)
	assert.Equal(t, "provider_kyc_incomplete", code)
	assert.Contains(t, msg, "Stripe")
}

func TestRetryFailedTransfer_NotProviderOrg_409(t *testing.T) {
	// The service maps "caller's org != provider's org" to
	// ErrTransferNotRetriable so the handler returns 409. This is the
	// cross-tenant defence — the attacker sees the same response code
	// as a "mission still active" caller, no oracle on ownership.
	retrier := &fakeRetrier{err: paymentdomain.ErrTransferNotRetriable}
	wh := handler.NewWalletHandler(nil, nil).WithTransferRetrier(retrier)

	req := retryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryFailedTransfer(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	code, _ := decodeRetryError(t, rec)
	assert.Equal(t, "transfer_not_retriable", code)
}

func TestRetryFailedTransfer_RecordNotFound_404(t *testing.T) {
	retrier := &fakeRetrier{err: paymentdomain.ErrPaymentRecordNotFound}
	wh := handler.NewWalletHandler(nil, nil).WithTransferRetrier(retrier)

	req := retryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryFailedTransfer(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	code, _ := decodeRetryError(t, rec)
	assert.Equal(t, "payment_record_not_found", code)
}

func TestRetryFailedTransfer_StripeUpstreamError_502(t *testing.T) {
	// Anything that isn't a known sentinel maps to 502 — the upstream
	// failed and the user can retry. NEVER bury this as a 500: the
	// client must see "try again later", not "permanent server error".
	retrier := &fakeRetrier{err: errors.New("stripe: retry stripe transfer: rate limited")}
	wh := handler.NewWalletHandler(nil, nil).WithTransferRetrier(retrier)

	req := retryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryFailedTransfer(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	code, _ := decodeRetryError(t, rec)
	assert.Equal(t, "retry_failed", code)
}

func TestRetryFailedTransfer_BadRecordID_400(t *testing.T) {
	retrier := &fakeRetrier{}
	wh := handler.NewWalletHandler(nil, nil).WithTransferRetrier(retrier)

	req := retryReq(t, "not-a-uuid", uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryFailedTransfer(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 0, retrier.calls, "retrier must not be called on a malformed id")
}

func TestRetryFailedTransfer_NoStripeAccount_403(t *testing.T) {
	retrier := &fakeRetrier{err: paymentdomain.ErrStripeAccountNotFound}
	wh := handler.NewWalletHandler(nil, nil).WithTransferRetrier(retrier)

	req := retryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryFailedTransfer(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	code, _ := decodeRetryError(t, rec)
	assert.Equal(t, "stripe_account_missing", code)
}
