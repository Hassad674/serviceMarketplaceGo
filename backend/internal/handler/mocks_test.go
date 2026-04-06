package handler

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

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
	generateAccessFn  func(userID uuid.UUID, role string, isAdmin bool) (string, error)
	generateRefreshFn func(userID uuid.UUID) (string, error)
	validateAccessFn  func(token string) (*service.TokenClaims, error)
	validateRefreshFn func(token string) (*service.TokenClaims, error)
}

func (m *mockTokenService) GenerateAccessToken(userID uuid.UUID, role string, isAdmin bool) (string, error) {
	if m.generateAccessFn != nil {
		return m.generateAccessFn(userID, role, isAdmin)
	}
	return "access_" + userID.String(), nil
}

func (m *mockTokenService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	if m.generateRefreshFn != nil {
		return m.generateRefreshFn(userID)
	}
	return "refresh_" + userID.String(), nil
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

// --- mockSessionService ---

type mockSessionService struct {
	createFn          func(ctx context.Context, userID uuid.UUID, role string, isAdmin bool) (*service.Session, error)
	getFn             func(ctx context.Context, sessionID string) (*service.Session, error)
	deleteFn          func(ctx context.Context, sessionID string) error
	createWSTokenFn   func(ctx context.Context, userID uuid.UUID) (string, error)
	validateWSTokenFn func(ctx context.Context, token string) (uuid.UUID, error)
}

func (m *mockSessionService) Create(ctx context.Context, userID uuid.UUID, role string, isAdmin bool) (*service.Session, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, role, isAdmin)
	}
	return &service.Session{ID: "session_123", UserID: userID, Role: role, IsAdmin: isAdmin}, nil
}

func (m *mockSessionService) Get(ctx context.Context, sessionID string) (*service.Session, error) {
	if m.getFn != nil {
		return m.getFn(ctx, sessionID)
	}
	return nil, user.ErrUnauthorized
}

func (m *mockSessionService) Delete(ctx context.Context, sessionID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, sessionID)
	}
	return nil
}

func (m *mockSessionService) CreateWSToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.createWSTokenFn != nil {
		return m.createWSTokenFn(ctx, userID)
	}
	return "ws_token_123", nil
}

func (m *mockSessionService) ValidateWSToken(ctx context.Context, token string) (uuid.UUID, error) {
	if m.validateWSTokenFn != nil {
		return m.validateWSTokenFn(ctx, token)
	}
	return uuid.Nil, user.ErrUnauthorized
}

// --- mockEmailService ---

type mockEmailService struct {
	sendPasswordResetFn func(ctx context.Context, to, resetURL string) error
}

func (m *mockEmailService) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	if m.sendPasswordResetFn != nil {
		return m.sendPasswordResetFn(ctx, to, resetURL)
	}
	return nil
}

func (m *mockEmailService) SendNotification(_ context.Context, _, _, _ string) error {
	return nil
}

// --- mockProfileRepo ---

type mockProfileRepo struct {
	createFn       func(ctx context.Context, p *profile.Profile) error
	getByUserIDFn  func(ctx context.Context, userID uuid.UUID) (*profile.Profile, error)
	updateFn       func(ctx context.Context, p *profile.Profile) error
	searchPublicFn func(ctx context.Context, roleFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error)
}

func (m *mockProfileRepo) Create(ctx context.Context, p *profile.Profile) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*profile.Profile, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return nil, profile.ErrProfileNotFound
}

func (m *mockProfileRepo) Update(ctx context.Context, p *profile.Profile) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) SearchPublic(ctx context.Context, roleFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error) {
	if m.searchPublicFn != nil {
		return m.searchPublicFn(ctx, roleFilter, referrerOnly, cursor, limit)
	}
	return []*profile.PublicProfile{}, "", nil
}

func (m *mockProfileRepo) GetPublicProfilesByUserIDs(_ context.Context, _ []uuid.UUID) ([]*profile.PublicProfile, error) {
	return []*profile.PublicProfile{}, nil
}

// --- mockJobRepo ---

type mockJobRepo struct {
	createFn        func(ctx context.Context, j *job.Job) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*job.Job, error)
	updateFn        func(ctx context.Context, j *job.Job) error
	listByCreatorFn func(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*job.Job, string, error)
}

func (m *mockJobRepo) Create(ctx context.Context, j *job.Job) error {
	if m.createFn != nil {
		return m.createFn(ctx, j)
	}
	return nil
}

func (m *mockJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*job.Job, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, job.ErrJobNotFound
}

func (m *mockJobRepo) Update(ctx context.Context, j *job.Job) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, j)
	}
	return nil
}

func (m *mockJobRepo) ListByCreator(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*job.Job, string, error) {
	if m.listByCreatorFn != nil {
		return m.listByCreatorFn(ctx, creatorID, cursor, limit)
	}
	return []*job.Job{}, "", nil
}

func (m *mockJobRepo) ListOpen(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*job.Job, string, error) {
	return []*job.Job{}, "", nil
}

func (m *mockJobRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// --- mockStorageService ---

type mockStorageService struct {
	uploadFn              func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	deleteFn              func(ctx context.Context, key string) error
	getPublicURLFn        func(key string) string
	getPresignedUploadFn  func(ctx context.Context, key, contentType string, expiry time.Duration) (string, error)
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

func (m *mockStorageService) GetPresignedUploadURL(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	if m.getPresignedUploadFn != nil {
		return m.getPresignedUploadFn(ctx, key, contentType, expiry)
	}
	return "https://storage.example.com/upload/" + key, nil
}

func (m *mockStorageService) Download(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

// --- helpers ---

func testUser(id uuid.UUID, role user.Role) *user.User {
	return &user.User{
		ID:          id,
		Email:       "test@example.com",
		FirstName:   "Test",
		LastName:    "User",
		DisplayName: "Test User",
		Role:        role,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func testProfile(userID uuid.UUID) *profile.Profile {
	return &profile.Profile{
		UserID:    userID,
		Title:     "Software Developer",
		About:     "Hello world",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func testJob(creatorID uuid.UUID) *job.Job {
	return &job.Job{
		ID:              uuid.New(),
		CreatorID:       creatorID,
		Title:           "Need a Go Developer",
		Description:     "Build a REST API",
		Skills:          []string{"go", "postgresql"},
		ApplicantType:   job.ApplicantAll,
		BudgetType:      job.BudgetOneShot,
		MinBudget:       1000,
		MaxBudget:       5000,
		Status:          job.StatusOpen,
		DescriptionType: job.DescriptionText,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func testCookieConfig() *CookieConfig {
	return &CookieConfig{
		Secure:   false,
		Domain:   "localhost",
		MaxAge:   3600,
		SameSite: 0,
	}
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
