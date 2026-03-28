package proposal

import (
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type ServiceDeps struct {
	Proposals     repository.ProposalRepository
	Users         repository.UserRepository
	Messages      service.MessageSender
	Storage       service.StorageService
	Notifications service.NotificationSender
	Payments      service.PaymentProcessor // nil if Stripe not configured
}

type Service struct {
	proposals     repository.ProposalRepository
	users         repository.UserRepository
	messages      service.MessageSender
	storage       service.StorageService
	notifications service.NotificationSender
	payments      service.PaymentProcessor
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		proposals:     deps.Proposals,
		users:         deps.Users,
		messages:      deps.Messages,
		storage:       deps.Storage,
		notifications: deps.Notifications,
		payments:      deps.Payments,
	}
}

type CreateProposalInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	RecipientID    uuid.UUID
	Title          string
	Description    string
	Amount         int64
	Deadline       *time.Time
	Documents      []DocumentInput
}

type DocumentInput struct {
	Filename string
	URL      string
	Size     int64
	MimeType string
}

type AcceptProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
}

type DeclineProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
}

type ModifyProposalInput struct {
	ProposalID  uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description string
	Amount      int64
	Deadline    *time.Time
	Documents   []DocumentInput
}

type PayProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
}

type RequestCompletionInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
}

type CompleteProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
}

type RejectCompletionInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
}
