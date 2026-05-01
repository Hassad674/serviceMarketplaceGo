package proposal

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// --- mockProposalRepo ---

type mockProposalRepo struct {
	createFn                 func(ctx context.Context, p *domain.Proposal) error
	createWithDocsFn         func(ctx context.Context, p *domain.Proposal, docs []*domain.ProposalDocument) error
	createWithDocsAndMilesFn func(ctx context.Context, p *domain.Proposal, docs []*domain.ProposalDocument, milestones []*milestone.Milestone) error
	getByIDFn                func(ctx context.Context, id uuid.UUID) (*domain.Proposal, error)
	updateFn                 func(ctx context.Context, p *domain.Proposal) error
	getLatestVersionFn       func(ctx context.Context, rootProposalID uuid.UUID) (*domain.Proposal, error)
	listByConversationFn     func(ctx context.Context, conversationID uuid.UUID) ([]*domain.Proposal, error)
	listActiveProjectsFn     func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Proposal, string, error)
	getDocumentsFn           func(ctx context.Context, proposalID uuid.UUID) ([]*domain.ProposalDocument, error)
	createDocumentFn         func(ctx context.Context, doc *domain.ProposalDocument) error
	isOrgAuthorizedFn        func(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error)
}

func (m *mockProposalRepo) Create(ctx context.Context, p *domain.Proposal) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil
}

func (m *mockProposalRepo) CreateWithDocuments(ctx context.Context, p *domain.Proposal, docs []*domain.ProposalDocument) error {
	if m.createWithDocsFn != nil {
		return m.createWithDocsFn(ctx, p, docs)
	}
	return nil
}

func (m *mockProposalRepo) CreateWithDocumentsAndMilestones(ctx context.Context, p *domain.Proposal, docs []*domain.ProposalDocument, milestones []*milestone.Milestone) error {
	if m.createWithDocsAndMilesFn != nil {
		return m.createWithDocsAndMilesFn(ctx, p, docs, milestones)
	}
	// Default behaviour in tests: delegate to createWithDocsFn so
	// existing tests that only stub CreateWithDocuments keep passing.
	if m.createWithDocsFn != nil {
		return m.createWithDocsFn(ctx, p, docs)
	}
	return nil
}

func (m *mockProposalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Proposal, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrProposalNotFound
}

func (m *mockProposalRepo) GetByIDs(context.Context, []uuid.UUID) ([]*domain.Proposal, error) {
	return nil, nil
}

func (m *mockProposalRepo) Update(ctx context.Context, p *domain.Proposal) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

func (m *mockProposalRepo) GetLatestVersion(ctx context.Context, rootProposalID uuid.UUID) (*domain.Proposal, error) {
	if m.getLatestVersionFn != nil {
		return m.getLatestVersionFn(ctx, rootProposalID)
	}
	return nil, domain.ErrProposalNotFound
}

func (m *mockProposalRepo) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*domain.Proposal, error) {
	if m.listByConversationFn != nil {
		return m.listByConversationFn(ctx, conversationID)
	}
	return nil, nil
}

func (m *mockProposalRepo) ListActiveProjectsByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Proposal, string, error) {
	if m.listActiveProjectsFn != nil {
		return m.listActiveProjectsFn(ctx, orgID, cursor, limit)
	}
	return []*domain.Proposal{}, "", nil
}

func (m *mockProposalRepo) ListCompletedByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Proposal, string, error) {
	return []*domain.Proposal{}, "", nil
}

func (m *mockProposalRepo) GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*domain.ProposalDocument, error) {
	if m.getDocumentsFn != nil {
		return m.getDocumentsFn(ctx, proposalID)
	}
	return []*domain.ProposalDocument{}, nil
}

func (m *mockProposalRepo) CreateDocument(ctx context.Context, doc *domain.ProposalDocument) error {
	if m.createDocumentFn != nil {
		return m.createDocumentFn(ctx, doc)
	}
	return nil
}

