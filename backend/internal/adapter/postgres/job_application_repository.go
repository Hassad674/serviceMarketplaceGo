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

// JobApplicationRepository implements repository.JobApplicationRepository.
type JobApplicationRepository struct {
	db *sql.DB
}

func NewJobApplicationRepository(db *sql.DB) *JobApplicationRepository {
	return &JobApplicationRepository{db: db}
}

func (r *JobApplicationRepository) Create(ctx context.Context, app *job.JobApplication) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertJobApplication,
		app.ID, app.JobID, app.ApplicantID, app.Message, app.VideoURL,
		app.CreatedAt, app.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return job.ErrAlreadyApplied
		}
		return fmt.Errorf("insert job application: %w", err)
	}
	return nil
}

func (r *JobApplicationRepository) GetByID(ctx context.Context, id uuid.UUID) (*job.JobApplication, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	app, err := scanJobApp(r.db.QueryRowContext(ctx, queryGetJobApplicationByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, job.ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job application by id: %w", err)
	}
	return app, nil
}

func (r *JobApplicationRepository) GetByJobAndApplicant(ctx context.Context, jobID, applicantID uuid.UUID) (*job.JobApplication, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	app, err := scanJobApp(r.db.QueryRowContext(ctx, queryGetJobApplicationByJobAndApplicant, jobID, applicantID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, job.ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job application by job and applicant: %w", err)
	}
	return app, nil
}

func (r *JobApplicationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryDeleteJobApplication, id)
	if err != nil {
		return fmt.Errorf("delete job application: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return job.ErrApplicationNotFound
	}
	return nil
}

func (r *JobApplicationRepository) ListByJob(ctx context.Context, jobID uuid.UUID, cursorStr string, limit int) ([]*job.JobApplication, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error
	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListJobAppsByJobFirst, jobID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListJobAppsByJobWithCursor, jobID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list job applications by job: %w", err)
	}
	defer rows.Close()
	return scanJobAppListWithCursor(rows, limit)
}

func (r *JobApplicationRepository) ListByApplicant(ctx context.Context, applicantID uuid.UUID, cursorStr string, limit int) ([]*job.JobApplication, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error
	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListJobAppsByApplicantFirst, applicantID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListJobAppsByApplicantWithCursor, applicantID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list job applications by applicant: %w", err)
	}
	defer rows.Close()
	return scanJobAppListWithCursor(rows, limit)
}

func (r *JobApplicationRepository) CountByJob(ctx context.Context, jobID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx, queryCountJobAppsByJob, jobID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count job applications: %w", err)
	}
	return count, nil
}

func scanJobApp(row *sql.Row) (*job.JobApplication, error) {
	app := &job.JobApplication{}
	err := row.Scan(
		&app.ID, &app.JobID, &app.ApplicantID, &app.Message,
		&app.VideoURL, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func scanJobAppFromRows(rows *sql.Rows) (*job.JobApplication, error) {
	app := &job.JobApplication{}
	err := rows.Scan(
		&app.ID, &app.JobID, &app.ApplicantID, &app.Message,
		&app.VideoURL, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func scanJobAppListWithCursor(rows *sql.Rows, limit int) ([]*job.JobApplication, string, error) {
	var results []*job.JobApplication
	for rows.Next() {
		app, err := scanJobAppFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("scan job application: %w", err)
		}
		results = append(results, app)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}
	if results == nil {
		results = []*job.JobApplication{}
	}
	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}
	return results, nextCursor, nil
}
