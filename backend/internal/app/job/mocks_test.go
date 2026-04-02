package job

import (
	"context"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// --- mockJobRepo ---

type mockJobRepo struct {
	createFn        func(ctx context.Context, j *domain.Job) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	updateFn        func(ctx context.Context, j *domain.Job) error
	listByCreatorFn func(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*domain.Job, string, error)
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

func (m *mockJobRepo) ListByCreator(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*domain.Job, string, error) {
	if m.listByCreatorFn != nil {
		return m.listByCreatorFn(ctx, creatorID, cursor, limit)
	}
	return []*domain.Job{}, "", nil
}

func (m *mockJobRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

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
	listByApplicantFn     func(ctx context.Context, applicantID uuid.UUID, cursor string, limit int) ([]*domain.JobApplication, string, error)
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

func (m *mockJobApplicationRepo) ListByApplicant(ctx context.Context, applicantID uuid.UUID, cursor string, limit int) ([]*domain.JobApplication, string, error) {
	if m.listByApplicantFn != nil {
		return m.listByApplicantFn(ctx, applicantID, cursor, limit)
	}
	return []*domain.JobApplication{}, "", nil
}

func (m *mockJobApplicationRepo) CountByJob(ctx context.Context, jobID uuid.UUID) (int, error) {
	if m.countByJobFn != nil {
		return m.countByJobFn(ctx, jobID)
	}
	return 0, nil
}

// --- mockProfileRepo ---

type mockProfileRepo struct {
	getPublicProfilesByUserIDsFn func(ctx context.Context, userIDs []uuid.UUID) ([]*profile.PublicProfile, error)
}

func (m *mockProfileRepo) Create(_ context.Context, _ *profile.Profile) error               { return nil }
func (m *mockProfileRepo) GetByUserID(_ context.Context, _ uuid.UUID) (*profile.Profile, error) { return nil, nil }
func (m *mockProfileRepo) Update(_ context.Context, _ *profile.Profile) error               { return nil }
func (m *mockProfileRepo) SearchPublic(_ context.Context, _ string, _ bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
	return nil, "", nil
}

func (m *mockProfileRepo) GetPublicProfilesByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]*profile.PublicProfile, error) {
	if m.getPublicProfilesByUserIDsFn != nil {
		return m.getPublicProfilesByUserIDsFn(ctx, userIDs)
	}
	return []*profile.PublicProfile{}, nil
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
	return &user.User{ID: id, Role: user.RoleEnterprise, DisplayName: "Test"}, nil
}

func (m *mockUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}

func (m *mockUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
