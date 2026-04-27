package handler

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/organization"
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
	// Default: return a stub user whose organization id is identical
	// to the user's own id. This matches the convention used by
	// proposalCtx (test helper) which stores orgID := userID in the
	// request context — so when the R14 proposal service resolves a
	// side's org and compares it to the caller's orgID, both sides
	// look like the same uuid and the directional check passes.
	// Tests that want a mismatch override getByIDFn explicitly.
	stubOrg := id
	return &user.User{ID: id, OrganizationID: &stubOrg}, nil
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
	generateAccessFn  func(input service.AccessTokenInput) (string, error)
	generateRefreshFn func(userID uuid.UUID) (string, error)
	validateAccessFn  func(token string) (*service.TokenClaims, error)
	validateRefreshFn func(token string) (*service.TokenClaims, error)
}

func (m *mockTokenService) GenerateAccessToken(input service.AccessTokenInput) (string, error) {
	if m.generateAccessFn != nil {
		return m.generateAccessFn(input)
	}
	return "access_" + input.UserID.String(), nil
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
	createFn          func(ctx context.Context, input service.CreateSessionInput) (*service.Session, error)
	getFn             func(ctx context.Context, sessionID string) (*service.Session, error)
	deleteFn          func(ctx context.Context, sessionID string) error
	createWSTokenFn   func(ctx context.Context, userID uuid.UUID) (string, error)
	validateWSTokenFn func(ctx context.Context, token string) (uuid.UUID, error)
}

func (m *mockSessionService) Create(ctx context.Context, input service.CreateSessionInput) (*service.Session, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	return &service.Session{
		ID:             "session_123",
		UserID:         input.UserID,
		Role:           input.Role,
		IsAdmin:        input.IsAdmin,
		OrganizationID: input.OrganizationID,
		OrgRole:        input.OrgRole,
	}, nil
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

func (m *mockSessionService) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
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

func (m *mockEmailService) SendTeamInvitation(_ context.Context, _ service.TeamInvitationEmailInput) error {
	return nil
}
func (m *mockEmailService) SendRolePermissionsChanged(_ context.Context, _ service.RolePermissionsChangedEmailInput) error {
	return nil
}

// --- mockProfileRepo ---

type mockProfileRepo struct {
	createFn                  func(ctx context.Context, p *profile.Profile) error
	getByOrgIDFn              func(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error)
	updateFn                  func(ctx context.Context, p *profile.Profile) error
	searchPublicFn            func(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error)
	updateClientDescriptionFn func(ctx context.Context, orgID uuid.UUID, clientDescription string) error
}

func (m *mockProfileRepo) Create(ctx context.Context, p *profile.Profile) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) GetByOrganizationID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	if m.getByOrgIDFn != nil {
		return m.getByOrgIDFn(ctx, orgID)
	}
	return nil, profile.ErrProfileNotFound
}

func (m *mockProfileRepo) Update(ctx context.Context, p *profile.Profile) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) SearchPublic(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error) {
	if m.searchPublicFn != nil {
		return m.searchPublicFn(ctx, orgTypeFilter, referrerOnly, cursor, limit)
	}
	return []*profile.PublicProfile{}, "", nil
}

func (m *mockProfileRepo) GetPublicProfilesByOrgIDs(_ context.Context, _ []uuid.UUID) ([]*profile.PublicProfile, error) {
	return []*profile.PublicProfile{}, nil
}

func (m *mockProfileRepo) OrgProfilesByUserIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
	return map[uuid.UUID]*profile.PublicProfile{}, nil
}

// Tier 1 completion stubs — handler unit tests do not exercise
// these paths but must satisfy the interface surface.
func (m *mockProfileRepo) UpdateLocation(_ context.Context, _ uuid.UUID, _ repository.LocationInput) error {
	return nil
}
func (m *mockProfileRepo) UpdateLanguages(_ context.Context, _ uuid.UUID, _, _ []string) error {
	return nil
}
func (m *mockProfileRepo) UpdateAvailability(_ context.Context, _ uuid.UUID, _ *profile.AvailabilityStatus, _ *profile.AvailabilityStatus) error {
	return nil
}

