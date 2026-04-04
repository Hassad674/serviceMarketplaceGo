package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/pkg/cursor"
)

// AdminJob represents a job with author info for admin listing.
type AdminJob struct {
	ID               uuid.UUID
	CreatorID        uuid.UUID
	Title            string
	Description      string
	Skills           []string
	ApplicantType    string
	BudgetType       string
	MinBudget        int
	MaxBudget        int
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ClosedAt         *time.Time
	PaymentFrequency *string
	DurationWeeks    *int
	IsIndefinite     bool
	DescriptionType  string
	VideoURL         *string
	ApplicationCount int

	AuthorDisplayName string
	AuthorEmail       string
	AuthorRole        string
}

// AdminJobApplication represents a job application with candidate and job info.
type AdminJobApplication struct {
	ID          uuid.UUID
	JobID       uuid.UUID
	ApplicantID uuid.UUID
	Message     string
	VideoURL    *string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	CandidateDisplayName string
	CandidateEmail       string
	CandidateRole        string

	JobTitle  string
	JobStatus string
}

// ListJobs returns paginated jobs for admin with author info and application counts.
func (s *Service) ListJobs(ctx context.Context, status, search, sort, cursorStr string, limit int) ([]AdminJob, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	total, err := s.countAdminJobs(ctx, status, search)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list jobs: %w", err)
	}

	jobs, nextCursor, err := s.queryAdminJobs(ctx, status, search, sort, cursorStr, limit)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list jobs: %w", err)
	}

	return jobs, nextCursor, total, nil
}

// GetJob returns a single job with full details for admin.
func (s *Service) GetJob(ctx context.Context, jobID uuid.UUID) (*AdminJob, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var j AdminJob
	var closedAt sql.NullTime
	var paymentFreq, descType, videoURL sql.NullString
	var durationWeeks sql.NullInt32

	err := s.db.QueryRowContext(ctx, queryAdminGetJob, jobID).Scan(
		&j.ID, &j.CreatorID, &j.Title, &j.Description, pq.Array(&j.Skills),
		&j.ApplicantType, &j.BudgetType, &j.MinBudget, &j.MaxBudget,
		&j.Status, &j.CreatedAt, &j.UpdatedAt, &closedAt,
		&paymentFreq, &durationWeeks, &j.IsIndefinite,
		&descType, &videoURL,
		&j.ApplicationCount,
		&j.AuthorDisplayName, &j.AuthorEmail, &j.AuthorRole,
	)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	applyNullables(&j, closedAt, paymentFreq, durationWeeks, descType, videoURL)
	return &j, nil
}

// DeleteJob removes a job by ID (admin action).
func (s *Service) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := s.db.ExecContext(ctx, "DELETE FROM jobs WHERE id = $1", jobID)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete job: check rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("delete job: not found")
	}
	return nil
}

// ListJobApplications returns paginated job applications for admin.
func (s *Service) ListJobApplications(ctx context.Context, jobID, search, sort, cursorStr string, limit int) ([]AdminJobApplication, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	total, err := s.countAdminApplications(ctx, jobID, search)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list applications: %w", err)
	}

	apps, nextCursor, err := s.queryAdminApplications(ctx, jobID, search, sort, cursorStr, limit)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list applications: %w", err)
	}

	return apps, nextCursor, total, nil
}

// DeleteJobApplication removes a job application by ID (admin action).
func (s *Service) DeleteJobApplication(ctx context.Context, applicationID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := s.db.ExecContext(ctx, "DELETE FROM job_applications WHERE id = $1", applicationID)
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete application: check rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("delete application: not found")
	}
	return nil
}

func (s *Service) countAdminJobs(ctx context.Context, status, search string) (int, error) {
	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString("SELECT COUNT(*) FROM jobs")
	where := buildJobWhereClause(&b, &paramIdx, &args, status, search)
	if where {
		// where clause already built
	}

	var total int
	if err := s.db.QueryRowContext(ctx, b.String(), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count jobs: %w", err)
	}
	return total, nil
}

