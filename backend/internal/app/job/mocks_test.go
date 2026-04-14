package job

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// --- mockOrgRepo (minimal, KYC-aware) ---

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
	// Default: return a stub non-blocked provider_personal org so any
	// test that doesn't care about KYC passes through the check.
	return &organization.Organization{ID: uuid.New(), Type: organization.OrgTypeProviderPersonal}, nil
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

// --- mockJobRepo ---

type mockJobRepo struct {
	createFn        func(ctx context.Context, j *domain.Job) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	updateFn        func(ctx context.Context, j *domain.Job) error
	listByOrgFn     func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Job, string, error)
	listOpenFn      func(ctx context.Context, filters repository.JobListFilters, cursor string, limit int) ([]*domain.Job, string, error)
}

func (m *mockJobRepo) Create(ctx context.Context, j *domain.Job) error {
	if m.createFn != nil {
		return m.createFn(ctx, j)
	}
	return nil
}

func (m *mockJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrJobNotFound
}

func (m *mockJobRepo) Update(ctx context.Context, j *domain.Job) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, j)
	}
	return nil
}

func (m *mockJobRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Job, string, error) {
	if m.listByOrgFn != nil {
		return m.listByOrgFn(ctx, orgID, cursor, limit)
	}
	return []*domain.Job{}, "", nil
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

func (m *mockJobRepo) ListOpen(ctx context.Context, filters repository.JobListFilters, cursor string, limit int) ([]*domain.Job, string, error) {
	if m.listOpenFn != nil {
		return m.listOpenFn(ctx, filters, cursor, limit)
	}
	return []*domain.Job{}, "", nil
}

// --- mockJobApplicationRepo ---

type mockJobApplicationRepo struct {
	createFn              func(ctx context.Context, app *domain.JobApplication) error
	getByIDFn             func(ctx context.Context, id uuid.UUID) (*domain.JobApplication, error)
	getByJobAndApplicantFn func(ctx context.Context, jobID, applicantID uuid.UUID) (*domain.JobApplication, error)
	deleteFn              func(ctx context.Context, id uuid.UUID) error
	listByJobFn           func(ctx context.Context, jobID uuid.UUID, cursor string, limit int) ([]*domain.JobApplication, string, error)
	listByApplicantOrgFn  func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.JobApplication, string, error)
	countByJobFn          func(ctx context.Context, jobID uuid.UUID) (int, error)
}

func (m *mockJobApplicationRepo) Create(ctx context.Context, app *domain.JobApplication) error {
	if m.createFn != nil {
		return m.createFn(ctx, app)
	}
	return nil
}

func (m *mockJobApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.JobApplication, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrApplicationNotFound
}

func (m *mockJobApplicationRepo) GetByJobAndApplicant(ctx context.Context, jobID, applicantID uuid.UUID) (*domain.JobApplication, error) {
	if m.getByJobAndApplicantFn != nil {
		return m.getByJobAndApplicantFn(ctx, jobID, applicantID)
	}
	return nil, domain.ErrApplicationNotFound
}

func (m *mockJobApplicationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockJobApplicationRepo) ListByJob(ctx context.Context, jobID uuid.UUID, cursor string, limit int) ([]*domain.JobApplication, string, error) {
	if m.listByJobFn != nil {
		return m.listByJobFn(ctx, jobID, cursor, limit)
	}
	return []*domain.JobApplication{}, "", nil
}

func (m *mockJobApplicationRepo) ListByApplicantOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.JobApplication, string, error) {
	if m.listByApplicantOrgFn != nil {
		return m.listByApplicantOrgFn(ctx, orgID, cursor, limit)
	}
	return []*domain.JobApplication{}, "", nil
}

func (m *mockJobApplicationRepo) CountByJob(ctx context.Context, jobID uuid.UUID) (int, error) {
	if m.countByJobFn != nil {
		return m.countByJobFn(ctx, jobID)
	}
	return 0, nil
}

func (m *mockJobApplicationRepo) ListAdmin(_ context.Context, _ repository.AdminApplicationFilters) ([]repository.AdminJobApplication, string, error) {
	return nil, "", nil
}

func (m *mockJobApplicationRepo) CountAdmin(_ context.Context, _ repository.AdminApplicationFilters) (int, error) {
	return 0, nil
}

// --- mockProfileRepo ---

type mockProfileRepo struct {
	orgProfilesByUserIDsFn func(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error)
}

