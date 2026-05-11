package service

import (
	"context"

	"github.com/google/uuid"
)

// ReferralCommissionRetryResult is the outcome of a retry attempt on a
// commission row stuck in pending_kyc or failed.
type ReferralCommissionRetryResult string

const (
	// ReferralCommissionRetryPaid — Stripe transfer succeeded on retry,
	// row is now in paid status.
	ReferralCommissionRetryPaid ReferralCommissionRetryResult = "paid"

	// ReferralCommissionRetryKYCRequired — the apporteur's Stripe
	// Connect account is still not ready (no account id OR
	// payouts_enabled=false). The handler returns 422 with this code
	// + the onboarding URL so the UI can route the user to /payment-info.
	ReferralCommissionRetryKYCRequired ReferralCommissionRetryResult = "kyc_required"

	// ReferralCommissionRetryFailed — Stripe returned an error on the
	// transfer call. The row's failure_reason is updated. The handler
	// returns 502 so the client can retry later.
	ReferralCommissionRetryFailed ReferralCommissionRetryResult = "failed"

	// ReferralCommissionRetryAlreadyPaid — the row was already in paid
	// status when the caller invoked retry (someone else retried
	// concurrently, or the kyc_listener drained it). The handler
	// returns 409.
	ReferralCommissionRetryAlreadyPaid ReferralCommissionRetryResult = "already_paid"

	// ReferralCommissionRetryNotRetriable — the row is in a state the
	// retry endpoint cannot drive (cancelled, clawed_back, …). The
	// handler returns 409.
	ReferralCommissionRetryNotRetriable ReferralCommissionRetryResult = "not_retriable"
)

// ReferralCommissionRetryOutcome bundles the retry result with the
// supplemental fields the handler needs to surface (the apporteur's
// onboarding URL when KYC is required, the failure reason on Stripe
// failure, etc.).
type ReferralCommissionRetryOutcome struct {
	Result        ReferralCommissionRetryResult
	StripeAccount string // current acct_* (when known) — useful for support
	FailureReason string // populated when Result == ReferralCommissionRetryFailed
}

// ReferralCommissionRetryService is implemented by the referral app
// service and called by the wallet handler when an apporteur clicks
// "Retirer" on a pending_kyc or failed commission row.
//
// Contract:
//   - The implementation MUST verify the requesting user OWNS the
//     commission (via the apporteur on the parent referral). The
//     handler also checks ownership but the service layer is the
//     authoritative gate.
//   - The implementation MUST be idempotent: a second retry call on a
//     commission that is already paid returns
//     ReferralCommissionRetryAlreadyPaid without any state change.
//   - The implementation re-runs the Connect-ready gate before calling
//     Stripe so we never burn an idempotency key on a doomed transfer.
type ReferralCommissionRetryService interface {
	RetryCommission(ctx context.Context, requestingUserID, commissionID uuid.UUID) (ReferralCommissionRetryOutcome, error)
}