// IsOrgAuthorizedForProposal mirrors the real adapter method used to
// gate GetProposal reads at org granularity. Default behaviour when
// no stub is set: deny — tests that exercise org auth MUST set the
// callback explicitly, so that a forgotten stub is surfaced as an
// ErrNotAuthorized rather than silently passing.
func (m *mockProposalRepo) IsOrgAuthorizedForProposal(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error) {
	if m.isOrgAuthorizedFn != nil {
		return m.isOrgAuthorizedFn(ctx, proposalID, orgID)
	}
	return false, nil
}

func (m *mockProposalRepo) CountAll(_ context.Context) (int, int, error) {
	return 0, 0, nil
}

func (m *mockProposalRepo) SumPaidByClientOrganization(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockProposalRepo) ListCompletedByClientOrganization(_ context.Context, _ uuid.UUID, _ int) ([]*domain.Proposal, error) {
	return nil, nil
}

var _ repository.ProposalRepository = (*mockProposalRepo)(nil)

// --- mockMilestoneRepo ---
//
// Hand-written stub of repository.MilestoneRepository following the
// same field-functions pattern as mockProposalRepo. The in-memory
// store backs the happy-path defaults so tests that only care about
// "there is a current milestone" can omit all the stub functions.

type mockMilestoneRepo struct {
	createBatchFn       func(ctx context.Context, milestones []*milestone.Milestone) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error)
	getByIDWithVersionFn  func(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error)
	listByProposalFn    func(ctx context.Context, proposalID uuid.UUID) ([]*milestone.Milestone, error)
	getCurrentActiveFn  func(ctx context.Context, proposalID uuid.UUID) (*milestone.Milestone, error)
	updateFn            func(ctx context.Context, m *milestone.Milestone) error
	createDeliverableFn func(ctx context.Context, d *milestone.Deliverable) error
	listDeliverablesFn  func(ctx context.Context, milestoneID uuid.UUID) ([]*milestone.Deliverable, error)
	deleteDeliverableFn func(ctx context.Context, id uuid.UUID) error
	listByProposalsFn   func(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error)

	// store is the in-memory backing map used when no stub function
	// is set, indexed by milestone id. Tests that want to rely on
	// default behaviour can populate this map to seed state.
	store map[uuid.UUID]*milestone.Milestone
	// byProposal keeps the proposal_id -> milestones[] index in sync
	// with store so ListByProposal and GetCurrentActive can walk it.
	byProposal map[uuid.UUID][]*milestone.Milestone

	// autoSynthStatus, when non-empty, makes GetCurrentActive and
	// ListByProposal lazily synthesise a single milestone in the
	// given status for any proposal id that has never been seeded.
	// This is a pragmatic shortcut for action-method tests that
	// previously only stubbed the proposal side and assumed the
	// macro status mapped directly onto the proposal. Tests that
	// need a specific milestone shape still call seedMilestone.
	autoSynthStatus milestone.MilestoneStatus
	autoSynthAmount int64
}

func (m *mockMilestoneRepo) init() {
	if m.store == nil {
		m.store = make(map[uuid.UUID]*milestone.Milestone)
	}
	if m.byProposal == nil {
		m.byProposal = make(map[uuid.UUID][]*milestone.Milestone)
	}
}

func (m *mockMilestoneRepo) CreateBatch(ctx context.Context, milestones []*milestone.Milestone) error {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, milestones)
	}
	m.init()
	for _, mm := range milestones {
		m.store[mm.ID] = mm
		m.byProposal[mm.ProposalID] = append(m.byProposal[mm.ProposalID], mm)
	}
	return nil
}

func (m *mockMilestoneRepo) GetByID(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	m.init()
	if mm, ok := m.store[id]; ok {
		return mm, nil
	}
	return nil, milestone.ErrMilestoneNotFound
}

func (m *mockMilestoneRepo) GetByIDWithVersion(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error) {
	if m.getByIDWithVersionFn != nil {
		return m.getByIDWithVersionFn(ctx, id)
	}
	return m.GetByID(ctx, id)
}

