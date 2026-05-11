package referral

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// PendingCommissionSweepGracePeriod is the minimum age a `pending`
// commission row must reach before the sweeper picks it up. We give
// the legacy DistributeIfApplicable path — which fires inside the
// provider auto-transfer flow — a small head start so we don't race
// it on the milestone the client just approved. Anything older than
// this window has measurably failed the auto-transfer eligibility
// gate and is safe for the sweeper to drain straight to Stripe.
const PendingCommissionSweepGracePeriod = 5 * time.Minute

// PendingCommissionSweepBatchSize caps the number of rows the sweeper
// processes per invocation. Bound mirrors the expirer batch size.
const PendingCommissionSweepBatchSize = 200

// SweepPendingCommissions drains `pending` commission rows that have
// been sitting longer than PendingCommissionSweepGracePeriod. Each row
// gets one Stripe transfer attempt — success flips it to paid, missing
// KYC flips it to pending_kyc (where OnStripeAccountReady takes over),
// failure flips it to failed with the Stripe reason. Failures are
// per-row and never abort the batch.
//
// Returns the number of rows processed (drained to a terminal state)
// so the cron can log a summary. CRIT-REF: this is the safety net for
// commissions that were prepared on milestone approval but whose
// provider auto-transfer never fired (new providers without consent).
func (s *Service) SweepPendingCommissions(ctx context.Context) (int, error) {
	if s.referrals == nil {
		return 0, nil
	}

	cutoff := time.Now().UTC().Add(-PendingCommissionSweepGracePeriod)
	rows, err := s.referrals.ListPendingCommissions(ctx, cutoff, PendingCommissionSweepBatchSize)
	if err != nil {
		return 0, fmt.Errorf("list pending commissions: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}

	slog.Info("referral: sweeping pending commissions",
		"count", len(rows), "grace_period", PendingCommissionSweepGracePeriod)

	processed := 0
	for _, c := range rows {
		if s.sweepPendingCommission(ctx, c) {
			processed++
		}
	}
	if processed > 0 {
		slog.Info("referral: pending commissions swept", "count", processed)
	}
	return processed, nil
}

// sweepPendingCommission attempts ONE Stripe transfer on a pending row
// and persists the resulting status (paid / pending_kyc / failed).
// Returns true when the row reached a terminal state, false on a
// transient lookup error (so the next sweep can retry without
// flapping the row through failed).
func (s *Service) sweepPendingCommission(ctx context.Context, c *referral.Commission) bool {
	if c.Status != referral.CommissionPending {
		// Defensive: another path (DistributeIfApplicable racing) may
		// have already drained the row between the list and the
		// iteration. Drop it silently.
		return false
	}

	att, err := s.referrals.FindAttributionByID(ctx, c.AttributionID)
	if err != nil || att == nil {
		slog.Warn("referral sweep: load attribution failed",
			"commission_id", c.ID, "error", err)
		return false
	}
	parent, err := s.referrals.GetByID(ctx, att.ReferralID)
	if err != nil || parent == nil {
		slog.Warn("referral sweep: load parent referral failed",
			"commission_id", c.ID, "referral_id", att.ReferralID, "error", err)
		return false
	}

	stripeAccount := ""
	if s.stripeAccounts != nil {
		stripeAccount, err = s.stripeAccounts.ResolveStripeAccountID(ctx, parent.ReferrerID)
		if err != nil {
			slog.Warn("referral sweep: resolve stripe account failed",
				"referrer_id", parent.ReferrerID, "error", err)
		}
	}

	if stripeAccount == "" {
		// Apporteur KYC not ready — park the row in pending_kyc and let
		// OnStripeAccountReady drain it later. We notify on the
		// transition so the apporteur sees the action-required cue.
		if mErr := c.MarkPendingKYC(); mErr != nil {
			slog.Warn("referral sweep: MarkPendingKYC state transition failed",
				"error", mErr, "commission_id", c.ID)
			return false
		}
		if uErr := s.referrals.UpdateCommission(ctx, c); uErr != nil {
			slog.Error("referral sweep: persist pending_kyc commission failed",
				"commission_id", c.ID, "error", uErr)
			return false
		}
		s.notifyCommissionPendingKYC(ctx, parent.ID, parent.ReferrerID, c.CommissionCents)
		return true
	}

	return s.sweepStripeTransfer(ctx, c, parent.ID, parent.ReferrerID, stripeAccount)
}

// sweepStripeTransfer fires the Stripe transfer for a sweep candidate
// and persists the outcome. Extracted to keep sweepPendingCommission
// under the function-size limit.
func (s *Service) sweepStripeTransfer(
	ctx context.Context,
	c *referral.Commission,
	referralID, referrerID uuid.UUID,
	stripeAccount string,
) bool {
	transferID, err := s.stripe.CreateTransfer(ctx, service.CreateTransferInput{
		Amount:             c.CommissionCents,
		Currency:           c.Currency,
		DestinationAccount: stripeAccount,
		TransferGroup:      fmt.Sprintf("referral_%s", referralID),
		IdempotencyKey:     fmt.Sprintf("referral_commission_%s", c.ID),
	})
	if err != nil {
		if mErr := c.MarkFailed(err.Error()); mErr != nil {
			slog.Warn("referral sweep: MarkFailed state transition failed",
				"error", mErr, "commission_id", c.ID)
		}
		if uErr := s.referrals.UpdateCommission(ctx, c); uErr != nil {
			slog.Error("referral sweep: persist failed commission failed",
				"commission_id", c.ID, "error", uErr)
		}
		slog.Error("referral sweep: stripe transfer failed",
			"commission_id", c.ID, "error", err)
		return true
	}

	if mErr := c.MarkPaid(transferID); mErr != nil {
		slog.Warn("referral sweep: MarkPaid state transition failed",
			"error", mErr, "commission_id", c.ID)
	}
	if uErr := s.referrals.UpdateCommission(ctx, c); uErr != nil {
		slog.Error("referral sweep: persist paid commission failed",
			"commission_id", c.ID, "error", uErr)
		return false
	}
	s.notifyCommissionPaid(ctx, referralID, referrerID, c.CommissionCents, transferID)
	return true
}
