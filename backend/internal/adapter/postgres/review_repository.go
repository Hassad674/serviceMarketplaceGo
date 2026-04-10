package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/review"
	"marketplace-backend/pkg/cursor"
)

// ReviewRepository implements repository.ReviewRepository using PostgreSQL.
type ReviewRepository struct {
	db *sql.DB
}

// NewReviewRepository creates a new PostgreSQL-backed review repository.
func NewReviewRepository(db *sql.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, rv *review.Review) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertReview,
		rv.ID, rv.ProposalID, rv.ReviewerID, rv.ReviewedID,
		rv.GlobalRating, rv.Timeliness, rv.Communication, rv.Quality,
		rv.Comment, rv.VideoURL, rv.TitleVisible, rv.CreatedAt, rv.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert review: %w", err)
	}
	return nil
}

func (r *ReviewRepository) GetByID(ctx context.Context, id uuid.UUID) (*review.Review, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rv, err := scanReview(r.db.QueryRowContext(ctx, queryGetReviewByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, review.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get review by id: %w", err)
	}
	return rv, nil
}

func (r *ReviewRepository) ListByReviewedUser(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*review.Review, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListReviewsByReviewedFirst, userID, limit+1)
	} else {
		c, decErr := cursor.Decode(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListReviewsByReviewedWithCursor, userID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*review.Review
	for rows.Next() {
		rv, scanErr := scanReview(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("scan review: %w", scanErr)
		}
		reviews = append(reviews, rv)
	}

	var nextCursor string
	if len(reviews) > limit {
		last := reviews[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		reviews = reviews[:limit]
	}

	return reviews, nextCursor, nil
}

func (r *ReviewRepository) GetAverageRating(ctx context.Context, userID uuid.UUID) (*review.AverageRating, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var avg float64
	var count int
	err := r.db.QueryRowContext(ctx, queryAverageRating, userID).Scan(&avg, &count)
	if err != nil {
		return nil, fmt.Errorf("get average rating: %w", err)
	}
	return &review.AverageRating{Average: avg, Count: count}, nil
}

func (r *ReviewRepository) UpdateReviewModeration(ctx context.Context, reviewID uuid.UUID, status string, score float64, labelsJSON []byte) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpdateReviewModeration, reviewID, status, score, labelsJSON)
	if err != nil {
		return fmt.Errorf("update review moderation: %w", err)
	}
	return nil
}

func (r *ReviewRepository) HasReviewed(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var exists bool
	err := r.db.QueryRowContext(ctx, queryHasReviewed, proposalID, reviewerID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check has reviewed: %w", err)
	}
	return exists, nil
}

// GetByProposalIDs fetches all (non-hidden) reviews for the given proposal IDs
// in a single query, and returns them keyed by proposal ID.
func (r *ReviewRepository) GetByProposalIDs(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID]*review.Review, error) {
	result := make(map[uuid.UUID]*review.Review, len(proposalIDs))
	if len(proposalIDs) == 0 {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Convert uuid.UUID slice to string slice for pq.Array.
	ids := make([]string, len(proposalIDs))
	for i, id := range proposalIDs {
		ids[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx, queryReviewsByProposalIDs, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("reviews by proposal ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		rv, scanErr := scanReview(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan review: %w", scanErr)
		}
		result[rv.ProposalID] = rv
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return result, nil
}

// scanner interface satisfied by both *sql.Row and *sql.Rows.
type reviewScanner interface {
	Scan(dest ...any) error
}

func scanReview(s reviewScanner) (*review.Review, error) {
	var rv review.Review
	err := s.Scan(
		&rv.ID, &rv.ProposalID, &rv.ReviewerID, &rv.ReviewedID,
		&rv.GlobalRating, &rv.Timeliness, &rv.Communication, &rv.Quality,
		&rv.Comment, &rv.VideoURL, &rv.TitleVisible, &rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}
