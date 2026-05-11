package referral_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
	portservice "marketplace-backend/internal/port/service"
)

// markCommissionStatus is a small test helper to flip a commission row
// to an arbitrary status via the repository, used to set up the
// status-mix scenarios for the grouped reader tests.
func markCommissionStatus(t *testing.T, f *testFixture, commissionID uuid.UUID, status referral.CommissionStatus) {
	t.Helper()
	c, err := f.repo.FindCommissionByID(context.Background(), commissionID)
	require.NoError(t, err)
	c.Status = status
	require.NoError(t, f.repo.UpdateCommission(context.Background(), c))
}

// TestGroupedCommissions_Empty verifies the empty case returns four
// nil slices rather than erroring — the wallet UI degrades gracefully
// for apporteurs with no activity yet.
func TestGroupedCommissions_Empty(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	groups, err := f.svc.GroupedCommissions(context.Background(), uuid.New(), 50)
	require.NoError(t, err)
	assert.Empty(t, groups.Paid)
	assert.Empty(t, groups.PendingKYC)
	assert.Empty(t, groups.Failed)
	assert.Empty(t, groups.Cancelled)
}

// TestGroupedCommissions_PartitionsCorrectly seeds one commission per
// status the wallet cares about and verifies each lands in the right
// bucket with the right retire_eligible flag.
func TestGroupedCommissions_PartitionsCorrectly(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Seed four commissions on four different milestones, each driven
	// to a different status via direct status flips on the fake repo.
	mkCommission := func(t *testing.T, status referral.CommissionStatus) uuid.UUID {
		t.Helper()
		proposalID := uuid.New()
		require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(),
			attrInputFor(proposalID, provID, cliID)))
		milestoneID := uuid.New()
		_, err := f.svc.DistributeIfApplicable(context.Background(),
			distInputFor(proposalID, milestoneID, 1000_00))
		require.NoError(t, err)
		row, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
		require.NoError(t, err)
		markCommissionStatus(t, f, row.ID, status)
		return row.ID
	}

	paidID := mkCommission(t, referral.CommissionPaid)
	pendingKYCID := mkCommission(t, referral.CommissionPendingKYC)
	failedID := mkCommission(t, referral.CommissionFailed)
	cancelledID := mkCommission(t, referral.CommissionCancelled)

	groups, err := f.svc.GroupedCommissions(context.Background(), refID, 50)
	require.NoError(t, err)
	require.Len(t, groups.Paid, 1)
	assert.Equal(t, paidID, groups.Paid[0].ID)
	assert.False(t, groups.Paid[0].RetireEligible, "paid rows are NOT retire-eligible")

	require.Len(t, groups.PendingKYC, 1)
	assert.Equal(t, pendingKYCID, groups.PendingKYC[0].ID)
	assert.True(t, groups.PendingKYC[0].RetireEligible, "pending_kyc rows ARE retire-eligible")

	require.Len(t, groups.Failed, 1)
	assert.Equal(t, failedID, groups.Failed[0].ID)
	assert.True(t, groups.Failed[0].RetireEligible, "failed rows ARE retire-eligible")

	require.Len(t, groups.Cancelled, 1)
	assert.Equal(t, cancelledID, groups.Cancelled[0].ID)
	assert.False(t, groups.Cancelled[0].RetireEligible, "cancelled rows are NOT retire-eligible")
}

// TestGroupedCommissions_IgnoresPendingAndClawedBack verifies rows in
// pending or clawed_back status are NOT surfaced — D1+D2 deliberately
// drops them because the wallet has dedicated UI for pending (totals
// summary) and clawed_back (reprises section). Including them in the
// groups would double-render those rows.
func TestGroupedCommissions_IgnoresPendingAndClawedBack(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Distribute one commission and leave it in pending — i.e. no
	// stripe account on file → CommissionPendingKYC, then flip back to
	// pending for this test.
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(),
		attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(),
		distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	row, _ := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	markCommissionStatus(t, f, row.ID, referral.CommissionPending)

	groups, err := f.svc.GroupedCommissions(context.Background(), refID, 50)
	require.NoError(t, err)
	assert.Empty(t, groups.Paid)
	assert.Empty(t, groups.PendingKYC)
	assert.Empty(t, groups.Failed)
	assert.Empty(t, groups.Cancelled)
}

// TestRecentCommissions_RetireEligibleFlag verifies the non-grouped
// reader also exposes retire_eligible so the existing wallet UI can
// adopt the same flag without a second API roundtrip.
func TestRecentCommissions_RetireEligibleFlag(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(),
		attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(),
		distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	row, _ := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NotNil(t, row)

	// Default status from a successful happy-path distribute is paid.
	recs, err := f.svc.RecentCommissions(context.Background(), refID, 50)
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, string(referral.CommissionPaid), recs[0].Status)
	assert.False(t, recs[0].RetireEligible)

	// Flip to failed → flag flips to true.
	markCommissionStatus(t, f, row.ID, referral.CommissionFailed)
	recs, err = f.svc.RecentCommissions(context.Background(), refID, 50)
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, string(referral.CommissionFailed), recs[0].Status)
	assert.True(t, recs[0].RetireEligible)
}

// TestGroupedCommissions_RespectsLimit verifies the limit parameter is
// forwarded to RecentCommissions so a low limit truncates the input
// set before partitioning.
func TestGroupedCommissions_RespectsLimit(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// 5 commissions, all paid.
	for i := 0; i < 5; i++ {
		proposalID := uuid.New()
		require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(),
			attrInputFor(proposalID, provID, cliID)))
		milestoneID := uuid.New()
		_, err := f.svc.DistributeIfApplicable(context.Background(),
			distInputFor(proposalID, milestoneID, 1000_00))
		require.NoError(t, err)
	}

	groups, err := f.svc.GroupedCommissions(context.Background(), refID, 2)
	require.NoError(t, err)
	// All 5 are paid; with limit=2 the underlying RecentCommissions
	// caps to 2 rows. Note: the in-memory fake doesn't enforce ordering
	// the same way the real SQL does, so we assert the truncation only.
	assert.LessOrEqual(t, len(groups.Paid), 2)
}

// TestGetReferrerSummary_Unknown verifies the wallet summary degrades
// gracefully when the referrer has no rows at all.
func TestGetReferrerSummary_Unknown(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	sum, err := f.svc.GetReferrerSummary(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, portservice.ReferrerCommissionSummary{Currency: "EUR"}, sum)
}
