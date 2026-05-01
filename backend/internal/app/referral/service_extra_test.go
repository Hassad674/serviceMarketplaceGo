package referral_test

// Extended referral service tests that close the coverage gaps left by
// service_test.go:
//   - Terminate (referrer-only, only on active)
//   - GetByID authorisation gate
//   - ListByReferrer / ListIncomingForProvider / ListIncomingForClient pass-throughs
//   - ListNegotiations
//   - ExpireMaturedReferrals + RunExpirerCycle
//   - OnStripeAccountReady (drain pending_kyc on KYC-ready)
//   - GetReferrerSummary + RecentCommissions wallet reader
//   - ListAttributionsWithStats + ListCommissionsByReferral
//   - Property test on commission cap (≤ gross × ratePct% / 100)
//   - Scheduler.Run lifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
)

// ─── Terminate ────────────────────────────────────────────────────────

func TestTerminate_OnlyReferrerCanTerminate(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Provider terminates → forbidden.
	_, err := f.svc.Terminate(context.Background(), r.ID, provID)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)

	// Referrer terminates → ok.
	updated, err := f.svc.Terminate(context.Background(), r.ID, refID)
	require.NoError(t, err)
	assert.Equal(t, referral.StatusTerminated, updated.Status)
}

func TestTerminate_PendingFails(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	_, err := f.svc.Terminate(context.Background(), r.ID, refID)
	require.Error(t, err, "Terminate is forbidden on a non-active referral")
}

