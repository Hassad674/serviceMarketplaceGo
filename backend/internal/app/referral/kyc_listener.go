package referral

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// OnStripeAccountReady implements service.ReferralKYCListener.
//
// Called by the embedded feature when a user's Stripe Connect account becomes
// payable. Walks the commissions parked in pending_kyc for that referrer and
// retries each one as a fresh Stripe transfer. Failures are logged per-row
// without aborting the batch — one bad row doesn't block the others.
func (s *Service) OnStripeAccountReady(ctx context.Context, userID uuid.UUID) error {
	pending, err := s.referrals.ListPendingKYCByReferrer(ctx, userID)
	if err != nil {
		return fmt.Errorf("list pending kyc commissions: %w", err)
	}
	if len(pending) == 0 {
		return nil
	}

	slog.Info("referral: draining pending_kyc commissions",
		"referrer_id", userID, "count", len(pending))

	accountID := ""
	if s.stripeAccounts != nil {
		accountID, err = s.stripeAccounts.ResolveStripeAccountID(ctx, userID)
		if err != nil || accountID == "" {
			// Account still not resolvable — bail. The hook will fire again
			// when the state really stabilises.
			return nil
		}
	}

	for _, c := range pending {
		s.drainCommission(ctx, c, userID, accountID)
	}
	return nil
}

// drainCommission retries one parked commission as a Stripe transfer and
// updates the row in-place. Non-fatal to the outer batch: a single failure
// marks the row 'failed' and continues.
func (s *Service) drainCommission(ctx context.Context, c *referral.Commission, userID uuid.UUID, accountID string) {
	att, err := s.referrals.FindAttributionByID(ctx, c.AttributionID)
	if err != nil {
		slog.Warn("referral: drain missing attribution",
			"commission_id", c.ID, "error", err)
		return
	}

	transferID, err := s.stripe.CreateTransfer(ctx, service.CreateTransferInput{
		Amount:             c.CommissionCents,
		Currency:           c.Currency,
		DestinationAccount: accountID,
		TransferGroup:      fmt.Sprintf("referral_%s", att.ReferralID),
		IdempotencyKey:     fmt.Sprintf("referral_commission_%s", c.ID),
	})
	if err != nil {
		_ = c.MarkFailed(err.Error())
		if uerr := s.referrals.UpdateCommission(ctx, c); uerr != nil {
			slog.Error("referral: drain update-failed failed",
				"commission_id", c.ID, "error", uerr)
		}
		slog.Error("referral: drain transfer failed",
			"commission_id", c.ID, "error", err)
		return
	}

	// Promote pending_kyc → paid. MarkPaid requires pending state, so we
	// bypass it with a direct status set: the commission was already
	// pending_kyc, not pending. This is the only place we skip the entity
	// guard, justified by the single transition pending_kyc → paid being
	// owned by this drain path.
	c.Status = referral.CommissionPaid
	c.StripeTransferID = transferID
	nowFromDomain := c // keep var for readability
	_ = nowFromDomain
	if err := s.referrals.UpdateCommission(ctx, c); err != nil {
		slog.Error("referral: drain update-paid failed",
			"commission_id", c.ID, "error", err)
		return
	}

	s.notifyCommissionPaid(ctx, att.ReferralID, userID, c.CommissionCents, transferID)
}
