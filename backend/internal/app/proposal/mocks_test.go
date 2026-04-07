package proposal

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"

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
	listActiveProjectsFn func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*domain.Proposal, string, error)
	getDocumentsFn       func(ctx context.Context, proposalID uuid.UUID) ([]*domain.ProposalDocument, error)
	createDocumentFn     func(ctx context.Context, doc *domain.ProposalDocument) error
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

func (m *mockProposalRepo) ListActiveProjects(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*domain.Proposal, string, error) {
	if m.listActiveProjectsFn != nil {
		return m.listActiveProjectsFn(ctx, userID, cursor, limit)
	}
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

func (m *mockProposalRepo) CountAll(_ context.Context) (int, int, error) {
	return 0, 0, nil
}

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

type mockJobCreditRepo struct {
	getOrCreateFn  func(ctx context.Context, userID uuid.UUID) (int, error)
	decrementFn    func(ctx context.Context, userID uuid.UUID) error
	addBonusFn     func(ctx context.Context, userID uuid.UUID, amount int, maxTokens int) error
	resetForUserFn func(ctx context.Context, userID uuid.UUID, minCredits int) error
	resetWeeklyFn  func(ctx context.Context, minCredits int) error

	addBonusCalls []addBonusCall
}

type addBonusCall struct {
	UserID    uuid.UUID
	Amount    int
	MaxTokens int
}

func (m *mockJobCreditRepo) GetOrCreate(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, userID)
	}
	return 10, nil
}

func (m *mockJobCreditRepo) Decrement(ctx context.Context, userID uuid.UUID) error {
	if m.decrementFn != nil {
		return m.decrementFn(ctx, userID)
	}
	return nil
}

func (m *mockJobCreditRepo) AddBonus(ctx context.Context, userID uuid.UUID, amount int, maxTokens int) error {
	m.addBonusCalls = append(m.addBonusCalls, addBonusCall{UserID: userID, Amount: amount, MaxTokens: maxTokens})
	if m.addBonusFn != nil {
		return m.addBonusFn(ctx, userID, amount, maxTokens)
	}
	return nil
}

func (m *mockJobCreditRepo) ResetForUser(ctx context.Context, userID uuid.UUID, minCredits int) error {
	if m.resetForUserFn != nil {
		return m.resetForUserFn(ctx, userID, minCredits)
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
