package referral

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// GetReferrerSummary implements service.ReferralWalletReader. It
// aggregates commissions across every referral of the given apporteur
// into the 4 wallet-relevant statuses. Unknown referrers (no rows)
// return a zero summary with no error so the wallet degrades gracefully.
func (s *Service) GetReferrerSummary(ctx context.Context, referrerID uuid.UUID) (service.ReferrerCommissionSummary, error) {
	if s.referrals == nil {
		return service.ReferrerCommissionSummary{Currency: "EUR"}, nil
	}
	byStatus, err := s.referrals.SumCommissionsByReferrer(ctx, referrerID)
	if err != nil {
		return service.ReferrerCommissionSummary{}, fmt.Errorf("sum commissions by referrer: %w", err)
	}
	return service.ReferrerCommissionSummary{
		PendingCents:    byStatus[referral.CommissionPending],
		PendingKYCCents: byStatus[referral.CommissionPendingKYC],
		PaidCents:       byStatus[referral.CommissionPaid],
		ClawedBackCents: byStatus[referral.CommissionClawedBack],
		Currency:        "EUR",
	}, nil
}

// RecentCommissions implements service.ReferralWalletReader. Returns
// the N most recent commission rows for the apporteur, enriched with
// the parent referral id + proposal id so the UI can deep-link into
// /referrals/{id} or /projects/{id}.
func (s *Service) RecentCommissions(ctx context.Context, referrerID uuid.UUID, limit int) ([]service.ReferralCommissionRecord, error) {
	if s.referrals == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.referrals.ListRecentCommissionsByReferrer(ctx, referrerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent commissions: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	// Resolve attribution → referral/proposal in one pass so the UI
	// can navigate from a commission row straight to its context
	// without a second round-trip.
	attIDs := make(map[uuid.UUID]struct{}, len(rows))
	for _, c := range rows {
		attIDs[c.AttributionID] = struct{}{}
	}
	atts := make(map[uuid.UUID]*referral.Attribution, len(attIDs))
	for id := range attIDs {
		if a, err := s.referrals.FindAttributionByID(ctx, id); err == nil && a != nil {
			atts[id] = a
		}
	}

	out := make([]service.ReferralCommissionRecord, 0, len(rows))
	for _, c := range rows {
		rec := service.ReferralCommissionRecord{
			ID:               c.ID,
			MilestoneID:      c.MilestoneID,
			GrossAmountCents: c.GrossAmountCents,
			CommissionCents:  c.CommissionCents,
			Currency:         c.Currency,
			Status:           string(c.Status),
			StripeTransferID: c.StripeTransferID,
			StripeReversalID: c.StripeReversalID,
			PaidAt:           c.PaidAt,
			ClawedBackAt:     c.ClawedBackAt,
			CreatedAt:        c.CreatedAt,
		}
		if a, ok := atts[c.AttributionID]; ok {
			rec.ReferralID = a.ReferralID
			rec.ProposalID = a.ProposalID
		}
		out = append(out, rec)
	}
	return out, nil
}
