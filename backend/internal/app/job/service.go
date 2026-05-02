package job

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	appmoderation "marketplace-backend/internal/app/moderation"
	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type ServiceDeps struct {
	Jobs         repository.JobRepository
	Applications repository.JobApplicationRepository
	Users        repository.UserReader
	// Organizations is narrowed to OrganizationReader — the job feature
	// only resolves the applicant's org by user id to gate KYC.
	Organizations repository.OrganizationReader
	Profiles      repository.ProfileRepository
	Messages      service.MessageSender
	JobViews      repository.JobViewRepository
	Credits       repository.JobCreditRepository
}

type Service struct {
	jobs         repository.JobRepository
	applications repository.JobApplicationRepository
	users        repository.UserReader
	orgs         repository.OrganizationReader
	profiles     repository.ProfileRepository
	messages               service.MessageSender
	jobViews               repository.JobViewRepository
	credits                repository.JobCreditRepository
	moderationOrchestrator *appmoderation.Service // optional sync moderation gate
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		jobs:         deps.Jobs,
		applications: deps.Applications,
		users:        deps.Users,
		orgs:         deps.Organizations,
		profiles:     deps.Profiles,
		messages:     deps.Messages,
		jobViews:     deps.JobViews,
		credits:      deps.Credits,
	}
}

// SetModerationOrchestrator wires the synchronous moderation gate.
// Optional: when nil, the create + update paths skip the gate (legacy
// behaviour). In production this MUST be set, otherwise toxic job
// listings make it through to the public marketplace.
func (s *Service) SetModerationOrchestrator(svc *appmoderation.Service) {
	s.moderationOrchestrator = svc
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

	// Synchronous moderation gate. Job listings are public + SEO-
	// indexed, so we refuse creation outright when the title or
	// description score above the blocking threshold. ContentID is
	// the freshly-minted job.ID — admin queue can later show the
	// blocked attempt and admin support can investigate the user.
	if err := s.moderateJobText(ctx, j.ID, input.Title, input.Description); err != nil {
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

// ListOrgJobs returns the jobs posted by the caller's organization.
// Every operator of the same org sees the exact same list — the
// Stripe Dashboard shared-workspace semantics.
func (s *Service) ListOrgJobs(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]*domain.Job, string, error) {
	jobs, nextCursor, err := s.jobs.ListByOrganization(ctx, orgID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list org jobs: %w", err)
	}
	return jobs, nextCursor, nil
}

// ListOrgJobsWithCounts returns org jobs enriched with application counts.
// Application view state (new-since-last-viewed) stays per-user since
// each operator's personal "I last looked at this at X" marker is still
// meaningful inside a shared org.
func (s *Service) ListOrgJobsWithCounts(ctx context.Context, orgID, viewerUserID uuid.UUID, cursorStr string, limit int) ([]JobWithCounts, string, error) {
	jobs, nextCursor, err := s.jobs.ListByOrganization(ctx, orgID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list org jobs: %w", err)
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
	counts, err := s.jobViews.GetApplicationCountsBatch(ctx, ids, viewerUserID)
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

	// Re-moderate edited title + description. Updates pass the same
	// gate as creates because an edit can flip a clean listing into
	// a toxic one (and we need to refuse it before persistence).
	if err := s.moderateJobText(ctx, j.ID, input.Title, input.Description); err != nil {
		return nil, err
	}

	if err := s.jobs.Update(ctx, j); err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}
	return j, nil
}

// moderateJobText runs the synchronous gate on title (strict 0.50) and
// description (lenient 0.85). Empty fields are skipped — partial
// updates are common for the "change budget only" flow.
func (s *Service) moderateJobText(ctx context.Context, jobID uuid.UUID, title, description string) error {
	if s.moderationOrchestrator == nil {
		return nil
	}
	if title = strings.TrimSpace(title); title != "" {
		_, err := s.moderationOrchestrator.Moderate(ctx, appmoderation.ModerateInput{
			ContentType:       moderation.ContentTypeJobTitle,
			ContentID:         jobID,
			Text:              title,
			BlockingMode:      true,
			BlockingThreshold: 0.50,
		})
		if errors.Is(err, moderation.ErrContentBlocked) {
			return domain.ErrJobTitleInappropriate
		}
		if err != nil {
			return fmt.Errorf("moderate job title: %w", err)
		}
	}
	if description = strings.TrimSpace(description); description != "" {
		_, err := s.moderationOrchestrator.Moderate(ctx, appmoderation.ModerateInput{
			ContentType:       moderation.ContentTypeJobDescription,
			ContentID:         jobID,
			Text:              description,
			BlockingMode:      true,
			BlockingThreshold: 0.85,
		})
		if errors.Is(err, moderation.ErrContentBlocked) {
			return domain.ErrJobDescriptionInappropriate
		}
		if err != nil {
			return fmt.Errorf("moderate job description: %w", err)
		}
	}
	return nil
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

// GetCredits returns the shared application credit balance for the
// given organization. Every operator of the same org sees the same
// number — that is the whole point of R12.
func (s *Service) GetCredits(ctx context.Context, orgID uuid.UUID) (int, error) {
	if s.credits == nil {
		return domain.WeeklyQuota, nil
	}
	credits, err := s.credits.GetOrCreate(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("get credits: %w", err)
	}
	return credits, nil
}

// ResetWeeklyCredits resets every org below the weekly quota back to
// the quota. Triggered by the external weekly cron.
func (s *Service) ResetWeeklyCredits(ctx context.Context) error {
	if s.credits == nil {
		return nil
	}
	return s.credits.ResetWeekly(ctx, domain.WeeklyQuota)
}

// ResetCreditsForUser accepts a user id for admin UX convenience
// (admins click "reset this user's credits" on a user row), but
// resolves the user's org under the hood and resets the shared pool —
// because that is where credits actually live after R12. If the user
// has no org, the call is a no-op.
func (s *Service) ResetCreditsForUser(ctx context.Context, userID uuid.UUID) error {
	if s.credits == nil {
		return nil
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user for credit reset: %w", err)
	}
	if u.OrganizationID == nil {
		return nil
	}
	return s.credits.ResetForOrg(ctx, *u.OrganizationID, domain.WeeklyQuota)
}

func canCreateJob(role user.Role) bool {
	return role == user.RoleEnterprise || role == user.RoleAgency
}
