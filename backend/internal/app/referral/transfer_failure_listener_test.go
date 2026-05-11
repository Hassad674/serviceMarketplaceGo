package referral_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

// TestOnTransferFailed_UnknownTransfer verifies the no-op path — a
// transfer.failed for a transfer id that does not match any commission
// row (e.g. milestone payout to a provider) returns nil without
// touching state.
func TestOnTransferFailed_UnknownTransfer(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	err := f.svc.OnTransferFailed(context.Background(), "tr_unknown", "card declined")
	require.NoError(t, err)
}

// TestOnTransferFailed_EmptyTransferID verifies the defensive guard
// against malformed payloads — empty transfer id is a silent no-op.
func TestOnTransferFailed_EmptyTransferID(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	err := f.svc.OnTransferFailed(context.Background(), "", "anything")
	require.NoError(t, err)
}

// TestOnTransferFailed_FlipsPaidCommission verifies the listener
// transitions a paid commission to failed when Stripe later reports
// the transfer failed. The failure_message ends up in failure_reason
// so support can investigate.
func TestOnTransferFailed_FlipsPaidCommission(t *testing.T) {
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
	require.Equal(t, referral.CommissionPaid, row.Status)
	require.NotEmpty(t, row.StripeTransferID)

	// Fire transfer.failed against that exact transfer id.
	err = f.svc.OnTransferFailed(context.Background(), row.StripeTransferID, "card_declined")
	require.NoError(t, err)

	after, _ := f.repo.FindCommissionByID(context.Background(), row.ID)
	require.NotNil(t, after)
	assert.Equal(t, referral.CommissionFailed, after.Status)
	assert.Equal(t, "card_declined", after.FailureReason)
}

// TestOnTransferFailed_Idempotent verifies that a second transfer.failed
// for the same transfer id is a silent no-op when the row is already
// in failed state.
func TestOnTransferFailed_Idempotent(t *testing.T) {
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

	require.NoError(t, f.svc.OnTransferFailed(context.Background(), row.StripeTransferID, "first"))
	// Second event — must be a no-op, must NOT overwrite the failure_reason.
	require.NoError(t, f.svc.OnTransferFailed(context.Background(), row.StripeTransferID, "second"))

	after, _ := f.repo.FindCommissionByID(context.Background(), row.ID)
	assert.Equal(t, referral.CommissionFailed, after.Status)
	assert.Equal(t, "first", after.FailureReason, "first failure message kept, second event ignored")
}
