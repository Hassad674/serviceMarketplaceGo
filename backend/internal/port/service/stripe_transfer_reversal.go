package service

import "context"

// StripeTransferReversalService reverses a previously executed Stripe transfer
// (full or partial). Used by the referral clawback flow when a milestone is
// refunded after the apporteur commission has already been transferred to the
// referrer's connected account.
//
// Defined as a single-method port (interface segregation) so consumers don't
// pull the wider StripeService surface just for the reversal call.
type StripeTransferReversalService interface {
	// CreateTransferReversal reverses `amount` cents from the given transfer.
	// Pass the original transfer's full amount for a full reversal, or a
	// smaller value for a partial one. The returned id is Stripe's reversal
	// id (trr_...) which the caller persists for audit and idempotency.
	CreateTransferReversal(ctx context.Context, input CreateTransferReversalInput) (reversalID string, err error)
}

// CreateTransferReversalInput is the payload for CreateTransferReversal.
type CreateTransferReversalInput struct {
	TransferID     string // Stripe transfer id (tr_…) to reverse
	Amount         int64  // cents to reverse — must be ≤ original transfer amount
	IdempotencyKey string // upstream-supplied key to dedupe retries
}