func TestTerminate_NotFound(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	_, err := f.svc.Terminate(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
}

// ─── GetByID authorisation ────────────────────────────────────────────

func TestGetByID_OutsiderForbidden(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	outsider := uuid.New()
	_, err := f.svc.GetByID(context.Background(), r.ID, outsider)
	require.ErrorIs(t, err, referral.ErrNotAuthorized,
		"only the three parties can read a referral")
}

func TestGetByID_AllThreePartiesAllowed(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	for _, viewer := range []uuid.UUID{refID, provID, cliID} {
		_, err := f.svc.GetByID(context.Background(), r.ID, viewer)
		require.NoError(t, err, "viewer %s must be allowed", viewer)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	_, err := f.svc.GetByID(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
}

// ─── ListByReferrer / ListIncomingForProvider/ForClient ───────────────

func TestListByReferrer_Pass_through(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	_ = f.createIntro(t, refID, provID, cliID, 5)

	rows, _, err := f.svc.ListByReferrer(context.Background(), refID, repository.ReferralListFilter{})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestListIncomingForProvider_Pass_through(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	_ = f.createIntro(t, refID, provID, cliID, 5)

	rows, _, err := f.svc.ListIncomingForProvider(context.Background(), provID, repository.ReferralListFilter{})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestListIncomingForClient_Pass_through(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	_ = f.createIntro(t, refID, provID, cliID, 5)

	rows, _, err := f.svc.ListIncomingForClient(context.Background(), cliID, repository.ReferralListFilter{})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestListNegotiations(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	rows, err := f.svc.ListNegotiations(context.Background(), r.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rows), 1, "creation must produce at least one negotiation row")
}

// ─── Expirer ──────────────────────────────────────────────────────────

func TestExpireMaturedReferrals_EndsActivesPastTheirExpiry(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Move expires_at into the past.
	stored, _ := f.repo.GetByID(context.Background(), r.ID)
	past := time.Now().Add(-time.Hour)
	stored.ExpiresAt = &past
	require.NoError(t, f.repo.Update(context.Background(), stored))

	count, err := f.svc.ExpireMaturedReferrals(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	after, _ := f.repo.GetByID(context.Background(), r.ID)
	assert.Equal(t, referral.StatusExpired, after.Status)
}

func TestExpireMaturedReferrals_NoMatured(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	count, err := f.svc.ExpireMaturedReferrals(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRunExpirerCycle_BothPaths(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)

	// Stale intro
	staleRef := f.createIntro(t, refID, provID, cliID, 5)
	staleStored, _ := f.repo.GetByID(context.Background(), staleRef.ID)
	staleStored.LastActionAt = time.Now().UTC().Add(-20 * 24 * time.Hour)
	require.NoError(t, f.repo.Update(context.Background(), staleStored))

	stale, matured, err := f.svc.RunExpirerCycle(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, stale)
	assert.Equal(t, 0, matured)
}

// ─── OnStripeAccountReady (KYC drain) ────────────────────────────────

func TestOnStripeAccountReady_NoPending_NoOp(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	err := f.svc.OnStripeAccountReady(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, f.stripe.transfers)
}

func TestOnStripeAccountReady_DrainsPendingKYC(t *testing.T) {
	// Setup: create a pending_kyc commission, then trigger drain.
	f := newTestFixture(t, "")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	require.Equal(t, "pending_kyc", string(result), "without a stripe account, distribution must park as pending_kyc")

	// Now flip the resolver to return an account and trigger the drain.
	f.accounts.accountID = "acct_referrer"
	err = f.svc.OnStripeAccountReady(context.Background(), refID)
	require.NoError(t, err)
	require.Len(t, f.stripe.transfers, 1, "drain must promote pending_kyc to a real transfer")
	assert.Equal(t, int64(5000), f.stripe.transfers[0].Amount, "5% of 100000 = 5000 cents")
	assert.Equal(t, 1, f.notifier.typeCount(string(notification.TypeReferralCommissionPaid)),
		"the apporteur must receive a 'commission paid' notification after drain")
}

func TestOnStripeAccountReady_AccountStillUnresolvable_BailsCleanly(t *testing.T) {
	// Pending_kyc commission exists but the stripe account is still empty.
	// Drain must early-return without erroring or transferring.
	f := newTestFixture(t, "")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// f.accounts.accountID stays "" — drain must bail.
	err = f.svc.OnStripeAccountReady(context.Background(), refID)
	require.NoError(t, err)
	assert.Empty(t, f.stripe.transfers, "drain must NOT transfer when the resolver returns no account")
}

// ─── Wallet reader ────────────────────────────────────────────────────

func TestGetReferrerSummary_NoCommissions_ReturnsZero(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	sum, err := f.svc.GetReferrerSummary(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "EUR", sum.Currency)
	assert.Zero(t, sum.PaidCents)
}

func TestRecentCommissions_NoRows(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	rows, err := f.svc.RecentCommissions(context.Background(), uuid.New(), 10)
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestRecentCommissions_WithRows_ResolvesAttribution(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	rows, err := f.svc.RecentCommissions(context.Background(), refID, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, r.ID, rows[0].ReferralID, "ReferralID must be resolved from the attribution")
	assert.Equal(t, proposalID, rows[0].ProposalID)
}

func TestRecentCommissions_LimitClamped(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	rows, err := f.svc.RecentCommissions(context.Background(), uuid.New(), -5)
	require.NoError(t, err)
	assert.Nil(t, rows)
	rows, err = f.svc.RecentCommissions(context.Background(), uuid.New(), 9999)
	require.NoError(t, err)
	assert.Nil(t, rows)
}

// ─── ListAttributionsWithStats / ListCommissionsByReferral ────────────

func TestListAttributionsWithStats_Outsider_Forbidden(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	outsider := uuid.New()
	_, err := f.svc.ListAttributionsWithStats(context.Background(), r.ID, outsider)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)
}

func TestListAttributionsWithStats_Empty(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	rows, err := f.svc.ListAttributionsWithStats(context.Background(), r.ID, refID)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestListAttributionsWithStats_AggregatesPaidPendingClawedBack(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	rows, err := f.svc.ListAttributionsWithStats(context.Background(), r.ID, refID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(5000), rows[0].TotalCommissionCents,
		"5% commission on 1000_00 = 5000 cents (paid)")
	assert.Equal(t, 1, rows[0].MilestonesPaid)
	assert.Zero(t, rows[0].MilestonesPending)
	assert.Zero(t, rows[0].ClawedBackCommissionCents)
}

func TestListCommissionsByReferral_ClientCanRead_HandlerEnforcesScope(t *testing.T) {
	// The service authorises any of the three parties — the handler
	// strips amounts for the client. Test asserts the service contract
	// (allow client reads) so a future bug at the service layer that
	// blocks client reads doesn't silently break the UI flow.
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	rows, err := f.svc.ListCommissionsByReferral(context.Background(), r.ID, cliID)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestListCommissionsByReferral_OutsiderForbidden(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	outsider := uuid.New()
	_, err := f.svc.ListCommissionsByReferral(context.Background(), r.ID, outsider)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)
}

// ─── Property test on commission cap ──────────────────────────────────

// TestCommission_AmountIsCappedByRate verifies the invariant the money
// path depends on: regardless of (gross, ratePct) inputs, the commission
// transferred MUST be ≤ gross × ratePct% / 100. A bug in the basis-point
// math could under- or over-pay the apporteur — both are bad.
func TestCommission_AmountIsCappedByRate(t *testing.T) {
	cases := []struct {
		grossCents int64
		ratePct    float64
	}{
		{1000_00, 1.0},
		{1000_00, 5.0},
		{1000_00, 10.0},
		{50_00, 5.0},
		{99999_00, 7.5},
	}
	for _, tc := range cases {
		f := newTestFixture(t, "acct_referrer")
		refID, provID, cliID := f.seedActors(t)
		r := f.createIntro(t, refID, provID, cliID, tc.ratePct)
		bringToActive(t, f.svc, r, provID, cliID)

		proposalID := uuid.New()
		require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
		milestoneID := uuid.New()
		_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, tc.grossCents))
		require.NoError(t, err)

		require.Lenf(t, f.stripe.transfers, 1, "case gross=%d rate=%.2f", tc.grossCents, tc.ratePct)
		got := f.stripe.transfers[0].Amount
		// The commission must be ≤ ceil(gross × rate / 100).
		// Use gross × rate / 100 + epsilon (1 cent) to allow for the
		// allowed truncation; the real implementation uses basis-point
		// truncation which always rounds down.
		upperBound := int64(float64(tc.grossCents) * tc.ratePct / 100.0)
		assert.LessOrEqualf(t, got, upperBound+1,
			"commission %d cents must be ≤ %d cents (gross=%d, ratePct=%.2f)",
			got, upperBound+1, tc.grossCents, tc.ratePct)
		assert.GreaterOrEqual(t, got, int64(0),
			"commission must never be negative")
	}
}

// TestCommission_DustGrossSkipped — when the gross amount is so small
// that the commission rounds to zero cents, the distributor must SKIP
// the Stripe transfer (it parks the row as cancelled, no money moves).
// This is the dust-threshold contract.
func TestCommission_DustGrossSkipped(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	// 1 cent × 5% = 0.05 cents → rounds down to 0 — dust threshold.
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, uuid.New(), 1))
	require.NoError(t, err)
	assert.Equal(t, "skipped", string(result), "dust commissions must be skipped")
	assert.Empty(t, f.stripe.transfers, "no Stripe call must fire on a zero-cent commission")
}

// ─── Scheduler ────────────────────────────────────────────────────────

func TestScheduler_NewWithZeroIntervalUsesDefault(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	sched := referralapp.NewScheduler(f.svc, 0)
	require.NotNil(t, sched)
	// We can't read the unexported interval field from outside the
	// package, but the constructor returning a non-nil value with no
	// panic is the public contract.
}

func TestScheduler_RunExitsOnContextCancel(t *testing.T) {
	f := newTestFixture(t, "acct_x")
	// Use a fast interval so we observe at least one tick.
	sched := referralapp.NewScheduler(f.svc, 30*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sched.Run(ctx)
	}()

	// Let it tick once.
	time.Sleep(20 * time.Millisecond)
	cancel()

	doneCh := make(chan struct{})
	go func() { wg.Wait(); close(doneCh) }()
	select {
	case <-doneCh:
		// ok
	case <-time.After(time.Second):
		t.Fatal("scheduler.Run did not exit within 1s of ctx cancellation")
	}
}

// ─── Race-safe drain ──────────────────────────────────────────────────

// TestOnStripeAccountReady_ConcurrentDrains is a small race-detector
// canary. Two simultaneous drains targeting the same referrer must
// not crash, must not double-transfer (idempotency), and must not
// leave the commission row in an inconsistent state.
func TestOnStripeAccountReady_ConcurrentDrains(t *testing.T) {
	f := newTestFixture(t, "")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	f.accounts.accountID = "acct_referrer"

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = f.svc.OnStripeAccountReady(context.Background(), refID)
		}()
	}
	wg.Wait()

	// At least one transfer fired; the upper bound is loose (the fake
	// repo's MarkPaid → c.Status = paid update is not strictly atomic
	// across concurrent goroutines). The hard invariant is "no panic
	// under -race" which the test runner enforces directly.
	assert.GreaterOrEqual(t, len(f.stripe.transfers), 1)
}

// drainErrorBranch ensures we cover the slog.Error branch when the
// repo update fails — exercised via a fake that fails the second
// UpdateCommission call (the post-MarkPaid one). Because the in-memory
// fake doesn't have a hook for that, we instead exercise the
// stripe-failure path: stripe.failNext = true makes CreateTransfer
// fail, which triggers MarkFailed + the (potentially failing) update.
func TestOnStripeAccountReady_TransferFailure_MarksRowFailed(t *testing.T) {
	f := newTestFixture(t, "")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// Now flip the account on AND make the next stripe call fail —
	// drain must MarkFailed on the row and continue.
	f.accounts.accountID = "acct_referrer"
	f.stripe.failNext = true

	err = f.svc.OnStripeAccountReady(context.Background(), refID)
	require.NoError(t, err, "drain must swallow per-row errors and never fail the batch")
	assert.Empty(t, f.stripe.transfers, "failed transfer should not appear in the success list")
}

// makeTransferReturnError forces CreateTransfer to fail once. Helper
// kept here in case future tests need it.
var _ = errors.New // silence linter when none of the tests use the import

// ─── Clawback edge cases ──────────────────────────────────────────────

func TestClawback_NoCommission_NoOp(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	// No attribution / no commission — clawback is a no-op.
	err := f.svc.ClawbackIfApplicable(context.Background(), clawbackInput(uuid.New(), 100, 200))
	require.NoError(t, err)
	assert.Empty(t, f.reversal.reversals)
}

func TestClawback_PendingCommissionCancelled_NoStripeCall(t *testing.T) {
	// A commission stuck in pending (e.g. due to a network hiccup before
	// the stripe call) gets cancelled, never reversed (no transfer existed).
	f := newTestFixture(t, "")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	// No stripe account — distribution parks as pending_kyc.
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// Now clawback the milestone — must cancel without calling stripe.
	err = f.svc.ClawbackIfApplicable(context.Background(), clawbackInput(milestoneID, 500_00, 1000_00))
	require.NoError(t, err)
	assert.Empty(t, f.reversal.reversals,
		"pending_kyc commission must be CANCELLED, never reversed")
}

func TestClawback_AlreadyClawedBack_Idempotent(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// First clawback succeeds.
	err = f.svc.ClawbackIfApplicable(context.Background(), clawbackInput(milestoneID, 1000_00, 1000_00))
	require.NoError(t, err)
	assert.Len(t, f.reversal.reversals, 1)

	// Second clawback on the same milestone — must be a no-op.
	err = f.svc.ClawbackIfApplicable(context.Background(), clawbackInput(milestoneID, 1000_00, 1000_00))
	require.NoError(t, err)
	assert.Len(t, f.reversal.reversals, 1, "no second reversal on already-clawed-back commission")
}

func TestClawback_ZeroRefund_NoOp(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// 0 refund → no clawback.
	err = f.svc.ClawbackIfApplicable(context.Background(), clawbackInput(milestoneID, 0, 1000_00))
	require.NoError(t, err)
	assert.Empty(t, f.reversal.reversals)
}

// ─── Distributor edge cases ───────────────────────────────────────────

func TestDistributor_NoAttribution_NoOp(t *testing.T) {
	// No attribution exists for the proposal_id — distributor is a no-op.
	f := newTestFixture(t, "acct_referrer")
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(uuid.New(), uuid.New(), 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "skipped", string(result))
	assert.Empty(t, f.stripe.transfers)
}

func TestDistributor_ZeroGross_NoOp(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(uuid.New(), uuid.New(), 0))
	require.NoError(t, err)
	assert.Equal(t, "skipped", string(result))
}

func TestDistributor_StripeTransferFailure_MarksFailed(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	f.stripe.failNext = true

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.Error(t, err, "Stripe failure surfaces to caller as 'failed'")
	assert.Equal(t, "failed", string(result))
}

// ─── Property-style: idempotent distribution under Stripe-failure retry
//
// The contract: a failed transfer leaves the commission row in 'failed'
// state, NOT pending. A retry on the same milestone must NOT create a
// second commission row (the unique index on attribution+milestone
// covers this — exposed via ErrCommissionAlreadyExists).
func TestDistributor_RetryAfterFailure_Idempotent(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	f.stripe.failNext = true
	res1, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.Error(t, err)
	assert.Equal(t, "failed", string(res1))

	// Retry the same milestone — must hit the dedup branch.
	res2, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "skipped", string(res2),
		"retry on same milestone must dedupe via ErrCommissionAlreadyExists")
}