func (m *mockMilestoneRepo) ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*milestone.Milestone, error) {
	if m.listByProposalFn != nil {
		return m.listByProposalFn(ctx, proposalID)
	}
	m.init()
	existing := m.byProposal[proposalID]
	if len(existing) == 0 && m.autoSynthStatus != "" {
		amount := m.autoSynthAmount
		if amount == 0 {
			amount = 100000
		}
		mm := m.seedMilestone(proposalID, m.autoSynthStatus, amount)
		return []*milestone.Milestone{mm}, nil
	}
	return existing, nil
}

func (m *mockMilestoneRepo) GetCurrentActive(ctx context.Context, proposalID uuid.UUID) (*milestone.Milestone, error) {
	if m.getCurrentActiveFn != nil {
		return m.getCurrentActiveFn(ctx, proposalID)
	}
	m.init()
	candidates := m.byProposal[proposalID]
	var current *milestone.Milestone
	for _, mm := range candidates {
		if mm.IsTerminal() {
			continue
		}
		if current == nil || mm.Sequence < current.Sequence {
			current = mm
		}
	}
	if current != nil {
		return current, nil
	}
	// Auto-synthesis fallback: when the test has set autoSynthStatus,
	// lazily create a single milestone in that status and remember
	// it so subsequent calls see the same instance. Default amount
	// is 100000 centimes (1000 EUR) unless overridden.
	if m.autoSynthStatus != "" {
		amount := m.autoSynthAmount
		if amount == 0 {
			amount = 100000
		}
		return m.seedMilestone(proposalID, m.autoSynthStatus, amount), nil
	}
	return nil, milestone.ErrMilestoneNotFound
}

func (m *mockMilestoneRepo) enableAutoSynth(status milestone.MilestoneStatus, amount int64) {
	m.autoSynthStatus = status
	m.autoSynthAmount = amount
}

func (m *mockMilestoneRepo) Update(ctx context.Context, mm *milestone.Milestone) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, mm)
	}
	m.init()
	mm.Version++
	m.store[mm.ID] = mm
	return nil
}

func (m *mockMilestoneRepo) CreateDeliverable(ctx context.Context, d *milestone.Deliverable) error {
	if m.createDeliverableFn != nil {
		return m.createDeliverableFn(ctx, d)
	}
	return nil
}

func (m *mockMilestoneRepo) ListDeliverables(ctx context.Context, milestoneID uuid.UUID) ([]*milestone.Deliverable, error) {
	if m.listDeliverablesFn != nil {
		return m.listDeliverablesFn(ctx, milestoneID)
	}
	return nil, nil
}

func (m *mockMilestoneRepo) DeleteDeliverable(ctx context.Context, id uuid.UUID) error {
	if m.deleteDeliverableFn != nil {
		return m.deleteDeliverableFn(ctx, id)
	}
	return nil
}

func (m *mockMilestoneRepo) ListByProposals(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error) {
	if m.listByProposalsFn != nil {
		return m.listByProposalsFn(ctx, proposalIDs)
	}
	m.init()
	out := make(map[uuid.UUID][]*milestone.Milestone)
	for _, id := range proposalIDs {
		if list, ok := m.byProposal[id]; ok {
			out[id] = list
		}
	}
	return out, nil
}

