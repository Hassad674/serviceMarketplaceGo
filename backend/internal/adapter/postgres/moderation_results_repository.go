package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/port/repository"
)

// ModerationResultsRepository is the PostgreSQL adapter for the
// moderation_results table introduced in migration 120. The table
// holds the latest moderation verdict per (content_type, content_id)
// pair — historical transitions live in audit_logs.
//
// Implementation notes:
//   - Upsert uses ON CONFLICT (content_type, content_id) DO UPDATE so
//     re-moderating an edited message does not require a separate
//     "exists?" query.
//   - List allows callers to filter by content_type, status and
//     author_user_id. Sort/limit/offset enable the admin queue
//     pagination. The total count is computed without limit/offset so
//     the UI can render a "X results" header.
//   - MarkReviewed is a single UPDATE that captures the admin's
//     override + identity in one round-trip; the previous status is
//     preserved in audit_logs by the calling app service.
type ModerationResultsRepository struct {
	db *sql.DB
}

// NewModerationResultsRepository wires the adapter to a database
// handle. The handle is reused across calls, matching the pattern of
// every other postgres adapter in the project.
func NewModerationResultsRepository(db *sql.DB) *ModerationResultsRepository {
	return &ModerationResultsRepository{db: db}
}

// maxListLimit caps List queries so a misconfigured admin page cannot
// pull the entire table in one shot. 100 mirrors the project-wide
// pagination cap from CLAUDE.md.
const maxListLimit = 100

func (r *ModerationResultsRepository) Upsert(ctx context.Context, result *moderation.Result) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpsertModerationResult,
		result.ID,
		string(result.ContentType),
		result.ContentID,
		result.AuthorUserID,
		string(result.Status),
		result.Score,
		result.Labels,
		result.Reason,
		result.DecidedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert moderation result: %w", err)
	}
	return nil
}

func (r *ModerationResultsRepository) GetByContent(
	ctx context.Context,
	contentType moderation.ContentType,
	contentID uuid.UUID,
) (*moderation.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, queryGetModerationResultByContent, string(contentType), contentID)

	out, err := scanModerationResult(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, moderation.ErrResultNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get moderation result: %w", err)
	}
	return out, nil
}

func (r *ModerationResultsRepository) List(
	ctx context.Context,
	filters repository.ModerationResultsFilters,
) ([]*moderation.Result, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := filters.Limit
	if limit <= 0 || limit > maxListLimit {
		limit = 20
	}

	whereSQL, args := buildModerationResultsWhere(filters)
	orderSQL := buildModerationResultsOrder(filters.Sort)

	listSQL := buildModerationResultsListQuery(whereSQL, orderSQL, limit, filters.Offset, len(args))
	rows, err := r.db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list moderation results: %w", err)
	}
	defer rows.Close()

	out := make([]*moderation.Result, 0, limit)
	for rows.Next() {
		item, scanErr := scanModerationResult(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan moderation result: %w", scanErr)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate moderation results: %w", err)
	}

	countSQL := buildModerationResultsCountQuery(whereSQL)
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count moderation results: %w", err)
	}

	return out, total, nil
}

func (r *ModerationResultsRepository) MarkReviewed(
	ctx context.Context,
	contentType moderation.ContentType,
	contentID uuid.UUID,
	reviewerID uuid.UUID,
	newStatus moderation.Status,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	res, err := r.db.ExecContext(ctx, queryMarkModerationReviewed,
		string(newStatus), reviewerID, string(contentType), contentID,
	)
	if err != nil {
		return fmt.Errorf("mark moderation reviewed: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return moderation.ErrResultNotFound
	}
	return nil
}

// rowScanner abstracts *sql.Row vs *sql.Rows so scanModerationResult
// can be reused by both Get and List code paths.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanModerationResult(row rowScanner) (*moderation.Result, error) {
	var (
		out         moderation.Result
		contentType string
		status      string
		labelsRaw   sql.NullString
		authorID    uuid.NullUUID
		reviewerID  uuid.NullUUID
		reviewedAt  sql.NullTime
	)
	err := row.Scan(
		&out.ID,
		&contentType,
		&out.ContentID,
		&authorID,
		&status,
		&out.Score,
		&labelsRaw,
		&out.Reason,
		&out.DecidedAt,
		&reviewerID,
		&reviewedAt,
	)
	if err != nil {
		return nil, err
	}
	out.ContentType = moderation.ContentType(contentType)
	out.Status = moderation.Status(status)
	if labelsRaw.Valid {
		out.Labels = []byte(labelsRaw.String)
	} else {
		out.Labels = []byte("[]")
	}
	if authorID.Valid {
		v := authorID.UUID
		out.AuthorUserID = &v
	}
	if reviewerID.Valid {
		v := reviewerID.UUID
		out.ReviewedBy = &v
	}
	if reviewedAt.Valid {
		v := reviewedAt.Time
		out.ReviewedAt = &v
	}
	return &out, nil
}

// buildModerationResultsWhere assembles the parameterised WHERE clause
// from a filter struct. The returned []any is positional ($1, $2, …)
// to match the SQL placeholders. Pure function for testability.
func buildModerationResultsWhere(f repository.ModerationResultsFilters) (string, []any) {
	var (
		clauses []string
		args    []any
		idx     = 1
	)
	if f.ContentType != "" {
		clauses = append(clauses, fmt.Sprintf("content_type = $%d", idx))
		args = append(args, f.ContentType)
		idx++
	}
	if f.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.AuthorUserID != nil {
		clauses = append(clauses, fmt.Sprintf("author_user_id = $%d", idx))
		args = append(args, *f.AuthorUserID)
		idx++
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func buildModerationResultsOrder(sort string) string {
	switch sort {
	case "oldest":
		return "ORDER BY decided_at ASC, id ASC"
	case "score":
		return "ORDER BY score DESC, decided_at DESC"
	default:
		return "ORDER BY decided_at DESC, id DESC"
	}
}
