package referral

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// PrepareCommissionForMilestone implements
// service.ReferralCommissionPreparer.
//
// Called by the proposal service when a milestone is APPROVED — independent
// from the provider auto-transfer eligibility gate that historically guarded
// commission creation. The row lands in pending so the referrer wallet shows
// the income immediately; the scheduler (or the legacy DistributeIfApplicable
// path when the provider transfer eventually fires) is responsible for the
// Stripe transfer itself.
//
// Idempotency strategy mirrors DistributeIfApplicable:
//
//  1. Look up the attribution row for the proposal_id. No attribution = no-op.
//  2. INSERT the commission row in pending state. The DB unique index on
//     (attribution_id, milestone_id) raises ErrCommissionAlreadyExists on a
//     retry, in which case we return without touching anything else.
//  3. Below-the-dust commissions (CommissionCents == 0) are marked cancelled
//     immediately so the operator can still audit the attempt.
func (s *Service) PrepareCommissionForMilestone(ctx context.Context, in service.ReferralCommissionPrepareInput) error {
	if in.GrossAmountCents <= 0 {
		return nil
	}

	att, err := s.referrals.FindAttributionByProposal(ctx, in.ProposalID)
	if errors.Is(err, referral.ErrAttributionNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find attribution: %w", err)
	}

	commission, err := referral.NewCommission(referral.NewCommissionInput{
		AttributionID:    att.ID,
		MilestoneID:      in.MilestoneID,
		GrossAmountCents: in.GrossAmountCents,
		RatePct:          att.RatePctSnapshot,
		Currency:         in.Currency,
	})
	if err != nil {
		return fmt.Errorf("build commission: %w", err)
	}

	if err := s.referrals.CreateCommission(ctx, commission); err != nil {
		if errors.Is(err, referral.ErrCommissionAlreadyExists) {
			// Already prepared / distributed — silent no-op.
			return nil
		}
		return fmt.Errorf("persist commission: %w", err)
	}

	// Dust threshold — keep the row for audit but mark it cancelled so
	// no one ever tries to transfer it.
	if commission.CommissionCents == 0 {
		if err := commission.MarkCancelled(); err != nil {
			slog.Warn("commission preparer: MarkCancelled state transition failed",
				"error", err, "commission_id", commission.ID)
		}
		if err := s.referrals.UpdateCommission(ctx, commission); err != nil {
			slog.Warn("commission preparer: persist cancelled commission failed",
				"error", err, "commission_id", commission.ID)
		}
		return nil
	}

	// Row stays in pending. The scheduler (DrainPendingCommissions) and the
	// legacy DistributeIfApplicable path are both authorized to drain it to
	// Stripe — whichever fires first wins, the other becomes a no-op via the
	// status check.
	slog.Info("referral: commission prepared (pending)",
		"commission_id", commission.ID,
		"attribution_id", att.ID,
		"proposal_id", in.ProposalID,
		"milestone_id", in.MilestoneID,
		"commission_cents", commission.CommissionCents)
	return nil
}

