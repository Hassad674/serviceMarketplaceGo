package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/job"
	"marketplace-backend/pkg/cursor"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(ctx context.Context, j *job.Job) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var paymentFreq *string
	if j.PaymentFrequency != nil {
		s := string(*j.PaymentFrequency)
		paymentFreq = &s
	}

	_, err := r.db.ExecContext(ctx, queryInsertJob,
		j.ID, j.CreatorID, j.Title, j.Description, pq.Array(j.Skills),
		string(j.ApplicantType), string(j.BudgetType), j.MinBudget, j.MaxBudget,
		string(j.Status), j.CreatedAt, j.UpdatedAt, j.ClosedAt,
		paymentFreq, j.DurationWeeks, j.IsIndefinite,
		string(j.DescriptionType), j.VideoURL,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*job.Job, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	j, err := scanJob(r.db.QueryRowContext(ctx, queryGetJobByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, job.ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job by id: %w", err)
	}
	return j, nil
}

func (r *JobRepository) Update(ctx context.Context, j *job.Job) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	result, err := r.db.ExecContext(ctx, queryUpdateJob, j.ID, string(j.Status), j.ClosedAt, j.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return job.ErrJobNotFound
	}
	return nil
}

func (r *JobRepository) ListByCreator(ctx context.Context, creatorID uuid.UUID, cursorStr string, limit int) ([]*job.Job, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows *sql.Rows
	var err error
	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListJobsByCreatorFirst, creatorID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListJobsByCreatorWithCursor, creatorID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list jobs by creator: %w", err)
	}
	defer rows.Close()
	return scanJobListWithCursor(rows, limit)
}

func scanJob(row *sql.Row) (*job.Job, error) {
	j := &job.Job{}
	var status, applicantType, budgetType string
	var paymentFreq, descType *string
	err := row.Scan(
		&j.ID, &j.CreatorID, &j.Title, &j.Description, pq.Array(&j.Skills),
		&applicantType, &budgetType, &j.MinBudget, &j.MaxBudget,
		&status, &j.CreatedAt, &j.UpdatedAt, &j.ClosedAt,
		&paymentFreq, &j.DurationWeeks, &j.IsIndefinite,
		&descType, &j.VideoURL,
	)
	if err != nil {
		return nil, err
	}
	j.Status = job.JobStatus(status)
	j.ApplicantType = job.ApplicantType(applicantType)
	j.BudgetType = job.BudgetType(budgetType)
	if paymentFreq != nil {
		f := job.PaymentFrequency(*paymentFreq)
		j.PaymentFrequency = &f
	}
	if descType != nil {
		j.DescriptionType = job.DescriptionType(*descType)
	} else {
		j.DescriptionType = job.DescriptionText
	}
	return j, nil
}

func scanJobFromRows(rows *sql.Rows) (*job.Job, error) {
	j := &job.Job{}
	var status, applicantType, budgetType string
	var paymentFreq, descType *string
	err := rows.Scan(
		&j.ID, &j.CreatorID, &j.Title, &j.Description, pq.Array(&j.Skills),
		&applicantType, &budgetType, &j.MinBudget, &j.MaxBudget,
		&status, &j.CreatedAt, &j.UpdatedAt, &j.ClosedAt,
		&paymentFreq, &j.DurationWeeks, &j.IsIndefinite,
		&descType, &j.VideoURL,
	)
	if err != nil {
		return nil, err
	}
	j.Status = job.JobStatus(status)
	j.ApplicantType = job.ApplicantType(applicantType)
	j.BudgetType = job.BudgetType(budgetType)
	if paymentFreq != nil {
		f := job.PaymentFrequency(*paymentFreq)
		j.PaymentFrequency = &f
	}
	if descType != nil {
		j.DescriptionType = job.DescriptionType(*descType)
	} else {
		j.DescriptionType = job.DescriptionText
	}
	return j, nil
}

func scanJobListWithCursor(rows *sql.Rows, limit int) ([]*job.Job, string, error) {
	var results []*job.Job
	for rows.Next() {
		j, err := scanJobFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("scan job: %w", err)
		}
		results = append(results, j)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}
	if results == nil {
		results = []*job.Job{}
	}
	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}
	return results, nextCursor, nil
}
