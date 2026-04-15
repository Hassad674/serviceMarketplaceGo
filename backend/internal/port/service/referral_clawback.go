package service

import (
	"context"

	"github.com/google/uuid"
)

// ReferralClawbackInput carries the milestone identity plus the amount that
// was just refunded to the client. The referral service derives the
// proportional commission amount to reverse and issues a Stripe transfer
// reversal.
type ReferralClawbackInput struct {
	MilestoneID     uuid.UUID
	RefundedCents   int64
	GrossCents      int64 // the original gross used to compute the commission
}

// ReferralClawback is implemented by the referral app service and called by
// the payment app service whenever a milestone is fully or partially refunded
// — both via the standard refund path and via dispute resolution when the
// dispute splits the funds in the client's favour.
//
// Contract:
//   - No-op if no commission exists for the milestone.
//   - No-op if the commission is in a non-paid state (pending or pending_kyc
//     are simply cancelled, no Stripe call is made).
//   - For paid commissions: the implementation issues a Stripe TransferReversal
//     for the proportional amount (truncated down) and updates the commission
//     to clawed_back.
//   - Idempotent: calling twice for the same milestone with the same refund
//     amount must not double-reverse. Implementation MAY rely on the commission
//     status check (clawed_back commissions are skipped).
//   - Errors MUST be returned but the caller treats them as non-blocking and
//     surfaces them via logging.
type ReferralClawback interface {
	ClawbackIfApplicable(ctx context.Context, input ReferralClawbackInput) error
}