// seedMilestone is a test helper that injects a single milestone at
// sequence=1 into the mock repository's in-memory store. Used by
// action-method tests to simulate the post-CreateProposal state
// (where exactly one milestone exists in pending_funding or later).
//
// The status argument lets each test express the precondition it
// needs: pending_funding for InitiatePayment/ConfirmPayment, funded
// for RequestCompletion, submitted for CompleteProposal and
// RejectCompletion. The milestone carries the supplied amount and
// a fixed title/description so debug output is readable.
func (m *mockMilestoneRepo) seedMilestone(proposalID uuid.UUID, status milestone.MilestoneStatus, amount int64) *milestone.Milestone {
	m.init()
	mm := &milestone.Milestone{
		ID:          uuid.New(),
		ProposalID:  proposalID,
		Sequence:    1,
		Title:       "Seeded milestone",
		Description: "Seeded for action-method tests",
		Amount:      amount,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	// Populate timestamps consistent with the status so the macro
	// status projection stays coherent.
	now := time.Now()
	switch status {
	case milestone.StatusFunded:
		mm.FundedAt = &now
	case milestone.StatusSubmitted:
		mm.FundedAt = &now
		mm.SubmittedAt = &now
	case milestone.StatusApproved:
		mm.FundedAt = &now
		mm.SubmittedAt = &now
		mm.ApprovedAt = &now
	case milestone.StatusReleased:
		mm.FundedAt = &now
		mm.SubmittedAt = &now
		mm.ApprovedAt = &now
		mm.ReleasedAt = &now
	}
	m.store[mm.ID] = mm
	m.byProposal[proposalID] = append(m.byProposal[proposalID], mm)
	return mm
}

var _ repository.MilestoneRepository = (*mockMilestoneRepo)(nil)

// --- mockOrgRepo (KYC-aware stub) ---

type mockOrgRepo struct {
	findByUserIDFn func(ctx context.Context, userID uuid.UUID) (*organization.Organization, error)
}

func (m *mockOrgRepo) Create(context.Context, *organization.Organization) error { return nil }
func (m *mockOrgRepo) CreateWithOwnerMembership(context.Context, *organization.Organization, *organization.Member) error {
	return nil
}
func (m *mockOrgRepo) FindByID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepo) FindByOwnerUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*organization.Organization, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	// Deterministic default: the org's ID equals the user id so tests
	// asserting that "the bonus landed on providerID" keep working
	// unchanged after R12 (credits are keyed by org id — but in this
	// default stub every user IS their own org).
	return &organization.Organization{ID: userID, Type: organization.OrgTypeProviderPersonal}, nil
}
func (m *mockOrgRepo) Update(context.Context, *organization.Organization) error { return nil }
func (m *mockOrgRepo) Delete(context.Context, uuid.UUID) error                  { return nil }
func (m *mockOrgRepo) CountAll(context.Context) (int, error)                    { return 0, nil }
func (m *mockOrgRepo) FindByStripeAccountID(context.Context, string) (*organization.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepo) ListKYCPending(context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockOrgRepo) GetStripeAccountByUserID(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockOrgRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (m *mockOrgRepo) ClearStripeAccount(context.Context, uuid.UUID) error { return nil }
func (m *mockOrgRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockOrgRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error { return nil }
func (m *mockOrgRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (m *mockOrgRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}
func (m *mockOrgRepo) SaveRoleOverrides(context.Context, uuid.UUID, organization.RoleOverrides) error {
	return nil
}
func (m *mockOrgRepo) ListWithStripeAccount(context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

var _ repository.OrganizationRepository = (*mockOrgRepo)(nil)

// --- mockUserRepo ---

type mockUserRepo struct {
	getByIDFn     func(ctx context.Context, id uuid.UUID) (*user.User, error)
	getByIDsFn    func(ctx context.Context, ids []uuid.UUID) ([]*user.User, error)
	// getByIDsCalls counts batch calls for PERF-B-02 N+1 regression
	// tests so we can assert the page hits the DB once (not 2*N times).
	getByIDsCalls int
}

func (m *mockUserRepo) Create(ctx context.Context, u *user.User) error      { return nil }
func (m *mockUserRepo) Update(ctx context.Context, u *user.User) error      { return nil }
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &user.User{ID: id, Role: user.RoleEnterprise, DisplayName: "Test User"}, nil
}

// GetByIDs satisfies UserBatchReader so the same fixture can be wired
// to UsersBatch (PERF-B-02). The default implementation reuses the
// per-id stub to keep behaviour aligned with GetByID, and increments
// getByIDsCalls so tests can assert the batch is invoked exactly once.
func (m *mockUserRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*user.User, error) {
	m.getByIDsCalls++
	if m.getByIDsFn != nil {
		return m.getByIDsFn(ctx, ids)
	}
	out := make([]*user.User, 0, len(ids))
	for _, id := range ids {
		u, err := m.GetByID(ctx, id)
		if err != nil {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

func (m *mockUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}

func (m *mockUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}

func (m *mockUserRepo) CountByRole(_ context.Context) (map[string]int, error) {
	return map[string]int{}, nil
}

func (m *mockUserRepo) CountByStatus(_ context.Context) (map[string]int, error) {
	return map[string]int{}, nil
}

func (m *mockUserRepo) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return nil, nil
}

// --- mockMessageSender ---

type mockMessageSender struct {
	sendSystemMessageFn func(ctx context.Context, input service.SystemMessageInput) error
	calls               []service.SystemMessageInput
}

func (m *mockMessageSender) SendSystemMessage(ctx context.Context, input service.SystemMessageInput) error {
	m.calls = append(m.calls, input)
	if m.sendSystemMessageFn != nil {
		return m.sendSystemMessageFn(ctx, input)
	}
	return nil
}

func (m *mockMessageSender) FindOrCreateConversation(_ context.Context, _ service.FindOrCreateConversationInput) (uuid.UUID, error) {
	return uuid.New(), nil
}

// --- mockStorageService ---

type mockStorageService struct {
	uploadFn             func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	deleteFn             func(ctx context.Context, key string) error
	getPublicURLFn       func(key string) string
	getPresignedUploadFn func(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error)
}

func (m *mockStorageService) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, reader, contentType, size)
	}
	return "https://storage.example.com/" + key, nil
}

func (m *mockStorageService) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func (m *mockStorageService) GetPublicURL(key string) string {
	if m.getPublicURLFn != nil {
		return m.getPublicURLFn(key)
	}
	return "https://storage.example.com/" + key
}

func (m *mockStorageService) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	if m.getPresignedUploadFn != nil {
		return m.getPresignedUploadFn(ctx, key, contentType, expiry)
	}
	return "https://storage.example.com/presigned/" + key, nil
}

func (m *mockStorageService) GetPresignedDownloadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://storage.example.com/download/" + key, nil
}

