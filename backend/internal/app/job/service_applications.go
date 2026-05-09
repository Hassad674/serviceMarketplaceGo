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
	// Kind is the persona under which the applicant is submitting.
	// Empty value defaults to a role-derived kind (agency → 'agency',
	// provider → 'freelance'); explicit 'referrer' is only allowed
	// for providers with referrer_enabled=true.
	Kind     domain.ApplicantKind
	Message  string
	VideoURL *string
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

	// Resolve the applicant kind. The handler may pass an explicit value
	// (radio selection in the apply modal) or leave it empty for the
	// default role-derived kind. The cross-check below is the authoritative
	// gate: an agency cannot fake a referrer kind, and a non-referrer
	// provider cannot apply as 'referrer'.
	kind, kErr := resolveApplicantKind(input.Kind, applicant.Role, applicant.ReferrerEnabled)
	if kErr != nil {
		return nil, kErr
	}

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
		ApplicantKind:           kind,
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

// ListJobApplicationsFilter narrows the candidates list. An empty Kind
// returns all applications; a non-empty Kind must validate via
// ApplicantKind.IsValid (the repository never receives an unchecked
// string, so we never build a SQL filter from user-controlled input
// without going through this guard).
type ListJobApplicationsFilter struct {
	Kind domain.ApplicantKind
}

// ListJobApplications returns applications for a job with enriched profiles.
func (s *Service) ListJobApplications(ctx context.Context, jobID, ownerID uuid.UUID, cursorStr string, limit int, filter ListJobApplicationsFilter) ([]ApplicationWithProfile, string, error) {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, "", fmt.Errorf("get job: %w", err)
	}
	if j.CreatorID != ownerID {
		return nil, "", domain.ErrNotOwner
	}

	if filter.Kind != "" && !filter.Kind.IsValid() {
		return nil, "", domain.ErrInvalidApplicantKind
	}

	apps, nextCursor, err := s.applications.ListByJob(ctx, jobID, cursorStr, limit, filter.Kind)
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

// ListOpenJobs returns open jobs matching the given filters, enriched
// with the public application count (social proof on the marketplace
// feed). new_applicants is intentionally NOT computed: that state is
// owner-only ("new since I last viewed") and has no meaning for a
// candidate browsing the public feed.
//
// The count is fetched via the same batch helper used by /jobs/mine,
// so this list endpoint stays N+1-free regardless of page size.
// When the optional JobView repository is not wired (e.g. legacy unit
// tests), the counts gracefully default to zero — the feed still
// renders, just without the social-proof badge.
func (s *Service) ListOpenJobs(ctx context.Context, filters repository.JobListFilters, cursorStr string, limit int) ([]JobWithCounts, string, error) {
	jobs, nextCursor, err := s.jobs.ListOpen(ctx, filters, cursorStr, limit)
	if err != nil {
		return nil, "", err
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
	// uuid.Nil for the viewer slot: the LEFT JOIN on job_views falls
	// through to the 1970 sentinel, so total stays accurate; the
	// new_count column is discarded below.
	counts, err := s.jobViews.GetApplicationCountsBatch(ctx, ids, uuid.Nil)
	if err != nil {
		return nil, "", fmt.Errorf("get application counts: %w", err)
	}

	result := make([]JobWithCounts, len(jobs))
	for i, j := range jobs {
		c := counts[j.ID]
		// Deliberately skip c.NewCount — public feed exposes total only.
		result[i] = JobWithCounts{Job: j, TotalApplicants: c.Total}
	}
	return result, nextCursor, nil
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

// resolveApplicantKind picks the persona under which the application
// is recorded. The default rule mirrors the user's role; an explicit
// kind overrides only when it is consistent with the role + referrer
// flag. This keeps the radio in the apply modal honest — a UI bug or
// a hostile client cannot persist a kind the applicant is not entitled
// to.
func resolveApplicantKind(requested domain.ApplicantKind, role user.Role, referrerEnabled bool) (domain.ApplicantKind, error) {
	switch role {
	case user.RoleAgency:
		// Agencies always submit as 'agency' — referrer mode is a
		// solo-provider feature.
		if requested != "" && requested != domain.ApplicantKindAgency {
			return "", domain.ErrInvalidApplicantKind
		}
		return domain.ApplicantKindAgency, nil
	case user.RoleProvider:
		switch requested {
		case "", domain.ApplicantKindFreelance:
			return domain.ApplicantKindFreelance, nil
		case domain.ApplicantKindReferrer:
			if !referrerEnabled {
				return "", domain.ErrInvalidApplicantKind
			}
			return domain.ApplicantKindReferrer, nil
		default:
			// Providers cannot apply as 'agency'.
			return "", domain.ErrInvalidApplicantKind
		}
	default:
		// Enterprise / unknown roles cannot apply at all.
		return "", domain.ErrApplicantTypeMismatch
	}
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
