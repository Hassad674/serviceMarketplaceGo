package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/proposal"
)

// ProposalRepository defines persistence operations for proposals.
type ProposalRepository interface {
	Create(ctx context.Context, p *proposal.Proposal) error
	CreateWithDocuments(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument) error
	// CreateWithDocumentsAndMilestones persists the proposal, its
	// documents, AND its milestone batch in a single transaction. The
	// milestones slice must be non-empty (a proposal always has ≥1
	// milestone since phase 4).
	CreateWithDocumentsAndMilestones(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument, milestones []*milestone.Milestone) error
	GetByID(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error)
	// GetByIDs batch-loads proposals for a set of ids in a single query.
	// Returns the slice in no particular order — callers must map by id
	// if they need a specific ordering. Missing ids are silently dropped
	// from the result; the primary caller (apporteur reputation) joins
	// this against its own ordered attribution list.
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*proposal.Proposal, error)
	Update(ctx context.Context, p *proposal.Proposal) error
	GetLatestVersion(ctx context.Context, rootProposalID uuid.UUID) (*proposal.Proposal, error)
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*proposal.Proposal, error)
	// ListActiveProjectsByOrganization returns active-or-later proposals
	// where the caller's organization is either the client or the
	// provider. Used for the org-wide "my projects" view.
	ListActiveProjectsByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposal.Proposal, string, error)
	// ListCompletedByOrganization returns proposals where the given
	// organization is the provider (via users.organization_id) and the
	// status is 'completed'. Ordered by completed_at DESC.
	ListCompletedByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposal.Proposal, string, error)
	GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*proposal.ProposalDocument, error)
	CreateDocument(ctx context.Context, doc *proposal.ProposalDocument) error
	// IsOrgAuthorizedForProposal returns true if the given organization is a
	// party to the proposal on either side: the client side (via the
	// denormalized proposals.organization_id column set from
	// organization_members on insert) OR the provider side (via
	// users.organization_id JOIN on the proposal's provider_id). Any
	// operator of such an org can read the proposal — the Stripe Dashboard
	// shared-workspace model. Directional mutations (accept, pay, complete)
	// are still narrowed to a single side at the service layer.
	IsOrgAuthorizedForProposal(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error)
	CountAll(ctx context.Context) (total int, active int, err error)

	// SumPaidByClientOrganization returns the total amount (in cents)
	// the given organization has spent as the client across all paid-
	// or-later proposals (paid, active, completion_requested, completed,
	// disputed). Keyed on the denormalized proposals.client_organization_id
	// column (migration 115) so the query hits the dedicated partial
	// index without a JOIN.
	SumPaidByClientOrganization(ctx context.Context, orgID uuid.UUID) (int64, error)

	// ListCompletedByClientOrganization returns the most recent
	// completed proposals where the given organization is the client.
	// Ordered by completed_at DESC, capped to limit. Used to power the
	// public client profile's project-history section.
	ListCompletedByClientOrganization(ctx context.Context, orgID uuid.UUID, limit int) ([]*proposal.Proposal, error)
}
