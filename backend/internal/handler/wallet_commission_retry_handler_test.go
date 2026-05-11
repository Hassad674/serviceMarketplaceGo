package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referralapp "marketplace-backend/internal/app/referral"
	referraldomain "marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
)

// fakeCommissionRetrier drives every RetryCommission branch from the
// handler tests without standing up the referral service.
type fakeCommissionRetrier struct {
	outcome portservice.ReferralCommissionRetryOutcome
	err     error
	calls   int
}

func (f *fakeCommissionRetrier) RetryCommission(_ context.Context, _, _ uuid.UUID) (portservice.ReferralCommissionRetryOutcome, error) {
	f.calls++
	return f.outcome, f.err
}

// fakeOnboardingURL returns a canned onboarding URL so the
// 422 kyc_required test can assert the field is forwarded into the
// response envelope.
type fakeOnboardingURL struct {
	url string
	err error
}

func (f *fakeOnboardingURL) GetOnboardingURL(_ context.Context, _ uuid.UUID) (string, error) {
	return f.url, f.err
}

// commissionRetryReq builds an authenticated commission retry request
// with the supplied id (UUID or raw string) wired into the chi URL
// param so the handler can read it via chi.URLParam(r, "id").
func commissionRetryReq(t *testing.T, commissionID string, userID, orgID uuid.UUID) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/wallet/commissions/"+commissionID+"/retry", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", commissionID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// decodeJSONBody is a small helper for the variable-shape responses the
// commission retry handler produces (success payload, error envelope,
// and the 422 kyc_required envelope all have different keys).
func decodeJSONBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	return body
}

func TestRetryCommission_NoUserContext_401(t *testing.T) {
	retrier := &fakeCommissionRetrier{}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/wallet/commissions/"+uuid.New().String()+"/retry", nil)
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 0, retrier.calls, "retrier must NOT be called when auth context is missing")
}

func TestRetryCommission_NoOrgContext_401(t *testing.T) {
	retrier := &fakeCommissionRetrier{}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/wallet/commissions/"+uuid.New().String()+"/retry", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req.WithContext(ctx))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 0, retrier.calls)
}

func TestRetryCommission_BadCommissionID_400(t *testing.T) {
	retrier := &fakeCommissionRetrier{}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, "not-a-uuid", uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 0, retrier.calls)
}

func TestRetryCommission_RetrierUnavailable_503(t *testing.T) {
	// No WithCommissionRetrier — wallet handler still boots but the
	// route degrades to 503. This is the "referral feature disabled
	// in this worktree" path.
	wh := handler.NewWalletHandler(nil, nil)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestRetryCommission_NotFound_404(t *testing.T) {
	retrier := &fakeCommissionRetrier{err: referraldomain.ErrCommissionNotFound}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	body := decodeJSONBody(t, rec)
	assert.Equal(t, "commission_not_found", body["error"])
}

func TestRetryCommission_NotOwner_403(t *testing.T) {
	retrier := &fakeCommissionRetrier{err: referralapp.ErrCommissionNotOwned}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	body := decodeJSONBody(t, rec)
	assert.Equal(t, "forbidden", body["error"])
}

func TestRetryCommission_AlreadyPaid_409(t *testing.T) {
	retrier := &fakeCommissionRetrier{outcome: portservice.ReferralCommissionRetryOutcome{
		Result: portservice.ReferralCommissionRetryAlreadyPaid,
	}}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	body := decodeJSONBody(t, rec)
	assert.Equal(t, "already_paid", body["error"])
}

func TestRetryCommission_NotRetriable_409(t *testing.T) {
	retrier := &fakeCommissionRetrier{outcome: portservice.ReferralCommissionRetryOutcome{
		Result: portservice.ReferralCommissionRetryNotRetriable,
	}}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	body := decodeJSONBody(t, rec)
	assert.Equal(t, "not_retriable", body["error"])
}

func TestRetryCommission_KYCRequired_422(t *testing.T) {
	retrier := &fakeCommissionRetrier{outcome: portservice.ReferralCommissionRetryOutcome{
		Result:        portservice.ReferralCommissionRetryKYCRequired,
		StripeAccount: "acct_apporteur",
	}}
	onboarding := &fakeOnboardingURL{url: "https://stripe.com/connect/onboarding/abc"}
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionRetrier(retrier).
		WithKYCOnboardingURLResolver(onboarding)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := decodeJSONBody(t, rec)
	errObj, _ := body["error"].(map[string]any)
	require.NotNil(t, errObj)
	assert.Equal(t, "kyc_required", errObj["code"])
	assert.Equal(t, "https://stripe.com/connect/onboarding/abc", body["onboarding_url"])
	assert.Equal(t, "acct_apporteur", body["stripe_account"])
	assert.Equal(t, "/payment-info", body["redirect"])
}

func TestRetryCommission_KYCRequired_NoResolver_422_OmitsURL(t *testing.T) {
	retrier := &fakeCommissionRetrier{outcome: portservice.ReferralCommissionRetryOutcome{
		Result: portservice.ReferralCommissionRetryKYCRequired,
	}}
	// No WithKYCOnboardingURLResolver → response still 422 but no URL.
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := decodeJSONBody(t, rec)
	assert.Equal(t, "", body["onboarding_url"], "no resolver → empty onboarding_url")
	assert.Equal(t, "/payment-info", body["redirect"])
}

func TestRetryCommission_HappyPath_200(t *testing.T) {
	retrier := &fakeCommissionRetrier{outcome: portservice.ReferralCommissionRetryOutcome{
		Result:        portservice.ReferralCommissionRetryPaid,
		StripeAccount: "acct_apporteur",
	}}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeJSONBody(t, rec)
	assert.Equal(t, "paid", body["status"])
	assert.Equal(t, "acct_apporteur", body["stripe_account"])
}

func TestRetryCommission_StripeFailure_502(t *testing.T) {
	retrier := &fakeCommissionRetrier{outcome: portservice.ReferralCommissionRetryOutcome{
		Result:        portservice.ReferralCommissionRetryFailed,
		FailureReason: "stripe: rate limited",
	}}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	body := decodeJSONBody(t, rec)
	errObj, _ := body["error"].(map[string]any)
	require.NotNil(t, errObj)
	assert.Equal(t, "retry_failed", errObj["code"])
	assert.Equal(t, "stripe: rate limited", body["failure_reason"])
}

func TestRetryCommission_GenericError_502(t *testing.T) {
	// Anything that is NOT a sentinel maps to 502 — the operation
	// failed but the user can retry. Hides the internal error from
	// the API surface (only the slog line carries the original).
	retrier := &fakeCommissionRetrier{err: errors.New("db down")}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := commissionRetryReq(t, uuid.New().String(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	wh.RetryCommission(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	body := decodeJSONBody(t, rec)
	assert.Equal(t, "retry_failed", body["error"])
}
