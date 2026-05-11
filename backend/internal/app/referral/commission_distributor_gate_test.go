package referral_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
	portservice "marketplace-backend/internal/port/service"
)

// TestDistributor_ConnectGate_PayoutsDisabled verifies the new D1+D2
// Connect-ready gate: when the apporteur has an account id on file but
// Stripe reports `payouts_enabled=false`, the commission lands in
// pending_kyc and NO transfer is attempted. This is the production-
// observed failure mode: the apporteur started onboarding but did not
// finish — burning a Stripe idempotency key on the doomed transfer
// would have made the eventual retry impossible.
func TestDistributor_ConnectGate_PayoutsDisabled(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Simulate "account exists but payouts not yet enabled".
	f.stripe.account = &portservice.StripeAccountInfo{
		ChargesEnabled: false,
		PayoutsEnabled: false,
	}

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "pending_kyc", string(result))
	assert.Empty(t, f.stripe.transfers, "no Stripe transfer when account not payable")

	row, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPendingKYC, row.Status)
}

// TestDistributor_ConnectGate_OnlyChargesEnabled verifies that an
// account with `charges_enabled=true` but `payouts_enabled=false` is
// still treated as not-ready. Both capabilities must be active before
// a transfer fires — this guards against the partial-readiness state
// Stripe reports during the onboarding ramp.
func TestDistributor_ConnectGate_OnlyChargesEnabled(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	f.stripe.account = &portservice.StripeAccountInfo{
		ChargesEnabled: true,
		PayoutsEnabled: false,
	}

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "pending_kyc", string(result))
	assert.Empty(t, f.stripe.transfers)
}

// TestDistributor_ConnectGate_StripeError verifies the fail-CLOSED
// behaviour: when the GetAccount probe returns an error (transient
// network failure, Stripe outage, etc.), the gate treats the account
// as not-ready and parks the commission in pending_kyc. A user click
// on Retirer will re-run the gate, and on a successful probe the
// transfer fires.
func TestDistributor_ConnectGate_StripeError(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Stripe probe failure → fail closed.
	f.stripe.account = nil
	f.stripe.accountErr = errors.New("stripe boom")

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "pending_kyc", string(result))
	assert.Empty(t, f.stripe.transfers, "no transfer on Stripe probe failure")
}

// TestDistributor_ConnectGate_NoAccountSnapshot verifies that an
// account_id present + GetAccount returning nil snapshot (no error
// but no payload — e.g. Stripe API version mismatch projecting an
// empty object) is treated as not-ready. Defense-in-depth: never
// CreateTransfer without an affirmative readiness signal.
func TestDistributor_ConnectGate_NoAccountSnapshot(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	f.stripe.account = nil
	f.stripe.accountErr = nil

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "pending_kyc", string(result))
}

// TestDistributor_ConnectGate_HappyPath restates the canonical happy
// case for completeness in this file: account id + payouts_enabled +
// charges_enabled → transfer fires, commission becomes paid. This is
// already covered by TestDistributor_PaysCommission but having it
// here makes the gate test file self-documenting for the next reader.
func TestDistributor_ConnectGate_HappyPath(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Default fixture already sets charges+payouts enabled; restate
	// for clarity.
	f.stripe.account = &portservice.StripeAccountInfo{
		ChargesEnabled: true,
		PayoutsEnabled: true,
	}

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "paid", string(result))
	require.Len(t, f.stripe.transfers, 1)

	row, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPaid, row.Status)
}
