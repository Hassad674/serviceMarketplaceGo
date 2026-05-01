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
	// GetByIDForOrg fetches a dispute by id under the caller's
	// organization tenant context. The adapter wraps the read in
	// RunInTxWithTenant so the RLS policy keyed on
	// app.current_org_id matches the dispute's client or provider
	// organization. Returns ErrDisputeNotFound when the row does
	// not exist OR when the caller's org is not party to the
	// dispute — RLS does not distinguish "missing" from "denied".
	//
	// User-facing app callers MUST use this method; the legacy
	// GetByID is retained for the dispute scheduler's auto-resolve
	// path which runs as a system actor.
	GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*dispute.Dispute, error)
	GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*dispute.Dispute, error)
	Update(ctx context.Context, d *dispute.Dispute) error

	// Listings
	ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*dispute.Dispute, string, error)
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

	// AI chat history (admin Q/A persisted append-only per dispute)
	CreateChatMessage(ctx context.Context, msg *dispute.ChatMessage) error
	ListChatMessages(ctx context.Context, disputeID uuid.UUID) ([]*dispute.ChatMessage, error)

	// Stats
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountAll(ctx context.Context) (total int, open int, escalated int, err error)
}
