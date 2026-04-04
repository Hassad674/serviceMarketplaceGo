package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/port/repository"
)

// ListMedia returns a paginated list of media for admin moderation.
func (s *Service) ListMedia(
	ctx context.Context,
	filters repository.AdminMediaFilters,
) ([]repository.AdminMediaItem, int, error) {
	items, err := s.mediaRepo.ListAdmin(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin media: %w", err)
	}

	count, err := s.mediaRepo.CountAdmin(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin media: %w", err)
	}

	return items, count, nil
}

// GetMedia returns a single media item for admin detail view.
func (s *Service) GetMedia(ctx context.Context, id uuid.UUID) (*mediadomain.Media, error) {
	m, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get admin media: %w", err)
	}
	return m, nil
}

// ApproveMedia marks a media item as approved by the admin.
func (s *Service) ApproveMedia(ctx context.Context, mediaID uuid.UUID, adminID uuid.UUID) error {
	m, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("approve media: %w", err)
	}

	m.Approve(adminID)

	if err := s.mediaRepo.Update(ctx, m); err != nil {
		return fmt.Errorf("approve media: save: %w", err)
	}
	return nil
}

// RejectMedia marks a media item as rejected by the admin.
func (s *Service) RejectMedia(ctx context.Context, mediaID uuid.UUID, adminID uuid.UUID) error {
	m, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("reject media: %w", err)
	}

	m.Reject(adminID)

	if err := s.mediaRepo.Update(ctx, m); err != nil {
		return fmt.Errorf("reject media: save: %w", err)
	}
	return nil
}

// DeleteMedia removes a media record and its file from storage.
func (s *Service) DeleteMedia(ctx context.Context, mediaID uuid.UUID) error {
	m, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("delete media: get: %w", err)
	}

	if s.storageSvc != nil {
		_ = s.storageSvc.Delete(ctx, m.FileURL)
	}

	if err := s.mediaRepo.Delete(ctx, mediaID); err != nil {
		return fmt.Errorf("delete media: remove: %w", err)
	}
	return nil
}
