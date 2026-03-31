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
	ListByApplicant(ctx context.Context, applicantID uuid.UUID, cursor string, limit int) ([]*job.JobApplication, string, error)
	CountByJob(ctx context.Context, jobID uuid.UUID) (int, error)
}