func (m *mockProfileRepo) UpdateClientDescription(ctx context.Context, orgID uuid.UUID, clientDescription string) error {
	if m.updateClientDescriptionFn != nil {
		return m.updateClientDescriptionFn(ctx, orgID, clientDescription)
	}
	return nil
}

// --- mockJobRepo ---

type mockJobRepo struct {
	createFn        func(ctx context.Context, j *job.Job) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*job.Job, error)
	updateFn        func(ctx context.Context, j *job.Job) error
	listByOrgFn     func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*job.Job, string, error)
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

func (m *mockJobRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*job.Job, string, error) {
	if m.listByOrgFn != nil {
		return m.listByOrgFn(ctx, orgID, cursor, limit)
	}
	return []*job.Job{}, "", nil
}

func (m *mockJobRepo) ListOpen(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*job.Job, string, error) {
	return []*job.Job{}, "", nil
}

func (m *mockJobRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockJobRepo) ListAdmin(_ context.Context, _ repository.AdminJobFilters) ([]repository.AdminJob, string, error) {
	return nil, "", nil
}

func (m *mockJobRepo) CountAdmin(_ context.Context, _ repository.AdminJobFilters) (int, error) {
	return 0, nil
}

func (m *mockJobRepo) GetAdmin(_ context.Context, _ uuid.UUID) (*repository.AdminJob, error) {
	return nil, nil
}

func (m *mockJobRepo) CountAll(_ context.Context) (int, int, error) {
	return 0, 0, nil
}

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

func (m *mockStorageService) GetPresignedDownloadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://storage.example.com/download/" + key, nil
}

func (m *mockStorageService) GetPresignedDownloadURLAsAttachment(_ context.Context, key string, filename string, _ time.Duration) (string, error) {
	return "https://storage.example.com/download/" + key + "?filename=" + filename, nil
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

func testProfile(orgID uuid.UUID) *profile.Profile {
	return &profile.Profile{
		OrganizationID: orgID,
		Title:          "Software Developer",
		About:          "Hello world",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
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

// --- Session version stubs (migration 056, Phase 3) ---
func (m *mockUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// --- Email notifications enabled (migration 076) ---
func (m *mockUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}

// --- last_active_at bump (migration 110) ---
func (m *mockUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error {
	return nil
}

// --- mockOrgRepo ---
//
// Shared minimal stub of repository.OrganizationRepository for handler
// tests. FindByID is the only hook most tests need — it resolves the
// org's owner user id, which StartConversation then uses as the real
// recipient user.

type mockOrgRepo struct {
	findByIDFn     func(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
	findByUserIDFn func(ctx context.Context, userID uuid.UUID) (*organization.Organization, error)
}

func (m *mockOrgRepo) Create(context.Context, *organization.Organization) error { return nil }
func (m *mockOrgRepo) CreateWithOwnerMembership(context.Context, *organization.Organization, *organization.Member) error {
	return nil
}
func (m *mockOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	// Default: treat the org id as the owner user id so simple tests
	// that reuse a single UUID for both don't need a custom hook.
	return &organization.Organization{ID: id, OwnerUserID: id, Type: organization.OrgTypeProviderPersonal}, nil
}
func (m *mockOrgRepo) FindByOwnerUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockOrgRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*organization.Organization, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	return nil, organization.ErrOrgNotFound
}
func (m *mockOrgRepo) Update(context.Context, *organization.Organization) error { return nil }
func (m *mockOrgRepo) Delete(context.Context, uuid.UUID) error                  { return nil }
func (m *mockOrgRepo) CountAll(context.Context) (int, error)                    { return 0, nil }
func (m *mockOrgRepo) FindByStripeAccountID(context.Context, string) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
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
func (m *mockOrgRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error {
	return nil
}
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
