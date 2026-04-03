package job

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type ServiceDeps struct {
	Jobs         repository.JobRepository
	Applications repository.JobApplicationRepository
	Users        repository.UserRepository
	Profiles     repository.ProfileRepository
	Messages     service.MessageSender
	JobViews     repository.JobViewRepository
	Credits      repository.JobCreditRepository
}

type Service struct {
	jobs         repository.JobRepository
	applications repository.JobApplicationRepository
	users        repository.UserRepository
	profiles     repository.ProfileRepository
	messages     service.MessageSender
	jobViews     repository.JobViewRepository
	credits      repository.JobCreditRepository
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		jobs:         deps.Jobs,
		applications: deps.Applications,
		users:        deps.Users,
		profiles:     deps.Profiles,
		messages:     deps.Messages,
		jobViews:     deps.JobViews,
		credits:      deps.Credits,
	}
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

// JobWithCounts enriches a job with application count info.
type JobWithCounts struct {
	Job              *domain.Job
	TotalApplicants  int
	NewApplicants    int
}

func (s *Service) ListMyJobs(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*domain.Job, string, error) {
	jobs, nextCursor, err := s.jobs.ListByCreator(ctx, userID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list my jobs: %w", err)
	}
	return jobs, nextCursor, nil
}

// ListMyJobsWithCounts returns jobs enriched with application counts.
func (s *Service) ListMyJobsWithCounts(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]JobWithCounts, string, error) {
	jobs, nextCursor, err := s.jobs.ListByCreator(ctx, userID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list my jobs: %w", err)
	}
	if len(jobs) == 0 || s.jobViews == nil {
		result := make([]JobWithCounts, len(jobs))
		for i, j := range jobs {
			result[i] = JobWithCounts{Job: j}
		}
		return result, nextCursor, nil
	}

	ids := make([]uuid.UUID, len(jobs))
	for i, j := range jobs {
		ids[i] = j.ID
	}
	counts, err := s.jobViews.GetApplicationCountsBatch(ctx, ids, userID)
	if err != nil {
		return nil, "", fmt.Errorf("get application counts: %w", err)
	}

	result := make([]JobWithCounts, len(jobs))
	for i, j := range jobs {
		c := counts[j.ID]
		result[i] = JobWithCounts{Job: j, TotalApplicants: c.Total, NewApplicants: c.NewCount}
	}
	return result, nextCursor, nil
}

// MarkApplicationsViewed updates the last_viewed_at timestamp for a job.
func (s *Service) MarkApplicationsViewed(ctx context.Context, jobID, userID uuid.UUID) error {
	if s.jobViews == nil {
		return nil
	}
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}
	if j.CreatorID != userID {
		return domain.ErrNotOwner
	}
	return s.jobViews.Upsert(ctx, jobID, userID)
}

// DeleteJob deletes a job and all its applications. Only the creator can delete.
func (s *Service) DeleteJob(ctx context.Context, jobID, userID uuid.UUID) error {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}
	if j.CreatorID != userID {
		return domain.ErrNotOwner
	}
	if err := s.jobs.Delete(ctx, jobID); err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	return nil
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

type UpdateJobInput struct {
	JobID            uuid.UUID
	UserID           uuid.UUID
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

func (s *Service) UpdateJob(ctx context.Context, input UpdateJobInput) (*domain.Job, error) {
	j, err := s.jobs.GetByID(ctx, input.JobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	updateInput := domain.UpdateJobInput{
		Title:         input.Title,
		Description:   input.Description,
		Skills:        input.Skills,
		ApplicantType: domain.ApplicantType(input.ApplicantType),
		BudgetType:    domain.BudgetType(input.BudgetType),
		MinBudget:     input.MinBudget,
		MaxBudget:     input.MaxBudget,
		IsIndefinite:  input.IsIndefinite,
		VideoURL:      input.VideoURL,
	}
	if input.PaymentFrequency != nil {
		f := domain.PaymentFrequency(*input.PaymentFrequency)
		updateInput.PaymentFrequency = &f
	}
	if input.DescriptionType != "" {
		updateInput.DescriptionType = domain.DescriptionType(input.DescriptionType)
	}
	updateInput.DurationWeeks = input.DurationWeeks

	if err := j.Update(input.UserID, updateInput); err != nil {
		return nil, err
	}
	if err := s.jobs.Update(ctx, j); err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}
	return j, nil
}

func (s *Service) ReopenJob(ctx context.Context, jobID, userID uuid.UUID) error {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}
	if err := j.Reopen(userID); err != nil {
		return err
	}
	if err := s.jobs.Update(ctx, j); err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	return nil
}

// GetCredits returns the current application credit balance for the user.
func (s *Service) GetCredits(ctx context.Context, userID uuid.UUID) (int, error) {
	if s.credits == nil {
		return domain.WeeklyQuota, nil
	}
	credits, err := s.credits.GetOrCreate(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get credits: %w", err)
	}
	return credits, nil
}

// ResetWeeklyCredits resets all users below the weekly quota back to the quota.
func (s *Service) ResetWeeklyCredits(ctx context.Context) error {
	if s.credits == nil {
		return nil
	}
	return s.credits.ResetWeekly(ctx, domain.WeeklyQuota)
}

func canCreateJob(role user.Role) bool {
	return role == user.RoleEnterprise || role == user.RoleAgency
}
