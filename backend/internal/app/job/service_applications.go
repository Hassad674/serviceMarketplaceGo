package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	appmoderation "marketplace-backend/internal/app/moderation"
	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/moderation"
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
//
// R12 — Credits are debited on the applicant's ORGANIZATION, not on
// their user row. All operators of the same org share a single pool.
// The debit is atomic (single SQL UPDATE with `WHERE credits > 0`)
// so two operators racing to apply cannot both pass the check. If the
// subsequent application INSERT fails, the credit is refunded so the
// shared balance stays consistent with reality.
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
	if applicant.OrganizationID == nil {
		return nil, fmt.Errorf("apply to job: applicant must belong to an organization")
	}
	orgID := *applicant.OrganizationID

	// KYC enforcement: the applicant's organization must not be
	// blocked (14-day deadline since first earning without Stripe
	// onboarding).
	if s.orgs != nil {
		if org, oErr := s.orgs.FindByUserID(ctx, input.ApplicantID); oErr == nil && org.IsKYCBlocked() {
			return nil, user.ErrKYCRestricted
		}
	}

	// Duplicate-check BEFORE we spend a credit so a known-duplicate
	// apply never touches the shared pool.
	if _, dupErr := s.applications.GetByJobAndApplicant(ctx, input.JobID, input.ApplicantID); dupErr == nil {
		return nil, domain.ErrAlreadyApplied
	} else if !errors.Is(dupErr, domain.ErrApplicationNotFound) {
		return nil, fmt.Errorf("check existing application: %w", dupErr)
	}

	// Atomic credit debit. The single-statement UPDATE in the
	// repository returns ErrNoCreditsLeft when the pool is empty —
	// that IS the authoritative check under concurrent applies.
	if s.credits != nil {
		if decErr := s.credits.Decrement(ctx, orgID); decErr != nil {
			if errors.Is(decErr, domain.ErrNoCreditsLeft) {
				return nil, domain.ErrNoCreditsLeft
			}
			return nil, fmt.Errorf("decrement credits: %w", decErr)
		}
	}

	app, err := domain.NewJobApplication(domain.NewApplicationInput{
		JobID:                   input.JobID,
		ApplicantID:             input.ApplicantID,
		ApplicantOrganizationID: orgID,
		Message:                 input.Message,
		VideoURL:                input.VideoURL,
	})
	if err != nil {
		s.refundCredit(ctx, orgID)
		return nil, err
	}

	if err := s.applications.Create(ctx, app); err != nil {
		s.refundCredit(ctx, orgID)
		return nil, fmt.Errorf("persist application: %w", err)
	}

	// Async moderation: run AFTER the application is persisted so a
	// transient OpenAI hiccup never blocks the apply flow. The
	// content_id is the application's own ID — that lets the admin
	// queue link straight back to the application detail page.
	s.moderateApplicationMessage(app.ID, &app.ApplicantID, input.Message)

	return app, nil
}

// moderateApplicationMessage fires a background scan on the cover
// letter. Empty messages skip the call so we do not waste an API call
// on the (rare) zero-content apply. The orchestrator owns persistence
// + audit + admin notifier — this helper just spawns the goroutine.
func (s *Service) moderateApplicationMessage(appID uuid.UUID, authorID *uuid.UUID, message string) {
	if s.moderationOrchestrator == nil || message == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := s.moderationOrchestrator.Moderate(ctx, appmoderation.ModerateInput{
			ContentType:  moderation.ContentTypeJobApplicationMessage,
			ContentID:    appID,
			AuthorUserID: authorID,
			Text:         message,
		})
		if err != nil {
			slog.Error("application message moderation failed",
				"error", err, "application_id", appID)
		}
	}()
}

// refundCredit returns one credit to the org pool after a failed
// application insert. Best-effort — if the refund itself fails we log
// loudly so operators can reconcile, but we do not propagate the error
// to the caller (the original error is more actionable).
func (s *Service) refundCredit(ctx context.Context, orgID uuid.UUID) {
	if s.credits == nil {
		return
	}
	if err := s.credits.Refund(ctx, orgID); err != nil {
		slog.Error("failed to refund application credit after apply failure",
			"org_id", orgID, "error", err)
	}
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

// ListOrgApplications returns the applications submitted by the
// caller's organization, each enriched with the target job.
// All operators of the same org see the same list (shared workspace).
func (s *Service) ListOrgApplications(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]ApplicationWithJob, string, error) {
	apps, nextCursor, err := s.applications.ListByApplicantOrganization(ctx, orgID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list org applications: %w", err)
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
