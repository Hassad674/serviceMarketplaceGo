package proposal

import (
	"time"

	"github.com/google/uuid"

	appmoderation "marketplace-backend/internal/app/moderation"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type ServiceDeps struct {
	// Proposals stays on the wide ProposalRepository — the proposal
	// service straddles all three segregated children (Reader for
	// GetByID/GetByIDForOrg/GetDocuments/IsOrgAuthorizedForProposal/
	// ListActiveProjectsByOrganization, Writer for
	// CreateWithDocuments/Update, MilestoneStore for
	// CreateWithDocumentsAndMilestones). Composing locally would
	// reproduce the wide port verbatim.
	Proposals           repository.ProposalRepository
	Milestones          repository.MilestoneRepository          // required since phase 4 — every proposal has ≥1 milestone
	MilestoneTransitions repository.MilestoneTransitionRepository // optional since phase 9 — when nil, audit writes are no-ops
	PendingEvents       repository.PendingEventRepository        // optional since phase 6 — when nil, scheduling is a no-op
	// Users is narrowed to UserReader — every proposal flow only
	// resolves users by id (GetByID) for naming, KYC and side
	// resolution. Persistence happens in dedicated mutation services.
	Users               repository.UserReader
	// UsersBatch is the bulk-fetch sibling consumed by
	// GetParticipantNamesBatch (PERF-B-02). Optional — when nil, the
	// list path falls back to per-id lookups so legacy test setups still
	// work. In production wiring the concrete *postgres.UserRepository
	// satisfies both Users (UserRepository) and UsersBatch
	// (UserBatchReader), so the same instance is passed twice.
	UsersBatch repository.UserBatchReader
	// Organizations is narrowed to proposalOrgs — proposal flows read
	// the provider's org for KYC gating and stamp the first-earning
	// timestamp on the StripeStore.
	Organizations proposalOrgs
	Messages      service.MessageSender
	Storage             service.StorageService
	Notifications       service.NotificationSender
	Payments            service.PaymentProcessor            // nil if Stripe not configured
	Credits             repository.JobCreditRepository      // nil if credits not configured
	BonusLog            repository.CreditBonusLogRepository // nil if not configured

	// Phase 6 timers (zero values use sensible defaults).
	AutoApprovalDelay time.Duration // default 7 days
	FundReminderDelay time.Duration // default 7 days
	AutoCloseDelay    time.Duration // default 14 days
}

// proposalOrgs is the local composite the proposal service needs:
// FindByUserID (Reader) for KYC gating on the create / action paths
// and SetKYCFirstEarning (StripeStore) for the milestone-funding
// success path. No segregated child covers both, so we compose locally.
type proposalOrgs interface {
	repository.OrganizationReader
	repository.OrganizationStripeStore
}

type Service struct {
	proposals            repository.ProposalRepository
	milestones           repository.MilestoneRepository
	milestoneTransitions repository.MilestoneTransitionRepository
	pendingEvents        repository.PendingEventRepository
	users                repository.UserReader
	usersBatch           repository.UserBatchReader
	orgs                 proposalOrgs
	messages             service.MessageSender
	storage              service.StorageService
	notifications        service.NotificationSender
	payments             service.PaymentProcessor
	credits              repository.JobCreditRepository
	bonusLog             repository.CreditBonusLogRepository

	// Referral hook — wired post-construction via SetReferralAttributor to
	// break the import cycle. Nil when the referral feature is not active.
	referralAttributor service.ReferralAttributor

	// moderationOrchestrator runs an async scan on the proposal title +
	// description after a successful create. Optional: when nil, the
	// proposal goes through unchecked (legacy behaviour).
	moderationOrchestrator *appmoderation.Service

	autoApprovalDelay time.Duration
	fundReminderDelay time.Duration
	autoCloseDelay    time.Duration
}

// SetReferralAttributor plugs the referral attributor in post-construction.
// Called on a CreateProposal success path to attribute the new proposal to
// any active referral covering the (provider, client) couple.
func (s *Service) SetReferralAttributor(a service.ReferralAttributor) {
	s.referralAttributor = a
}

// SetModerationOrchestrator wires the central moderation pipeline.
// Optional: when nil, proposal title/description moderation is
// skipped. Same setter pattern as messaging + review for parity.
func (s *Service) SetModerationOrchestrator(svc *appmoderation.Service) {
	s.moderationOrchestrator = svc
}

func NewService(deps ServiceDeps) *Service {
	autoApproval := deps.AutoApprovalDelay
	if autoApproval <= 0 {
		autoApproval = 7 * 24 * time.Hour
	}
	fundReminder := deps.FundReminderDelay
	if fundReminder <= 0 {
		fundReminder = 7 * 24 * time.Hour
	}
	autoClose := deps.AutoCloseDelay
	if autoClose <= 0 {
		autoClose = 14 * 24 * time.Hour
	}
	return &Service{
		proposals:            deps.Proposals,
		milestones:           deps.Milestones,
		milestoneTransitions: deps.MilestoneTransitions,
		pendingEvents:        deps.PendingEvents,
		users:                deps.Users,
		usersBatch:           deps.UsersBatch,
		orgs:                 deps.Organizations,
		messages:             deps.Messages,
		storage:              deps.Storage,
		notifications:        deps.Notifications,
		payments:             deps.Payments,
		credits:              deps.Credits,
		bonusLog:             deps.BonusLog,
		autoApprovalDelay:    autoApproval,
		fundReminderDelay:    fundReminder,
		autoCloseDelay:       autoClose,
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
