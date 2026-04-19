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

// Create inserts a new review row (published_at = NULL). Used by paths
// that do not need the atomic reveal (e.g. tests, seeds). Production
// writes should go through CreateAndMaybeReveal.
func (r *ReviewRepository) Create(ctx context.Context, rv *review.Review) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if _, err := r.db.ExecContext(ctx, queryInsertReview, insertReviewArgs(rv)...); err != nil {
		return fmt.Errorf("insert review: %w", err)
	}
	return nil
}

// CreateAndMaybeReveal is the production write path: INSERT the review
// and, inside the same transaction, flip every pending review on the
// proposal to published_at = NOW() whenever the submission completes
// the pair (or re-veils against a backfilled, already-published
// client→provider review).
func (r *ReviewRepository) CreateAndMaybeReveal(ctx context.Context, rv *review.Review) (*review.Review, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin reveal tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, queryInsertReview, insertReviewArgs(rv)...); err != nil {
		return nil, fmt.Errorf("insert review: %w", err)
	}

	var total, published int
	if err := tx.QueryRowContext(ctx, queryCountReviewsForProposal, rv.ProposalID).Scan(&total, &published); err != nil {
		return nil, fmt.Errorf("count reviews for proposal: %w", err)
	}

	// Reveal when we have both sides (total >= 2) OR when a prior review
	// on this proposal was already published (e.g. a backfilled client
	// review from before the feature shipped — the provider's fresh
	// submission should not re-blind the earlier one).
	shouldReveal := total >= 2 || published >= 1

	if shouldReveal {
		if _, err := tx.ExecContext(ctx, queryRevealPendingReviews, rv.ProposalID); err != nil {
			return nil, fmt.Errorf("reveal pending reviews: %w", err)
		}
	}

	// Re-read the row we just inserted so the returned entity reflects
	// the post-transaction published_at — the caller uses it to decide
	// whether to notify the reviewer about the reveal.
	var refreshed review.Review
	if err := scanReviewRow(tx.QueryRowContext(ctx, queryGetReviewByID, rv.ID), &refreshed); err != nil {
		return nil, fmt.Errorf("reload review after reveal: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit reveal tx: %w", err)
	}

	return &refreshed, nil
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

// ListByReviewedOrganization returns the non-hidden, published client→
// provider reviews received by the given organization, ordered by
// created_at DESC. Before running the SELECT it performs a lazy
// auto-publish sweep against proposals whose 14-day window has
// elapsed — this amortizes the deadline reveal across reads and
// removes the need for a background worker.
func (r *ReviewRepository) ListByReviewedOrganization(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]*review.Review, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Best-effort auto-publish sweep. A failure here is non-fatal —
	// the SELECT still returns what is visible today; the sweep will
	// be retried on the next public read.
	if _, err := r.db.ExecContext(ctx, queryAutoPublishDeadlineElapsed); err != nil {
		return nil, "", fmt.Errorf("auto-publish deadline sweep: %w", err)
	}

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListReviewsByReviewedOrgFirst, orgID, limit+1)
	} else {
		c, decErr := cursor.Decode(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListReviewsByReviewedOrgWithCursor, orgID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list reviews by organization: %w", err)
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

// GetAverageRatingByOrganization returns the aggregated rating for an
// organization (published, non-hidden, client→provider reviews only).
func (r *ReviewRepository) GetAverageRatingByOrganization(ctx context.Context, orgID uuid.UUID) (*review.AverageRating, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var avg float64
	var count int
	err := r.db.QueryRowContext(ctx, queryAverageRatingByOrg, orgID).Scan(&avg, &count)
	if err != nil {
		return nil, fmt.Errorf("get average rating by organization: %w", err)
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

// GetByProposalIDs fetches the published, non-hidden reviews for the
// given proposal IDs in a single query, keyed by proposal ID and
// filtered to the requested side.
//
// A proposal carries up to two reviews (double-blind: one per side).
// Without a side filter the two rows would collide in the result map
// keyed by proposal_id — whichever scan wins last silently overwrites
// the other. All user-facing callers MUST pass an explicit side.
//
// Hidden and unpublished reviews are filtered out at the SQL layer so
// blind submissions cannot leak into public surfaces.
func (r *ReviewRepository) GetByProposalIDs(ctx context.Context, proposalIDs []uuid.UUID, side string) (map[uuid.UUID]*review.Review, error) {
	result := make(map[uuid.UUID]*review.Review, len(proposalIDs))
	if len(proposalIDs) == 0 {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	ids := make([]string, len(proposalIDs))
	for i, id := range proposalIDs {
		ids[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx, queryReviewsByProposalIDs, pq.Array(ids), side)
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

// insertReviewArgs returns the positional args for queryInsertReview in
// the exact order expected by the SQL statement.
func insertReviewArgs(rv *review.Review) []any {
	return []any{
		rv.ID, rv.ProposalID, rv.ReviewerID, rv.ReviewedID,
		rv.ReviewerOrganizationID, rv.ReviewedOrganizationID,
		rv.Side,
		rv.GlobalRating, rv.Timeliness, rv.Communication, rv.Quality,
		rv.Comment, rv.VideoURL, rv.TitleVisible, rv.CreatedAt, rv.UpdatedAt,
	}
}

// scanner interface satisfied by both *sql.Row and *sql.Rows.
type reviewScanner interface {
	Scan(dest ...any) error
}

func scanReview(s reviewScanner) (*review.Review, error) {
	var rv review.Review
	if err := scanReviewRow(s, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func scanReviewRow(s reviewScanner, rv *review.Review) error {
	return s.Scan(
		&rv.ID, &rv.ProposalID, &rv.ReviewerID, &rv.ReviewedID,
		&rv.ReviewerOrganizationID, &rv.ReviewedOrganizationID,
		&rv.Side,
		&rv.GlobalRating, &rv.Timeliness, &rv.Communication, &rv.Quality,
		&rv.Comment, &rv.VideoURL, &rv.TitleVisible, &rv.CreatedAt, &rv.UpdatedAt,
		&rv.PublishedAt,
	)
}

// ListAdmin, CountAdmin, GetAdminByID, DeleteAdmin — admin operations are
// delegated to review_admin.go to keep this repo file focused on the
// org-facing surface.
