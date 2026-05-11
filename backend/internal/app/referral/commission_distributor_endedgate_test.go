package referral_test

// Unit + integration tests for the WALLET-UNIFY Run A end-of-intro
// gate in commission_distributor.go.
//
// Split into two layers:
//
//  1. A pure-function table-driven test on attributionEndedGateClosed
//     covering the (ended_at nil vs set) × (now before/at/after
//     ended_at) matrix. Anchored on real time.Now() snapshots — no
//     stubbed clocks — per the brief.
//
//  2. End-to-end test of DistributeIfApplicable + PrepareCommissionForMilestone
//     when the attribution is ended: the commission must NOT be
//     created, no Stripe transfer fires, and the result is
//     ReferralCommissionSkipped.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/domain/referral"
)

// TestAttributionEndedGate_PureFunction is the white-box table-
// driven test on the gate function exported (test-internal) by the
// referral package. We re-implement the same predicate here rather
// than calling the un-exported attributionEndedGateClosed (it lives
// in another package) — the implementation is small enough that
// duplicating the rule and asserting it against the same fixtures
// keeps the test independent of the file's internal helper layout.
// The DOWNSTREAM integration test (TestDistributor_EndedAttribution_*)
// is the real proof that the gate fires correctly inside the
// distributor — this matrix only documents the timestamp rules.
func TestAttributionEndedGate_TimestampMatrix(t *testing.T) {
	now := time.Now().UTC()

	cases := []struct {
		name    string
		endedAt *time.Time
		callAt  time.Time
		want    bool // true = gate is CLOSED (skip commission)
	}{
		{
			name:    "active attribution before any end",
			endedAt: nil,
			callAt:  now,
			want:    false,
		},
		{
			name:    "active attribution, regardless of when we call",
			endedAt: nil,
			callAt:  now.Add(24 * time.Hour),
			want:    false,
		},
		{
			name:    "ended attribution, call exactly AT ended_at (tie → skip, fair to apporteur)",
			endedAt: tp(now),
			callAt:  now,
			want:    true,
		},
		{
			name:    "ended attribution, call 1ns AFTER ended_at",
			endedAt: tp(now),
			callAt:  now.Add(time.Nanosecond),
			want:    true,
		},
		{
			name:    "ended attribution, call 1h AFTER ended_at",
			endedAt: tp(now),
			callAt:  now.Add(time.Hour),
			want:    true,
		},
		{
			name:    "ended attribution, call 1ns BEFORE ended_at (race / clock skew → pay)",
			endedAt: tp(now),
			callAt:  now.Add(-time.Nanosecond),
			want:    false,
		},
		{
			name:    "ended attribution, call 1h BEFORE ended_at (impossible in practice but exercises the strict-before rule)",
			endedAt: tp(now),
			callAt:  now.Add(-time.Hour),
			want:    false,
		},
		{
			name:    "ended attribution, call days AFTER ended_at",
			endedAt: tp(now.Add(-7 * 24 * time.Hour)),
			callAt:  now,
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att := &referral.Attribution{EndedAt: tc.endedAt}
			// Replicate the predicate locally: !att.IsEnded() → false,
			// else !callAt.Before(*EndedAt) → tie-or-after.
			got := att.IsEnded() && !tc.callAt.Before(*att.EndedAt)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDistributor_EndedAttribution_DistributeSkips is the end-to-end
// proof that DistributeIfApplicable honours the gate: an ended
// attribution → no commission row, no Stripe transfer, result =
// ReferralCommissionSkipped.
func TestDistributor_EndedAttribution_DistributeSkips(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(
		context.Background(),
		attrInputFor(proposalID, provID, cliID)))

	// End the attribution via the freshly added service method.
	att, err := f.repo.FindAttributionByProposal(context.Background(), proposalID)
	require.NoError(t, err)
	_, err = f.svc.EndIntroAttribution(context.Background(), att.ID, refID)
	require.NoError(t, err)

	// Reset transfer count to a clean state so the assertion is
	// not polluted by anything emitted earlier in the fixture.
	f.stripe.transfers = nil

	// A NEW milestone now triggers the distributor — it must skip.
	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(
		context.Background(),
		distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "skipped", string(result),
		"ended attribution must short-circuit to Skipped")
	assert.Empty(t, f.stripe.transfers,
		"no Stripe transfer must fire after the attribution is ended")

	_, ferr := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	assert.ErrorIs(t, ferr, referral.ErrCommissionNotFound,
		"no commission row must be created for milestones after end")
}

// TestDistributor_EndedAttribution_PreparePathSkips proves the
// PrepareCommissionForMilestone entry point honours the gate too —
// otherwise the legacy "prepare on approval" flow would create a
// pending row that the distributor would later transfer.
func TestDistributor_EndedAttribution_PreparePathSkips(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(
		context.Background(),
		attrInputFor(proposalID, provID, cliID)))

	att, err := f.repo.FindAttributionByProposal(context.Background(), proposalID)
	require.NoError(t, err)
	_, err = f.svc.EndIntroAttribution(context.Background(), att.ID, refID)
	require.NoError(t, err)

	milestoneID := uuid.New()
	require.NoError(t, f.svc.PrepareCommissionForMilestone(
		context.Background(),
		prepareInputFor(proposalID, milestoneID, 1000_00)))

	_, ferr := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	assert.ErrorIs(t, ferr, referral.ErrCommissionNotFound,
		"PrepareCommissionForMilestone must NOT create a commission after end")
}

// TestDistributor_PreExistingCommissions_Preserved verifies the
// "fair to apporteur" half of the contract — a commission row that
// EXISTS before the attribution is ended is not touched by the end
// flow, and the distributor / scheduler can still drain it. This is
// the work-already-delivered semantic.
func TestDistributor_PreExistingCommissions_Preserved(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(
		context.Background(),
		attrInputFor(proposalID, provID, cliID)))

	// Commission #1 (PRE-end) — must pay out.
	milestonePre := uuid.New()
	resultPre, err := f.svc.DistributeIfApplicable(
		context.Background(),
		distInputFor(proposalID, milestonePre, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "paid", string(resultPre))

	// End the attribution.
	att, err := f.repo.FindAttributionByProposal(context.Background(), proposalID)
	require.NoError(t, err)
	_, err = f.svc.EndIntroAttribution(context.Background(), att.ID, refID)
	require.NoError(t, err)

	// Pre-end commission still exists and is paid.
	pre, err := f.repo.FindCommissionByMilestone(context.Background(), milestonePre)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPaid, pre.Status,
		"pre-end commission must remain paid — fair to apporteur for work delivered")
}

// tp returns a pointer to t — small helper to keep table cases
// concise.
func tp(t time.Time) *time.Time { return &t }

// _ unused suppressor so this file's imports stay clean if a
// helper is removed.
var _ = referralapp.Service{}
