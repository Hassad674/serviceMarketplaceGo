package referral

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// ClawbackIfApplicable implements service.ReferralClawback.
//
// Called by the payment service when a milestone is fully or partially
// refunded — both via the normal refund path and via dispute resolution.
//
// Behaviour by commission state:
//
//   - no commission        : no-op
//   - pending or pending_kyc: cancel without touching Stripe
//   - paid                 : compute proportional clawback amount, issue a
//                            Stripe TransferReversal, mark clawed_back
//   - already clawed_back  : no-op (idempotent)
//   - failed/cancelled     : no-op (already terminal money-wise)
func (s *Service) ClawbackIfApplicable(ctx context.Context, in service.ReferralClawbackInput) error {
	commission, err := s.referrals.FindCommissionByMilestone(ctx, in.MilestoneID)
	if errors.Is(err, referral.ErrCommissionNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find commission for clawback: %w", err)
	}

	switch commission.Status {
	case referral.CommissionPending, referral.CommissionPendingKYC:
		if err := commission.MarkCancelled(); err != nil {
			return err
		}
		if err := s.referrals.UpdateCommission(ctx, commission); err != nil {
			return fmt.Errorf("cancel commission: %w", err)
		}
		return nil

	case referral.CommissionPaid:
		// proceed below

	case referral.CommissionClawedBack, referral.CommissionFailed, referral.CommissionCancelled:
		return nil
	}

	if commission.StripeTransferID == "" {
		// Defensive — should never happen for paid status, but guard anyway.
		slog.Warn("referral: paid commission missing stripe transfer id",
			"commission_id", commission.ID)
		return nil
	}

	clawbackAmount := referral.ClawbackAmountCents(
		commission.CommissionCents, in.GrossCents, in.RefundedCents,
	)
	if clawbackAmount <= 0 {
		return nil
	}

	reversalID := ""
	if s.reversals != nil {
		reversalID, err = s.reversals.CreateTransferReversal(ctx, service.CreateTransferReversalInput{
			TransferID:     commission.StripeTransferID,
			Amount:         clawbackAmount,
			IdempotencyKey: fmt.Sprintf("referral_clawback_%s", commission.ID),
		})
		if err != nil {
			return fmt.Errorf("stripe transfer reversal: %w", err)
		}
	}

	if err := commission.ApplyClawback(reversalID); err != nil {
		return err
	}
	if err := s.referrals.UpdateCommission(ctx, commission); err != nil {
		return fmt.Errorf("update commission to clawed_back: %w", err)
	}

	// Notify the referrer of the clawback so they understand their balance
	// dropped. Lookup goes: commission → attribution (by id) → referral → referrer.
	if att, aerr := s.referrals.FindAttributionByID(ctx, commission.AttributionID); aerr == nil && att != nil {
		if parent, perr := s.referrals.GetByID(ctx, att.ReferralID); perr == nil {
			s.notify(ctx, parent.ReferrerID, notification.TypeReferralCommissionClawedBack,
				"Commission reprise",
				fmt.Sprintf("⚠️ %.2f € de commission ont été repris suite à un remboursement.", float64(clawbackAmount)/100),
				map[string]any{
					"referral_id":    parent.ID.String(),
					"commission_id":  commission.ID.String(),
					"clawback_cents": clawbackAmount,
					"reversal_id":    reversalID,
				})
		}
	}
	return nil
}