// DistributeIfApplicable implements service.ReferralCommissionDistributor.
//
// Called by the payment service after a milestone has been transferred to the
// provider. Computes the apporteur's slice and either sends a Stripe transfer
// (referrer has KYC) or parks the row as pending_kyc to be drained later.
//
// Idempotency strategy:
//
//  1. Look up the attribution row for the proposal_id. No attribution = no-op.
//  2. INSERT the commission row in pending state. The DB unique index on
//     (attribution_id, milestone_id) raises ErrCommissionAlreadyExists on a
//     retry, in which case we return ReferralCommissionSkipped without
//     touching Stripe.
//  3. Resolve the referrer's stripe account. Empty → mark pending_kyc, return.
//  4. Call Stripe.CreateTransfer with an idempotency key derived from the
//     commission id, so even if Stripe sees the same call twice it returns
//     the same transfer.
//  5. Persist the result (paid or failed) and notify the referrer.
func (s *Service) DistributeIfApplicable(ctx context.Context, in service.ReferralCommissionDistributorInput) (service.ReferralCommissionResult, error) {
	if in.GrossAmountCents <= 0 {
		return service.ReferralCommissionSkipped, nil
	}

	att, err := s.referrals.FindAttributionByProposal(ctx, in.ProposalID)
	if errors.Is(err, referral.ErrAttributionNotFound) {
		return service.ReferralCommissionSkipped, nil
	}
	if err != nil {
		return service.ReferralCommissionFailed, fmt.Errorf("find attribution: %w", err)
	}

	commission, err := referral.NewCommission(referral.NewCommissionInput{
		AttributionID:    att.ID,
		MilestoneID:      in.MilestoneID,
		GrossAmountCents: in.GrossAmountCents,
		RatePct:          att.RatePctSnapshot,
		Currency:         in.Currency,
	})
	if err != nil {
		return service.ReferralCommissionFailed, fmt.Errorf("build commission: %w", err)
	}

	if err := s.referrals.CreateCommission(ctx, commission); err != nil {
		if errors.Is(err, referral.ErrCommissionAlreadyExists) {
			// A commission row already exists — either because
			// PrepareCommissionForMilestone ran on milestone APPROVAL (the
			// expected post-fix flow) or because this distributor itself was
			// retried. Reload the existing row and continue with the Stripe
			// transfer if (and only if) it is still in pending state — paid /
			// pending_kyc / failed / cancelled / clawed_back rows are
			// terminal-or-owned-elsewhere and must NOT be re-driven from
			// here.
			existing, ferr := s.referrals.FindCommissionByMilestone(ctx, in.MilestoneID)
			if ferr != nil {
				slog.Warn("commission distributor: load existing pending commission failed",
					"error", ferr, "milestone_id", in.MilestoneID)
				return service.ReferralCommissionSkipped, nil
			}
			if existing == nil || existing.Status != referral.CommissionPending {
				return service.ReferralCommissionSkipped, nil
			}
			commission = existing
		} else {
			return service.ReferralCommissionFailed, fmt.Errorf("persist commission: %w", err)
		}
	}

	// Below the dust threshold — skip the Stripe call but keep the row in
	// pending so the operator can audit. We could mark it cancelled but
	// keeping it around is cheaper than reasoning about it later.
	if commission.CommissionCents == 0 {
		if err := commission.MarkCancelled(); err != nil {
			slog.Warn("commission distributor: MarkCancelled state transition failed",
				"error", err, "commission_id", commission.ID)
		}
		if err := s.referrals.UpdateCommission(ctx, commission); err != nil {
			slog.Warn("commission distributor: persist cancelled commission failed",
				"error", err, "commission_id", commission.ID)
		}
		return service.ReferralCommissionSkipped, nil
	}

	parent, err := s.referrals.GetByID(ctx, att.ReferralID)
	if err != nil {
		return service.ReferralCommissionFailed, fmt.Errorf("load parent referral: %w", err)
	}

	stripeAccount := ""
	if s.stripeAccounts != nil {
		stripeAccount, err = s.stripeAccounts.ResolveStripeAccountID(ctx, parent.ReferrerID)
		if err != nil {
			slog.Warn("referral: resolve stripe account failed",
				"referrer_id", parent.ReferrerID, "error", err)
		}
	}

	// Connect-ready gate: the apporteur must have a connected account
	// AND that account must have payouts enabled before we burn a Stripe
	// idempotency key on a doomed transfer. When the gate trips, the
	// commission stays in pending_kyc so the apporteur can come back and
	// retire it after completing onboarding (D1+D2).
	if !s.connectReadyForReferrer(ctx, stripeAccount) {
		if err := commission.MarkPendingKYC(); err != nil {
			slog.Warn("commission distributor: MarkPendingKYC state transition failed",
				"error", err, "commission_id", commission.ID)
		}
		if err := s.referrals.UpdateCommission(ctx, commission); err != nil {
			return service.ReferralCommissionFailed, fmt.Errorf("update commission to pending_kyc: %w", err)
		}
		slog.Info("referral: commission parked pending_kyc",
			"commission_id", commission.ID,
			"referrer_id", parent.ReferrerID,
			"has_account", stripeAccount != "")
		s.notifyCommissionPendingKYC(ctx, parent.ID, parent.ReferrerID, commission.CommissionCents)
		return service.ReferralCommissionPendingKYC, nil
	}

	transferID, err := s.stripe.CreateTransfer(ctx, service.CreateTransferInput{
		Amount:             commission.CommissionCents,
		Currency:           commission.Currency,
		DestinationAccount: stripeAccount,
		TransferGroup:      fmt.Sprintf("referral_%s", parent.ID),
		IdempotencyKey:     fmt.Sprintf("referral_commission_%s", commission.ID),
	})
	if err != nil {
		if mErr := commission.MarkFailed(err.Error()); mErr != nil {
			slog.Warn("commission distributor: MarkFailed state transition failed",
				"error", mErr, "commission_id", commission.ID)
		}
		if uErr := s.referrals.UpdateCommission(ctx, commission); uErr != nil {
			slog.Warn("commission distributor: persist failed-status commission failed",
				"error", uErr, "commission_id", commission.ID)
		}
		slog.Error("referral: stripe transfer failed",
			"commission_id", commission.ID, "error", err)
		return service.ReferralCommissionFailed, err
	}

	if err := commission.MarkPaid(transferID); err != nil {
		slog.Warn("commission distributor: MarkPaid state transition failed",
			"error", err, "commission_id", commission.ID)
	}
	if err := s.referrals.UpdateCommission(ctx, commission); err != nil {
		return service.ReferralCommissionFailed, fmt.Errorf("update commission to paid: %w", err)
	}
	s.notifyCommissionPaid(ctx, parent.ID, parent.ReferrerID, commission.CommissionCents, transferID)
	return service.ReferralCommissionPaid, nil
}
