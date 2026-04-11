package job

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/user"
)

func newTestService(jobRepo *mockJobRepo, userRepo *mockUserRepo) *Service {
	return NewService(ServiceDeps{
		Jobs:  jobRepo,
		Users: userRepo,
	})
}

func TestCreateJob_Success(t *testing.T) {
	var persisted *domain.Job
	jobRepo := &mockJobRepo{
		createFn: func(_ context.Context, j *domain.Job) error {
			persisted = j
			return nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id, Role: user.RoleEnterprise}, nil
		},
	}

	svc := newTestService(jobRepo, userRepo)
	j, err := svc.CreateJob(context.Background(), CreateJobInput{
		CreatorID:     uuid.New(),
		Title:         "Go Developer",
		Description:   "Build APIs",
		Skills:        []string{"Go"},
		ApplicantType: "all",
		BudgetType:    "one_shot",
		MinBudget:     1000,
		MaxBudget:     5000,
	})

	assert.NoError(t, err)
	assert.NotNil(t, j)
	assert.Equal(t, domain.StatusOpen, j.Status)
	assert.NotNil(t, persisted)
}

func TestCreateJob_ProviderForbidden(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id, Role: user.RoleProvider}, nil
		},
	}
	svc := newTestService(&mockJobRepo{}, userRepo)

	j, err := svc.CreateJob(context.Background(), CreateJobInput{
		CreatorID:     uuid.New(),
		Title:         "Test",
		Description:   "Test",
		ApplicantType: "all",
		BudgetType:    "one_shot",
		MinBudget:     100,
		MaxBudget:     200,
	})

	assert.ErrorIs(t, err, domain.ErrUnauthorizedRole)
	assert.Nil(t, j)
}

func TestCreateJob_AgencyAllowed(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id, Role: user.RoleAgency}, nil
		},
	}
	svc := newTestService(&mockJobRepo{}, userRepo)

	j, err := svc.CreateJob(context.Background(), CreateJobInput{
		CreatorID:     uuid.New(),
		Title:         "Designer needed",
		Description:   "UX work",
		Skills:        []string{"UX"},
		ApplicantType: "freelancers",
		BudgetType:    "long_term",
		MinBudget:     500,
		MaxBudget:     1000,
	})

	assert.NoError(t, err)
	assert.NotNil(t, j)
}

func TestCreateJob_ValidationError(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id, Role: user.RoleEnterprise}, nil
		},
	}
	svc := newTestService(&mockJobRepo{}, userRepo)

	j, err := svc.CreateJob(context.Background(), CreateJobInput{
		CreatorID:     uuid.New(),
		Title:         "",
		Description:   "Desc",
		ApplicantType: "all",
		BudgetType:    "one_shot",
		MinBudget:     100,
		MaxBudget:     200,
	})

	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
	assert.Nil(t, j)
}

func TestCloseJob_Success(t *testing.T) {
	creatorID := uuid.New()
	existingJob := &domain.Job{
		ID:        uuid.New(),
		CreatorID: creatorID,
		Status:    domain.StatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jobRepo := &mockJobRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Job, error) {
			return existingJob, nil
		},
		updateFn: func(_ context.Context, j *domain.Job) error {
			assert.Equal(t, domain.StatusClosed, j.Status)
			return nil
		},
	}
	svc := newTestService(jobRepo, &mockUserRepo{})

	err := svc.CloseJob(context.Background(), existingJob.ID, creatorID)
	assert.NoError(t, err)
}

func TestCloseJob_NotOwner(t *testing.T) {
	creatorID := uuid.New()
	existingJob := &domain.Job{
		ID:        uuid.New(),
		CreatorID: creatorID,
		Status:    domain.StatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jobRepo := &mockJobRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Job, error) {
			return existingJob, nil
		},
	}
	svc := newTestService(jobRepo, &mockUserRepo{})

	err := svc.CloseJob(context.Background(), existingJob.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotOwner)
}

func TestCloseJob_AlreadyClosed(t *testing.T) {
	creatorID := uuid.New()
	now := time.Now()
	existingJob := &domain.Job{
		ID:        uuid.New(),
		CreatorID: creatorID,
		Status:    domain.StatusClosed,
		ClosedAt:  &now,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jobRepo := &mockJobRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Job, error) {
			return existingJob, nil
		},
	}
	svc := newTestService(jobRepo, &mockUserRepo{})

	err := svc.CloseJob(context.Background(), existingJob.ID, creatorID)
	assert.ErrorIs(t, err, domain.ErrAlreadyClosed)
}

func TestListOrgJobs_Empty(t *testing.T) {
	svc := newTestService(&mockJobRepo{}, &mockUserRepo{})
	jobs, cursor, err := svc.ListOrgJobs(context.Background(), uuid.New(), "", 20)
	assert.NoError(t, err)
	assert.Empty(t, jobs)
	assert.Empty(t, cursor)
}