func (m *mockStorageService) GetPresignedDownloadURLAsAttachment(_ context.Context, key string, _ string, _ time.Duration) (string, error) {
	return "https://storage.example.com/download/" + key, nil
}

func (m *mockStorageService) Download(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

// --- mockNotificationSender ---

type mockNotificationSender struct {
	sendFn func(ctx context.Context, input service.NotificationInput) error
	calls  []service.NotificationInput
}

func (m *mockNotificationSender) Send(ctx context.Context, input service.NotificationInput) error {
	m.calls = append(m.calls, input)
	if m.sendFn != nil {
		return m.sendFn(ctx, input)
	}
	return nil
}

// --- mockJobCreditRepo ---
//
// R12 — the repository now takes org ids. Existing proposal tests
// assert on `addBonusCall.UserID` — we keep the field name for minimal
// diff, but it now carries an org id under the hood. Callers that
// match on a specific value must be updated (done via stubs so the
// provider->org resolution returns a deterministic id).

type mockJobCreditRepo struct {
	getOrCreateFn  func(ctx context.Context, orgID uuid.UUID) (int, error)
	decrementFn    func(ctx context.Context, orgID uuid.UUID) error
	refundFn       func(ctx context.Context, orgID uuid.UUID) error
	addBonusFn     func(ctx context.Context, orgID uuid.UUID, amount int, maxTokens int) error
	resetForOrgFn  func(ctx context.Context, orgID uuid.UUID, minCredits int) error
	resetWeeklyFn  func(ctx context.Context, minCredits int) error

	addBonusCalls []addBonusCall
}

// addBonusCall captures a single AddBonus invocation. The `UserID`
// field is a legacy name — after R12 it actually holds the ORG id.
// Keeping the name to avoid churning every call site in one go; the
// tests that need to assert on a specific provider's org set the
// mockOrgRepo stub so the resolved org id is predictable.
type addBonusCall struct {
	UserID    uuid.UUID
	Amount    int
	MaxTokens int
}

func (m *mockJobCreditRepo) GetOrCreate(ctx context.Context, orgID uuid.UUID) (int, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, orgID)
	}
	return 10, nil
}

