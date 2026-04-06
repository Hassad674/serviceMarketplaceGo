package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/proposal"
)

// ProposalRepository defines persistence operations for proposals.
type ProposalRepository interface {
	Create(ctx context.Context, p *proposal.Proposal) error
	CreateWithDocuments(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument) error
	GetByID(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error)
	Update(ctx context.Context, p *proposal.Proposal) error
	GetLatestVersion(ctx context.Context, rootProposalID uuid.UUID) (*proposal.Proposal, error)
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*proposal.Proposal, error)
	ListActiveProjects(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*proposal.Proposal, string, error)
	GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*proposal.ProposalDocument, error)
	CreateDocument(ctx context.Context, doc *proposal.ProposalDocument) error
	CountAll(ctx context.Context) (total int, active int, err error)
}
