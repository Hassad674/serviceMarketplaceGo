package service

import "context"

// ReferralTransferFailureListener is invoked by the Stripe webhook
// handler when a `transfer.failed` event lands on the platform. The
// referral feature uses the projected transfer id to look up the
// matching commission row (referral_commissions.stripe_transfer_id)
// and mark it as failed with the Stripe failure_message stored in
// failure_reason.
//
// Contract:
//   - The implementation MUST be idempotent on (transfer_id). Stripe
//     can retry the same event; a second call must be a silent no-op
//     once the row is already in `failed`.
//   - When no commission row matches the transfer id, the call is a
//     silent no-op (the transfer might belong to a different feature,
//     e.g. provider milestone payouts).
//   - Errors are non-blocking for the webhook handler: the handler
//     logs and returns 200 so Stripe does not flood retries.
type ReferralTransferFailureListener interface {
	OnTransferFailed(ctx context.Context, transferID, failureMessage string) error
}
