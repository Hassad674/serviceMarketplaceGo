package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/port/repository"
)

// ListReviews returns a paginated list of reviews with user info for admin.
func (s *Service) ListReviews(ctx context.Context, filters repository.AdminReviewFilters) ([]repository.AdminReview, int, error) {
	items, err := s.reviews.ListAdmin(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin reviews: %w", err)
	}

	count, err := s.reviews.CountAdmin(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin reviews: %w", err)
	}

	reportCounts, err := s.loadReviewPendingReportCounts(ctx, items)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin reviews: %w", err)
	}
	for i := range items {
		items[i].PendingReportCount = reportCounts[items[i].ID]
	}

	return items, count, nil
}

// GetReview returns a single review with user info for admin detail view.
func (s *Service) GetReview(ctx context.Context, id uuid.UUID) (*repository.AdminReview, error) {
	item, err := s.reviews.GetAdminByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get admin review: %w", err)
	}
	return item, nil
}

// DeleteReview removes a review by ID (admin action).
func (s *Service) DeleteReview(ctx context.Context, id uuid.UUID) error {
	if err := s.reviews.DeleteAdmin(ctx, id); err != nil {
		return fmt.Errorf("delete admin review: %w", err)
	}
	return nil
}

// ListReviewReports returns all reports targeting a specific review.
func (s *Service) ListReviewReports(ctx context.Context, reviewID uuid.UUID) ([]*report.Report, error) {
	reports, err := s.reports.ListByTarget(ctx, string(report.TargetReview), reviewID)
	if err != nil {
		return nil, fmt.Errorf("list review reports: %w", err)
	}
	return reports, nil
}

// ApproveReviewModeration clears the moderation flag on a review, marking it clean.
func (s *Service) ApproveReviewModeration(ctx context.Context, reviewID uuid.UUID) error {
	if err := s.reviews.UpdateReviewModeration(ctx, reviewID, "clean", 0, nil); err != nil {
		return fmt.Errorf("approve review moderation: %w", err)
	}
	return nil
}

func (s *Service) loadReviewPendingReportCounts(ctx context.Context, reviews []repository.AdminReview) (map[uuid.UUID]int, error) {
	if len(reviews) == 0 {
		return make(map[uuid.UUID]int), nil
	}

	ids := make([]uuid.UUID, len(reviews))
	for i, rv := range reviews {
		ids[i] = rv.ID
	}

	return s.reports.PendingCountsByTargets(ctx, "review", ids)
}
