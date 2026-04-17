package referral

import (
	"context"

	"github.com/google/uuid"
)

// ProposalSummary is a tiny projection of a proposal carrying the
// fields the referral UI needs to render an "attributed missions"
// section: the apporteur sees the title + status of every proposal
// that landed during the exclusivity window, plus the authoritative
// milestone count so the UI can render "{paid}/{total} jalons"
// instead of inferring the total from heterogeneous commission rows.
//
// FundedAmountCents is the gross amount (in cents) of milestones the
// client has already funded but the provider has not yet had released
// — i.e. money currently held in escrow for the provider. Feeding this
// to the apporteur lets the UI show an "en séquestre" commission line
// for in-progress missions even before a single Stripe transfer has
// gone out.
//
// Kept in this package (not domain/proposal) because the referral
// feature must NOT import the proposal package — cross-feature
// modularity rule. The adapter in wiring_adapters.go maps from
// proposal.Proposal + milestone.Milestone to this shape.
type ProposalSummary struct {
	ID                uuid.UUID
	Title             string
	Status            string
	MilestonesTotal   int
	MilestonesFunded  int   // informational — milestones currently in escrow-ish states
	FundedAmountCents int64 // sum of Amount for milestones in non-released escrow states
}

// ProposalSummaryResolver batches proposal summaries by id for the
// referral detail view. Returning a map rather than a slice makes
// "proposal not found" (e.g. deleted, or still being created) degrade
// gracefully: the enriched DTO simply omits title/status for that id.
//
// Defined as a port so the referral feature stays decoupled — the
// concrete resolver in wiring_adapters.go reads from the
// ProposalRepository without the referral package ever importing it.
type ProposalSummaryResolver interface {
	ResolveProposalSummaries(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*ProposalSummary, error)
}
