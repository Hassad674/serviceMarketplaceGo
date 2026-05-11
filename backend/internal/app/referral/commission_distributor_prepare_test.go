package referral_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
	portservice "marketplace-backend/internal/port/service"
)

// prepareInputFor mirrors distInputFor for the new preparer port.
func prepareInputFor(proposalID, milestoneID uuid.UUID, grossCents int64) portservice.ReferralCommissionPrepareInput {
	return portservice.ReferralCommissionPrepareInput{
		ProposalID:       proposalID,
		MilestoneID:      milestoneID,
		GrossAmountCents: grossCents,
		Currency:         "EUR",
	}
}

// TestPrepareCommissionForMilestone_CreatesPendingRow verifies the
// happy path: when an attribution exists for the proposal, the
// preparer inserts a commission row in `pending` status and does NOT
// touch Stripe (the transfer is somebody else's job).
func TestPrepareCommissionForMilestone_CreatesPendingRow(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	err := f.svc.PrepareCommissionForMilestone(context.Background(), prepareInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// The commission row exists in pending status, no Stripe transfer fired.
	row, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPending, row.Status)
	assert.Equal(t, int64(5000), row.CommissionCents) // 5% of 100000 = 5000
	assert.Empty(t, f.stripe.transfers, "preparer must NOT call Stripe")
}

// TestPrepareCommissionForMilestone_NoAttribution_NoOp verifies the
// no-op path: when no attribution exists for the proposal, the
// preparer silently returns without persisting anything.
func TestPrepareCommissionForMilestone_NoAttribution_NoOp(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	proposalID := uuid.New()
	milestoneID := uuid.New()

	err := f.svc.PrepareCommissionForMilestone(context.Background(), prepareInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err, "missing attribution must NOT raise an error")

	_, ferr := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.ErrorIs(t, ferr, referral.ErrCommissionNotFound,
		"no commission row may be created without an attribution")
}

// TestPrepareCommissionForMilestone_Idempotent verifies that calling
// the preparer twice on the same milestone is a no-op (the unique
// index on (attribution_id, milestone_id) shields the row).
func TestPrepareCommissionForMilestone_Idempotent(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	input := prepareInputFor(proposalID, milestoneID, 1000_00)
	require.NoError(t, f.svc.PrepareCommissionForMilestone(context.Background(), input))
	// Second call: silent no-op.
	require.NoError(t, f.svc.PrepareCommissionForMilestone(context.Background(), input))

	// Exactly one commission row exists.
	all, err := f.repo.ListCommissionsByReferral(context.Background(), r.ID)
	require.NoError(t, err)
	assert.Len(t, all, 1, "idempotent preparer must not duplicate the commission row")
}

// TestDistributeIfApplicable_DrainsPreparedRow verifies the
// post-preparer contract on DistributeIfApplicable: when a pending
// commission row already exists (because PrepareCommissionForMilestone
// ran on milestone approval), the distributor reloads it and fires
// the Stripe transfer on the existing row instead of skipping.
func TestDistributeIfApplicable_DrainsPreparedRow(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	// 1) PrepareCommissionForMilestone (proposal-side, on approval).
	require.NoError(t, f.svc.PrepareCommissionForMilestone(context.Background(), prepareInputFor(proposalID, milestoneID, 1000_00)))
	// 2) DistributeIfApplicable (payment-side, on provider transfer)
	//    must drain the existing pending row to Stripe.
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "paid", string(result))
	require.Len(t, f.stripe.transfers, 1)

	// The row is now `paid` and tagged with the Stripe transfer id.
	row, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPaid, row.Status)
	assert.NotEmpty(t, row.StripeTransferID)
}

// TestSweepPendingCommissions_DrainsAfterGracePeriod verifies the
// scheduler safety net: a pending commission row older than the
// grace period is drained to Stripe on the next sweep tick.
func TestSweepPendingCommissions_DrainsAfterGracePeriod(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	require.NoError(t, f.svc.PrepareCommissionForMilestone(context.Background(), prepareInputFor(proposalID, milestoneID, 1000_00)))

	// Backdate the commission row to make it older than the sweeper
	// grace period — the fake repo gives us direct access.
	f.repo.backdateCommissionsBy(time.Hour)

	processed, err := f.svc.SweepPendingCommissions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	require.Len(t, f.stripe.transfers, 1)

	row, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPaid, row.Status)
}

// TestSweepPendingCommissions_PendingKYCWhenNoAccount verifies that
// the sweeper parks the row as pending_kyc when the referrer has no
// Stripe Connect account yet — same outcome as DistributeIfApplicable
// for parity.
func TestSweepPendingCommissions_PendingKYCWhenNoAccount(t *testing.T) {
	f := newTestFixture(t, "") // empty stripe account
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	require.NoError(t, f.svc.PrepareCommissionForMilestone(context.Background(), prepareInputFor(proposalID, milestoneID, 1000_00)))
	f.repo.backdateCommissionsBy(time.Hour)

	processed, err := f.svc.SweepPendingCommissions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.Empty(t, f.stripe.transfers, "no transfer must fire without a destination account")

	row, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPendingKYC, row.Status)
}

// TestSweepPendingCommissions_RespectsGracePeriod verifies that a
// freshly-created pending row is NOT swept (the grace period gives
// the legacy DistributeIfApplicable path a chance to fire first).
func TestSweepPendingCommissions_RespectsGracePeriod(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	require.NoError(t, f.svc.PrepareCommissionForMilestone(context.Background(), prepareInputFor(proposalID, milestoneID, 1000_00)))
	// Do NOT backdate — the row is "fresh".

	processed, err := f.svc.SweepPendingCommissions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, processed, "fresh pending rows must remain untouched")
	assert.Empty(t, f.stripe.transfers)
}
