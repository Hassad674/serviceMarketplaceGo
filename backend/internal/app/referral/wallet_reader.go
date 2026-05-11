package referral

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// ProjectionStatus is the bucket the wallet groups a projected
// commission into. Mirrors the wallet's "escrowed / pending / paid /
// failed" grammar so the UI can render projections alongside paid rows
// without a second taxonomy.
//
// Run B (WALLET-UNIFY) — introduced by ProjectedCommissions so the
// /wallet/summary endpoint can compose missions + commissions with the
// same status shape.
type ProjectionStatus string

const (
	// ProjectionEscrowed — the milestone holds escrow (funded /
	// submitted / disputed). The commission would be paid out on
	// milestone release; for now it sits in the apporteur's "à venir"
	// pile and the UI flags it accordingly.
	ProjectionEscrowed ProjectionStatus = "escrowed"
	// ProjectionPending — the milestone has been approved by the
	// client (commission row should exist) but the row could not be
	// found for race-condition / safety-net reasons. Treated as
	// "pending Stripe transfer" by the UI.
	ProjectionPending ProjectionStatus = "pending"
	// ProjectionPaid — the commission row exists and is in paid
	// status. Money has reached the apporteur's Stripe.
	ProjectionPaid ProjectionStatus = "paid"
	// ProjectionFailed — the commission row exists but its Stripe
	// transfer failed. Retry-eligible.
	ProjectionFailed ProjectionStatus = "failed"
)

// ProjectionSource discriminates whether a row in the response was
// computed from an actual `referral_commissions` row (source=row) or
// projected speculatively from an active milestone with no commission
// row yet (source=projection). The UI uses this to decide between
// "Versée" and "À venir" labelling without re-deriving the status
// from the bucket.
type ProjectionSource string

const (
	// SourceProjection — no DB row backs this entry; computed from
	// milestone.amount × attribution.rate_pct_snapshot.
	SourceProjection ProjectionSource = "projection"
	// SourceRow — backed by an existing referral_commissions row.
	SourceRow ProjectionSource = "row"
)

// ProjectedCommission is one row of the apporteur's "projected
// commissions" wallet section. Carries enough context for the UI to
// deep-link to the parent referral / proposal and to label the line
// with the right mission title.
//
// ProjectedCents is computed snapshot-style:
//   - SourceRow rows expose the commission row's CommissionCents
//     directly (which itself was snapshot-locked to the rate at
//     milestone-approval time).
//   - SourceProjection rows compute milestone.Amount *
//     attribution.RatePctSnapshot / 10000 so a future rate change
//     never retroactively shifts the projected amount.
type ProjectedCommission struct {
	AttributionID  uuid.UUID
	MilestoneID    uuid.UUID
	ProposalID     uuid.UUID
	MissionTitle   string
	ProjectedCents int64
	Currency       string
	Status         ProjectionStatus
	Source         ProjectionSource
	ProjectedAt    time.Time
}

// MaxProjections is the hard cap on the size of the slice returned by
// ProjectedCommissions. A noisy apporteur with thousands of attributed
// milestones cannot blow up the /wallet/summary response payload. Set
// to 200 — at 20 visible rows on a wallet tile, that's 10 pages of
// history, well past what any UI surfaces.
const MaxProjections = 200

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
