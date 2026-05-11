package referral

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// paid30dWindow is the rolling window used for the "Versées 30j" tile
// on the apporteur wallet. Kept as a package-level constant so tests
// can compare against the same boundary without re-deriving it.
const paid30dWindow = 30 * 24 * time.Hour

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
	paid := byStatus[referral.CommissionPaid]
	clawed := byStatus[referral.CommissionClawedBack]
	// Paid30d is computed by walking the recent commission rows (capped
	// at 100 — beyond that the apporteur is busy enough that the tile is
	// meaningful from the most recent slice alone). The sweep is bounded
	// and runs inside the same handler that already calls RecentCommissions
	// so the read amplification stays low. Errors fall through to zero
	// rather than failing the whole summary — the tile degrades to "0 €"
	// in that case.
	paid30d := s.computePaid30d(ctx, referrerID)
	return service.ReferrerCommissionSummary{
		PendingCents:    byStatus[referral.CommissionPending],
		PendingKYCCents: byStatus[referral.CommissionPendingKYC],
		PaidCents:       paid,
		ClawedBackCents: clawed,
		Paid30dCents:    paid30d,
		LifetimeCents:   paid + clawed,
		Currency:        "EUR",
	}, nil
}

// computePaid30d returns the sum of commission_cents on rows whose
// status is paid AND whose paid_at falls within the last 30 days.
// Walks at most 100 recent rows — beyond that the cost outweighs the
// benefit for a wallet-tile aggregate. Returns 0 on any error so the
// caller can still serve the rest of the summary.
func (s *Service) computePaid30d(ctx context.Context, referrerID uuid.UUID) int64 {
	rows, err := s.referrals.ListRecentCommissionsByReferrer(ctx, referrerID, 100)
	if err != nil || len(rows) == 0 {
		return 0
	}
	cutoff := time.Now().Add(-paid30dWindow)
	var sum int64
	for _, c := range rows {
		if c.Status != referral.CommissionPaid {
			continue
		}
		if c.PaidAt == nil || c.PaidAt.Before(cutoff) {
			continue
		}
		sum += c.CommissionCents
	}
	return sum
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
			RetireEligible:   isCommissionRetireEligible(c.Status),
		}
		if a, ok := atts[c.AttributionID]; ok {
			rec.ReferralID = a.ReferralID
			rec.ProposalID = a.ProposalID
		}
		out = append(out, rec)
	}
	return out, nil
}

// isCommissionRetireEligible mirrors the retry orchestrator's
// eligibility rule: only pending_kyc and failed rows can be retried.
// Centralised here so wallet payloads and the retry endpoint agree on
// the same definition — drift would let the UI render Retirer buttons
// that the backend immediately refuses.
func isCommissionRetireEligible(status referral.CommissionStatus) bool {
	switch status {
	case referral.CommissionPendingKYC, referral.CommissionFailed:
		return true
	default:
		return false
	}
}

// GroupedCommissions implements service.ReferralWalletReader. Returns
// the apporteur's recent commission rows partitioned by the four
// wallet-relevant statuses. The cap is shared with RecentCommissions
// so a noisy apporteur cannot blow up the wallet payload.
//
// The implementation calls RecentCommissions internally — there is no
// dedicated SQL query because the cardinality of an apporteur's
// recent commissions is bounded (50-100 rows) and a single read with
// post-partition is simpler, cheaper to test, and avoids drift between
// the two query paths.
func (s *Service) GroupedCommissions(ctx context.Context, referrerID uuid.UUID, limit int) (service.ReferralCommissionGroups, error) {
	recent, err := s.RecentCommissions(ctx, referrerID, limit)
	if err != nil {
		return service.ReferralCommissionGroups{}, err
	}
	groups := service.ReferralCommissionGroups{}
	for _, rec := range recent {
		switch rec.Status {
		case string(referral.CommissionPaid):
			groups.Paid = append(groups.Paid, rec)
		case string(referral.CommissionPendingKYC):
			groups.PendingKYC = append(groups.PendingKYC, rec)
		case string(referral.CommissionFailed):
			groups.Failed = append(groups.Failed, rec)
		case string(referral.CommissionCancelled):
			groups.Cancelled = append(groups.Cancelled, rec)
		default:
			// pending / clawed_back are not surfaced as groups in
			// D1+D2 — the wallet already has a separate summary
			// section for them. Skipping keeps the contract
			// focused on the retire-able statuses.
		}
	}
	return groups, nil
}
