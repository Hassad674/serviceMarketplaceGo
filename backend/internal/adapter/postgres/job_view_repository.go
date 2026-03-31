package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/port/repository"
)

type JobViewRepository struct {
	db *sql.DB
}

func NewJobViewRepository(db *sql.DB) *JobViewRepository {
	return &JobViewRepository{db: db}
}

func (r *JobViewRepository) Upsert(ctx context.Context, jobID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO job_views (job_id, user_id, last_viewed_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (job_id, user_id)
		 DO UPDATE SET last_viewed_at = now()`,
		jobID, userID,
	)
	if err != nil {
		return fmt.Errorf("upsert job view: %w", err)
	}
	return nil
}

func (r *JobViewRepository) GetApplicationCounts(ctx context.Context, jobID, userID uuid.UUID) (int, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var total, newCount int
	err := r.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE ja.created_at > COALESCE(jv.last_viewed_at, '1970-01-01'::timestamptz)) AS new_count
		 FROM job_applications ja
		 LEFT JOIN job_views jv ON jv.job_id = ja.job_id AND jv.user_id = $2
		 WHERE ja.job_id = $1`,
		jobID, userID,
	).Scan(&total, &newCount)
	if err != nil {
		return 0, 0, fmt.Errorf("get application counts: %w", err)
	}
	return total, newCount, nil
}

func (r *JobViewRepository) GetApplicationCountsBatch(ctx context.Context, jobIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]repository.ApplicationCounts, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if len(jobIDs) == 0 {
		return map[uuid.UUID]repository.ApplicationCounts{}, nil
	}

	ids := make([]string, len(jobIDs))
	for i, id := range jobIDs {
		ids[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT
			ja.job_id,
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE ja.created_at > COALESCE(jv.last_viewed_at, '1970-01-01'::timestamptz)) AS new_count
		 FROM job_applications ja
		 LEFT JOIN job_views jv ON jv.job_id = ja.job_id AND jv.user_id = $2
		 WHERE ja.job_id = ANY($1)
		 GROUP BY ja.job_id`,
		pq.Array(ids), userID,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get application counts: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]repository.ApplicationCounts, len(jobIDs))
	for rows.Next() {
		var jobID uuid.UUID
		var total, newCount int
		if err := rows.Scan(&jobID, &total, &newCount); err != nil {
			return nil, fmt.Errorf("scan counts: %w", err)
		}
		result[jobID] = repository.ApplicationCounts{Total: total, NewCount: newCount}
	}
	return result, rows.Err()
}