func (s *Service) queryAdminJobs(ctx context.Context, status, search, sort, cursorStr string, limit int) ([]AdminJob, string, error) {
	query, args := buildAdminJobListQuery(status, search, sort, cursorStr, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	return scanAdminJobs(rows, limit)
}

func (s *Service) countAdminApplications(ctx context.Context, jobID, search string) (int, error) {
	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString("SELECT COUNT(*) FROM job_applications ja JOIN users u ON u.id = ja.applicant_id")

	hasWhere := false
	if jobID != "" {
		parsed, err := uuid.Parse(jobID)
		if err == nil {
			fmt.Fprintf(&b, " WHERE ja.job_id = $%d", paramIdx)
			args = append(args, parsed)
			paramIdx++
			hasWhere = true
		}
	}
	if search != "" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
		}
		fmt.Fprintf(&b, " (COALESCE(u.display_name, u.first_name || ' ' || u.last_name) ILIKE $%d OR u.email ILIKE $%d)", paramIdx, paramIdx+1)
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, b.String(), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count applications: %w", err)
	}
	return total, nil
}

func (s *Service) queryAdminApplications(ctx context.Context, jobID, search, sort, cursorStr string, limit int) ([]AdminJobApplication, string, error) {
	query, args := buildAdminApplicationListQuery(jobID, search, sort, cursorStr, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query applications: %w", err)
	}
	defer rows.Close()

	return scanAdminApplications(rows, limit)
}

func scanAdminJobs(rows *sql.Rows, limit int) ([]AdminJob, string, error) {
	var results []AdminJob

	for rows.Next() {
		var j AdminJob
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
			return nil, "", fmt.Errorf("scan job: %w", err)
		}

		applyNullables(&j, closedAt, paymentFreq, durationWeeks, descType, videoURL)
		results = append(results, j)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []AdminJob{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func scanAdminApplications(rows *sql.Rows, limit int) ([]AdminJobApplication, string, error) {
	var results []AdminJobApplication

	for rows.Next() {
		var a AdminJobApplication
		var videoURL sql.NullString

		if err := rows.Scan(
			&a.ID, &a.JobID, &a.ApplicantID, &a.Message, &videoURL,
			&a.CreatedAt, &a.UpdatedAt,
			&a.CandidateDisplayName, &a.CandidateEmail, &a.CandidateRole,
			&a.JobTitle, &a.JobStatus,
		); err != nil {
			return nil, "", fmt.Errorf("scan application: %w", err)
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
		results = []AdminJobApplication{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func applyNullables(j *AdminJob, closedAt sql.NullTime, paymentFreq sql.NullString, durationWeeks sql.NullInt32, descType sql.NullString, videoURL sql.NullString) {
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

// buildJobWhereClause appends WHERE conditions for status and search.
func buildJobWhereClause(b *strings.Builder, paramIdx *int, args *[]any, status, search string) bool {
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

func buildAdminJobListQuery(status, search, sort, cursorStr string, limit int) (string, []any) {
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
	if status != "" {
		fmt.Fprintf(&b, " WHERE j.status = $%d", paramIdx)
		args = append(args, status)
		paramIdx++
		hasWhere = true
	}
	if search != "" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		fmt.Fprintf(&b, " j.title ILIKE $%d", paramIdx)
		args = append(args, "%"+search+"%")
		paramIdx++
	}
	if cursorStr != "" {
		c, err := cursor.Decode(cursorStr)
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

	b.WriteString(adminJobOrderClause(sort))
	fmt.Fprintf(&b, " LIMIT $%d", paramIdx)
	args = append(args, limit+1)

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

func buildAdminApplicationListQuery(jobID, search, sort, cursorStr string, limit int) (string, []any) {
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
	if jobID != "" {
		parsed, err := uuid.Parse(jobID)
		if err == nil {
			fmt.Fprintf(&b, " WHERE ja.job_id = $%d", paramIdx)
			args = append(args, parsed)
			paramIdx++
			hasWhere = true
		}
	}
	if search != "" {
		if hasWhere {
			b.WriteString(" AND")
		} else {
			b.WriteString(" WHERE")
			hasWhere = true
		}
		fmt.Fprintf(&b, " (COALESCE(u.display_name, u.first_name || ' ' || u.last_name) ILIKE $%d OR u.email ILIKE $%d)", paramIdx, paramIdx+1)
		args = append(args, "%"+search+"%", "%"+search+"%")
		paramIdx += 2
	}
	if cursorStr != "" {
		c, err := cursor.Decode(cursorStr)
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

	b.WriteString(adminApplicationOrderClause(sort))
	fmt.Fprintf(&b, " LIMIT $%d", paramIdx)
	args = append(args, limit+1)

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
