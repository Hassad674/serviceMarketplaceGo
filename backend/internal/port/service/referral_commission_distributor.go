package service

import (
	"context"

	"github.com/google/uuid"
)

// ReferralCommissionDistributorInput is the minimal payload the payment service
// hands to the referral feature when a milestone has just been transferred to
// the provider. The referral service uses MilestoneID to find the matching
// attribution (via the proposal's existing attribution row).
type ReferralCommissionDistributorInput struct {
	ProposalID            uuid.UUID
	MilestoneID           uuid.UUID
	GrossAmountCents      int64
	Currency              string
	IdempotencyKeySuffix  string // optional, forwarded to Stripe to dedupe retries
}

// ReferralCommissionResult tells the caller what happened. The payment service
// uses this for logging and never branches on it.
type ReferralCommissionResult string

const (
	// ReferralCommissionSkipped — no attribution exists for this proposal,
	// or commission was already created (idempotent retry). No-op.
	ReferralCommissionSkipped ReferralCommissionResult = "skipped"

	// ReferralCommissionPaid — Stripe transfer to the referrer succeeded.
	ReferralCommissionPaid ReferralCommissionResult = "paid"

	// ReferralCommissionPendingKYC — referrer has no Stripe Connect account
	// yet; commission row was inserted in pending_kyc state and will be
	// drained by OnStripeAccountReady once KYC completes.
	ReferralCommissionPendingKYC ReferralCommissionResult = "pending_kyc"

	// ReferralCommissionFailed — Stripe call returned an error; the row is
	// in failed state with a failure_reason for operator review.
	ReferralCommissionFailed ReferralCommissionResult = "failed"
)

// ReferralCommissionDistributor is implemented by the referral app service and
// called by the payment app service AFTER a successful provider transfer on a
// milestone payout.
//
// Contract:
//   - Implementation MUST be idempotent on milestone_id (DB unique index +
//     Stripe idempotency key).
//   - Implementation MUST NOT block the payment flow: the caller logs the
//     result and continues on error.
//   - The gross amount is the milestone's gross (the same number from which
//     the platform fee was already deducted before paying the provider). The
//     commission is computed as gross × rate_pct (the referrer takes a slice
//     of the gross, not of the post-platform amount).
//   - Failures must be observable: implementations should at minimum log via
//     slog and persist failure_reason in the commission row.
type ReferralCommissionDistributor interface {
	DistributeIfApplicable(ctx context.Context, input ReferralCommissionDistributorInput) (ReferralCommissionResult, error)
}
