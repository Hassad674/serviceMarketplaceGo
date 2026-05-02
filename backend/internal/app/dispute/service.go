// Package dispute implements the dispute resolution feature.
//
// Disputes freeze funds on active proposals and allow negotiation
// between client and provider. Unresolved disputes escalate to
// admin mediation with AI-assisted analysis.
package dispute

import (
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// disputeProposals is the local composite the dispute service needs:
// it reads a proposal (GetByID / GetByIDForOrg) before negotiating
// terms and writes it back (Update) when the resolution flow flips
// the dispute_status / status / amount fields. No segregated child
// covers both, so we compose locally and keep the wide port out of
// the dependency graph.
type disputeProposals interface {
	repository.ProposalReader
	repository.ProposalWriter
}

type ServiceDeps struct {
	// Disputes stays on the wide DisputeRepository — the dispute
	// service straddles all three segregated children (Reader for the
	// many lookup paths, Writer for create/update + counter-proposals,
	// EvidenceStore for evidence + chat-message appends). Composing
	// locally would reproduce the wide port verbatim.
	Disputes      repository.DisputeRepository
	Proposals     disputeProposals
	Milestones    repository.MilestoneRepository // phase 8 — required so disputes scope to a specific milestone
	Users         repository.UserReader
	MessageRepo   repository.MessageReader // read-side, used by AI summary (ListMessagesSinceTime)
	Messages      service.MessageSender
	Notifications service.NotificationSender
	Payments      service.PaymentProcessor
	AI            service.AIAnalyzer
}

type Service struct {
	disputes      repository.DisputeRepository
	proposals     disputeProposals
	milestones    repository.MilestoneRepository
	users         repository.UserReader
	messageRepo   repository.MessageReader
	messages      service.MessageSender
	notifications service.NotificationSender
	payments      service.PaymentProcessor
	ai            service.AIAnalyzer
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		disputes:      deps.Disputes,
		proposals:     deps.Proposals,
		milestones:    deps.Milestones,
		users:         deps.Users,
		messageRepo:   deps.MessageRepo,
		messages:      deps.Messages,
		notifications: deps.Notifications,
		payments:      deps.Payments,
		ai:            deps.AI,
	}
}
