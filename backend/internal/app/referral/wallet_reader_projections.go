package referral

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
)

// ─── Narrow ports used by ProjectedCommissions ────────────────────────────

// MilestonesByProposalLister batch-loads milestones for a slice of
// proposal ids. Defined as a narrow port (single method) so the
// referral feature stays decoupled from the milestone package — the
// adapter in wiring_adapters.go satisfies it from
// repository.MilestoneRepository.
//
// SYSTEM-ACTOR note: like ProposalSummaryResolver, the projection path
// straddles multiple proposals owned by different organizations. The
// implementation MUST tag the context with system.WithSystemActor so
// the per-proposal RLS gate admits every row reachable through the
// apporteur's attribution chain. The attribution table is the
// authoritative ACL — we already filtered to rows the apporteur owns
// before we ever ask for milestones.
type MilestonesByProposalLister interface {
	ListByProposals(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error)
}

// OrgMemberLister returns the user ids of every active member of each
// given organization. Narrow port so the referral feature does not
// import the team / organization service directly — the bootstrap
// wires the postgres organization_members adapter.
//
// Used by ProjectedCommissions to fan-out an org-keyed wallet query
// onto the underlying users (since referrals.referrer_id is a user id,
// not an org id — see ServiceDeps).
type OrgMemberLister interface {
	ListMemberUserIDsByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
}

// ─── ProjectedCommissions ─────────────────────────────────────────────────

// ProjectedCommissions returns the apporteur's projected + recorded
// commissions for the requesting organisation. Composes
// referral_attributions × proposal_milestones × referral_commissions
// in a single bounded sweep so the wallet UI can render "à venir" and
// "versée" rows together with consistent ordering.
//
// Algorithm (high level):
//
//	1. Resolve orgID → member user ids (apporteur is a user, an org can
//	   have multiple).
//	2. Batch-fetch every referral those users own (cursor-paginated,
//	   capped at MaxProjections × 5 inputs).
//	3. Batch-fetch attributions for those referrals.
//	4. Batch-fetch milestones for the attributed proposals.
//	5. Batch-fetch commissions per referral (one call per ref, NOT per
//	   milestone — the N+1 trap).
//	6. For each attribution × milestone, dispatch the milestone status
//	   into a projection / row lookup / skip decision.
//	7. Cap at MaxProjections, sort DESC by ProjectedAt.
//	8. Resolve mission titles via ProposalSummaryResolver (single batch
//	   per referral).
//
// Returns nil on missing dependencies (graceful degradation — wallet
// feature degrades to "no projections" when the referral / milestone
// stack is not wired in a worktree).
func (s *Service) ProjectedCommissions(ctx context.Context, orgID uuid.UUID) ([]ProjectedCommission, error) {
	if s.referrals == nil || s.milestonesByProposal == nil || s.orgMemberLister == nil {
		return nil, nil
	}
	memberMap, err := s.orgMemberLister.ListMemberUserIDsByOrgIDs(ctx, []uuid.UUID{orgID})
	if err != nil {
		return nil, fmt.Errorf("resolve org members: %w", err)
	}
	userIDs := memberMap[orgID]
	if len(userIDs) == 0 {
		return nil, nil
	}

	refs, err := s.collectReferralsForUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, nil
	}

	atts, err := s.referrals.ListAttributionsByReferralIDs(ctx, referralIDsOf(refs))
	if err != nil {
		return nil, fmt.Errorf("list attributions: %w", err)
	}
	if len(atts) == 0 {
		return nil, nil
	}

	proposalIDs := proposalIDsOf(atts)
	milestonesMap, err := s.milestonesByProposal.ListByProposals(ctx, proposalIDs)
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}

	commissionsByMilestone, err := s.fetchCommissionsByRefs(ctx, refs)
	if err != nil {
		return nil, err
	}

	titles := s.resolveMissionTitles(ctx, refs, atts)

	rows := buildProjections(atts, milestonesMap, commissionsByMilestone, titles)
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].ProjectedAt.After(rows[j].ProjectedAt)
	})
	if len(rows) > MaxProjections {
		rows = rows[:MaxProjections]
	}
	return rows, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────

// collectReferralsForUsers fans out a ListByReferrer call per user id
// and dedupes by referral ID. The page size mirrors MaxProjections × 5
// so even an apporteur with a long history is captured fully — the
// per-projection cap downstream handles the rendering side.
func (s *Service) collectReferralsForUsers(ctx context.Context, userIDs []uuid.UUID) ([]*referral.Referral, error) {
	seen := map[uuid.UUID]struct{}{}
	out := make([]*referral.Referral, 0, len(userIDs))
	for _, uid := range userIDs {
		rows, _, err := s.referrals.ListByReferrer(ctx, uid, repository.ReferralListFilter{Limit: MaxProjections * 5})
		if err != nil {
			return nil, fmt.Errorf("list referrals by referrer %s: %w", uid, err)
		}
		for _, r := range rows {
			if r == nil {
				continue
			}
			if _, dup := seen[r.ID]; dup {
				continue
			}
			seen[r.ID] = struct{}{}
			out = append(out, r)
		}
	}
	return out, nil
}

