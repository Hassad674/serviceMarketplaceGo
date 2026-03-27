package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
)

// JobRepository defines persistence operations for job postings.
type JobRepository interface {
	Create(ctx context.Context, j *job.Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*job.Job, error)
	Update(ctx context.Context, j *job.Job) error
	ListByCreator(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*job.Job, string, error)
}
