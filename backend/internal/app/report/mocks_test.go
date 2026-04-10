package report

import (
	"context"
	"time"

	"github.com/google/uuid"

	messagedomain "marketplace-backend/internal/domain/message"
	domain "marketplace-backend/internal/domain/report"
	userdomain "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// --- mockReportRepo ---

type mockReportRepo struct {
	createFn           func(ctx context.Context, r *domain.Report) error
	getByIDFn          func(ctx context.Context, id uuid.UUID) (*domain.Report, error)
	listByStatusFn     func(ctx context.Context, status string, cursor string, limit int) ([]*domain.Report, string, error)
	listByReporterFn   func(ctx context.Context, reporterID uuid.UUID, cursor string, limit int) ([]*domain.Report, string, error)
	listByTargetFn     func(ctx context.Context, targetType string, targetID uuid.UUID) ([]*domain.Report, error)
	updateStatusFn     func(ctx context.Context, id uuid.UUID, status string, adminNote string, resolvedBy uuid.UUID) error
	hasPendingReportFn func(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error)
}

func (m *mockReportRepo) Create(ctx context.Context, r *domain.Report) error {
	if m.createFn != nil {
		return m.createFn(ctx, r)
	}
	return nil
}

func (m *mockReportRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockReportRepo) ListByStatus(ctx context.Context, status string, cursor string, limit int) ([]*domain.Report, string, error) {
	if m.listByStatusFn != nil {
		return m.listByStatusFn(ctx, status, cursor, limit)
	}
	return nil, "", nil
}

func (m *mockReportRepo) ListByReporter(ctx context.Context, reporterID uuid.UUID, cursor string, limit int) ([]*domain.Report, string, error) {
	if m.listByReporterFn != nil {
		return m.listByReporterFn(ctx, reporterID, cursor, limit)
	}
	return nil, "", nil
}

func (m *mockReportRepo) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]*domain.Report, error) {
	if m.listByTargetFn != nil {
		return m.listByTargetFn(ctx, targetType, targetID)
	}
	return nil, nil
}

func (m *mockReportRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, adminNote string, resolvedBy uuid.UUID) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, adminNote, resolvedBy)
	}
	return nil
}

func (m *mockReportRepo) HasPendingReport(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error) {
	if m.hasPendingReportFn != nil {
		return m.hasPendingReportFn(ctx, reporterID, targetType, targetID)
	}
	return false, nil
}

func (m *mockReportRepo) ListByConversation(_ context.Context, _ uuid.UUID) ([]*domain.Report, error) {
	return nil, nil
}

func (m *mockReportRepo) ListByUserInvolved(_ context.Context, _ uuid.UUID) ([]*domain.Report, []*domain.Report, error) {
	return nil, nil, nil
}

func (m *mockReportRepo) PendingCountsByTargets(_ context.Context, _ string, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return map[uuid.UUID]int{}, nil
}

// Compile-time check.
var _ repository.ReportRepository = (*mockReportRepo)(nil)

// --- mockUserRepo ---

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*userdomain.User, error)
}

func (m *mockUserRepo) Create(_ context.Context, _ *userdomain.User) error { return nil }
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &userdomain.User{ID: id}, nil
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*userdomain.User, error) {
	return nil, userdomain.ErrUserNotFound
}
func (m *mockUserRepo) Update(_ context.Context, _ *userdomain.User) error  { return nil }
func (m *mockUserRepo) Delete(_ context.Context, _ uuid.UUID) error         { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*userdomain.User, string, error) {
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

func (m *mockUserRepo) RecentSignups(_ context.Context, _ int) ([]*userdomain.User, error) {
	return nil, nil
}

// Compile-time check.
var _ repository.UserRepository = (*mockUserRepo)(nil)

// --- mockMessageRepo ---

type mockMessageRepo struct {
	getMessageFn func(ctx context.Context, id uuid.UUID) (*messagedomain.Message, error)
}

func (m *mockMessageRepo) FindOrCreateConversation(_ context.Context, _, _ uuid.UUID) (uuid.UUID, bool, error) {
	return uuid.Nil, false, nil
}
func (m *mockMessageRepo) GetConversation(_ context.Context, _ uuid.UUID) (*messagedomain.Conversation, error) {
	return nil, nil
}
func (m *mockMessageRepo) ListConversations(_ context.Context, _ repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
	return nil, "", nil
}
func (m *mockMessageRepo) IsParticipant(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockMessageRepo) CreateMessage(_ context.Context, _ *messagedomain.Message) error {
	return nil
}
func (m *mockMessageRepo) GetMessage(ctx context.Context, id uuid.UUID) (*messagedomain.Message, error) {
	if m.getMessageFn != nil {
		return m.getMessageFn(ctx, id)
	}
	now := time.Now()
	return &messagedomain.Message{ID: id, CreatedAt: now}, nil
}
func (m *mockMessageRepo) ListMessages(_ context.Context, _ repository.ListMessagesParams) ([]*messagedomain.Message, string, error) {
	return nil, "", nil
}
func (m *mockMessageRepo) GetMessagesSinceSeq(_ context.Context, _ uuid.UUID, _ int, _ int) ([]*messagedomain.Message, error) {
	return nil, nil
}
func (m *mockMessageRepo) ListMessagesSinceTime(_ context.Context, _ uuid.UUID, _ time.Time, _ int) ([]*messagedomain.Message, error) {
	return nil, nil
}
func (m *mockMessageRepo) UpdateMessage(_ context.Context, _ *messagedomain.Message) error {
	return nil
}
func (m *mockMessageRepo) IncrementUnread(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockMessageRepo) MarkAsRead(_ context.Context, _, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockMessageRepo) GetTotalUnread(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockMessageRepo) GetTotalUnreadBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return nil, nil
}
func (m *mockMessageRepo) GetParticipantIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockMessageRepo) UpdateMessageStatus(_ context.Context, _ uuid.UUID, _ messagedomain.MessageStatus) error {
	return nil
}
func (m *mockMessageRepo) MarkMessagesAsRead(_ context.Context, _, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockMessageRepo) GetContactIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockMessageRepo) SaveMessageHistory(_ context.Context, _, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockMessageRepo) UpdateMessageModeration(_ context.Context, _ uuid.UUID, _ string, _ float64, _ []byte) error {
	return nil
}

// Compile-time check.
var _ repository.MessageRepository = (*mockMessageRepo)(nil)

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
func (m *mockUserRepo) GetKYCPendingUsers(_ context.Context) ([]*userdomain.User, error) {
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
