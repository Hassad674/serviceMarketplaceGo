package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
)

// JobListFilters contains optional filters for listing open jobs.
type JobListFilters struct {
	Skills        []string
	ApplicantType string
	BudgetType    string
	MinBudget     *int
	MaxBudget     *int
	Search        string
}

// JobRepository defines persistence operations for job postings.
type JobRepository interface {
	Create(ctx context.Context, j *job.Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*job.Job, error)
	Update(ctx context.Context, j *job.Job) error
	ListByCreator(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*job.Job, string, error)
	ListOpen(ctx context.Context, filters JobListFilters, cursor string, limit int) ([]*job.Job, string, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
