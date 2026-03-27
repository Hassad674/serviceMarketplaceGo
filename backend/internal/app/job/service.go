package job

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// ServiceDeps contains all dependencies for the job service.
type ServiceDeps struct {
	Jobs  repository.JobRepository
	Users repository.UserRepository
}

// Service orchestrates job-related use cases.
type Service struct {
	jobs  repository.JobRepository
	users repository.UserRepository
}

// NewService creates a new job application service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		jobs:  deps.Jobs,
		users: deps.Users,
	}
}

// CreateJobInput holds the data required to create a new job.
type CreateJobInput struct {
	CreatorID     uuid.UUID
	Title         string
	Description   string
	Skills        []string
	ApplicantType string
	BudgetType    string
	MinBudget     int
	MaxBudget     int
}

// CreateJob validates the creator role and persists a new job posting.
func (s *Service) CreateJob(ctx context.Context, input CreateJobInput) (*domain.Job, error) {
	creator, err := s.users.GetByID(ctx, input.CreatorID)
	if err != nil {
		return nil, fmt.Errorf("get creator: %w", err)
	}

	if !canCreateJob(creator.Role) {
		return nil, domain.ErrUnauthorizedRole
	}

	j, err := domain.NewJob(domain.NewJobInput{
		CreatorID:     input.CreatorID,
		Title:         input.Title,
		Description:   input.Description,
		Skills:        input.Skills,
		ApplicantType: domain.ApplicantType(input.ApplicantType),
		BudgetType:    domain.BudgetType(input.BudgetType),
		MinBudget:     input.MinBudget,
		MaxBudget:     input.MaxBudget,
	})
	if err != nil {
		return nil, err
	}

	if err := s.jobs.Create(ctx, j); err != nil {
		return nil, fmt.Errorf("persist job: %w", err)
	}

	return j, nil
}

// GetJob returns a single job by ID.
func (s *Service) GetJob(ctx context.Context, jobID uuid.UUID) (*domain.Job, error) {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}

// ListMyJobs returns jobs created by the given user with cursor pagination.
func (s *Service) ListMyJobs(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*domain.Job, string, error) {
	jobs, nextCursor, err := s.jobs.ListByCreator(ctx, userID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list my jobs: %w", err)
	}
	return jobs, nextCursor, nil
}

// CloseJob closes an open job. Only the creator may close their own job.
func (s *Service) CloseJob(ctx context.Context, jobID, userID uuid.UUID) error {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	if err := j.Close(userID); err != nil {
		return err
	}

	if err := s.jobs.Update(ctx, j); err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	return nil
}

func canCreateJob(role user.Role) bool {
	return role == user.RoleEnterprise || role == user.RoleAgency
}
