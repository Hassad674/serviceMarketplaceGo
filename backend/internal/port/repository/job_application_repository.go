package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
)

// JobApplicationRepository defines persistence operations for job applications.
type JobApplicationRepository interface {
	Create(ctx context.Context, app *job.JobApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*job.JobApplication, error)
	GetByJobAndApplicant(ctx context.Context, jobID, applicantID uuid.UUID) (*job.JobApplication, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByJob(ctx context.Context, jobID uuid.UUID, cursor string, limit int) ([]*job.JobApplication, string, error)
	// ListByApplicantOrganization returns job applications submitted by
	// any member of the given organization. All operators of the same
	// applying org see the same list.
	ListByApplicantOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*job.JobApplication, string, error)
	CountByJob(ctx context.Context, jobID uuid.UUID) (int, error)

	// Admin methods
	ListAdmin(ctx context.Context, filters AdminApplicationFilters) ([]AdminJobApplication, string, error)
	CountAdmin(ctx context.Context, filters AdminApplicationFilters) (int, error)
}

// JobViewRepository tracks when users last viewed a job's applications.
type JobViewRepository interface {
	Upsert(ctx context.Context, jobID, userID uuid.UUID) error
	GetApplicationCounts(ctx context.Context, jobID, userID uuid.UUID) (total int, newCount int, err error)
	GetApplicationCountsBatch(ctx context.Context, jobIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]ApplicationCounts, error)
}

// ApplicationCounts holds total and new application counts for a job.
type ApplicationCounts struct {
	Total    int
	NewCount int
}
