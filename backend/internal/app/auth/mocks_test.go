package auth

import (
	"context"

	"github.com/google/uuid"

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
	generateAccessFn   func(userID uuid.UUID, role string, isAdmin bool) (string, error)
	generateRefreshFn  func(userID uuid.UUID) (string, error)
	validateAccessFn   func(token string) (*service.TokenClaims, error)
	validateRefreshFn  func(token string) (*service.TokenClaims, error)
}

func (m *mockTokenService) GenerateAccessToken(userID uuid.UUID, role string, isAdmin bool) (string, error) {
	if m.generateAccessFn != nil {
		return m.generateAccessFn(userID, role, isAdmin)
	}
	return "access_token_" + userID.String(), nil
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
