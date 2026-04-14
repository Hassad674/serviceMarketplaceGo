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

type ServiceDeps struct {
	Disputes      repository.DisputeRepository
	Proposals     repository.ProposalRepository
	Milestones    repository.MilestoneRepository // phase 8 — required so disputes scope to a specific milestone
	Users         repository.UserRepository
	MessageRepo   repository.MessageRepository // read-side, used by AI summary
	Messages      service.MessageSender
	Notifications service.NotificationSender
	Payments      service.PaymentProcessor
	AI            service.AIAnalyzer
}

type Service struct {
	disputes      repository.DisputeRepository
	proposals     repository.ProposalRepository
	milestones    repository.MilestoneRepository
	users         repository.UserRepository
	messageRepo   repository.MessageRepository
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
