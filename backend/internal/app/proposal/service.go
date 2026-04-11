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
	Organizations repository.OrganizationRepository
	Messages      service.MessageSender
	Storage       service.StorageService
	Notifications service.NotificationSender
	Payments      service.PaymentProcessor            // nil if Stripe not configured
	Credits       repository.JobCreditRepository      // nil if credits not configured
	BonusLog      repository.CreditBonusLogRepository // nil if not configured
}

type Service struct {
	proposals     repository.ProposalRepository
	users         repository.UserRepository
	orgs          repository.OrganizationRepository
	messages      service.MessageSender
	storage       service.StorageService
	notifications service.NotificationSender
	payments      service.PaymentProcessor
	credits       repository.JobCreditRepository
	bonusLog      repository.CreditBonusLogRepository
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		proposals:     deps.Proposals,
		users:         deps.Users,
		orgs:          deps.Organizations,
		messages:      deps.Messages,
		storage:       deps.Storage,
		notifications: deps.Notifications,
		payments:      deps.Payments,
		credits:       deps.Credits,
		bonusLog:      deps.BonusLog,
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

// AcceptProposalInput, DeclineProposalInput, and every other mutation
// input carry BOTH the acting user id (for audit / notifications) and
// the caller's organization id (for authorization). Since the R1 team
// refactor, a proposal is shared between every operator of the
// client-side and provider-side orgs — so the authorization check is
// "does your org own this side", not "are you the exact sender/recipient
// user". OrgID is the uuid stored in the JWT context under
// middleware.ContextKeyOrganizationID.
type AcceptProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
}

type DeclineProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
}

type ModifyProposalInput struct {
	ProposalID  uuid.UUID
	UserID      uuid.UUID
	OrgID       uuid.UUID
	Title       string
	Description string
	Amount      int64
	Deadline    *time.Time
	Documents   []DocumentInput
}

type PayProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
}

type RequestCompletionInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
}

type CompleteProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
}

type RejectCompletionInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
}