// fetchCommissionsByRefs batches ListCommissionsByReferral calls
// (one per referral, the segregated SQL path). Returns a milestone-id
// keyed map so the dispatcher can look up the existing row in O(1).
func (s *Service) fetchCommissionsByRefs(ctx context.Context, refs []*referral.Referral) (map[uuid.UUID]*referral.Commission, error) {
	out := map[uuid.UUID]*referral.Commission{}
	for _, r := range refs {
		if r == nil {
			continue
		}
		commissions, err := s.referrals.ListCommissionsByReferral(ctx, r.ID)
		if err != nil {
			return nil, fmt.Errorf("list commissions for referral %s: %w", r.ID, err)
		}
		for _, c := range commissions {
			if c == nil {
				continue
			}
			out[c.MilestoneID] = c
		}
	}
	return out, nil
}

// resolveMissionTitles batch-resolves proposal titles for every
// attributed proposal id, scoped by referral so the NF-12 attribution
// gate in ProposalSummaryResolver fires. Returns a proposal-id keyed
// map. Failures are swallowed — missing titles degrade the UI to an
// empty string, never break the wallet.
func (s *Service) resolveMissionTitles(ctx context.Context, refs []*referral.Referral, atts []*referral.Attribution) map[uuid.UUID]string {
	out := map[uuid.UUID]string{}
	if s.proposalSummaries == nil {
		return out
	}
	idsByReferral := map[uuid.UUID][]uuid.UUID{}
	for _, a := range atts {
		if a == nil {
			continue
		}
		idsByReferral[a.ReferralID] = append(idsByReferral[a.ReferralID], a.ProposalID)
	}
	for _, r := range refs {
		ids := idsByReferral[r.ID]
		if len(ids) == 0 {
			continue
		}
		summaries, err := s.proposalSummaries.ResolveProposalSummaries(ctx, r.ID, ids)
		if err != nil {
			continue
		}
		for id, s := range summaries {
			if s != nil {
				out[id] = s.Title
			}
		}
	}
	return out
}

// buildProjections is the pure dispatcher: it walks every attribution
// and emits zero, one, or many ProjectedCommission rows depending on
// the milestone × commission × ended_at matrix. Extracted out of
// ProjectedCommissions so the dispatch logic is straightforwardly
// table-testable without standing up the full service.
func buildProjections(
	atts []*referral.Attribution,
	milestonesMap map[uuid.UUID][]*milestone.Milestone,
	commissionsByMilestone map[uuid.UUID]*referral.Commission,
	titles map[uuid.UUID]string,
) []ProjectedCommission {
	out := make([]ProjectedCommission, 0, len(atts))
	for _, a := range atts {
		if a == nil {
			continue
		}
		title := titles[a.ProposalID]
		for _, m := range milestonesMap[a.ProposalID] {
			if m == nil {
				continue
			}
			row, ok := dispatchMilestone(a, m, commissionsByMilestone[m.ID], title)
			if !ok {
				continue
			}
			out = append(out, row)
		}
	}
	return out
}

