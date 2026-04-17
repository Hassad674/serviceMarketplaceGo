package referral

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
)

// ListByReferrer returns the paginated list of referrals where the given
// user is the apporteur. Thin pass-through to the repository — the filter
// values are validated by the handler layer.
func (s *Service) ListByReferrer(ctx context.Context, referrerID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return s.referrals.ListByReferrer(ctx, referrerID, filter)
}

// ListIncomingForProvider returns the paginated list of referrals where
// the given user is the provider party.
func (s *Service) ListIncomingForProvider(ctx context.Context, providerID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return s.referrals.ListIncomingForProvider(ctx, providerID, filter)
}

// ListIncomingForClient returns the paginated list of referrals where the
// given user is the client party.
func (s *Service) ListIncomingForClient(ctx context.Context, clientID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return s.referrals.ListIncomingForClient(ctx, clientID, filter)
}

// ListNegotiations returns the audit trail of negotiation events for a
// given referral. The handler is expected to first verify the caller is
// one of the three parties before calling this.
func (s *Service) ListNegotiations(ctx context.Context, referralID uuid.UUID) ([]*referral.Negotiation, error) {
	return s.referrals.ListNegotiations(ctx, referralID)
}

// AttributionWithStats is a projection of one attribution enriched with
// the parent proposal's title + status and the aggregate commission
// stats (how much has been paid to the apporteur so far, how many
// milestones are still in flight). Used by the referral detail page's
// "Missions attribuées pendant cette intro" section.
//
// Escrow commission = apporteur's share of the amount currently held
// in escrow on funded-but-not-released milestones. It is purely
// derived (gross escrow × rate_pct_snapshot) and not stored in the
// commissions table — we never INSERT a referral_commissions row
// until Stripe actually releases the milestone. Showing it upfront
// gives the apporteur a preview of "what's coming" on in-progress
// missions so the card does not read "0 €" on a live contract.
//
// MilestonesTotal is the authoritative count of milestones on the
// proposal (≥ 1 by domain invariant) — replaces the old
// MilestonesPaid + MilestonesPending math, which was missing every
// milestone that never produced a commission row yet (the common
// "in-progress" case).
//
// ClawedBackCommissionCents sums the apporteur commissions that were
// paid out then clawed back after a dispute — lets the UI render a
// red "- X € reprises" line under the main commission amount.
type AttributionWithStats struct {
	Attribution               *referral.Attribution
	ProposalTitle             string
	ProposalStatus            string
	TotalCommissionCents      int64 // sum of commissions in status=paid
	PendingCommissionCents    int64 // sum of commissions in status=pending or pending_kyc
	ClawedBackCommissionCents int64 // sum of commissions in status=clawed_back
	EscrowCommissionCents     int64 // apporteur share of funded-but-not-released milestones
	MilestonesPaid            int
	MilestonesPending         int
	MilestonesTotal           int
}

// ListAttributionsWithStats returns the attributions of a referral
// enriched with proposal summary + commission aggregates. Only the
// three parties (referrer, provider, client) may read. The client's
// DTO is trimmed upstream (handler layer) to strip commission amounts
// — Modèle A confidentiality.
func (s *Service) ListAttributionsWithStats(ctx context.Context, referralID, viewerID uuid.UUID) ([]*AttributionWithStats, error) {
	// Ownership check — reuse GetByID so the service stays the single
	// gate keeping these reads scoped to the three parties.
	if _, err := s.GetByID(ctx, referralID, viewerID); err != nil {
		return nil, err
	}

	atts, err := s.referrals.ListAttributionsByReferral(ctx, referralID)
	if err != nil {
		return nil, err
	}
	if len(atts) == 0 {
		return nil, nil
	}

	// Resolve proposal summaries in one pass.
	ids := make([]uuid.UUID, 0, len(atts))
	for _, a := range atts {
		ids = append(ids, a.ProposalID)
	}
	summaries := map[uuid.UUID]*ProposalSummary{}
	if s.proposalSummaries != nil {
		summaries, _ = s.proposalSummaries.ResolveProposalSummaries(ctx, ids)
	}

	// Aggregate commission stats per attribution.
	allCommissions, err := s.referrals.ListCommissionsByReferral(ctx, referralID)
	if err != nil {
		return nil, err
	}
	type agg struct {
		paid       int64
		pending    int64
		clawedBack int64
		countP     int
		countQ     int
	}
	byAtt := make(map[uuid.UUID]*agg, len(atts))
	for _, c := range allCommissions {
		a, ok := byAtt[c.AttributionID]
		if !ok {
			a = &agg{}
			byAtt[c.AttributionID] = a
		}
		switch c.Status {
		case referral.CommissionPaid:
			a.paid += c.CommissionCents
			a.countP++
		case referral.CommissionPending, referral.CommissionPendingKYC:
			a.pending += c.CommissionCents
			a.countQ++
		case referral.CommissionClawedBack:
			a.clawedBack += c.CommissionCents
		}
	}

	out := make([]*AttributionWithStats, 0, len(atts))
	for _, att := range atts {
		row := &AttributionWithStats{Attribution: att}
		if sum, ok := summaries[att.ProposalID]; ok {
			row.ProposalTitle = sum.Title
			row.ProposalStatus = sum.Status
			row.MilestonesTotal = sum.MilestonesTotal
			// Apporteur share of the gross amount currently in escrow
			// on this proposal. Uses the same basis-points truncation
			// as the domain's commission math so the preview lines up
			// with the eventual real commission to the cent.
			row.EscrowCommissionCents = escrowCommissionCents(sum.FundedAmountCents, att.RatePctSnapshot)
		}
		if a, ok := byAtt[att.ID]; ok {
			row.TotalCommissionCents = a.paid
			row.PendingCommissionCents = a.pending
			row.ClawedBackCommissionCents = a.clawedBack
			row.MilestonesPaid = a.countP
			row.MilestonesPending = a.countQ
		}
		out = append(out, row)
	}
	return out, nil
}

// escrowCommissionCents previews the apporteur's commission on a gross
// escrow amount using the same basis-points truncation as the domain
// layer (see domain/referral.commission.computeCommissionCents). Kept
// as a package-private helper to avoid exposing a second commission
// math path — the domain function is unexported by design.
func escrowCommissionCents(grossCents int64, ratePct float64) int64 {
	rateBp := int64(ratePct * 100)
	return grossCents * rateBp / 10_000
}

// ListCommissionsByReferral returns every commission row attached to a
// referral, across all its attributions. Reserved for the apporteur and
// the provider party — the client never sees individual commission
// amounts (Modèle A). The handler enforces that scope via the DTO
// tailoring step; the service authorises any of the three parties so
// downstream tooling (admin, tests) can read without duplication.
func (s *Service) ListCommissionsByReferral(ctx context.Context, referralID, viewerID uuid.UUID) ([]*referral.Commission, error) {
	if _, err := s.GetByID(ctx, referralID, viewerID); err != nil {
		return nil, err
	}
	return s.referrals.ListCommissionsByReferral(ctx, referralID)
}
