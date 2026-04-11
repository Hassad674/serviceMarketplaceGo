package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ApplyToJobInput contains the data required to apply to a job.
type ApplyToJobInput struct {
	JobID       uuid.UUID
	ApplicantID uuid.UUID
	Message     string
	VideoURL    *string
}

// ApplicationWithProfile pairs an application with the applicant's public profile.
type ApplicationWithProfile struct {
	Application *domain.JobApplication
	Profile     *profile.PublicProfile
}

// ApplicationWithJob pairs an application with the job it was submitted to.
type ApplicationWithJob struct {
	Application *domain.JobApplication
	Job         *domain.Job
}

// ApplyToJob creates a new application to a job posting.
func (s *Service) ApplyToJob(ctx context.Context, input ApplyToJobInput) (*domain.JobApplication, error) {
	j, err := s.jobs.GetByID(ctx, input.JobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	if j.Status != domain.StatusOpen {
		return nil, domain.ErrCannotApplyToClosed
	}
	if j.CreatorID == input.ApplicantID {
		return nil, domain.ErrCannotApplyToOwnJob
	}

	applicant, err := s.users.GetByID(ctx, input.ApplicantID)
	if err != nil {
		return nil, fmt.Errorf("get applicant: %w", err)
	}
	if !canApply(j.ApplicantType, applicant.Role) {
		return nil, domain.ErrApplicantTypeMismatch
	}
	// KYC enforcement: blocked providers/agencies cannot apply
	if applicant.IsKYCBlocked() {
		return nil, user.ErrKYCRestricted
	}

	// Check application credits before proceeding.
	if s.credits != nil {
		credits, credErr := s.credits.GetOrCreate(ctx, input.ApplicantID)
		if credErr != nil {
			return nil, fmt.Errorf("check credits: %w", credErr)
		}
		if credits <= 0 {
			return nil, domain.ErrNoCreditsLeft
		}
	}

	_, err = s.applications.GetByJobAndApplicant(ctx, input.JobID, input.ApplicantID)
	if err == nil {
		return nil, domain.ErrAlreadyApplied
	}
	if !errors.Is(err, domain.ErrApplicationNotFound) {
		return nil, fmt.Errorf("check existing application: %w", err)
	}

	app, err := domain.NewJobApplication(domain.NewApplicationInput{
		JobID:       input.JobID,
		ApplicantID: input.ApplicantID,
		Message:     input.Message,
		VideoURL:    input.VideoURL,
	})
	if err != nil {
		return nil, err
	}

	if err := s.applications.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("persist application: %w", err)
	}

	// Decrement credit after successful application.
	if s.credits != nil {
		if decErr := s.credits.Decrement(ctx, input.ApplicantID); decErr != nil {
			// Log but do not fail the application — the app was already created.
			slog.Warn("failed to decrement credits", "user_id", input.ApplicantID, "error", decErr)
		}
	}

	return app, nil
}

// WithdrawApplication removes an application. Only the applicant can withdraw.
func (s *Service) WithdrawApplication(ctx context.Context, applicationID, applicantID uuid.UUID) error {
	app, err := s.applications.GetByID(ctx, applicationID)
	if err != nil {
		return fmt.Errorf("get application: %w", err)
	}
	if app.ApplicantID != applicantID {
		return domain.ErrNotApplicant
	}
	if err := s.applications.Delete(ctx, applicationID); err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	return nil
}

// ListJobApplications returns applications for a job with enriched profiles.
func (s *Service) ListJobApplications(ctx context.Context, jobID, ownerID uuid.UUID, cursorStr string, limit int) ([]ApplicationWithProfile, string, error) {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, "", fmt.Errorf("get job: %w", err)
	}
	if j.CreatorID != ownerID {
		return nil, "", domain.ErrNotOwner
	}

	apps, nextCursor, err := s.applications.ListByJob(ctx, jobID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list applications: %w", err)
	}
	if len(apps) == 0 {
		return []ApplicationWithProfile{}, "", nil
	}

	profileMap, err := s.fetchProfileMap(ctx, apps)
	if err != nil {
		return nil, "", err
	}

	results := make([]ApplicationWithProfile, len(apps))
	for i, app := range apps {
		results[i] = ApplicationWithProfile{
			Application: app,
			Profile:     profileMap[app.ApplicantID],
		}
	}
	return results, nextCursor, nil
}

// ListMyApplications returns the current user's applications with job details.
func (s *Service) ListMyApplications(ctx context.Context, applicantID uuid.UUID, cursorStr string, limit int) ([]ApplicationWithJob, string, error) {
	apps, nextCursor, err := s.applications.ListByApplicant(ctx, applicantID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list my applications: %w", err)
	}
	if len(apps) == 0 {
		return []ApplicationWithJob{}, "", nil
	}

	results := make([]ApplicationWithJob, 0, len(apps))
	for _, app := range apps {
		j, jErr := s.jobs.GetByID(ctx, app.JobID)
		if jErr != nil {
			continue // skip if job was deleted
		}
		results = append(results, ApplicationWithJob{
			Application: app,
			Job:         j,
		})
	}
	return results, nextCursor, nil
}

// ContactApplicant finds or creates a conversation with an applicant.
func (s *Service) ContactApplicant(ctx context.Context, jobID, ownerID, applicantID uuid.UUID) (uuid.UUID, error) {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get job: %w", err)
	}
	if j.CreatorID != ownerID {
		return uuid.Nil, domain.ErrNotOwner
	}

	_, err = s.applications.GetByJobAndApplicant(ctx, jobID, applicantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get application: %w", err)
	}

	if s.messages == nil {
		return uuid.Nil, fmt.Errorf("messaging not available")
	}

	convID, err := s.messages.FindOrCreateConversation(ctx, service.FindOrCreateConversationInput{
		UserA:   ownerID,
		UserB:   applicantID,
		Content: fmt.Sprintf("Contact initiated regarding your application to \"%s\"", j.Title),
		Type:    "candidature_contact",
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create conversation: %w", err)
	}
	return convID, nil
}

// ListOpenJobs returns open jobs matching the given filters.
func (s *Service) ListOpenJobs(ctx context.Context, filters repository.JobListFilters, cursorStr string, limit int) ([]*domain.Job, string, error) {
	return s.jobs.ListOpen(ctx, filters, cursorStr, limit)
}

// HasApplied checks if the user has already applied to the given job.
func (s *Service) HasApplied(ctx context.Context, jobID, applicantID uuid.UUID) (bool, error) {
	_, err := s.applications.GetByJobAndApplicant(ctx, jobID, applicantID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, domain.ErrApplicationNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("check application: %w", err)
}

func (s *Service) fetchProfileMap(ctx context.Context, apps []*domain.JobApplication) (map[uuid.UUID]*profile.PublicProfile, error) {
	if s.profiles == nil {
		return map[uuid.UUID]*profile.PublicProfile{}, nil
	}
	ids := make([]uuid.UUID, len(apps))
	for i, a := range apps {
		ids[i] = a.ApplicantID
	}
	return s.profiles.OrgProfilesByUserIDs(ctx, ids)
}

func canApply(applicantType domain.ApplicantType, role user.Role) bool {
	switch applicantType {
	case domain.ApplicantFreelancers:
		return role == user.RoleProvider
	case domain.ApplicantAgencies:
		return role == user.RoleAgency
	case domain.ApplicantAll:
		return role == user.RoleProvider || role == user.RoleAgency
	default:
		return false
	}
}
