package dispute

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	disputedomain "marketplace-backend/internal/domain/dispute"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// mockDisputeRepo
// ---------------------------------------------------------------------------

type mockDisputeRepo struct {
	createFn           func(ctx context.Context, d *disputedomain.Dispute) error
	getByIDFn          func(ctx context.Context, id uuid.UUID) (*disputedomain.Dispute, error)
	getByProposalIDFn  func(ctx context.Context, proposalID uuid.UUID) (*disputedomain.Dispute, error)
	updateFn           func(ctx context.Context, d *disputedomain.Dispute) error
	listByOrganizationFn func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*disputedomain.Dispute, string, error)
	listPendingFn      func(ctx context.Context) ([]*disputedomain.Dispute, error)
	listAllFn          func(ctx context.Context, cursor string, limit int, statusFilter string) ([]*disputedomain.Dispute, string, error)
	createEvidenceFn   func(ctx context.Context, e *disputedomain.Evidence) error
	listEvidenceFn     func(ctx context.Context, disputeID uuid.UUID) ([]*disputedomain.Evidence, error)
	createCPFn         func(ctx context.Context, cp *disputedomain.CounterProposal) error
	getCPByIDFn        func(ctx context.Context, id uuid.UUID) (*disputedomain.CounterProposal, error)
	updateCPFn         func(ctx context.Context, cp *disputedomain.CounterProposal) error
	listCPsFn          func(ctx context.Context, disputeID uuid.UUID) ([]*disputedomain.CounterProposal, error)
	supersedeAllFn     func(ctx context.Context, disputeID uuid.UUID) error
	createChatMsgFn    func(ctx context.Context, m *disputedomain.ChatMessage) error
	listChatMsgsFn     func(ctx context.Context, disputeID uuid.UUID) ([]*disputedomain.ChatMessage, error)
	countByUserIDFn    func(ctx context.Context, userID uuid.UUID) (int, error)
	countAllFn         func(ctx context.Context) (int, int, int, error)
}

func (m *mockDisputeRepo) Create(ctx context.Context, d *disputedomain.Dispute) error {
	if m.createFn != nil {
		return m.createFn(ctx, d)
	}
	return nil
}
func (m *mockDisputeRepo) GetByID(ctx context.Context, id uuid.UUID) (*disputedomain.Dispute, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, disputedomain.ErrDisputeNotFound
}
func (m *mockDisputeRepo) GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*disputedomain.Dispute, error) {
	if m.getByProposalIDFn != nil {
		return m.getByProposalIDFn(ctx, proposalID)
	}
	return nil, nil
}
func (m *mockDisputeRepo) Update(ctx context.Context, d *disputedomain.Dispute) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, d)
	}
	return nil
}
func (m *mockDisputeRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*disputedomain.Dispute, string, error) {
	if m.listByOrganizationFn != nil {
		return m.listByOrganizationFn(ctx, orgID, cursor, limit)
	}
	return nil, "", nil
}
func (m *mockDisputeRepo) ListPendingForScheduler(ctx context.Context) ([]*disputedomain.Dispute, error) {
	if m.listPendingFn != nil {
		return m.listPendingFn(ctx)
	}
	return nil, nil
}
func (m *mockDisputeRepo) ListAll(ctx context.Context, cursor string, limit int, statusFilter string) ([]*disputedomain.Dispute, string, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, cursor, limit, statusFilter)
	}
	return nil, "", nil
}
func (m *mockDisputeRepo) CreateEvidence(ctx context.Context, e *disputedomain.Evidence) error {
	if m.createEvidenceFn != nil {
		return m.createEvidenceFn(ctx, e)
	}
	return nil
}
func (m *mockDisputeRepo) ListEvidence(ctx context.Context, disputeID uuid.UUID) ([]*disputedomain.Evidence, error) {
	if m.listEvidenceFn != nil {
		return m.listEvidenceFn(ctx, disputeID)
	}
	return []*disputedomain.Evidence{}, nil
}
func (m *mockDisputeRepo) CreateCounterProposal(ctx context.Context, cp *disputedomain.CounterProposal) error {
	if m.createCPFn != nil {
		return m.createCPFn(ctx, cp)
	}
	return nil
}
func (m *mockDisputeRepo) GetCounterProposalByID(ctx context.Context, id uuid.UUID) (*disputedomain.CounterProposal, error) {
	if m.getCPByIDFn != nil {
		return m.getCPByIDFn(ctx, id)
	}
	return nil, disputedomain.ErrCounterProposalNotFound
}
func (m *mockDisputeRepo) UpdateCounterProposal(ctx context.Context, cp *disputedomain.CounterProposal) error {
	if m.updateCPFn != nil {
		return m.updateCPFn(ctx, cp)
	}
	return nil
}
func (m *mockDisputeRepo) ListCounterProposals(ctx context.Context, disputeID uuid.UUID) ([]*disputedomain.CounterProposal, error) {
	if m.listCPsFn != nil {
		return m.listCPsFn(ctx, disputeID)
	}
	return []*disputedomain.CounterProposal{}, nil
}
func (m *mockDisputeRepo) SupersedeAllPending(ctx context.Context, disputeID uuid.UUID) error {
	if m.supersedeAllFn != nil {
		return m.supersedeAllFn(ctx, disputeID)
	}
	return nil
}
func (m *mockDisputeRepo) CreateChatMessage(ctx context.Context, msg *disputedomain.ChatMessage) error {
	if m.createChatMsgFn != nil {
		return m.createChatMsgFn(ctx, msg)
	}
	return nil
}
func (m *mockDisputeRepo) ListChatMessages(ctx context.Context, disputeID uuid.UUID) ([]*disputedomain.ChatMessage, error) {
	if m.listChatMsgsFn != nil {
		return m.listChatMsgsFn(ctx, disputeID)
	}
	return []*disputedomain.ChatMessage{}, nil
}
func (m *mockDisputeRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.countByUserIDFn != nil {
		return m.countByUserIDFn(ctx, userID)
	}
	return 0, nil
}
func (m *mockDisputeRepo) CountAll(ctx context.Context) (int, int, int, error) {
	if m.countAllFn != nil {
		return m.countAllFn(ctx)
	}
	return 0, 0, 0, nil
}

