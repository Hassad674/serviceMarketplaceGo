package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

func (r *JobApplicationRepository) ListAdmin(ctx context.Context, filters repository.AdminApplicationFilters) ([]repository.AdminJobApplication, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	useOffset := filters.Page > 0 && filters.Cursor == ""
	query, args := buildAdminApplicationListQuery(filters, useOffset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query admin applications: %w", err)
	}
	defer rows.Close()

	return scanAdminApplications(rows, filters.Limit)
}

func (r *JobApplicationRepository) CountAdmin(ctx context.Context, filters repository.AdminApplicationFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString("SELECT COUNT(*) FROM job_applications ja JOIN users u ON u.id = ja.applicant_id")

	hasWhere := false
	if filters.JobID != "" {
		parsed, err := uuid.Parse(filters.JobID)
		if err == nil {
			fmt.Fprintf(&b, " WHERE ja.job_id = $%d", paramIdx)
			args = append(args, parsed)
			paramIdx++
			hasWhere = true
		}
	}
	if filters.Search != "" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		fmt.Fprintf(&b, " (COALESCE(u.display_name, u.first_name || ' ' || u.last_name) ILIKE $%d OR u.email ILIKE $%d)", paramIdx, paramIdx+1)
		args = append(args, "%"+filters.Search+"%", "%"+filters.Search+"%")
		paramIdx += 2
	}
	if filters.Filter == "reported" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
		}
		b.WriteString(` EXISTS (SELECT 1 FROM reports r WHERE r.target_type = 'job_application' AND r.target_id = ja.id AND r.status = 'pending')`)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, b.String(), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count admin applications: %w", err)
	}
	return total, nil
}

func buildAdminApplicationListQuery(filters repository.AdminApplicationFilters, useOffset bool) (string, []any) {
	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString(`SELECT
		ja.id, ja.job_id, ja.applicant_id, ja.message, ja.video_url,
		ja.created_at, ja.updated_at,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name),
		u.email, u.role,
		j.title, j.status
	FROM job_applications ja
	JOIN users u ON u.id = ja.applicant_id
	JOIN jobs j ON j.id = ja.job_id`)

	hasWhere := false
	if filters.JobID != "" {
		parsed, err := uuid.Parse(filters.JobID)
		if err == nil {
			fmt.Fprintf(&b, " WHERE ja.job_id = $%d", paramIdx)
			args = append(args, parsed)
			paramIdx++
			hasWhere = true
		}
	}
	if filters.Search != "" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		fmt.Fprintf(&b, " (COALESCE(u.display_name, u.first_name || ' ' || u.last_name) ILIKE $%d OR u.email ILIKE $%d)", paramIdx, paramIdx+1)
		args = append(args, "%"+filters.Search+"%", "%"+filters.Search+"%")
		paramIdx += 2
	}
	if filters.Filter == "reported" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		b.WriteString(` EXISTS (SELECT 1 FROM reports r WHERE r.target_type = 'job_application' AND r.target_id = ja.id AND r.status = 'pending')`)
	}
	if !useOffset && filters.Cursor != "" {
		c, err := cursor.Decode(filters.Cursor)
		if err == nil {
			if hasWhere {
				b.WriteString(" AND")
			} else {
				b.WriteString(" WHERE")
			}
			fmt.Fprintf(&b, " (ja.created_at, ja.id) < ($%d, $%d)", paramIdx, paramIdx+1)
			args = append(args, c.CreatedAt, c.ID)
			paramIdx += 2
		}
	}

	b.WriteString(adminApplicationOrderClause(filters.Sort))
	fmt.Fprintf(&b, " LIMIT $%d", paramIdx)
	args = append(args, filters.Limit+1)
	paramIdx++

	if useOffset {
		fmt.Fprintf(&b, " OFFSET $%d", paramIdx)
		args = append(args, (filters.Page-1)*filters.Limit)
	}

	return b.String(), args
}

func adminApplicationOrderClause(sort string) string {
	switch sort {
	case "oldest":
		return " ORDER BY ja.created_at ASC, ja.id ASC"
	default:
		return " ORDER BY ja.created_at DESC, ja.id DESC"
	}
}

func scanAdminApplications(rows *sql.Rows, limit int) ([]repository.AdminJobApplication, string, error) {
	var results []repository.AdminJobApplication

	for rows.Next() {
		var a repository.AdminJobApplication
		var videoURL sql.NullString

		if err := rows.Scan(
			&a.ID, &a.JobID, &a.ApplicantID, &a.Message, &videoURL,
			&a.CreatedAt, &a.UpdatedAt,
			&a.CandidateDisplayName, &a.CandidateEmail, &a.CandidateRole,
			&a.JobTitle, &a.JobStatus,
		); err != nil {
			return nil, "", fmt.Errorf("scan admin application: %w", err)
		}

		if videoURL.Valid {
			a.VideoURL = &videoURL.String
		}
		results = append(results, a)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []repository.AdminJobApplication{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}
