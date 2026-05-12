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
// service wired but a commission projector populated. Verifies the
// breakdown maths — projections are the canonical source for the
// commission aggregates (records still feed the timeline).
func TestSummary_CommissionsOnly(t *testing.T) {
	paidAt := time.Now().UTC().Add(-1 * time.Hour)
	// Each projection contributes to ONE bucket only; records still
	// flow into recent_transactions but no longer drive aggregates.
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
			// Paid → TransmittedCents (the row's commission is exposed
			// via SourceRow with ProjectionPaid status).
			{
				AttributionID:  uuid.New(),
				MilestoneID:    uuid.New(),
				ProjectedCents: 100_00,
				Currency:       "EUR",
				Status:         referralapp.ProjectionPaid,
				Source:         referralapp.SourceRow,
				ProjectedAt:    paidAt,
			},
			// Failed → AvailableCents (retire-eligible).
			{
				AttributionID:  uuid.New(),
				MilestoneID:    uuid.New(),
				ProjectedCents: 50_00,
				Currency:       "EUR",
				Status:         referralapp.ProjectionFailed,
				Source:         referralapp.SourceRow,
				ProjectedAt:    paidAt.Add(-2 * time.Hour),
			},
			// Escrowed → EscrowedCents (active milestone, no row yet).
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

// TestSummary_RecordsDoNotDoubleCountProjections is the regression pin
// for the user-reported bug: a single pending_kyc commission appeared
// as 1298 € in BOTH "séquestre" AND "disponible" cards. Root cause was
// the previous aggregator summing records AND SourceProjection
// projections in parallel — the same commission contributed to two
// buckets. After the fix, projections own the aggregates and records
// only feed the timeline. The same commission must appear in exactly
// one bucket.
func TestSummary_RecordsDoNotDoubleCountProjections(t *testing.T) {
	now := time.Now().UTC()
	milestoneID := uuid.New()
	// The row exists in `referral_commissions` with status pending_kyc
	// (UI shows "Retirer"). It is ALSO surfaced by the projection
	// stream as a SourceRow ProjectionPending entry (canonical).
	commissionID := uuid.New()
	recorder := &fakeCommissionRecorder{
		rows: []portservice.ReferralCommissionRecord{
			{
				ID:              commissionID,
				MilestoneID:     milestoneID,
				CommissionCents: 1298_00,
				Currency:        "EUR",
				Status:          "pending_kyc",
				CreatedAt:       now,
			},
		},
	}
	projector := &fakeCommissionProjector{
		rows: []referralapp.ProjectedCommission{
			{
				AttributionID:  uuid.New(),
				MilestoneID:    milestoneID,
				ProjectedCents: 1298_00,
				Currency:       "EUR",
				Status:         referralapp.ProjectionPending,
				Source:         referralapp.SourceRow,
				ProjectedAt:    now,
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
	// Available only — escrowed must be zero. Total = available.
	assert.Equal(t, float64(1298_00), commissions["available_cents"],
		"pending_kyc commission must surface in available only")
	assert.Equal(t, float64(0), commissions["escrowed_cents"],
		"pending_kyc commission must NOT also appear in escrowed (double-count regression)")
	assert.Equal(t, float64(0), commissions["transmitted_cents"])
	assert.Equal(t, float64(1298_00), commissions["total_cents"])
}

// TestSummary_NoDoubleCount is the explicit table-driven matrix asked
// for in the fix brief. For every (commission state × milestone state)
// pair the projection algorithm covers, the corresponding cents must
// land in EXACTLY ONE bucket — never in two.
func TestSummary_NoDoubleCount(t *testing.T) {
	tests := []struct {
		name                string
		projection          referralapp.ProjectedCommission
		record              *portservice.ReferralCommissionRecord
		wantAvailableCents  int64
		wantEscrowedCents   int64
		wantTransmitted     int64
	}{
		{
			name: "paid (row-sourced projection + paid record)",
			projection: referralapp.ProjectedCommission{
				MilestoneID:    uuid.New(),
				ProjectedCents: 100_00,
				Status:         referralapp.ProjectionPaid,
				Source:         referralapp.SourceRow,
				ProjectedAt:    time.Now(),
				Currency:       "EUR",
			},
			record: &portservice.ReferralCommissionRecord{
				ID: uuid.New(), CommissionCents: 100_00, Status: "paid",
				PaidAt: ptrTime(time.Now()), CreatedAt: time.Now(), Currency: "EUR",
			},
			wantTransmitted: 100_00,
		},
		{
			name: "pending_kyc (row-sourced projection + pending_kyc record)",
			projection: referralapp.ProjectedCommission{
				MilestoneID:    uuid.New(),
				ProjectedCents: 1298_00,
				Status:         referralapp.ProjectionPending,
				Source:         referralapp.SourceRow,
				ProjectedAt:    time.Now(),
				Currency:       "EUR",
			},
			record: &portservice.ReferralCommissionRecord{
				ID: uuid.New(), CommissionCents: 1298_00, Status: "pending_kyc",
				CreatedAt: time.Now(), Currency: "EUR",
			},
			wantAvailableCents: 1298_00,
		},
		{
			name: "failed (row-sourced projection + failed record)",
			projection: referralapp.ProjectedCommission{
				MilestoneID:    uuid.New(),
				ProjectedCents: 50_00,
				Status:         referralapp.ProjectionFailed,
				Source:         referralapp.SourceRow,
				ProjectedAt:    time.Now(),
				Currency:       "EUR",
			},
			record: &portservice.ReferralCommissionRecord{
				ID: uuid.New(), CommissionCents: 50_00, Status: "failed",
				CreatedAt: time.Now(), Currency: "EUR",
			},
			wantAvailableCents: 50_00,
		},
		{
			name: "escrowed (active milestone, no record yet — projection only)",
			projection: referralapp.ProjectedCommission{
				MilestoneID:    uuid.New(),
				ProjectedCents: 75_00,
				Status:         referralapp.ProjectionEscrowed,
				Source:         referralapp.SourceProjection,
				ProjectedAt:    time.Now(),
				Currency:       "EUR",
			},
			record:            nil,
			wantEscrowedCents: 75_00,
		},
		{
			name: "approved with missing row (safety-net pending projection)",
			projection: referralapp.ProjectedCommission{
				MilestoneID:    uuid.New(),
				ProjectedCents: 200_00,
				Status:         referralapp.ProjectionPending,
				Source:         referralapp.SourceProjection,
				ProjectedAt:    time.Now(),
				Currency:       "EUR",
			},
			record:             nil,
			wantAvailableCents: 200_00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &fakeCommissionRecorder{rows: nil}
			if tt.record != nil {
				recorder.rows = []portservice.ReferralCommissionRecord{*tt.record}
			}
			projector := &fakeCommissionProjector{
				rows: []referralapp.ProjectedCommission{tt.projection},
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
			assert.Equal(t, float64(tt.wantAvailableCents),
				commissions["available_cents"], "available_cents")
			assert.Equal(t, float64(tt.wantEscrowedCents),
				commissions["escrowed_cents"], "escrowed_cents")
			assert.Equal(t, float64(tt.wantTransmitted),
				commissions["transmitted_cents"], "transmitted_cents")
			// Total must equal the single non-zero bucket — never the sum
			// of two.
			expectedTotal := tt.wantAvailableCents + tt.wantEscrowedCents + tt.wantTransmitted
			assert.Equal(t, float64(expectedTotal),
				commissions["total_cents"], "total_cents must equal exactly one bucket's value")
		})
	}
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

// fakeProposalTitleResolver satisfies the wallet handler's
// proposalTitleResolver port for unit tests, driven by a map keyed by
// proposal id.
type fakeProposalTitleResolver struct {
	titles map[uuid.UUID]string
	err    error
}

func (f *fakeProposalTitleResolver) TitleForProposal(_ context.Context, id uuid.UUID) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.titles[id], nil
}

// fakeMissionWalletLoader satisfies the narrow missionWalletLoader
// port so tests can drive the mission-side overview without
// instantiating the full payment service.
type fakeMissionWalletLoader struct {
	overview *paymentapp.WalletOverview
	err      error
}

func (f *fakeMissionWalletLoader) GetWalletOverview(_ context.Context, _, _ uuid.UUID) (*paymentapp.WalletOverview, error) {
	return f.overview, f.err
}

// TestWalletSummary_RecentTransactionsHaveTitle pins Bug 3: every
// row in recent_transactions must carry the proposal.title so the
// frontend renders the mission name instead of falling back to "Sans
// titre". Covers BOTH mission rows (records → payment_records →
// proposals.title) and commission rows (commission → proposal_id →
// proposals.title) in one go.
func TestWalletSummary_RecentTransactionsHaveTitle(t *testing.T) {
	now := time.Now().UTC()
	missionProposalA := uuid.New()
	missionProposalB := uuid.New()
	commissionProposal := uuid.New()
	missionRecA := paymentapp.WalletRecord{
		ID:             uuid.New().String(),
		ProposalID:     missionProposalA.String(),
		ProviderPayout: 100_00,
		PaymentStatus:  "succeeded",
		TransferStatus: "completed",
		CreatedAt:      now.Format("2006-01-02T15:04:05Z"),
	}
	missionRecB := paymentapp.WalletRecord{
		ID:             uuid.New().String(),
		ProposalID:     missionProposalB.String(),
		ProviderPayout: 50_00,
		PaymentStatus:  "succeeded",
		TransferStatus: "completed",
		CreatedAt:      now.Add(-1 * time.Hour).Format("2006-01-02T15:04:05Z"),
	}
	commissionRec := portservice.ReferralCommissionRecord{
		ID:              uuid.New(),
		ProposalID:      commissionProposal,
		CommissionCents: 25_00,
		Currency:        "EUR",
		Status:          "paid",
		PaidAt:          ptrTime(now.Add(-30 * time.Minute)),
		CreatedAt:       now.Add(-30 * time.Minute),
	}

	// Stub the payment service via the test export — the handler
	// resolves missions via h.paymentSvc.GetWalletOverview at request
	// time. Since *paymentapp.Service is concrete, we bypass it here
	// and verify the title path through the timeline construction
	// path directly when the records are wired into a real overview.
	// In the e2e wallet_summary path, paymentSvc must be non-nil for
	// missions to flow. For this unit test, we drive the path via a
	// new fakePaymentService injected via the WithPaymentOverview
	// builder — but the cleaner path is to use the existing
	// WithCommissionRecorder for commissions and reuse the
	// MissionLegForTest helper to assert mission-side title is
	// surfaced. Here we exercise the full handler path through the
	// commission side (which has its own seam) + the timeline
	// integration test for the mission side below.

	t.Run("mission rows carry proposal title (full handler integration)", func(t *testing.T) {
		// Drive the full Summary path through the narrow
		// missionWalletLoader port — verifies the enrichment fills
		// `mission_title` for every mission row on the page.
		loader := &fakeMissionWalletLoader{
			overview: &paymentapp.WalletOverview{
				Records: []paymentapp.WalletRecord{missionRecA, missionRecB},
			},
		}
		titles := &fakeProposalTitleResolver{
			titles: map[uuid.UUID]string{
				missionProposalA: "Mission Alpha",
				missionProposalB: "Mission Beta",
			},
		}
		wh := handler.NewWalletHandler(nil, nil).
			WithMissionWalletLoader(loader).
			WithProposalTitleResolver(titles)
		req := summaryRequest(t, uuid.New(), uuid.New(), "")
		rec := httptest.NewRecorder()
		wh.Summary(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		data := decodeSummary(t, rec)
		txs := data["recent_transactions"].([]any)
		require.Len(t, txs, 2)
		got := map[string]string{}
		for _, tx := range txs {
			row := tx.(map[string]any)
			got[row["reference_id"].(string)] = row["mission_title"].(string)
		}
		assert.Equal(t, "Mission Alpha", got[missionRecA.ID])
		assert.Equal(t, "Mission Beta", got[missionRecB.ID])
	})

	t.Run("commission rows carry proposal title via enrichment", func(t *testing.T) {
		recorder := &fakeCommissionRecorder{
			rows: []portservice.ReferralCommissionRecord{commissionRec},
		}
		titles := &fakeProposalTitleResolver{
			titles: map[uuid.UUID]string{
				commissionProposal: "Refonte landing corail",
			},
		}
		wh := handler.NewWalletHandler(nil, nil).
			WithCommissionRecorder(recorder).
			WithProposalTitleResolver(titles)
		req := summaryRequest(t, uuid.New(), uuid.New(), "")
		rec := httptest.NewRecorder()
		wh.Summary(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		data := decodeSummary(t, rec)
		txs := data["recent_transactions"].([]any)
		require.Len(t, txs, 1)
		row := txs[0].(map[string]any)
		assert.Equal(t, "Refonte landing corail", row["mission_title"],
			"commission row must surface the proposal title from the resolver")
	})

	t.Run("missing title degrades to empty string (no panic)", func(t *testing.T) {
		recorder := &fakeCommissionRecorder{
			rows: []portservice.ReferralCommissionRecord{commissionRec},
		}
		// Resolver returns "" for everything (proposal not found).
		titles := &fakeProposalTitleResolver{titles: map[uuid.UUID]string{}}
		wh := handler.NewWalletHandler(nil, nil).
			WithCommissionRecorder(recorder).
			WithProposalTitleResolver(titles)
		req := summaryRequest(t, uuid.New(), uuid.New(), "")
		rec := httptest.NewRecorder()
		wh.Summary(rec, req)
		data := decodeSummary(t, rec)
		txs := data["recent_transactions"].([]any)
		require.Len(t, txs, 1)
		row := txs[0].(map[string]any)
		_, hasTitle := row["mission_title"]
		// The handler emits the JSON field with omitempty, so an
		// empty title can either be absent or "" — both are valid.
		// Either way the UI falls back to "Sans titre".
		if hasTitle {
			assert.Equal(t, "", row["mission_title"])
		}
	})

	t.Run("resolver error degrades to empty title, no 500", func(t *testing.T) {
		recorder := &fakeCommissionRecorder{
			rows: []portservice.ReferralCommissionRecord{commissionRec},
		}
		titles := &fakeProposalTitleResolver{err: fmt.Errorf("boom-rls-denied")}
		wh := handler.NewWalletHandler(nil, nil).
			WithCommissionRecorder(recorder).
			WithProposalTitleResolver(titles)
		req := summaryRequest(t, uuid.New(), uuid.New(), "")
		rec := httptest.NewRecorder()
		wh.Summary(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code,
			"resolver error must not break the summary response")
	})
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
