package service

import (
	"context"

	"github.com/google/uuid"
)

// ReferralKYCListener is implemented by the referral app service and called by
// the embedded (Stripe Connect KYC) feature when a user's connected account
// transitions to a payable state — that is, the account exists, the KYC is
// verified, and transfers can be sent to it.
//
// The referral feature uses this hook to drain commissions that were parked
// in CommissionPendingKYC at distribution time (because the referrer hadn't
// completed KYC yet). Each parked commission is retried as a fresh Stripe
// transfer.
//
// Contract:
//   - Implementation MUST be idempotent: calling it twice with the same userID
//     must not double-pay any commission.
//   - The implementation MUST NOT block the embedded flow: errors are logged
//     by the caller and the embedded request continues.
//   - Implementation SHOULD process pending_kyc commissions in deterministic
//     order (oldest first) so the apporteur sees their payouts arrive in a
//     predictable sequence.
type ReferralKYCListener interface {
	OnStripeAccountReady(ctx context.Context, userID uuid.UUID) error
}
