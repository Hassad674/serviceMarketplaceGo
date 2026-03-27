package job

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

type ServiceDeps struct {
	Jobs  repository.JobRepository
	Users repository.UserRepository
}

type Service struct {
	jobs  repository.JobRepository
	users repository.UserRepository
}

func NewService(deps ServiceDeps) *Service {
	return &Service{jobs: deps.Jobs, users: deps.Users}
}

type CreateJobInput struct {
	CreatorID        uuid.UUID
	Title            string
	Description      string
	Skills           []string
	ApplicantType    string
	BudgetType       string
	MinBudget        int
	MaxBudget        int
	PaymentFrequency *string
	DurationWeeks    *int
	IsIndefinite     bool
	DescriptionType  string
	VideoURL         *string
}

func (s *Service) CreateJob(ctx context.Context, input CreateJobInput) (*domain.Job, error) {
	creator, err := s.users.GetByID(ctx, input.CreatorID)
	if err != nil {
		return nil, fmt.Errorf("get creator: %w", err)
	}
	if !canCreateJob(creator.Role) {
		return nil, domain.ErrUnauthorizedRole
	}

	newInput := domain.NewJobInput{
		CreatorID:     input.CreatorID,
		Title:         input.Title,
		Description:   input.Description,
		Skills:        input.Skills,
		ApplicantType: domain.ApplicantType(input.ApplicantType),
		BudgetType:    domain.BudgetType(input.BudgetType),
		MinBudget:     input.MinBudget,
		MaxBudget:     input.MaxBudget,
		IsIndefinite:  input.IsIndefinite,
		DurationWeeks: input.DurationWeeks,
		VideoURL:      input.VideoURL,
	}
	if input.PaymentFrequency != nil {
		f := domain.PaymentFrequency(*input.PaymentFrequency)
		newInput.PaymentFrequency = &f
	}
	if input.DescriptionType != "" {
		newInput.DescriptionType = domain.DescriptionType(input.DescriptionType)
	}

	j, err := domain.NewJob(newInput)
	if err != nil {
		return nil, err
	}
	if err := s.jobs.Create(ctx, j); err != nil {
		return nil, fmt.Errorf("persist job: %w", err)
	}
	return j, nil
}

func (s *Service) GetJob(ctx context.Context, jobID uuid.UUID) (*domain.Job, error) {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}

func (s *Service) ListMyJobs(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*domain.Job, string, error) {
	jobs, nextCursor, err := s.jobs.ListByCreator(ctx, userID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list my jobs: %w", err)
	}
	return jobs, nextCursor, nil
}

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
