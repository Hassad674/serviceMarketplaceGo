package proposal

import (
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type ServiceDeps struct {
	Proposals     repository.ProposalRepository
	Milestones    repository.MilestoneRepository // required since phase 4 — every proposal has ≥1 milestone
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
	milestones    repository.MilestoneRepository
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
		milestones:    deps.Milestones,
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

// CreateProposalInput is the payload the create handler passes in.
//
// Since phase 4, a proposal ALWAYS has at least one milestone. Callers
// have two ways of expressing this:
//
//  1. Milestone mode (new): PaymentMode="milestone" and Milestones is a
//     non-empty slice. The proposal's total amount is derived from the
//     sum of milestone amounts (the Amount field is ignored).
//  2. One-time mode (legacy + default): PaymentMode="" or "one_time"
//     and Milestones is empty. The service synthesises a single
//     milestone at sequence=1 covering the full Amount.
//
// Either way, the downstream code path is identical — there is no
// dual-branch logic beyond the CreateProposal factory.
type CreateProposalInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	RecipientID    uuid.UUID
	Title          string
	Description    string
	Amount         int64 // ignored when Milestones is non-empty
	Deadline       *time.Time
	Documents      []DocumentInput
	PaymentMode    string // "one_time" (default) or "milestone"
	Milestones     []MilestoneInput
}

type DocumentInput struct {
	Filename string
	URL      string
	Size     int64
	MimeType string
}

// MilestoneInput is the frontend-facing payload for a single milestone
// in a multi-step proposal. Sequences must be consecutive starting at 1
// (enforced by milestone.NewMilestoneBatch).
type MilestoneInput struct {
	Sequence    int
	Title       string
	Description string
	Amount      int64
	Deadline    *time.Time
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