// ---------------------------------------------------------------------------
// mockProposalRepo (minimal — only methods used by dispute service)
// ---------------------------------------------------------------------------

type mockProposalRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error)
	updateFn  func(ctx context.Context, p *proposal.Proposal) error
}

func (m *mockProposalRepo) Create(context.Context, *proposal.Proposal) error               { return nil }
func (m *mockProposalRepo) CreateWithDocuments(context.Context, *proposal.Proposal, []*proposal.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) CreateWithDocumentsAndMilestones(context.Context, *proposal.Proposal, []*proposal.ProposalDocument, []*milestonedomain.Milestone) error {
	return nil
}
func (m *mockProposalRepo) GetByID(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, proposal.ErrProposalNotFound
}
func (m *mockProposalRepo) Update(ctx context.Context, p *proposal.Proposal) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}
func (m *mockProposalRepo) GetLatestVersion(context.Context, uuid.UUID) (*proposal.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListByConversation(context.Context, uuid.UUID) ([]*proposal.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListActiveProjectsByOrganization(context.Context, uuid.UUID, string, int) ([]*proposal.Proposal, string, error) {
	return nil, "", nil
}
func (m *mockProposalRepo) ListCompletedByOrganization(context.Context, uuid.UUID, string, int) ([]*proposal.Proposal, string, error) {
	return nil, "", nil
}
func (m *mockProposalRepo) GetDocuments(context.Context, uuid.UUID) ([]*proposal.ProposalDocument, error) {
	return nil, nil
}
func (m *mockProposalRepo) CreateDocument(context.Context, *proposal.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) IsOrgAuthorizedForProposal(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockProposalRepo) CountAll(context.Context) (int, int, error) { return 0, 0, nil }

// ---------------------------------------------------------------------------
// mockUserRepo (minimal)
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
}

func (m *mockUserRepo) Create(context.Context, *user.User) error { return nil }
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	// Every user in tests has a stub personal org so the OpenDispute
	// flow (which resolves both parties' orgs) can proceed.
	stubOrgID := uuid.New()
	return &user.User{ID: id, DisplayName: "Test User", OrganizationID: &stubOrgID}, nil
}
func (m *mockUserRepo) GetByEmail(context.Context, string) (*user.User, error) { return nil, nil }
func (m *mockUserRepo) Update(context.Context, *user.User) error               { return nil }
func (m *mockUserRepo) Delete(context.Context, uuid.UUID) error                { return nil }
func (m *mockUserRepo) ExistsByEmail(context.Context, string) (bool, error)    { return false, nil }
func (m *mockUserRepo) ListAdmin(context.Context, repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (m *mockUserRepo) CountAdmin(context.Context, repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) CountByRole(context.Context) (map[string]int, error)      { return nil, nil }
func (m *mockUserRepo) CountByStatus(context.Context) (map[string]int, error)    { return nil, nil }
func (m *mockUserRepo) RecentSignups(context.Context, int) ([]*user.User, error) { return nil, nil }
func (m *mockUserRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockUserRepo) FindUserIDByStripeAccount(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockUserRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error { return nil }
func (m *mockUserRepo) ClearStripeAccount(context.Context, uuid.UUID) error               { return nil }
func (m *mockUserRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error)     { return nil, nil }
func (m *mockUserRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error      { return nil }
func (m *mockUserRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error    { return nil }
func (m *mockUserRepo) GetKYCPendingUsers(context.Context) ([]*user.User, error)          { return nil, nil }
func (m *mockUserRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}

// ---------------------------------------------------------------------------
// mockMessageSender
// ---------------------------------------------------------------------------

type mockMessageSender struct {
	lastInput *service.SystemMessageInput
}

func (m *mockMessageSender) SendSystemMessage(_ context.Context, input service.SystemMessageInput) error {
	m.lastInput = &input
	return nil
}
func (m *mockMessageSender) FindOrCreateConversation(_ context.Context, _ service.FindOrCreateConversationInput) (uuid.UUID, error) {
	return uuid.New(), nil
}

// ---------------------------------------------------------------------------
// mockNotificationSender
// ---------------------------------------------------------------------------

type mockNotificationSender struct {
	sent []service.NotificationInput
}

func (m *mockNotificationSender) Send(_ context.Context, input service.NotificationInput) error {
	m.sent = append(m.sent, input)
	return nil
}

// ---------------------------------------------------------------------------
// mockPaymentProcessor
// ---------------------------------------------------------------------------

type mockPaymentProcessor struct {
	transferCalled bool
}

func (m *mockPaymentProcessor) CreatePaymentIntent(context.Context, service.PaymentIntentInput) (*service.PaymentIntentOutput, error) {
	return nil, nil
}
func (m *mockPaymentProcessor) TransferToProvider(_ context.Context, _ uuid.UUID) error {
	m.transferCalled = true
	return nil
}
func (m *mockPaymentProcessor) TransferMilestone(_ context.Context, _ uuid.UUID) error {
	m.transferCalled = true
	return nil
}
func (m *mockPaymentProcessor) HandlePaymentSucceeded(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockPaymentProcessor) TransferPartialToProvider(_ context.Context, _ uuid.UUID, _ int64) error {
	m.transferCalled = true
	return nil
}
func (m *mockPaymentProcessor) RefundToClient(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}

// ---------------------------------------------------------------------------
// mockAIAnalyzer
// ---------------------------------------------------------------------------

type mockAIAnalyzer struct{}

func (m *mockAIAnalyzer) AnalyzeDispute(_ context.Context, _ service.DisputeAnalysisInput, _ int) (string, service.AIUsage, error) {
	return "Mock AI analysis: test summary", service.AIUsage{InputTokens: 1000, OutputTokens: 200}, nil
}

func (m *mockAIAnalyzer) ChatAboutDispute(_ context.Context, _ service.DisputeAnalysisInput, _ []service.ChatTurn, _ string, _ int) (string, service.AIUsage, error) {
	return "Mock AI chat answer", service.AIUsage{InputTokens: 800, OutputTokens: 150}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService() (*Service, *mockDisputeRepo, *mockProposalRepo, *mockMessageSender, *mockNotificationSender, *mockPaymentProcessor) {
	dr := &mockDisputeRepo{}
	pr := &mockProposalRepo{}
	mr := &mockMilestoneRepo{}
	ur := &mockUserRepo{}
	ms := &mockMessageSender{}
	ns := &mockNotificationSender{}
	pp := &mockPaymentProcessor{}
	ai := &mockAIAnalyzer{}

	svc := NewService(ServiceDeps{
		Disputes:      dr,
		Proposals:     pr,
		Milestones:    mr,
		Users:         ur,
		Messages:      ms,
		Notifications: ns,
		Payments:      pp,
		AI:            ai,
	})

	return svc, dr, pr, ms, ns, pp
}

// mockMilestoneRepo is a minimal stub satisfying the milestone
// repository port. It always returns a synthetic submitted milestone
// matching the requested proposal id so the dispute happy-path tests
// (which only exercise the proposal-level flow) keep passing without
// per-test seeding.
type mockMilestoneRepo struct{}

func (m *mockMilestoneRepo) CreateBatch(_ context.Context, _ []*milestonedomain.Milestone) error {
	return nil
}

func (m *mockMilestoneRepo) GetByID(_ context.Context, id uuid.UUID) (*milestonedomain.Milestone, error) {
	return synthDisputeMilestone(id), nil
}

func (m *mockMilestoneRepo) GetByIDForUpdate(_ context.Context, id uuid.UUID) (*milestonedomain.Milestone, error) {
	return synthDisputeMilestone(id), nil
}

func (m *mockMilestoneRepo) ListByProposal(_ context.Context, _ uuid.UUID) ([]*milestonedomain.Milestone, error) {
	return nil, nil
}

func (m *mockMilestoneRepo) GetCurrentActive(_ context.Context, proposalID uuid.UUID) (*milestonedomain.Milestone, error) {
	return synthDisputeMilestoneForProposal(proposalID), nil
}

func (m *mockMilestoneRepo) Update(_ context.Context, _ *milestonedomain.Milestone) error {
	return nil
}

func (m *mockMilestoneRepo) CreateDeliverable(_ context.Context, _ *milestonedomain.Deliverable) error {
	return nil
}

func (m *mockMilestoneRepo) ListDeliverables(_ context.Context, _ uuid.UUID) ([]*milestonedomain.Deliverable, error) {
	return nil, nil
}

func (m *mockMilestoneRepo) DeleteDeliverable(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockMilestoneRepo) ListByProposals(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]*milestonedomain.Milestone, error) {
	return map[uuid.UUID][]*milestonedomain.Milestone{}, nil
}

// synthDisputeMilestone returns a freshly-built submitted milestone
// at sequence=1 with a deterministic amount large enough to satisfy
// every existing dispute test's RequestedAmount value (most tests
// use small constants well under 100000 cents).
func synthDisputeMilestone(id uuid.UUID) *milestonedomain.Milestone {
	now := time.Now()
	return &milestonedomain.Milestone{
		ID:          id,
		Sequence:    1,
		Title:       "Synthetic milestone",
		Description: "test fixture",
		Amount:      1000000,
		Status:      milestonedomain.StatusSubmitted,
		FundedAt:    &now,
		SubmittedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func synthDisputeMilestoneForProposal(proposalID uuid.UUID) *milestonedomain.Milestone {
	m := synthDisputeMilestone(uuid.New())
	m.ProposalID = proposalID
	return m
}

func makeActiveProposal(clientID, providerID uuid.UUID) *proposal.Proposal {
	return &proposal.Proposal{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		SenderID:       clientID,
		RecipientID:    providerID,
		ClientID:       clientID,
		ProviderID:     providerID,
		Title:          "Test Proposal",
		Description:    "Test description",
		Amount:         100000,
		Status:         proposal.StatusActive,
		Version:        1,
		Metadata:       json.RawMessage("{}"),
	}
}

// Suppress unused import warnings
var (
	_ = time.Now
	_ = repository.AdminUserFilters{}
)

// --- Session version stubs (migration 056, Phase 3) ---
func (m *mockUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error {
	return nil
}