func (m *mockProfileRepo) Create(_ context.Context, _ *profile.Profile) error { return nil }
func (m *mockProfileRepo) GetByOrganizationID(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) Update(_ context.Context, _ *profile.Profile) error { return nil }
func (m *mockProfileRepo) SearchPublic(_ context.Context, _ string, _ bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
	return nil, "", nil
}

func (m *mockProfileRepo) GetPublicProfilesByOrgIDs(_ context.Context, _ []uuid.UUID) ([]*profile.PublicProfile, error) {
	return nil, nil
}

func (m *mockProfileRepo) OrgProfilesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
	if m.orgProfilesByUserIDsFn != nil {
		return m.orgProfilesByUserIDsFn(ctx, userIDs)
	}
	return map[uuid.UUID]*profile.PublicProfile{}, nil
}

// Tier 1 completion stubs — the job feature does not exercise
// these paths but must satisfy the interface surface.
func (m *mockProfileRepo) UpdateLocation(_ context.Context, _ uuid.UUID, _ repository.LocationInput) error {
	return nil
}
func (m *mockProfileRepo) UpdateLanguages(_ context.Context, _ uuid.UUID, _, _ []string) error {
	return nil
}
func (m *mockProfileRepo) UpdateAvailability(_ context.Context, _ uuid.UUID, _ profile.AvailabilityStatus, _ *profile.AvailabilityStatus) error {
	return nil
}

// --- mockMsgSender ---

type mockMsgSender struct {
	findOrCreateConversationFn func(ctx context.Context, input portservice.FindOrCreateConversationInput) (uuid.UUID, error)
}

func (m *mockMsgSender) SendSystemMessage(_ context.Context, _ portservice.SystemMessageInput) error {
	return nil
}

func (m *mockMsgSender) FindOrCreateConversation(ctx context.Context, input portservice.FindOrCreateConversationInput) (uuid.UUID, error) {
	if m.findOrCreateConversationFn != nil {
		return m.findOrCreateConversationFn(ctx, input)
	}
	return uuid.New(), nil
}

// --- mockUserRepo ---

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
}

func (m *mockUserRepo) Create(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepo) Update(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepo) Delete(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	stubOrg := uuid.New()
	return &user.User{ID: id, Role: user.RoleEnterprise, DisplayName: "Test", OrganizationID: &stubOrg}, nil
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

// --- mockJobCreditRepo ---
//
// R12 — credit ops are now keyed by org id. The mock keeps the same
// shape as the old per-user mock so existing tests only need minor
// rewiring (the same `uuid.UUID` slot is now semantically "orgID").

type mockJobCreditRepo struct {
	getOrCreateFn func(ctx context.Context, orgID uuid.UUID) (int, error)
	decrementFn   func(ctx context.Context, orgID uuid.UUID) error
	refundFn      func(ctx context.Context, orgID uuid.UUID) error
	addBonusFn    func(ctx context.Context, orgID uuid.UUID, amount int, maxTokens int) error
	resetWeeklyFn func(ctx context.Context, minCredits int) error

	decrementCalls []uuid.UUID
	refundCalls    []uuid.UUID
}

func (m *mockJobCreditRepo) GetOrCreate(ctx context.Context, orgID uuid.UUID) (int, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, orgID)
	}
	return domain.WeeklyQuota, nil
}

func (m *mockJobCreditRepo) Decrement(ctx context.Context, orgID uuid.UUID) error {
	m.decrementCalls = append(m.decrementCalls, orgID)
	if m.decrementFn != nil {
		return m.decrementFn(ctx, orgID)
	}
	return nil
}

func (m *mockJobCreditRepo) Refund(ctx context.Context, orgID uuid.UUID) error {
	m.refundCalls = append(m.refundCalls, orgID)
	if m.refundFn != nil {
		return m.refundFn(ctx, orgID)
	}
	return nil
}

func (m *mockJobCreditRepo) AddBonus(ctx context.Context, orgID uuid.UUID, amount int, maxTokens int) error {
	if m.addBonusFn != nil {
		return m.addBonusFn(ctx, orgID, amount, maxTokens)
	}
	return nil
}

func (m *mockJobCreditRepo) ResetForOrg(_ context.Context, _ uuid.UUID, _ int) error { return nil }

func (m *mockJobCreditRepo) ResetWeekly(ctx context.Context, minCredits int) error {
	if m.resetWeeklyFn != nil {
		return m.resetWeeklyFn(ctx, minCredits)
	}
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
