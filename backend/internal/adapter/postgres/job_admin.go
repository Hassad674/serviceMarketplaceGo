package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

func (r *JobRepository) ListAdmin(ctx context.Context, filters repository.AdminJobFilters) ([]repository.AdminJob, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	useOffset := filters.Page > 0 && filters.Cursor == ""
	query, args := buildAdminJobListQuery(filters, useOffset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query admin jobs: %w", err)
	}
	defer rows.Close()

	return scanAdminJobs(rows, filters.Limit)
}

func (r *JobRepository) CountAdmin(ctx context.Context, filters repository.AdminJobFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString("SELECT COUNT(*) FROM jobs")
	hasWhere := appendJobWhereClause(&b, &paramIdx, &args, filters.Status, filters.Search)
	if filters.Filter == "reported" {
		appendCondition(&b, hasWhere, ` EXISTS (SELECT 1 FROM reports r WHERE r.target_type = 'job' AND r.target_id = jobs.id AND r.status = 'pending')`)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, b.String(), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count admin jobs: %w", err)
	}
	return total, nil
}

func (r *JobRepository) GetAdmin(ctx context.Context, id uuid.UUID) (*repository.AdminJob, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var j repository.AdminJob
	var closedAt sql.NullTime
	var paymentFreq, descType, videoURL sql.NullString
	var durationWeeks sql.NullInt32

	err := r.db.QueryRowContext(ctx, queryAdminGetJob, id).Scan(
		&j.ID, &j.CreatorID, &j.Title, &j.Description, pq.Array(&j.Skills),
		&j.ApplicantType, &j.BudgetType, &j.MinBudget, &j.MaxBudget,
		&j.Status, &j.CreatedAt, &j.UpdatedAt, &closedAt,
		&paymentFreq, &durationWeeks, &j.IsIndefinite,
		&descType, &videoURL,
		&j.ApplicationCount,
		&j.AuthorDisplayName, &j.AuthorEmail, &j.AuthorRole,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin job: %w", err)
	}

	applyAdminJobNullables(&j, closedAt, paymentFreq, durationWeeks, descType, videoURL)
	return &j, nil
}

func (r *JobRepository) CountAll(ctx context.Context) (total int, open int, err error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs").Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("count total jobs: %w", err)
	}

	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE status = 'open'").Scan(&open)
	if err != nil {
		return 0, 0, fmt.Errorf("count open jobs: %w", err)
	}
	return total, open, nil
}

func buildAdminJobListQuery(filters repository.AdminJobFilters, useOffset bool) (string, []any) {
	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString(`SELECT
		j.id, j.creator_id, j.title, j.description, j.skills,
		j.applicant_type, j.budget_type, j.min_budget, j.max_budget,
		j.status, j.created_at, j.updated_at, j.closed_at,
		j.payment_frequency, j.duration_weeks, j.is_indefinite,
		j.description_type, j.video_url,
		(SELECT COUNT(*) FROM job_applications WHERE job_id = j.id) AS application_count,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name),
		u.email, u.role
	FROM jobs j
	JOIN users u ON u.id = j.creator_id`)

	hasWhere := false
	if filters.Status != "" {
		fmt.Fprintf(&b, " WHERE j.status = $%d", paramIdx)
		args = append(args, filters.Status)
		paramIdx++
		hasWhere = true
	}
	if filters.Search != "" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		fmt.Fprintf(&b, " j.title ILIKE $%d", paramIdx)
		args = append(args, "%"+filters.Search+"%")
		paramIdx++
	}
	if filters.Filter == "reported" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		b.WriteString(` EXISTS (SELECT 1 FROM reports r WHERE r.target_type = 'job' AND r.target_id = j.id AND r.status = 'pending')`)
	}
	if !useOffset && filters.Cursor != "" {
		c, err := cursor.Decode(filters.Cursor)
		if err == nil {
			if hasWhere {
				b.WriteString(" AND")
			} else {
				b.WriteString(" WHERE")
			}
			fmt.Fprintf(&b, " (j.created_at, j.id) < ($%d, $%d)", paramIdx, paramIdx+1)
			args = append(args, c.CreatedAt, c.ID)
			paramIdx += 2
		}
	}

	b.WriteString(adminJobOrderClause(filters.Sort))
	fmt.Fprintf(&b, " LIMIT $%d", paramIdx)
	args = append(args, filters.Limit+1)
	paramIdx++

	if useOffset {
		fmt.Fprintf(&b, " OFFSET $%d", paramIdx)
		args = append(args, (filters.Page-1)*filters.Limit)
	}

	return b.String(), args
}

