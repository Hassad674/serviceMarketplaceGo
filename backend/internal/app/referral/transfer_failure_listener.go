package referral

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"marketplace-backend/internal/domain/referral"
)

// OnTransferFailed implements service.ReferralTransferFailureListener.
//
// Called by the Stripe webhook handler (D1+D2) when a `transfer.failed`
// event lands on a commission row. The listener:
//
//  1. Looks up the commission row by stripe_transfer_id.
//     ErrCommissionNotFound = silent no-op (the transfer belongs to a
//     different platform flow, not a referral commission).
//  2. Idempotent: a row already in `failed` is left untouched.
//  3. Otherwise records the Stripe failure message and flips the row
//     to `failed`, then fans the "commission failed" event out to
//     every member of the referrer org so they can act on it.
//
// Errors are returned to the webhook handler which logs them; the
// handler always responds 200 to Stripe to avoid retry storms — the
// audit trail in slog is the source of truth for forensic review.
func (s *Service) OnTransferFailed(ctx context.Context, transferID, failureMessage string) error {
	if transferID == "" {
		return nil
	}
	commission, err := s.referrals.FindCommissionByStripeTransferID(ctx, transferID)
	if err != nil {
		if errors.Is(err, referral.ErrCommissionNotFound) {
			// Not our transfer — silent no-op.
			return nil
		}
		return fmt.Errorf("load commission by transfer id: %w", err)
	}
	if commission.Status == referral.CommissionFailed {
		// Idempotent — Stripe retried the same event.
		return nil
	}
	// Force-set failed status. MarkFailed only allows pending /
	// pending_kyc transitions per the domain rules but on
	// transfer.failed we must mark even a paid row as failed (Stripe
	// already reversed the money flow at that point).
	prevStatus := commission.Status
	commission.Status = referral.CommissionFailed
	commission.FailureReason = failureMessage
	if uerr := s.referrals.UpdateCommission(ctx, commission); uerr != nil {
		return fmt.Errorf("persist failed-status commission: %w", uerr)
	}
	slog.Info("referral: transfer.failed applied to commission",
		"commission_id", commission.ID,
		"transfer_id", transferID,
		"prev_status", prevStatus,
		"failure_message", failureMessage)
	return nil
}
