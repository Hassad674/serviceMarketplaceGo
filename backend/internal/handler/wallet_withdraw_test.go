package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentapp "marketplace-backend/internal/app/payment"
	paymentdomain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
)

// fakeAuditLogger records every entry so tests can assert the
// audit row was emitted on successful drains.
type fakeAuditLogger struct {
	mu      sync.Mutex
	entries []map[string]any
}

func (f *fakeAuditLogger) Log(_ context.Context, e *handler.AuditEntryExposed) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, map[string]any{
		"action":        e.Action,
		"user_id":       e.UserID,
		"org_id":        e.OrgID,
		"resource_type": e.ResourceType,
		"metadata":      e.Metadata,
	})
	return nil
}

// fakeWithdrawCommissionRetrier drives RetryCommission outcomes for
// withdraw tests. Programs an ordered list of outcomes returned in
// sequence as RetryCommission is invoked.
type fakeWithdrawCommissionRetrier struct {
	mu       sync.Mutex
	outcomes []portservice.ReferralCommissionRetryOutcome
	errs     []error
	calls    int
}

func (f *fakeWithdrawCommissionRetrier) RetryCommission(_ context.Context, _, _ uuid.UUID) (portservice.ReferralCommissionRetryOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.calls
	f.calls++
	var out portservice.ReferralCommissionRetryOutcome
	if idx < len(f.outcomes) {
		out = f.outcomes[idx]
	}
	var err error
	if idx < len(f.errs) {
		err = f.errs[idx]
	}
	return out, err
}

// withdrawAuthReq builds an authenticated withdraw request with the
// supplied body.
func withdrawAuthReq(t *testing.T, userID, orgID uuid.UUID, body any) *http.Request {
	t.Helper()
	var bodyReader *bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewBuffer(raw)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/withdraw", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	return req.WithContext(ctx)
}

// TestWithdraw_Unauthorized_NoUserContext returns 401.
func TestWithdraw_Unauthorized_NoUserContext(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/withdraw", nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestWithdraw_NegativeAmount returns 400.
func TestWithdraw_NegativeAmount(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil)
	req := withdrawAuthReq(t, uuid.New(), uuid.New(), map[string]any{"amount_cents": -100})
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestWithdraw_InvalidBody returns 400.
func TestWithdraw_InvalidBody(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/withdraw",
		strings.NewReader("not json"))
	req.ContentLength = int64(len("not json"))
	req.Header.Set("Content-Type", "application/json")
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, uuid.New())
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uuid.New())
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req.WithContext(ctx))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestWithdraw_KYCRequired returns 422 with onboarding URL when the
// KYC gate trips.
func TestWithdraw_KYCRequired(t *testing.T) {
	probe := &fakeKYCProbe{ready: false}
	urlResolver := &fakeOnboardingURL{url: "https://stripe.example/connect/abc"}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithKYCOnboardingURLResolver(urlResolver)

	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	errObj := body["error"].(map[string]any)
	assert.Equal(t, "kyc_required", errObj["code"])
	assert.Equal(t, "https://stripe.example/connect/abc", body["onboarding_url"])
}