// dispatchMilestone is the pure per-milestone decision: given an
// attribution, a milestone and the optional commission row, return
// the projection to emit (and ok=true) — or ok=false to SKIP the row.
//
// Skip cases:
//   - milestone in pending_funding (brief's "draft") or cancelled or
//     refunded — no commission ever attaches to these
//   - attribution ended AND milestone was approved/released AFTER the
//     ended_at cutoff (mirrors the gate enforced in commission_distributor)
//
// Decision matrix (see project plan for the full table):
//
//	| milestone.Status     | commission row | projection |
//	| pending_funding      | —              | SKIP       |
//	| funded/submitted/disputed (active) | (irrelevant) | ProjectionEscrowed (source=projection) |
//	| approved             | exists         | from row (paid|failed|pending) |
//	| approved             | missing        | ProjectionPending (source=projection) |
//	| released             | exists         | from row |
//	| released             | missing        | ProjectionPending (safety net) |
//	| cancelled / refunded | —              | SKIP       |
func dispatchMilestone(
	a *referral.Attribution,
	m *milestone.Milestone,
	c *referral.Commission,
	title string,
) (ProjectedCommission, bool) {
	switch m.Status {
	case milestone.StatusPendingFunding, milestone.StatusCancelled, milestone.StatusRefunded:
		return ProjectedCommission{}, false
	}

	// Run A end-gate: drop milestones approved AFTER attribution.ended_at.
	// The approval timestamp is the boundary used by the commission
	// distributor — we mirror it here so projections agree with what
	// the apporteur would actually be paid.
	if a.IsEnded() && m.ApprovedAt != nil && !m.ApprovedAt.Before(*a.EndedAt) {
		return ProjectedCommission{}, false
	}

	switch m.Status {
	case milestone.StatusFunded, milestone.StatusSubmitted, milestone.StatusDisputed:
		// Active escrow — commission row does not exist yet. Emit a
		// pure projection from the rate snapshot.
		if a.IsEnded() {
			// Attribution ended and we're in active escrow → no
			// commission will accrue. Skip.
			return ProjectedCommission{}, false
		}
		return ProjectedCommission{
			AttributionID:  a.ID,
			MilestoneID:    m.ID,
			ProposalID:     a.ProposalID,
			MissionTitle:   title,
			ProjectedCents: projectAmount(m.Amount, a.RatePctSnapshot),
			Currency:       "EUR",
			Status:         ProjectionEscrowed,
			Source:         SourceProjection,
			ProjectedAt:    m.CreatedAt,
		}, true
	case milestone.StatusApproved, milestone.StatusReleased:
		if c != nil {
			return fromCommissionRow(a, m, c, title), true
		}
		// Safety net: milestone is in a state where a commission
		// SHOULD exist (commission preparer runs on approval) but the
		// row was not found. Emit a projection rather than dropping
		// the line so the apporteur is not silently shorted.
		return ProjectedCommission{
			AttributionID:  a.ID,
			MilestoneID:    m.ID,
			ProposalID:     a.ProposalID,
			MissionTitle:   title,
			ProjectedCents: projectAmount(m.Amount, a.RatePctSnapshot),
			Currency:       "EUR",
			Status:         ProjectionPending,
			Source:         SourceProjection,
			ProjectedAt:    timeOrNow(m.ApprovedAt, m.CreatedAt),
		}, true
	}
	return ProjectedCommission{}, false
}

// fromCommissionRow maps an existing commission DB row onto the
// projection shape. Status maps the commission lifecycle into the
// four wallet buckets.
func fromCommissionRow(
	a *referral.Attribution,
	m *milestone.Milestone,
	c *referral.Commission,
	title string,
) ProjectedCommission {
	row := ProjectedCommission{
		AttributionID:  a.ID,
		MilestoneID:    m.ID,
		ProposalID:     a.ProposalID,
		MissionTitle:   title,
		ProjectedCents: c.CommissionCents,
		Currency:       c.Currency,
		Source:         SourceRow,
		ProjectedAt:    timeOrNow(c.PaidAt, c.CreatedAt),
	}
	switch c.Status {
	case referral.CommissionPaid:
		row.Status = ProjectionPaid
	case referral.CommissionFailed:
		row.Status = ProjectionFailed
	case referral.CommissionPendingKYC, referral.CommissionPending:
		row.Status = ProjectionPending
	case referral.CommissionClawedBack, referral.CommissionCancelled:
		// Reuse the failed bucket so the UI surfaces the line in the
		// "à corriger / résolu" section. The status string is what
		// the UI keys off; downstream rendering already handles the
		// clawed_back nuance via the commission record list.
		row.Status = ProjectionFailed
	default:
		row.Status = ProjectionPending
	}
	if row.Currency == "" {
		row.Currency = "EUR"
	}
	return row
}

// projectAmount applies the snapshot rate (stored as a percentage,
// e.g. 5.0 for 5%) to a milestone amount in cents. The /10000 divisor
// converts (cents × rate_pct) into cents — rate_pct is whole-number
// percent so a 5% commission on a 100,00 € milestone yields 500 cents
// (= 5,00 €).
func projectAmount(amountCents int64, ratePct float64) int64 {
	return int64(float64(amountCents) * ratePct / 100.0)
}

// timeOrNow returns *primary when non-nil, else fallback. Used to
// pick the most relevant "projected_at" timestamp without nil-guards
// scattered across dispatchMilestone.
func timeOrNow(primary *time.Time, fallback time.Time) time.Time {
	if primary != nil {
		return *primary
	}
	return fallback
}

// referralIDsOf flattens a slice of referrals to their ids.
func referralIDsOf(refs []*referral.Referral) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(refs))
	for _, r := range refs {
		if r != nil {
			out = append(out, r.ID)
		}
	}
	return out
}

// proposalIDsOf flattens a slice of attributions to their proposal ids,
// dedup-ing along the way. One attribution per proposal so duplicates
// should not happen — defensive cleanup makes the function safe to
// call on inputs from any source.
func proposalIDsOf(atts []*referral.Attribution) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, len(atts))
	for _, a := range atts {
		if a == nil {
			continue
		}
		if _, dup := seen[a.ProposalID]; dup {
			continue
		}
		seen[a.ProposalID] = struct{}{}
		out = append(out, a.ProposalID)
	}
	return out
}
