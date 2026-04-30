package auth

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ensure mockUserRepo implements the full interface
var _ repository.UserRepository = (*mockUserRepo)(nil)

// --- mockUserRepo ---

type mockUserRepo struct {
	createFn        func(ctx context.Context, u *user.User) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*user.User, error)
	getByEmailFn    func(ctx context.Context, email string) (*user.User, error)
	updateFn        func(ctx context.Context, u *user.User) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
	existsByEmailFn func(ctx context.Context, email string) (bool, error)
	listAdminFn     func(ctx context.Context, filters repository.AdminUserFilters) ([]*user.User, string, error)
	countAdminFn    func(ctx context.Context, filters repository.AdminUserFilters) (int, error)
}

func (m *mockUserRepo) Create(ctx context.Context, u *user.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, u *user.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}

func (m *mockUserRepo) ListAdmin(ctx context.Context, filters repository.AdminUserFilters) ([]*user.User, string, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, filters)
	}
	return []*user.User{}, "", nil
}

func (m *mockUserRepo) CountAdmin(ctx context.Context, filters repository.AdminUserFilters) (int, error) {
	if m.countAdminFn != nil {
		return m.countAdminFn(ctx, filters)
	}
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

// --- mockPasswordResetRepo ---

type mockPasswordResetRepo struct {
	createFn        func(ctx context.Context, pr *repository.PasswordReset) error
	getByTokenFn    func(ctx context.Context, token string) (*repository.PasswordReset, error)
	markUsedFn      func(ctx context.Context, id uuid.UUID) error
	deleteExpiredFn func(ctx context.Context) error
}

func (m *mockPasswordResetRepo) Create(ctx context.Context, pr *repository.PasswordReset) error {
	if m.createFn != nil {
		return m.createFn(ctx, pr)
	}
	return nil
}

func (m *mockPasswordResetRepo) GetByToken(ctx context.Context, token string) (*repository.PasswordReset, error) {
	if m.getByTokenFn != nil {
		return m.getByTokenFn(ctx, token)
	}
	return nil, user.ErrUnauthorized
}

func (m *mockPasswordResetRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(ctx, id)
	}
	return nil
}

func (m *mockPasswordResetRepo) DeleteExpired(ctx context.Context) error {
	if m.deleteExpiredFn != nil {
		return m.deleteExpiredFn(ctx)
	}
	return nil
}

// --- mockHasher ---

type mockHasher struct {
	hashFn    func(password string) (string, error)
	compareFn func(hashed, password string) error
}

func (m *mockHasher) Hash(password string) (string, error) {
	if m.hashFn != nil {
		return m.hashFn(password)
	}
	return "hashed_" + password, nil
}

func (m *mockHasher) Compare(hashed, password string) error {
	if m.compareFn != nil {
		return m.compareFn(hashed, password)
	}
	if hashed == "hashed_"+password {
		return nil
	}
	return user.ErrInvalidCredentials
}

// --- mockTokenService ---

type mockTokenService struct {
	generateAccessFn   func(input service.AccessTokenInput) (string, error)
	generateRefreshFn  func(userID uuid.UUID) (string, error)
	validateAccessFn   func(token string) (*service.TokenClaims, error)
	validateRefreshFn  func(token string) (*service.TokenClaims, error)
}

func (m *mockTokenService) GenerateAccessToken(input service.AccessTokenInput) (string, error) {
	if m.generateAccessFn != nil {
		return m.generateAccessFn(input)
	}
	return "access_token_" + input.UserID.String(), nil
}

func (m *mockTokenService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	if m.generateRefreshFn != nil {
		return m.generateRefreshFn(userID)
	}
	return "refresh_token_" + userID.String(), nil
}

func (m *mockTokenService) ValidateAccessToken(token string) (*service.TokenClaims, error) {
	if m.validateAccessFn != nil {
		return m.validateAccessFn(token)
	}
	return nil, user.ErrUnauthorized
}

func (m *mockTokenService) ValidateRefreshToken(token string) (*service.TokenClaims, error) {
	if m.validateRefreshFn != nil {
		return m.validateRefreshFn(token)
	}
	return nil, user.ErrUnauthorized
}

// --- mockRefreshBlacklist ---

// mockRefreshBlacklist is an in-memory implementation of
// service.RefreshBlacklistService used by the auth service tests. It
// tracks Add calls + TTLs so the test can assert "blacklist was
// invoked exactly once after a successful refresh" without relying on
// Redis or fake clocks.
type mockRefreshBlacklist struct {
	mu      sync.Mutex
	entries map[string]time.Duration
	hasErr  error
	addErr  error
}

func newMockRefreshBlacklist() *mockRefreshBlacklist {
	return &mockRefreshBlacklist{entries: make(map[string]time.Duration)}
}

var _ service.RefreshBlacklistService = (*mockRefreshBlacklist)(nil)

func (m *mockRefreshBlacklist) Add(_ context.Context, jti string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.addErr != nil {
		return m.addErr
	}
	if jti == "" || ttl <= 0 {
		return nil
	}
	m.entries[jti] = ttl
	return nil
}

func (m *mockRefreshBlacklist) Has(_ context.Context, jti string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hasErr != nil {
		return false, m.hasErr
	}
	if jti == "" {
		return false, nil
	}
	_, ok := m.entries[jti]
	return ok, nil
}

func (m *mockRefreshBlacklist) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

// --- mockAuditRepo ---

// mockAuditRepo records every Log call in memory so tests can assert
// the action + resource + metadata of audit emissions without touching
// Postgres.
type mockAuditRepo struct {
	mu      sync.Mutex
	entries []*audit.Entry
	logErr  error
}

func newMockAuditRepo() *mockAuditRepo {
	return &mockAuditRepo{}
}

var _ repository.AuditRepository = (*mockAuditRepo)(nil)

func (m *mockAuditRepo) Log(_ context.Context, entry *audit.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logErr != nil {
		return m.logErr
	}
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditRepo) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

func (m *mockAuditRepo) ListByUser(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

func (m *mockAuditRepo) Snapshot() []*audit.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*audit.Entry, len(m.entries))
	copy(out, m.entries)
	return out
}

// --- mockEmailService ---

type mockEmailService struct {
	sendPasswordResetFn func(ctx context.Context, to string, resetURL string) error
}

func (m *mockEmailService) SendPasswordReset(ctx context.Context, to string, resetURL string) error {
	if m.sendPasswordResetFn != nil {
		return m.sendPasswordResetFn(ctx, to, resetURL)
	}
	return nil
}

func (m *mockEmailService) SendNotification(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockEmailService) SendTeamInvitation(_ context.Context, _ service.TeamInvitationEmailInput) error {
	return nil
}
func (m *mockEmailService) SendRolePermissionsChanged(_ context.Context, _ service.RolePermissionsChangedEmailInput) error {
	return nil
}

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
