package referral

import (
	"context"

	"github.com/google/uuid"
)

// ProposalSummary is a tiny projection of a proposal carrying only the
// fields the referral UI needs to render an "attributed missions" section:
// the apporteur sees the title + status of every proposal that landed
// during the exclusivity window.
//
// Kept in this package (not domain/proposal) because the referral feature
// must NOT import the proposal package — cross-feature modularity rule.
// The adapter in wiring_adapters.go maps from proposal.Proposal to this
// shape.
type ProposalSummary struct {
	ID     uuid.UUID
	Title  string
	Status string
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
