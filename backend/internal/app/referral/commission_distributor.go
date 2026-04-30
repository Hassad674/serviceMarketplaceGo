package referral

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

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
			return service.ReferralCommissionSkipped, nil
		}
		return service.ReferralCommissionFailed, fmt.Errorf("persist commission: %w", err)
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

	if stripeAccount == "" {
		if err := commission.MarkPendingKYC(); err != nil {
			slog.Warn("commission distributor: MarkPendingKYC state transition failed",
				"error", err, "commission_id", commission.ID)
		}
		if err := s.referrals.UpdateCommission(ctx, commission); err != nil {
			return service.ReferralCommissionFailed, fmt.Errorf("update commission to pending_kyc: %w", err)
		}
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
