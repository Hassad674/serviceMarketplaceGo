package referral_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/referral"
	portservice "marketplace-backend/internal/port/service"
	referralapp "marketplace-backend/internal/app/referral"
)

// seedCommissionStuckInPendingKYC creates a fully-active referral +
// attribution, distributes a commission while the apporteur has no
// Stripe account, and returns the parked commission row id. Convenience
// helper so every retry test starts from the same "stuck pending_kyc"
// baseline.
func seedCommissionStuckInPendingKYC(t *testing.T) (*testFixture, uuid.UUID, uuid.UUID) {
	t.Helper()
	f := newTestFixture(t, "") // no account → pending_kyc
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	res, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	require.Equal(t, "pending_kyc", string(res))

	commission, err := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NoError(t, err)
	return f, commission.ID, refID
}

// TestRetryCommission_NotFound verifies that an unknown commission id
// returns referral.ErrCommissionNotFound — the handler maps this to
// 404.
func TestRetryCommission_NotFound(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	_, err := f.svc.RetryCommission(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, referral.ErrCommissionNotFound)
}

// TestRetryCommission_NotOwner verifies that a user who is NOT the
// apporteur on the parent referral receives ErrCommissionNotOwned —
// the handler maps this to 403.
func TestRetryCommission_NotOwner(t *testing.T) {
	f, commissionID, _ := seedCommissionStuckInPendingKYC(t)
	stranger := uuid.New()
	_, err := f.svc.RetryCommission(context.Background(), stranger, commissionID)
	require.ErrorIs(t, err, referralapp.ErrCommissionNotOwned)
}

// TestRetryCommission_KYCStillMissing verifies that retrying a
// pending_kyc row when the apporteur still has no Stripe account
// returns kyc_required and does NOT fire a Stripe transfer.
func TestRetryCommission_KYCStillMissing(t *testing.T) {
	f, commissionID, referrerID := seedCommissionStuckInPendingKYC(t)

	// Account resolver still returns empty string → gate trips.
	outcome, err := f.svc.RetryCommission(context.Background(), referrerID, commissionID)
	require.NoError(t, err)
	assert.Equal(t, portservice.ReferralCommissionRetryKYCRequired, outcome.Result)
	assert.Empty(t, f.stripe.transfers)
}

// TestRetryCommission_KYCReadyAfterOnboarding verifies the happy path:
// the apporteur completes onboarding (account id + payouts_enabled),
// clicks Retirer, and the commission flips to paid with a Stripe
// transfer fired.
func TestRetryCommission_KYCReadyAfterOnboarding(t *testing.T) {
	f, commissionID, referrerID := seedCommissionStuckInPendingKYC(t)

	// Simulate apporteur completing KYC.
	f.accounts.accountID = "acct_apporteur"
	f.stripe.account = &portservice.StripeAccountInfo{
		ChargesEnabled: true,
		PayoutsEnabled: true,
	}

	outcome, err := f.svc.RetryCommission(context.Background(), referrerID, commissionID)
	require.NoError(t, err)
	assert.Equal(t, portservice.ReferralCommissionRetryPaid, outcome.Result)
	require.Len(t, f.stripe.transfers, 1, "exactly one transfer fired on retry")

	// Commission row is now paid.
	row, ferr := f.repo.FindCommissionByID(context.Background(), commissionID)
	require.NoError(t, ferr)
	assert.Equal(t, referral.CommissionPaid, row.Status)
	assert.NotEmpty(t, row.StripeTransferID)
	require.NotNil(t, row.PaidAt)
}

// TestRetryCommission_AlreadyPaid verifies that retrying a row already
// in paid status is a no-op that returns already_paid. The handler
// surfaces this as 409.
func TestRetryCommission_AlreadyPaid(t *testing.T) {
	f, commissionID, referrerID := seedCommissionStuckInPendingKYC(t)
	// First retry — succeeds and flips to paid.
	f.accounts.accountID = "acct_apporteur"
	f.stripe.account = &portservice.StripeAccountInfo{ChargesEnabled: true, PayoutsEnabled: true}
	_, err := f.svc.RetryCommission(context.Background(), referrerID, commissionID)
	require.NoError(t, err)

	// Second retry on a paid row — no Stripe call, returns already_paid.
	outcome, err := f.svc.RetryCommission(context.Background(), referrerID, commissionID)
	require.NoError(t, err)
	assert.Equal(t, portservice.ReferralCommissionRetryAlreadyPaid, outcome.Result)
	assert.Len(t, f.stripe.transfers, 1, "no duplicate transfer fired on retry of already-paid row")
}

