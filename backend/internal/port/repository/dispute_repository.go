package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/dispute"
)

type DisputeRepository interface {
	// Core CRUD
	Create(ctx context.Context, d *dispute.Dispute) error
	GetByID(ctx context.Context, id uuid.UUID) (*dispute.Dispute, error)
	GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*dispute.Dispute, error)
	Update(ctx context.Context, d *dispute.Dispute) error

	// Listings
	ListByUserID(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*dispute.Dispute, string, error)
	ListPendingForScheduler(ctx context.Context) ([]*dispute.Dispute, error)
	ListAll(ctx context.Context, cursor string, limit int, statusFilter string) ([]*dispute.Dispute, string, error)

	// Evidence
	CreateEvidence(ctx context.Context, e *dispute.Evidence) error
	ListEvidence(ctx context.Context, disputeID uuid.UUID) ([]*dispute.Evidence, error)

	// Counter-proposals
	CreateCounterProposal(ctx context.Context, cp *dispute.CounterProposal) error
	GetCounterProposalByID(ctx context.Context, id uuid.UUID) (*dispute.CounterProposal, error)
	UpdateCounterProposal(ctx context.Context, cp *dispute.CounterProposal) error
	ListCounterProposals(ctx context.Context, disputeID uuid.UUID) ([]*dispute.CounterProposal, error)
	SupersedeAllPending(ctx context.Context, disputeID uuid.UUID) error

	// Stats
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountAll(ctx context.Context) (total int, open int, escalated int, err error)
}