// TestWithdraw_KYCProbeErrorBlocks — probe error fails closed.
func TestWithdraw_KYCProbeErrorBlocks(t *testing.T) {
	probe := &fakeKYCProbe{err: errors.New("probe-boom")}
	wh := handler.NewWalletHandler(nil, nil).WithPayoutReadinessProbe(probe)
	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// TestWithdraw_EmptyWallet — no payment service, no commission
// recorder/retrier → 200 OK with drained=0.
func TestWithdraw_EmptyWallet(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	wh := handler.NewWalletHandler(nil, nil).WithPayoutReadinessProbe(probe)
	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeWithdraw(t, rec)
	assert.Equal(t, float64(0), data["drained_cents"])
	assert.Equal(t, float64(0), data["missions_cents"])
	assert.Equal(t, float64(0), data["commissions_cents"])
}

// TestWithdraw_CommissionsOnly_HappyPath drains commissions when
// missions are empty. Verifies audit is emitted.
func TestWithdraw_CommissionsOnly_HappyPath(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	audit := &fakeAuditLogger{}

	commission := portservice.ReferralCommissionRecord{
		ID:              uuid.New(),
		CommissionCents: 100_00,
		Currency:        "EUR",
		Status:          "pending_kyc",
		RetireEligible:  true,
		CreatedAt:       time.Now(),
	}
	recorder := &fakeCommissionRecorder{rows: []portservice.ReferralCommissionRecord{commission}}
	retrier := &fakeWithdrawCommissionRetrier{
		outcomes: []portservice.ReferralCommissionRetryOutcome{
			{Result: portservice.ReferralCommissionRetryPaid, StripeAccount: "acct_xxx"},
		},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithCommissionRecorder(recorder).
		WithCommissionRetrier(retrier).
		WithAuditLogger(audit)

	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeWithdraw(t, rec)
	assert.Equal(t, float64(100_00), data["drained_cents"])
	assert.Equal(t, float64(100_00), data["commissions_cents"])
	// Audit emitted.
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "wallet.withdraw_executed", audit.entries[0]["action"])
}

// TestWithdraw_PartialAmount stops draining when amount_cents is met.
func TestWithdraw_PartialAmount(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	commissions := []portservice.ReferralCommissionRecord{
		{ID: uuid.New(), CommissionCents: 50_00, Currency: "EUR", Status: "pending_kyc", RetireEligible: true, CreatedAt: time.Now()},
		{ID: uuid.New(), CommissionCents: 50_00, Currency: "EUR", Status: "pending_kyc", RetireEligible: true, CreatedAt: time.Now()},
		{ID: uuid.New(), CommissionCents: 50_00, Currency: "EUR", Status: "pending_kyc", RetireEligible: true, CreatedAt: time.Now()},
	}
	recorder := &fakeCommissionRecorder{rows: commissions}
	retrier := &fakeWithdrawCommissionRetrier{
		outcomes: []portservice.ReferralCommissionRetryOutcome{
			{Result: portservice.ReferralCommissionRetryPaid, StripeAccount: "a1"},
			{Result: portservice.ReferralCommissionRetryPaid, StripeAccount: "a2"},
		},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithCommissionRecorder(recorder).
		WithCommissionRetrier(retrier)

	req := withdrawAuthReq(t, uuid.New(), uuid.New(), map[string]any{"amount_cents": 100_00})
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeWithdraw(t, rec)
	assert.Equal(t, float64(100_00), data["drained_cents"])
	// We expected exactly 2 retry calls — the third would have
	// exceeded the amount cap.
	assert.Equal(t, 2, retrier.calls, "retrier must stop once amount_cents is reached")
}

// TestWithdrawWithdraw_RefusesAmountAboveAvailable pins the safety
// behavior the user verified manually after the UI crash on /wallet:
// when the client asks for MORE cents than are actually drainable,
// the backend NEVER overdraws. It caps the drained amount at what is
// authoritatively eligible (retire-eligible commissions + completed-
// mission payouts).
//
// Setup: 1 retire-eligible commission of 100 cents + 1 escrowed
// (non-retire-eligible) commission of 200 cents. Client asks for
// 300 cents — that's available + escrowed.
//
// Expected:
//   - response.drained_cents == 100 (NOT 300), proving the backend
//     refused to transfer the escrowed funds.
//   - The retrier is called exactly once (on the eligible row only) —
//     proving the escrowed row was never tapped for drain.
//   - HTTP status is 200 (the implementation treats overshoot as a
//     non-error: drain what you can, return the breakdown).
//
// This is the regression pin for the user-reported scenario: the
// frontend crashed but the funds stayed put — the backend's safety
// behavior was the unsung hero, and we lock it in here so a future
// refactor can't accidentally make `amount_cents` an authoritative
// drain instruction.
func TestWalletWithdraw_RefusesAmountAboveAvailable(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	availableCommission := portservice.ReferralCommissionRecord{
		ID:              uuid.New(),
		CommissionCents: 100,
		Currency:        "EUR",
		Status:          "pending_kyc",
		RetireEligible:  true,
		CreatedAt:       time.Now(),
	}
	// Non-retire-eligible — represents money in escrow that can't be
	// withdrawn yet (mission not approved / commission not crystallised).
	escrowedCommission := portservice.ReferralCommissionRecord{
		ID:              uuid.New(),
		CommissionCents: 200,
		Currency:        "EUR",
		Status:          "pending",
		RetireEligible:  false,
		CreatedAt:       time.Now(),
	}
	recorder := &fakeCommissionRecorder{
		rows: []portservice.ReferralCommissionRecord{
			availableCommission,
			escrowedCommission,
		},
	}
	retrier := &fakeWithdrawCommissionRetrier{
		outcomes: []portservice.ReferralCommissionRetryOutcome{
			{Result: portservice.ReferralCommissionRetryPaid, StripeAccount: "acct_xxx"},
		},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithCommissionRecorder(recorder).
		WithCommissionRetrier(retrier)

	// Ask for 300 cents — that's available (100) + escrowed (200).
	req := withdrawAuthReq(t, uuid.New(), uuid.New(), map[string]any{"amount_cents": 300})
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code,
		"backend must respond 200 — the implementation drains what it can, not what was asked")
	data := decodeWithdraw(t, rec)
	assert.Equal(t, float64(100), data["drained_cents"],
		"drained must be CAPPED at the available 100 cents, never the requested 300")
	assert.Equal(t, float64(100), data["commissions_cents"],
		"commission leg must report 100 cents drained")
	assert.Equal(t, float64(0), data["missions_cents"],
		"mission leg unchanged (no payment service wired)")
	// The retrier is called exactly once — on the eligible row only.
	// The escrowed row was never tapped, proving the funds did not move.
	assert.Equal(t, 1, retrier.calls,
		"retrier must be called once — once for the eligible row, never for the escrowed row")
}

// TestWithdraw_SkipsNonRetireEligible — commissions without
// RetireEligible=true are skipped (e.g. paid, clawed_back).
func TestWithdraw_SkipsNonRetireEligible(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	commissions := []portservice.ReferralCommissionRecord{
		{ID: uuid.New(), CommissionCents: 100_00, Currency: "EUR", Status: "paid", RetireEligible: false, CreatedAt: time.Now()},
	}
	recorder := &fakeCommissionRecorder{rows: commissions}
	retrier := &fakeWithdrawCommissionRetrier{}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithCommissionRecorder(recorder).
		WithCommissionRetrier(retrier)

	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeWithdraw(t, rec)
	assert.Equal(t, float64(0), data["drained_cents"])
	assert.Equal(t, 0, retrier.calls, "paid commission must not be retried")
}

// TestWithdraw_PartialFailure_207 — one commission succeeds, the
// next errors. Expected: HTTP 207 + breakdown shows the partial
// drain + errors[] populated.
func TestWithdraw_PartialFailure_207(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	commissions := []portservice.ReferralCommissionRecord{
		{ID: uuid.New(), CommissionCents: 50_00, Currency: "EUR", Status: "pending_kyc", RetireEligible: true, CreatedAt: time.Now()},
		{ID: uuid.New(), CommissionCents: 50_00, Currency: "EUR", Status: "failed", RetireEligible: true, CreatedAt: time.Now()},
	}
	recorder := &fakeCommissionRecorder{rows: commissions}
	retrier := &fakeWithdrawCommissionRetrier{
		outcomes: []portservice.ReferralCommissionRetryOutcome{
			{Result: portservice.ReferralCommissionRetryPaid, StripeAccount: "a1"},
			{}, // error on second call
		},
		errs: []error{nil, errors.New("stripe-boom")},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithCommissionRecorder(recorder).
		WithCommissionRetrier(retrier)

	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusMultiStatus, rec.Code, "partial drain must surface as 207")
	data := decodeWithdraw(t, rec)
	assert.Equal(t, float64(50_00), data["drained_cents"])
	errs, _ := data["errors"].([]any)
	require.Len(t, errs, 1)
}

// TestWithdraw_AllFailures_500 — every leg fails, drained=0.
// Expected: HTTP 500 with error envelope.
func TestWithdraw_AllFailures_500(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	commissions := []portservice.ReferralCommissionRecord{
		{ID: uuid.New(), CommissionCents: 50_00, Currency: "EUR", Status: "failed", RetireEligible: true, CreatedAt: time.Now()},
	}
	recorder := &fakeCommissionRecorder{rows: commissions}
	retrier := &fakeWithdrawCommissionRetrier{
		errs: []error{errors.New("stripe-boom")},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithCommissionRecorder(recorder).
		WithCommissionRetrier(retrier)

	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestWithdraw_NoAuditOnEmpty — when nothing was drained, no
// audit row is emitted. Audit captures executed withdraws only.
func TestWithdraw_NoAuditOnEmpty(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	audit := &fakeAuditLogger{}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithAuditLogger(audit)
	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, audit.entries, "no audit row when nothing was drained")
}

// TestWithdraw_BillingProfileIncomplete — billing gate fires.
func TestWithdraw_BillingProfileIncomplete(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	wh, userID, orgID := gatedWalletHarnessWith(t, gatedWalletHarnessOpts{
		kycProbe: probe,
		// No profile fixture seeded → billing gate is "incomplete".
	})
	req := withdrawAuthReq(t, userID, orgID, nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	errObj := body["error"].(map[string]any)
	assert.Equal(t, "billing_profile_incomplete", errObj["code"])
}

// TestWithdraw_NonRetriableCommissionResult is the no-op branch:
// outcome=AlreadyPaid / NotRetriable / KYCRequired must NOT add to
// the drain total even though the retrier ran cleanly.
func TestWithdraw_NonRetriableCommissionResult(t *testing.T) {
	probe := &fakeKYCProbe{ready: true}
	commissions := []portservice.ReferralCommissionRecord{
		{ID: uuid.New(), CommissionCents: 50_00, Currency: "EUR", Status: "pending_kyc", RetireEligible: true, CreatedAt: time.Now()},
	}
	recorder := &fakeCommissionRecorder{rows: commissions}
	retrier := &fakeWithdrawCommissionRetrier{
		outcomes: []portservice.ReferralCommissionRetryOutcome{
			{Result: portservice.ReferralCommissionRetryKYCRequired},
		},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithPayoutReadinessProbe(probe).
		WithCommissionRecorder(recorder).
		WithCommissionRetrier(retrier)

	req := withdrawAuthReq(t, uuid.New(), uuid.New(), nil)
	rec := httptest.NewRecorder()
	wh.Withdraw(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeWithdraw(t, rec)
	assert.Equal(t, float64(0), data["drained_cents"])
}

// TestParseMissionAmountFromMessage_KnownShapes pins the parser
// against the two known PayoutResult message shapes.
func TestParseMissionAmountFromMessage_KnownShapes(t *testing.T) {
	cases := []struct {
		msg  string
		want int64
	}{
		{"Transferred 12345 centimes to your account", 12345},
		{"Transferred 0 centimes to your account", 0},
		{"Transferred 99999999 centimes — bank transfer pending", 99999999},
		{"unrelated message", 0},
		{"", 0},
	}
	for _, c := range cases {
		got := handler.ParseMissionAmountFromMessageForTest(c.msg)
		assert.Equal(t, c.want, got, c.msg)
	}
}

// TestWithdraw_MissionsErrorMapping — a payment domain error is
// mapped to the right machine code in the errors[] envelope.
func TestWithdraw_MissionsErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{paymentdomain.ErrStripeAccountNotFound, "stripe_account_missing"},
		{paymentdomain.ErrProviderPayoutsDisabled, "provider_kyc_incomplete"},
		{errors.New("other"), "missions_drain_failed"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c.err), func(t *testing.T) {
			got := handler.MissionErrCodeForTest(c.err)
			assert.Equal(t, c.code, got)
		})
	}
}

// TestWithdraw_MissionDrainResult_ParsesAmount validates the
// drained_cents derivation from a PayoutResult — the handler's
// missionDrainedFromResult helper sums via the message parser.
func TestWithdraw_MissionDrainResult_ParsesAmount(t *testing.T) {
	cases := []struct {
		result *paymentapp.PayoutResult
		want   int64
	}{
		{&paymentapp.PayoutResult{Status: "transferred", Message: "Transferred 12345 centimes to your account"}, 12345},
		{&paymentapp.PayoutResult{Status: "transferred_pending_bank", Message: "Transferred 4567 centimes — bank transfer pending"}, 4567},
		{&paymentapp.PayoutResult{Status: "nothing_to_transfer", Message: "No funds available for transfer"}, 0},
		{nil, 0},
	}
	for _, c := range cases {
		got := handler.MissionDrainedFromResultForTest(c.result)
		assert.Equal(t, c.want, got)
	}
}

// ─── helpers ──────────────────────────────────────────────────────

func decodeWithdraw(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data, ok := body["data"].(map[string]any)
	require.True(t, ok, "response must wrap payload in `data`")
	return data
}
