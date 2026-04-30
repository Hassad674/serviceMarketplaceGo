package repository

// Segregated reader / writer / milestone-store interfaces for the
// proposal feature. Carved out of ProposalRepository (16 methods).
//
// Three families:
//   - ProposalReader   — read paths over the proposals + documents
//     tables, batch loads, dashboard aggregations.
//   - ProposalWriter   — mutation paths: create (single + with docs),
//     update, document append.
//   - ProposalMilestoneStore — the single composite write that persists
//     a proposal AND its milestone batch atomically. Owned by the
//     milestone feature semantically; pulled into its own port so a
//     read-only consumer (project-history aggregator) does not pull in
//     a write surface it never uses.

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/proposal"
)

// ProposalReader exposes read paths over the proposals + proposal
// documents tables.
type ProposalReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*proposal.Proposal, error)
	GetLatestVersion(ctx context.Context, rootProposalID uuid.UUID) (*proposal.Proposal, error)
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*proposal.Proposal, error)
	ListActiveProjectsByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposal.Proposal, string, error)
	ListCompletedByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposal.Proposal, string, error)
	GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*proposal.ProposalDocument, error)
	IsOrgAuthorizedForProposal(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error)
	CountAll(ctx context.Context) (total int, active int, err error)
	SumPaidByClientOrganization(ctx context.Context, orgID uuid.UUID) (int64, error)
	ListCompletedByClientOrganization(ctx context.Context, orgID uuid.UUID, limit int) ([]*proposal.Proposal, error)
}

// ProposalWriter exposes the mutation paths over the proposals and
// proposal documents tables. The composite "create with milestones"
// write lives in ProposalMilestoneStore because the milestone batch is
// owned by the milestone feature.
type ProposalWriter interface {
	Create(ctx context.Context, p *proposal.Proposal) error
	CreateWithDocuments(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument) error
	Update(ctx context.Context, p *proposal.Proposal) error
	CreateDocument(ctx context.Context, doc *proposal.ProposalDocument) error
}

// ProposalMilestoneStore covers the single "create proposal + milestone
// batch" composite transaction. Pulled into its own port so a consumer
// that only reads proposals does not pull in milestone-aware writes.
type ProposalMilestoneStore interface {
	CreateWithDocumentsAndMilestones(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument, milestones []*milestone.Milestone) error
}

// Compile-time guarantee that ProposalRepository is always equivalent
// to the union of its segregated children.
var _ ProposalRepository = (interface {
	ProposalReader
	ProposalWriter
	ProposalMilestoneStore
})(nil)