// TestRetryCommission_FailedRowReadyNow verifies a row in `failed`
// state is upgraded to paid when the retry succeeds. Covers the
// production case where the first transfer failed due to a Stripe
// outage and the user clicks Retirer once Stripe is healthy.
func TestRetryCommission_FailedRowReadyNow(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	// Force the first distributor call to fail at the Stripe layer →
	// row lands in `failed`.
	f.stripe.failNext = true
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.Error(t, err) // distributor surfaces Stripe error
	row, _ := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NotNil(t, row)
	require.Equal(t, referral.CommissionFailed, row.Status)

	// Now Stripe is healthy — retry succeeds.
	outcome, err := f.svc.RetryCommission(context.Background(), refID, row.ID)
	require.NoError(t, err)
	assert.Equal(t, portservice.ReferralCommissionRetryPaid, outcome.Result)

	after, _ := f.repo.FindCommissionByID(context.Background(), row.ID)
	assert.Equal(t, referral.CommissionPaid, after.Status)
	assert.Empty(t, after.FailureReason, "failure_reason must be cleared on successful retry")
}

// TestRetryCommission_FailedRowStripeStillBroken verifies a `failed`
// retry that hits another Stripe error stays failed with a refreshed
// failure_reason. Handler surfaces this as 502.
func TestRetryCommission_FailedRowStripeStillBroken(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	f.stripe.failNext = true
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.Error(t, err)
	row, _ := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NotNil(t, row)
	require.Equal(t, referral.CommissionFailed, row.Status)

	// Retry hits another Stripe error.
	f.stripe.failNext = true
	outcome, err := f.svc.RetryCommission(context.Background(), refID, row.ID)
	require.NoError(t, err)
	assert.Equal(t, portservice.ReferralCommissionRetryFailed, outcome.Result)
	assert.NotEmpty(t, outcome.FailureReason)

	after, _ := f.repo.FindCommissionByID(context.Background(), row.ID)
	assert.Equal(t, referral.CommissionFailed, after.Status)
	assert.NotEmpty(t, after.FailureReason)
}

// TestRetryCommission_Cancelled verifies a cancelled row cannot be
// retried — outcome is not_retriable (handler returns 409).
func TestRetryCommission_Cancelled(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()

	// Distribute with 0 gross → row lands as cancelled (dust threshold path).
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1))
	require.NoError(t, err)
	row, _ := f.repo.FindCommissionByMilestone(context.Background(), milestoneID)
	require.NotNil(t, row)
	require.Equal(t, referral.CommissionCancelled, row.Status)

	outcome, err := f.svc.RetryCommission(context.Background(), refID, row.ID)
	require.NoError(t, err)
	assert.Equal(t, portservice.ReferralCommissionRetryNotRetriable, outcome.Result)
}

// TestRetryCommission_AuditLogged verifies that every retry attempt
// emits exactly one audit row with the correct action + metadata —
// regardless of whether the attempt succeeded, was gated, or failed.
func TestRetryCommission_AuditLogged(t *testing.T) {
	f, commissionID, referrerID := seedCommissionStuckInPendingKYC(t)
	// kyc still missing → outcome should be kyc_required + audit emitted.
	outcome, err := f.svc.RetryCommission(context.Background(), referrerID, commissionID)
	require.NoError(t, err)
	require.Equal(t, portservice.ReferralCommissionRetryKYCRequired, outcome.Result)

	entries := f.audits.entriesOfAction(audit.ActionCommissionRetryAttempted)
	require.Len(t, entries, 1, "exactly one audit row for the retry attempt")
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, referrerID, *entries[0].UserID)
	assert.Equal(t, audit.ResourceTypeReferralCommission, entries[0].ResourceType)
	require.NotNil(t, entries[0].ResourceID)
	assert.Equal(t, commissionID, *entries[0].ResourceID)
	assert.Equal(t, "pending_kyc", entries[0].Metadata["prev_status"])
	assert.Equal(t, "kyc_required", entries[0].Metadata["retry_result"])
}
