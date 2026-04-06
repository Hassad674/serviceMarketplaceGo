package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
)

// ListAdmin returns a paginated list of reviews with reviewer + reviewed user info.
func (r *ReviewRepository) ListAdmin(ctx context.Context, filters repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query, args := buildAdminReviewListQuery(filters)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin reviews: %w", err)
	}
	defer rows.Close()

	return scanAdminReviews(rows, filters.Limit)
}

// CountAdmin returns the total count of reviews matching the given filters.
func (r *ReviewRepository) CountAdmin(ctx context.Context, filters repository.AdminReviewFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query, args := buildAdminReviewCountQuery(filters)

	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count admin reviews: %w", err)
	}
	return total, nil
}

// GetAdminByID returns a single review with reviewer + reviewed user info.
func (r *ReviewRepository) GetAdminByID(ctx context.Context, id uuid.UUID) (*repository.AdminReview, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, queryAdminGetReview, id)
	ar, err := scanSingleAdminReview(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, review.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get admin review: %w", err)
	}
	return ar, nil
}

// DeleteAdmin removes a review by ID (admin action).
func (r *ReviewRepository) DeleteAdmin(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, "DELETE FROM reviews WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete review: check rows: %w", err)
	}
	if rows == 0 {
		return review.ErrNotFound
	}
	return nil
}

func scanAdminReviews(rows *sql.Rows, limit int) ([]repository.AdminReview, error) {
	var results []repository.AdminReview

	for rows.Next() {
		ar, err := scanAdminReviewRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan admin review: %w", err)
		}
		results = append(results, *ar)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []repository.AdminReview{}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func scanAdminReviewRow(s reviewScanner) (*repository.AdminReview, error) {
	var ar repository.AdminReview
	err := s.Scan(
		&ar.ID, &ar.ProposalID, &ar.ReviewerID, &ar.ReviewedID,
		&ar.GlobalRating, &ar.Timeliness, &ar.Communication, &ar.Quality,
		&ar.Comment, &ar.VideoURL, &ar.CreatedAt, &ar.UpdatedAt,
		&ar.ReviewerDisplayName, &ar.ReviewerEmail, &ar.ReviewerRole,
		&ar.ReviewedDisplayName, &ar.ReviewedEmail, &ar.ReviewedRole,
	)
	if err != nil {
		return nil, err
	}
	return &ar, nil
}

func scanSingleAdminReview(row *sql.Row) (*repository.AdminReview, error) {
	var ar repository.AdminReview
	err := row.Scan(
		&ar.ID, &ar.ProposalID, &ar.ReviewerID, &ar.ReviewedID,
		&ar.GlobalRating, &ar.Timeliness, &ar.Communication, &ar.Quality,
		&ar.Comment, &ar.VideoURL, &ar.CreatedAt, &ar.UpdatedAt,
		&ar.ReviewerDisplayName, &ar.ReviewerEmail, &ar.ReviewerRole,
		&ar.ReviewedDisplayName, &ar.ReviewedEmail, &ar.ReviewedRole,
	)
	if err != nil {
		return nil, err
	}
	return &ar, nil
}

func buildAdminReviewListQuery(filters repository.AdminReviewFilters) (string, []any) {
	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString(baseAdminReviewSelect)

	hasWhere := appendReviewWhereClause(&b, &paramIdx, &args, filters)

	if filters.Filter == "reported" {
		appendAndOrWhere(&b, hasWhere)
		b.WriteString(` EXISTS (SELECT 1 FROM reports rp WHERE rp.target_type = 'review' AND rp.target_id = rv.id AND rp.status = 'pending')`)
	}

	b.WriteString(adminReviewOrderClause(filters.Sort))

	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	fmt.Fprintf(&b, " LIMIT $%d", paramIdx)
	args = append(args, limit+1)
	paramIdx++

	if filters.Page > 0 {
		fmt.Fprintf(&b, " OFFSET $%d", paramIdx)
		args = append(args, (filters.Page-1)*limit)
	}

	return b.String(), args
}

func buildAdminReviewCountQuery(filters repository.AdminReviewFilters) (string, []any) {
	var b strings.Builder
	args := []any{}
	paramIdx := 1

	b.WriteString(`SELECT COUNT(*) FROM reviews rv
		JOIN users reviewer ON reviewer.id = rv.reviewer_id
		JOIN users reviewed ON reviewed.id = rv.reviewed_id`)

	hasWhere := appendReviewWhereClause(&b, &paramIdx, &args, filters)

	if filters.Filter == "reported" {
		appendAndOrWhere(&b, hasWhere)
		b.WriteString(` EXISTS (SELECT 1 FROM reports rp WHERE rp.target_type = 'review' AND rp.target_id = rv.id AND rp.status = 'pending')`)
	}

	return b.String(), args
}

func appendReviewWhereClause(b *strings.Builder, paramIdx *int, args *[]any, filters repository.AdminReviewFilters) bool {
	hasWhere := false

	if filters.Search != "" {
		b.WriteString(" WHERE")
		hasWhere = true
		fmt.Fprintf(b, ` (COALESCE(reviewer.display_name, reviewer.first_name || ' ' || reviewer.last_name) ILIKE $%d OR reviewer.email ILIKE $%d OR COALESCE(reviewed.display_name, reviewed.first_name || ' ' || reviewed.last_name) ILIKE $%d OR reviewed.email ILIKE $%d)`,
			*paramIdx, *paramIdx+1, *paramIdx+2, *paramIdx+3)
		search := "%" + filters.Search + "%"
		*args = append(*args, search, search, search, search)
		*paramIdx += 4
	}

	if filters.Rating > 0 && filters.Rating <= 5 {
		appendAndOrWhere(b, hasWhere)
		hasWhere = true
		fmt.Fprintf(b, " rv.global_rating = $%d", *paramIdx)
		*args = append(*args, filters.Rating)
		*paramIdx++
	}

	return hasWhere
}

func appendAndOrWhere(b *strings.Builder, hasWhere bool) {
	if hasWhere {
		b.WriteString(" AND")
	} else {
		b.WriteString(" WHERE")
	}
}

func adminReviewOrderClause(sort string) string {
	switch sort {
	case "oldest":
		return " ORDER BY rv.created_at ASC, rv.id ASC"
	case "rating_high":
		return " ORDER BY rv.global_rating DESC, rv.created_at DESC, rv.id DESC"
	case "rating_low":
		return " ORDER BY rv.global_rating ASC, rv.created_at DESC, rv.id DESC"
	default:
		return " ORDER BY rv.created_at DESC, rv.id DESC"
	}
}

const baseAdminReviewSelect = `SELECT
	rv.id, rv.proposal_id, rv.reviewer_id, rv.reviewed_id,
	rv.global_rating, rv.timeliness, rv.communication, rv.quality,
	rv.comment, rv.video_url, rv.created_at, rv.updated_at,
	COALESCE(reviewer.display_name, reviewer.first_name || ' ' || reviewer.last_name),
	reviewer.email, reviewer.role,
	COALESCE(reviewed.display_name, reviewed.first_name || ' ' || reviewed.last_name),
	reviewed.email, reviewed.role
FROM reviews rv
JOIN users reviewer ON reviewer.id = rv.reviewer_id
JOIN users reviewed ON reviewed.id = rv.reviewed_id`

const queryAdminGetReview = baseAdminReviewSelect + `
WHERE rv.id = $1`
