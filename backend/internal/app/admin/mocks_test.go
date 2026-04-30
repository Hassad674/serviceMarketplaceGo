package admin

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// --- mockUserRepo ---
//
// Records BumpSessionVersion and Update calls for the SEC-05 / SEC-13
// admin tests. Only the methods exercised by the suspend/ban/unban
// flow are expressive — the rest return zero values to satisfy the
// repository.UserRepository contract.

var _ repository.UserRepository = (*mockUserRepo)(nil)

type mockUserRepo struct {
	mu sync.Mutex

	getByIDFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
	updateFn  func(ctx context.Context, u *user.User) error

	bumpCalls    []uuid.UUID
	bumpResult   int
	bumpErr      error
	updateCalls  []*user.User
}

func (m *mockUserRepo) Create(_ context.Context, _ *user.User) error { return nil }

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, u *user.User) error {
	m.mu.Lock()
	m.updateCalls = append(m.updateCalls, u)
	m.mu.Unlock()
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) { return false, nil }

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

func (m *mockUserRepo) BumpSessionVersion(_ context.Context, userID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bumpCalls = append(m.bumpCalls, userID)
	return m.bumpResult, m.bumpErr
}

func (m *mockUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}

func (m *mockUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error { return nil }

// snapshotBumpCalls returns a copy of the userIDs BumpSessionVersion
// was called for, in invocation order.
func (m *mockUserRepo) snapshotBumpCalls() []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]uuid.UUID, len(m.bumpCalls))
	copy(out, m.bumpCalls)
	return out
}

// snapshotUpdateCalls returns a copy of the User pointers Update was
// called with, in invocation order.
func (m *mockUserRepo) snapshotUpdateCalls() []*user.User {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*user.User, len(m.updateCalls))
	copy(out, m.updateCalls)
	return out
}

// --- mockAuditRepo ---

var _ repository.AuditRepository = (*mockAuditRepo)(nil)

type mockAuditRepo struct {
	mu      sync.Mutex
	entries []*audit.Entry
}

func (m *mockAuditRepo) Log(_ context.Context, entry *audit.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditRepo) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

func (m *mockAuditRepo) ListByUser(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

func (m *mockAuditRepo) snapshot() []*audit.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*audit.Entry, len(m.entries))
	copy(out, m.entries)
	return out
}

// --- mockSessionService ---

var _ portservice.SessionService = (*mockSessionService)(nil)

type mockSessionService struct {
	mu                sync.Mutex
	deleteByUserCalls []uuid.UUID
}

func (m *mockSessionService) Create(_ context.Context, _ portservice.CreateSessionInput) (*portservice.Session, error) {
	return &portservice.Session{ID: "session-id"}, nil
}
func (m *mockSessionService) Get(_ context.Context, _ string) (*portservice.Session, error) {
	return nil, nil
}
func (m *mockSessionService) Delete(_ context.Context, _ string) error { return nil }
func (m *mockSessionService) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteByUserCalls = append(m.deleteByUserCalls, userID)
	return nil
}
func (m *mockSessionService) CreateWSToken(_ context.Context, _ uuid.UUID) (string, error) {
	return "ws", nil
}
func (m *mockSessionService) ValidateWSToken(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockSessionService) snapshotDeleteCalls() []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]uuid.UUID, len(m.deleteByUserCalls))
	copy(out, m.deleteByUserCalls)
	return out
}

// --- mockBroadcaster ---

var _ portservice.MessageBroadcaster = (*mockBroadcaster)(nil)

type mockBroadcaster struct {
	mu              sync.Mutex
	suspensionCalls []uuid.UUID
}

func (m *mockBroadcaster) BroadcastNewMessage(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastTyping(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastStatusUpdate(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastUnreadCount(_ context.Context, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockBroadcaster) BroadcastPresence(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastNotification(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastMessageEdited(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastMessageDeleted(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastAccountSuspended(_ context.Context, userID uuid.UUID, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suspensionCalls = append(m.suspensionCalls, userID)
	return nil
}
func (m *mockBroadcaster) BroadcastAdminNotification(_ context.Context, _ []uuid.UUID) error {
	return nil
}
