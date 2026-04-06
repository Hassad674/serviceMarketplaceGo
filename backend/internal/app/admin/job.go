package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
)

// ListJobs returns paginated jobs for admin with author info and application counts.
func (s *Service) ListJobs(ctx context.Context, status, search, sort, filter, cursorStr string, limit int, page int) ([]repository.AdminJob, string, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	filters := repository.AdminJobFilters{
		Status: status, Search: search, Sort: sort,
		Filter: filter, Cursor: cursorStr, Limit: limit, Page: page,
	}

	total, err := s.jobs.CountAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list jobs: %w", err)
	}

	jobs, nextCursor, err := s.jobs.ListAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list jobs: %w", err)
	}

	if err := s.loadJobReportCounts(ctx, jobs); err != nil {
		return nil, "", 0, fmt.Errorf("list jobs: %w", err)
	}

	return jobs, nextCursor, total, nil
}

// GetJob returns a single job with full details for admin.
func (s *Service) GetJob(ctx context.Context, jobID uuid.UUID) (*repository.AdminJob, error) {
	j, err := s.jobs.GetAdmin(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}

// DeleteJob removes a job by ID (admin action).
func (s *Service) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	if err := s.jobs.Delete(ctx, jobID); err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	return nil
}

// ListJobApplications returns paginated job applications for admin.
func (s *Service) ListJobApplications(ctx context.Context, jobID, search, sort, filter, cursorStr string, limit int, page int) ([]repository.AdminJobApplication, string, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	filters := repository.AdminApplicationFilters{
		JobID: jobID, Search: search, Sort: sort,
		Filter: filter, Cursor: cursorStr, Limit: limit, Page: page,
	}

	total, err := s.applications.CountAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list applications: %w", err)
	}

	apps, nextCursor, err := s.applications.ListAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list applications: %w", err)
	}

	if err := s.loadApplicationReportCounts(ctx, apps); err != nil {
		return nil, "", 0, fmt.Errorf("list applications: %w", err)
	}

	return apps, nextCursor, total, nil
}

// DeleteJobApplication removes a job application by ID (admin action).
func (s *Service) DeleteJobApplication(ctx context.Context, applicationID uuid.UUID) error {
	if err := s.applications.Delete(ctx, applicationID); err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	return nil
}

func (s *Service) loadJobReportCounts(ctx context.Context, jobs []repository.AdminJob) error {
	if len(jobs) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(jobs))
	for i := range jobs {
		ids[i] = jobs[i].ID
	}
	counts, err := s.reports.PendingCountsByTargets(ctx, "job", ids)
	if err != nil {
		return err
	}
	for i := range jobs {
		jobs[i].PendingReportCount = counts[jobs[i].ID]
	}
	return nil
}

func (s *Service) loadApplicationReportCounts(ctx context.Context, apps []repository.AdminJobApplication) error {
	if len(apps) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(apps))
	for i := range apps {
		ids[i] = apps[i].ID
	}
	counts, err := s.reports.PendingCountsByTargets(ctx, "job_application", ids)
	if err != nil {
		return err
	}
	for i := range apps {
		apps[i].PendingReportCount = counts[apps[i].ID]
	}
	return nil
}
