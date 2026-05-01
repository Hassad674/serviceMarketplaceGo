package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
)

// MilestoneRepository defines persistence operations for proposal milestones.
//
// Every mutation that transitions a milestone's status must go through the
// optimistic-locked Update path: callers fetch with GetByIDWithVersion,
// perform the domain transition in memory, then call Update. A concurrent
// modification causes Update to return milestone.ErrConcurrentUpdate,
// which the app layer can choose to surface or retry.
type MilestoneRepository interface {
	// CreateBatch inserts every milestone of a proposal in a single
	// transaction. The slice must not be empty and must come from
	// milestone.NewMilestoneBatch (which enforces sequence 1..N and the
	// MaxMilestonesPerProposal cap).
	CreateBatch(ctx context.Context, milestones []*milestone.Milestone) error

	// GetByID fetches a milestone without taking a lock. Suitable for
	// read-only queries (listings, detail views, projections).
	GetByID(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error)

	// GetByIDForOrg fetches a milestone by id under the caller's
	// organization tenant context. The adapter wraps the read in
	// RunInTxWithTenant so the RLS policy on proposal_milestones
	// (which JOINs through to the parent proposal's stakeholder
	// orgs) admits the row. Returns ErrMilestoneNotFound when the
	// row does not exist OR when the caller's org is not party to
	// the parent proposal — RLS does not distinguish "missing"
	// from "denied".
	//
	// User-facing app callers MUST use this method; the legacy
	// GetByID is retained for the proposal scheduler's
	// auto-approve / fund-reminder paths which run as system
	// actors.
	GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*milestone.Milestone, error)

	// GetByIDWithVersion fetches a milestone and returns its current
	// Version field for optimistic-concurrency control by the caller.
	//
	// CONTRACT: this is a plain SELECT — it does NOT take a row-level
	// pessimistic lock. The previous name GetByIDForUpdate was misleading
	// because the implementation opened a transaction, ran SELECT FOR
	// UPDATE, and immediately committed — which RELEASES the lock at
	// commit. The actual race protection comes from Update's
	// `WHERE id = $1 AND version = $2` clause and the
	// milestone.ErrConcurrentUpdate sentinel.
	//
	// Concurrency model: callers fetch with GetByIDWithVersion → mutate
	// the in-memory copy → call Update. If two callers fetch the same
	// version, both reach Update; one wins (rows affected = 1) and
	// bumps the version, the other loses (rows affected = 0) and
	// receives ErrConcurrentUpdate so it can refetch and retry.
	//
	// BUG-11 background: dropping the SELECT FOR UPDATE simplifies the
	// code and clarifies the semantics — the lock was doing nothing
	// useful (committed immediately) and the misleading name made
	// readers believe the concurrency model was pessimistic when it
	// has always been optimistic.
	GetByIDWithVersion(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error)

	// ListByProposal returns every milestone of a proposal, ordered by
	// ascending sequence. Used to render the milestone tracker and compute
	// the proposal's macro status.
	ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*milestone.Milestone, error)

	// GetCurrentActive returns the first non-terminal milestone of the
	// proposal by ascending sequence, or milestone.ErrMilestoneNotFound if
	// every milestone is terminal. This is the milestone that owns the
	// current client/provider CTA.
	GetCurrentActive(ctx context.Context, proposalID uuid.UUID) (*milestone.Milestone, error)

	// Update persists a domain transition on the given milestone,
	// enforcing optimistic concurrency. The WHERE clause MUST include
	// "id = $1 AND version = $2" and bump version on success; a zero-row
	// result returns milestone.ErrConcurrentUpdate.
	Update(ctx context.Context, m *milestone.Milestone) error

	// CreateDeliverable registers a file attached to a specific milestone.
	// The caller is responsible for checking that the milestone is in a
	// mutable status (deliverable.IsMutableStatus).
	CreateDeliverable(ctx context.Context, d *milestone.Deliverable) error

	// ListDeliverables returns every deliverable attached to a milestone,
	// ordered by created_at ASC.
	ListDeliverables(ctx context.Context, milestoneID uuid.UUID) ([]*milestone.Deliverable, error)

	// DeleteDeliverable removes a deliverable by ID. Mutability is enforced
	// at the app layer before calling this method.
	DeleteDeliverable(ctx context.Context, id uuid.UUID) error

	// ListByProposals returns milestones for multiple proposals in a single
	// query, keyed by proposal_id. Used by list endpoints to avoid N+1
	// when rendering a page of proposals with their milestone summaries.
	ListByProposals(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error)
}

// MilestoneTransitionRepository is the append-only audit trail for
// milestone state changes (phase 9). Production grants the application
// DB user INSERT and SELECT only on this table — Update and Delete
// are forbidden so the timeline cannot be rewritten.
type MilestoneTransitionRepository interface {
	// Insert persists a single transition row. Errors are non-fatal
	// at the call site (the milestone update has already committed)
	// but should be logged for incident review.
	Insert(ctx context.Context, t *milestone.Transition) error

	// ListByMilestone returns every transition for a milestone in
	// chronological order. Used by admin dashboards and dispute
	// arbitration to reconstruct the milestone timeline.
	ListByMilestone(ctx context.Context, milestoneID uuid.UUID) ([]*milestone.Transition, error)
}
