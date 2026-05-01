package repository

// Segregated reader / writer / evidence-store interfaces for the
// dispute feature. Carved out of DisputeRepository (18 methods).
//
// Three families:
//   - DisputeReader  — read paths: lookup, listings (per-org, scheduler,
//     all), counter-proposal listings, AI chat history reads, stats.
//   - DisputeWriter  — life-cycle mutations on dispute rows and
//     counter-proposals.
//   - DisputeEvidenceStore — evidence + chat-message append-only stores.
//
// The single postgres adapter implements ALL three. Wiring stays the
// same; consumers narrow their declared dependency type.

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/dispute"
)

// DisputeReader exposes read paths over the disputes/counter_proposals/
// chat_messages tables.
type DisputeReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*dispute.Dispute, error)
	// GetByIDForOrg is the tenant-aware sibling of GetByID — see the
	// wide-port doc on DisputeRepository.GetByIDForOrg.
	GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*dispute.Dispute, error)
	GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*dispute.Dispute, error)
	ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*dispute.Dispute, string, error)
	ListPendingForScheduler(ctx context.Context) ([]*dispute.Dispute, error)
	ListAll(ctx context.Context, cursor string, limit int, statusFilter string) ([]*dispute.Dispute, string, error)
	GetCounterProposalByID(ctx context.Context, id uuid.UUID) (*dispute.CounterProposal, error)
	ListCounterProposals(ctx context.Context, disputeID uuid.UUID) ([]*dispute.CounterProposal, error)
	ListChatMessages(ctx context.Context, disputeID uuid.UUID) ([]*dispute.ChatMessage, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountAll(ctx context.Context) (total int, open int, escalated int, err error)
}

// DisputeWriter exposes mutation paths on dispute rows and counter
// proposals (the negotiation aggregate inside a dispute).
type DisputeWriter interface {
	Create(ctx context.Context, d *dispute.Dispute) error
	Update(ctx context.Context, d *dispute.Dispute) error
	CreateCounterProposal(ctx context.Context, cp *dispute.CounterProposal) error
	UpdateCounterProposal(ctx context.Context, cp *dispute.CounterProposal) error
	SupersedeAllPending(ctx context.Context, disputeID uuid.UUID) error
}

// DisputeEvidenceStore covers the append-only evidence + chat message
// stores. Evidence is uploaded by parties; chat messages are produced
// by the admin AI Q/A loop.
type DisputeEvidenceStore interface {
	CreateEvidence(ctx context.Context, e *dispute.Evidence) error
	ListEvidence(ctx context.Context, disputeID uuid.UUID) ([]*dispute.Evidence, error)
	CreateChatMessage(ctx context.Context, msg *dispute.ChatMessage) error
}

// Compile-time guarantee that DisputeRepository is always equivalent to
// the union of its segregated children.
var _ DisputeRepository = (interface {
	DisputeReader
	DisputeWriter
	DisputeEvidenceStore
})(nil)
