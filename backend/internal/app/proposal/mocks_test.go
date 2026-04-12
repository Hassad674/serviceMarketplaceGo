package proposal

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// --- mockProposalRepo ---

type mockProposalRepo struct {
	createFn             func(ctx context.Context, p *domain.Proposal) error
	createWithDocsFn     func(ctx context.Context, p *domain.Proposal, docs []*domain.ProposalDocument) error
	getByIDFn            func(ctx context.Context, id uuid.UUID) (*domain.Proposal, error)
	updateFn             func(ctx context.Context, p *domain.Proposal) error
	getLatestVersionFn   func(ctx context.Context, rootProposalID uuid.UUID) (*domain.Proposal, error)
	listByConversationFn func(ctx context.Context, conversationID uuid.UUID) ([]*domain.Proposal, error)
	listActiveProjectsFn func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Proposal, string, error)
	getDocumentsFn       func(ctx context.Context, proposalID uuid.UUID) ([]*domain.ProposalDocument, error)
	createDocumentFn     func(ctx context.Context, doc *domain.ProposalDocument) error
	isOrgAuthorizedFn    func(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error)
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

func (m *mockProposalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Proposal, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrProposalNotFound
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

var _ repository.ProposalRepository = (*mockProposalRepo)(nil)

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

var _ repository.OrganizationRepository = (*mockOrgRepo)(nil)

// --- mockUserRepo ---

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
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