func adminJobOrderClause(sort string) string {
	switch sort {
	case "oldest":
		return " ORDER BY j.created_at ASC, j.id ASC"
	case "title":
		return " ORDER BY j.title ASC, j.id ASC"
	case "budget":
		return " ORDER BY j.max_budget DESC, j.id DESC"
	default:
		return " ORDER BY j.created_at DESC, j.id DESC"
	}
}

func scanAdminJobs(rows *sql.Rows, limit int) ([]repository.AdminJob, string, error) {
	var results []repository.AdminJob

	for rows.Next() {
		var j repository.AdminJob
		var closedAt sql.NullTime
		var paymentFreq, descType, videoURL sql.NullString
		var durationWeeks sql.NullInt32

		if err := rows.Scan(
			&j.ID, &j.CreatorID, &j.Title, &j.Description, pq.Array(&j.Skills),
			&j.ApplicantType, &j.BudgetType, &j.MinBudget, &j.MaxBudget,
			&j.Status, &j.CreatedAt, &j.UpdatedAt, &closedAt,
			&paymentFreq, &durationWeeks, &j.IsIndefinite,
			&descType, &videoURL,
			&j.ApplicationCount,
			&j.AuthorDisplayName, &j.AuthorEmail, &j.AuthorRole,
		); err != nil {
			return nil, "", fmt.Errorf("scan admin job: %w", err)
		}

		applyAdminJobNullables(&j, closedAt, paymentFreq, durationWeeks, descType, videoURL)
		results = append(results, j)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []repository.AdminJob{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func applyAdminJobNullables(j *repository.AdminJob, closedAt sql.NullTime, paymentFreq sql.NullString, durationWeeks sql.NullInt32, descType sql.NullString, videoURL sql.NullString) {
	if closedAt.Valid {
		j.ClosedAt = &closedAt.Time
	}
	if paymentFreq.Valid {
		j.PaymentFrequency = &paymentFreq.String
	}
	if durationWeeks.Valid {
		w := int(durationWeeks.Int32)
		j.DurationWeeks = &w
	}
	if descType.Valid {
		j.DescriptionType = descType.String
	} else {
		j.DescriptionType = "text"
	}
	if videoURL.Valid {
		j.VideoURL = &videoURL.String
	}
}

// appendJobWhereClause appends WHERE conditions for status and search.
func appendJobWhereClause(b *strings.Builder, paramIdx *int, args *[]any, status, search string) bool {
	hasWhere := false

	if status != "" {
		b.WriteString(" WHERE status = $1")
		*args = append(*args, status)
		*paramIdx = 2
		hasWhere = true
	}
	if search != "" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		fmt.Fprintf(b, " title ILIKE $%d", *paramIdx)
		*args = append(*args, "%"+search+"%")
		*paramIdx++
	}

	return hasWhere
}

// appendCondition appends a WHERE or AND condition to the builder.
func appendCondition(b *strings.Builder, hasWhere bool, condition string) {
	if hasWhere {
		b.WriteString(" AND")
	} else {
		b.WriteString(" WHERE")
	}
	b.WriteString(condition)
}

const queryAdminGetJob = `
	SELECT
		j.id, j.creator_id, j.title, j.description, j.skills,
		j.applicant_type, j.budget_type, j.min_budget, j.max_budget,
		j.status, j.created_at, j.updated_at, j.closed_at,
		j.payment_frequency, j.duration_weeks, j.is_indefinite,
		j.description_type, j.video_url,
		(SELECT COUNT(*) FROM job_applications WHERE job_id = j.id) AS application_count,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name),
		u.email, u.role
	FROM jobs j
	JOIN users u ON u.id = j.creator_id
	WHERE j.id = $1`
