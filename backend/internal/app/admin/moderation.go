package admin

import (
	"context"
	"fmt"

	"marketplace-backend/internal/port/repository"
)

// ListModerationItems returns a paginated list of unified moderation items.
func (s *Service) ListModerationItems(ctx context.Context, filters repository.ModerationFilters) ([]repository.ModerationItem, int, error) {
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.Page < 1 {
		filters.Page = 1
	}

	items, err := s.moderationRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("list moderation items: %w", err)
	}

	count, err := s.moderationRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("count moderation items: %w", err)
	}

	return items, count, nil
}

// ModerationPendingCount returns the total number of pending items across all sources.
func (s *Service) ModerationPendingCount(ctx context.Context) (int, error) {
	count, err := s.moderationRepo.PendingCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("moderation pending count: %w", err)
	}
	return count, nil
}
