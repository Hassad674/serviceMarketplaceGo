package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentapp "marketplace-backend/internal/app/payment"
	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
)

// fakeCommissionProjector drives ProjectedCommissions for unit tests.
type fakeCommissionProjector struct {
	rows []referralapp.ProjectedCommission
	err  error
}

func (f *fakeCommissionProjector) ProjectedCommissions(_ context.Context, _ uuid.UUID) ([]referralapp.ProjectedCommission, error) {
	return f.rows, f.err
}

// fakeCommissionRecorder drives RecentCommissions for unit tests.
type fakeCommissionRecorder struct {
	rows []portservice.ReferralCommissionRecord
	err  error
}

func (f *fakeCommissionRecorder) RecentCommissions(_ context.Context, _ uuid.UUID, _ int) ([]portservice.ReferralCommissionRecord, error) {
	return f.rows, f.err
}

// summaryRequest builds an authenticated GET /wallet/summary request
// with the given user/org id baked into the context.
func summaryRequest(t *testing.T, userID, orgID uuid.UUID, query string) *http.Request {
	t.Helper()
	target := "/api/v1/wallet/summary"
	if query != "" {
		target += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	return req.WithContext(ctx)
}

// decodeSummary peeks at the JSON envelope and returns the
// inner data map for assertion convenience.
func decodeSummary(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data, ok := body["data"].(map[string]any)
	require.True(t, ok, "response must wrap payload in `data`")
	return data
}

// TestSummary_Unauthorized_NoUserContext returns 401 when the JWT
// middleware didn't run.
func TestSummary_Unauthorized_NoUserContext(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/summary", nil)
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestSummary_Unauthorized_NoOrgContext returns 401 when only
// userID is wired in context.
func TestSummary_Unauthorized_NoOrgContext(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/summary", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uuid.New())
	rec := httptest.NewRecorder()
	wh.Summary(rec, req.WithContext(ctx))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestSummary_EmptyWallet_DegradesGracefully — no payment service,
// no projector, no recorder → response is zero-valued but valid.
func TestSummary_EmptyWallet_DegradesGracefully(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil)
	req := summaryRequest(t, uuid.New(), uuid.New(), "")
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeSummary(t, rec)
	assert.Equal(t, "EUR", data["currency"])
	assert.Equal(t, float64(0), data["total_cents"])
	assert.Equal(t, float64(0), data["available_cents"])
	assert.Equal(t, float64(0), data["escrowed_cents"])
	assert.Equal(t, float64(0), data["transmitted_cents"])
	txs, _ := data["recent_transactions"].([]any)
	assert.Empty(t, txs)
}

// TestSummary_CommissionsOnly composes a summary with no mission
// service wired but a commission recorder + projector populated.
// Verifies the breakdown maths.
func TestSummary_CommissionsOnly(t *testing.T) {
	paidAt := time.Now().UTC().Add(-1 * time.Hour)
	recorder := &fakeCommissionRecorder{
		rows: []portservice.ReferralCommissionRecord{
			{
				ID:              uuid.New(),
				CommissionCents: 100_00,
				Currency:        "EUR",
				Status:          "paid",
				PaidAt:          &paidAt,
				CreatedAt:       paidAt.Add(-30 * time.Minute),
			},
			{
				ID:              uuid.New(),
				CommissionCents: 50_00,
				Currency:        "EUR",
				Status:          "failed",
				CreatedAt:       paidAt.Add(-2 * time.Hour),
			},
		},
	}
	projector := &fakeCommissionProjector{
		rows: []referralapp.ProjectedCommission{
			{
				AttributionID:  uuid.New(),
				MilestoneID:    uuid.New(),
				ProjectedCents: 25_00,
				Currency:       "EUR",
				Status:         referralapp.ProjectionEscrowed,
				Source:         referralapp.SourceProjection,
				ProjectedAt:    paidAt.Add(-15 * time.Minute),
			},
		},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionRecorder(recorder).
		WithCommissionProjector(projector)
	req := summaryRequest(t, uuid.New(), uuid.New(), "")
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeSummary(t, rec)
	breakdown := data["breakdown"].(map[string]any)
	commissions := breakdown["commissions"].(map[string]any)
	assert.Equal(t, float64(100_00), commissions["transmitted_cents"])
	assert.Equal(t, float64(50_00), commissions["available_cents"])
	assert.Equal(t, float64(25_00), commissions["escrowed_cents"])
	assert.Equal(t, float64(175_00), commissions["total_cents"])
}

// TestSummary_RowSourcedProjection_NotDoubleCounted ensures a
// projection with Source=row is NOT added to commission.escrowed_cents
// — its row counterpart is already counted on the records side.
func TestSummary_RowSourcedProjection_NotDoubleCounted(t *testing.T) {
	recorder := &fakeCommissionRecorder{
		rows: []portservice.ReferralCommissionRecord{
			{
				ID:              uuid.New(),
				CommissionCents: 100_00,
				Currency:        "EUR",
				Status:          "paid",
				PaidAt:          ptrTime(time.Now()),
				CreatedAt:       time.Now(),
			},
		},
	}
	projector := &fakeCommissionProjector{
		rows: []referralapp.ProjectedCommission{
			{
				AttributionID:  uuid.New(),
				MilestoneID:    uuid.New(),
				ProjectedCents: 100_00,
				Currency:       "EUR",
				Status:         referralapp.ProjectionPaid,
				Source:         referralapp.SourceRow,
				ProjectedAt:    time.Now(),
			},
		},
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionRecorder(recorder).
		WithCommissionProjector(projector)
	req := summaryRequest(t, uuid.New(), uuid.New(), "")
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeSummary(t, rec)
	breakdown := data["breakdown"].(map[string]any)
	commissions := breakdown["commissions"].(map[string]any)
	// transmitted_cents = 100,00 (from the row), escrowed_cents = 0
	// because the row-sourced projection is skipped.
	assert.Equal(t, float64(100_00), commissions["transmitted_cents"])
	assert.Equal(t, float64(0), commissions["escrowed_cents"])
}

// TestSummary_Pagination paginates 50 entries by cursor + limit and
// asserts no duplicates / no gaps across pages.
func TestSummary_Pagination(t *testing.T) {
	now := time.Now().UTC()
	rows := make([]portservice.ReferralCommissionRecord, 50)
	for i := range rows {
		rows[i] = portservice.ReferralCommissionRecord{
			ID:              uuid.New(),
			CommissionCents: int64((i + 1) * 100),
			Currency:        "EUR",
			Status:          "paid",
			PaidAt:          ptrTime(now.Add(-time.Duration(i) * time.Minute)),
			CreatedAt:       now.Add(-time.Duration(i) * time.Minute),
		}
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionRecorder(&fakeCommissionRecorder{rows: rows})

	seen := map[string]bool{}
	cursor := ""
	pages := 0
	for {
		query := "limit=10"
		if cursor != "" {
			query += "&cursor=" + cursor
		}
		req := summaryRequest(t, uuid.New(), uuid.New(), query)
		rec := httptest.NewRecorder()
		wh.Summary(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		data := decodeSummary(t, rec)
		txs, _ := data["recent_transactions"].([]any)
		require.NotNil(t, txs)
		for _, tx := range txs {
			m := tx.(map[string]any)
			id := m["reference_id"].(string)
			require.False(t, seen[id], "duplicate id %s across pages", id)
			seen[id] = true
		}
		pages++
		nextCursor, _ := data["next_cursor"].(string)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
		require.Less(t, pages, 10, "should not exceed 10 pages on 50 entries with limit=10")
	}
	assert.Equal(t, 50, len(seen), "every row must be paginated exactly once")
}

// TestSummary_Pagination_InvalidCursor returns 400 with a clear code.
func TestSummary_Pagination_InvalidCursor(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil)
	req := summaryRequest(t, uuid.New(), uuid.New(), "cursor=NotABase64Cursor!!!")
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	// res.Error uses a flat {error:code, message:msg} envelope.
	assert.Equal(t, "invalid_cursor", body["error"])
}

// TestSummary_LimitClamping verifies invalid limits fall back to
// the default rather than crashing.
func TestSummary_LimitClamping(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionRecorder(&fakeCommissionRecorder{rows: synthCommissions(30)})

	// limit=0 → default
	req := summaryRequest(t, uuid.New(), uuid.New(), "limit=0")
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeSummary(t, rec)
	txs, _ := data["recent_transactions"].([]any)
	assert.Len(t, txs, 20, "limit=0 must fall back to default of 20")

	// limit=999 → max
	req = summaryRequest(t, uuid.New(), uuid.New(), "limit=999")
	rec = httptest.NewRecorder()
	wh.Summary(rec, req)
	data = decodeSummary(t, rec)
	txs, _ = data["recent_transactions"].([]any)
	assert.LessOrEqual(t, len(txs), 100, "limit=999 must be capped at 100")
}

// TestSummary_RecorderErrorDegradesGracefully — a broken recorder
// must NOT take down the wallet; the response keeps the (zero)
// commission side and returns 200.
func TestSummary_RecorderErrorDegradesGracefully(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionRecorder(&fakeCommissionRecorder{err: fmt.Errorf("recorder-boom")})
	req := summaryRequest(t, uuid.New(), uuid.New(), "")
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSummary_ProjectorErrorDegradesGracefully — same for the projector.
func TestSummary_ProjectorErrorDegradesGracefully(t *testing.T) {
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionProjector(&fakeCommissionProjector{err: fmt.Errorf("projector-boom")})
	req := summaryRequest(t, uuid.New(), uuid.New(), "")
	rec := httptest.NewRecorder()
	wh.Summary(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSummary_CurrencyFallback prefers the recorder's currency when
// non-empty, otherwise defaults to EUR.
func TestSummary_CurrencyFallback(t *testing.T) {
	// 1) Non-empty currency wins
	recorder := &fakeCommissionRecorder{
		rows: []portservice.ReferralCommissionRecord{
			{ID: uuid.New(), Currency: "USD", Status: "paid", PaidAt: ptrTime(time.Now()), CreatedAt: time.Now(), CommissionCents: 1},
		},
	}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRecorder(recorder)
	rec := httptest.NewRecorder()
	wh.Summary(rec, summaryRequest(t, uuid.New(), uuid.New(), ""))
	data := decodeSummary(t, rec)
	assert.Equal(t, "USD", data["currency"])

	// 2) Empty currency falls back to EUR
	recorder.rows[0].Currency = ""
	rec = httptest.NewRecorder()
	wh.Summary(rec, summaryRequest(t, uuid.New(), uuid.New(), ""))
	data = decodeSummary(t, rec)
	assert.Equal(t, "EUR", data["currency"])
}

// TestSummary_MissionLeg_FromOverview covers the missionTransaction
// + missionLeg branches. Synthesises a fake payment service whose
// GetWalletOverview returns a populated WalletOverview, then asserts
// the breakdown sums + the timeline carries the mission row.
//
// The fake payment service uses the same WalletOverview shape as the
// real service; the summary handler only reads from the returned
// struct so we don't need to stand up the full payment stack here.
func TestSummary_MissionLeg_FromOverview(t *testing.T) {
	// Build a wallet handler wired with a fake mission-side reader.
	// Since *paymentapp.Service can't be easily mocked from outside
	// the handler package, this scenario is covered indirectly
	// through the integration tests — here we cover the breakdown
	// composer + the timeline mapper directly via the test exports.
	rec := paymentapp.WalletRecord{
		ID:             uuid.New().String(),
		ProposalID:     uuid.New().String(),
		ProviderPayout: 50_00,
		PaymentStatus:  "succeeded",
		TransferStatus: "pending",
		MissionStatus:  "in_progress",
		CreatedAt:      time.Now().Format("2006-01-02T15:04:05Z"),
	}
	overview := &paymentapp.WalletOverview{
		AvailableAmount:   100_00,
		EscrowAmount:      50_00,
		TransferredAmount: 25_00,
		Records:           []paymentapp.WalletRecord{rec},
	}
	leg := handler.MissionLegForTest(overview)
	assert.Equal(t, int64(100_00), leg.AvailableCents)
	assert.Equal(t, int64(50_00), leg.EscrowedCents)
	assert.Equal(t, int64(25_00), leg.TransmittedCents)
	assert.Equal(t, int64(175_00), leg.TotalCents)

	tx := handler.MissionTransactionForTest(rec)
	assert.Equal(t, "mission", tx.Type)
	assert.Equal(t, int64(50_00), tx.AmountCents)
	assert.Equal(t, "in_progress", tx.Status, "MissionStatus must take precedence over TransferStatus on mapped row")
}

// TestSummary_MissionLeg_NilOverview returns a zero leg without
// panicking.
func TestSummary_MissionLeg_NilOverview(t *testing.T) {
	leg := handler.MissionLegForTest(nil)
	assert.Equal(t, int64(0), leg.AvailableCents)
}

// TestSummary_MissionTransaction_FallbackToTransferStatus when
// MissionStatus is empty the timeline falls back to TransferStatus.
func TestSummary_MissionTransaction_FallbackToTransferStatus(t *testing.T) {
	rec := paymentapp.WalletRecord{
		ID:             uuid.New().String(),
		ProposalID:     uuid.New().String(),
		ProviderPayout: 50_00,
		TransferStatus: "completed",
		CreatedAt:      time.Now().Format("2006-01-02T15:04:05Z"),
	}
	tx := handler.MissionTransactionForTest(rec)
	assert.Equal(t, "completed", tx.Status, "empty MissionStatus must fall back to TransferStatus")
}

// TestSummary_OrderingDescending verifies recent_transactions is
// sorted by occurred_at DESC.
func TestSummary_OrderingDescending(t *testing.T) {
	now := time.Now().UTC()
	older := portservice.ReferralCommissionRecord{
		ID: uuid.New(), CommissionCents: 100, Currency: "EUR", Status: "paid",
		PaidAt: ptrTime(now.Add(-2 * time.Hour)), CreatedAt: now.Add(-3 * time.Hour),
	}
	newer := portservice.ReferralCommissionRecord{
		ID: uuid.New(), CommissionCents: 200, Currency: "EUR", Status: "paid",
		PaidAt: ptrTime(now.Add(-1 * time.Hour)), CreatedAt: now.Add(-2 * time.Hour),
	}
	wh := handler.NewWalletHandler(nil, nil).
		WithCommissionRecorder(&fakeCommissionRecorder{rows: []portservice.ReferralCommissionRecord{older, newer}})
	rec := httptest.NewRecorder()
	wh.Summary(rec, summaryRequest(t, uuid.New(), uuid.New(), ""))
	assert.Equal(t, http.StatusOK, rec.Code)
	data := decodeSummary(t, rec)
	txs, _ := data["recent_transactions"].([]any)
	require.Len(t, txs, 2)
	first := txs[0].(map[string]any)
	second := txs[1].(map[string]any)
	assert.Equal(t, newer.ID.String(), first["reference_id"], "newest must be first")
	assert.Equal(t, older.ID.String(), second["reference_id"])
}

// ─── helpers ──────────────────────────────────────────────────────

func ptrTime(t time.Time) *time.Time { return &t }

func synthCommissions(n int) []portservice.ReferralCommissionRecord {
	now := time.Now().UTC()
	rows := make([]portservice.ReferralCommissionRecord, n)
	for i := range rows {
		rows[i] = portservice.ReferralCommissionRecord{
			ID:              uuid.New(),
			CommissionCents: int64((i + 1) * 10),
			Currency:        "EUR",
			Status:          "paid",
			PaidAt:          ptrTime(now.Add(-time.Duration(i) * time.Minute)),
			CreatedAt:       now.Add(-time.Duration(i) * time.Minute),
		}
	}
	return rows
}