func (m *mockJobCreditRepo) Decrement(ctx context.Context, orgID uuid.UUID) error {
	if m.decrementFn != nil {
		return m.decrementFn(ctx, orgID)
	}
	return nil
}

func (m *mockJobCreditRepo) Refund(ctx context.Context, orgID uuid.UUID) error {
	if m.refundFn != nil {
		return m.refundFn(ctx, orgID)
	}
	return nil
}

func (m *mockJobCreditRepo) AddBonus(ctx context.Context, orgID uuid.UUID, amount int, maxTokens int) error {
	m.addBonusCalls = append(m.addBonusCalls, addBonusCall{UserID: orgID, Amount: amount, MaxTokens: maxTokens})
	if m.addBonusFn != nil {
		return m.addBonusFn(ctx, orgID, amount, maxTokens)
	}
	return nil
}

func (m *mockJobCreditRepo) ResetForOrg(ctx context.Context, orgID uuid.UUID, minCredits int) error {
	if m.resetForOrgFn != nil {
		return m.resetForOrgFn(ctx, orgID, minCredits)
	}
	return nil
}

func (m *mockJobCreditRepo) ResetWeekly(ctx context.Context, minCredits int) error {
	if m.resetWeeklyFn != nil {
		return m.resetWeeklyFn(ctx, minCredits)
	}
	return nil
}

// suppress unused import warning
var _ = json.RawMessage{}

// --- Stripe account stubs (migration 040) ---
func (m *mockUserRepo) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockUserRepo) FindUserIDByStripeAccount(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockUserRepo) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockUserRepo) ClearStripeAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockUserRepo) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockUserRepo) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}

// --- KYC enforcement stubs (migration 044) ---
func (m *mockUserRepo) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockUserRepo) GetKYCPendingUsers(_ context.Context) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepo) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}

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

// --- mockPaymentProcessor ---
//
// Stub of service.PaymentProcessor used by tests that need to exercise
// the proposal milestone-release path with a non-nil payments dependency
// (e.g. the provider-KYC pre-check). Methods that mutate state record
// the call so the test can assert "no Stripe transfer happened" alongside
// "no DB write happened".
type mockPaymentProcessor struct {
	canProviderReceiveFn   func(ctx context.Context, providerOrgID uuid.UUID) (bool, error)
	hasAutoPayoutConsentFn func() (bool, error)
	transferMilestoneCalls int
	transferProposalCalls  int
}

func (m *mockPaymentProcessor) CreatePaymentIntent(context.Context, service.PaymentIntentInput) (*service.PaymentIntentOutput, error) {
	return nil, nil
}
func (m *mockPaymentProcessor) TransferToProvider(_ context.Context, _ uuid.UUID) error {
	m.transferProposalCalls++
	return nil
}
func (m *mockPaymentProcessor) TransferMilestone(_ context.Context, _ uuid.UUID) error {
	m.transferMilestoneCalls++
	return nil
}
func (m *mockPaymentProcessor) HandlePaymentSucceeded(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockPaymentProcessor) TransferPartialToProvider(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}
func (m *mockPaymentProcessor) RefundToClient(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}
func (m *mockPaymentProcessor) CanProviderReceivePayouts(ctx context.Context, providerOrgID uuid.UUID) (bool, error) {
	if m.canProviderReceiveFn != nil {
		return m.canProviderReceiveFn(ctx, providerOrgID)
	}
	return true, nil
}
func (m *mockPaymentProcessor) HasAutoPayoutConsent(_ context.Context, _ uuid.UUID) (bool, error) {
	if m.hasAutoPayoutConsentFn != nil {
		return m.hasAutoPayoutConsentFn()
	}
	return false, nil
}

var _ service.PaymentProcessor = (*mockPaymentProcessor)(nil)
