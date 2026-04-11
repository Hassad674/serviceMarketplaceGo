package portfolio

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/portfolio"
	"marketplace-backend/internal/port/repository"
)

// ServiceDeps groups the dependencies for the portfolio service.
type ServiceDeps struct {
	Portfolios repository.PortfolioRepository
}

// Service orchestrates portfolio use cases.
type Service struct {
	portfolios repository.PortfolioRepository
}

// NewService creates a new portfolio service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		portfolios: deps.Portfolios,
	}
}

// CreateItemInput holds data for creating a portfolio item.
type CreateItemInput struct {
	OrganizationID uuid.UUID
	Title          string
	Description    string
	LinkURL        string
	Position       int
	Media          []MediaInput
}

// MediaInput describes a single media attachment.
type MediaInput struct {
	MediaURL     string
	MediaType    string
	ThumbnailURL string // Optional custom thumbnail (videos only)
	Position     int
}

// CreateItem creates a new portfolio item after validating limits.
func (s *Service) CreateItem(ctx context.Context, in CreateItemInput) (*domain.PortfolioItem, error) {
	count, err := s.portfolios.CountByOrganization(ctx, in.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("count items: %w", err)
	}
	if count >= domain.MaxItemsPerOrg {
		return nil, domain.ErrTooManyItems
	}

	mediaInputs := make([]domain.NewMediaInput, len(in.Media))
	for i, m := range in.Media {
		mediaInputs[i] = domain.NewMediaInput{
			MediaURL:     m.MediaURL,
			MediaType:    domain.MediaType(m.MediaType),
			ThumbnailURL: m.ThumbnailURL,
			Position:     m.Position,
		}
	}

	item, err := domain.NewPortfolioItem(domain.NewItemInput{
		OrganizationID: in.OrganizationID,
		Title:          in.Title,
		Description:    in.Description,
		LinkURL:        in.LinkURL,
		Position:       in.Position,
		Media:          mediaInputs,
	})
	if err != nil {
		return nil, err
	}

	if err := s.portfolios.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}

	return item, nil
}

// GetByID returns a portfolio item by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.PortfolioItem, error) {
	return s.portfolios.GetByID(ctx, id)
}

// ListByOrganization returns portfolio items for an org with pagination.
func (s *Service) ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.PortfolioItem, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.portfolios.ListByOrganization(ctx, orgID, cursor, limit)
}

// UpdateItemInput holds data for updating a portfolio item.
type UpdateItemInput struct {
	Title       *string
	Description *string
	LinkURL     *string
	Media       []MediaInput // nil = no media change
}

// UpdateItem updates a portfolio item after ownership verification.
// The item must belong to the caller's organization.
func (s *Service) UpdateItem(ctx context.Context, orgID, itemID uuid.UUID, in UpdateItemInput) (*domain.PortfolioItem, error) {
	item, err := s.portfolios.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item.OrganizationID != orgID {
		return nil, domain.ErrNotOwner
	}

	title := item.Title
	if in.Title != nil {
		title = *in.Title
	}
	desc := item.Description
	if in.Description != nil {
		desc = *in.Description
	}
	linkURL := item.LinkURL
	if in.LinkURL != nil {
		linkURL = *in.LinkURL
	}

	if err := item.UpdateItem(title, desc, linkURL); err != nil {
		return nil, err
	}

	if err := s.portfolios.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}

	// Replace media if provided.
	if in.Media != nil {
		media := make([]*domain.PortfolioMedia, 0, len(in.Media))
		now := time.Now()
		for _, m := range in.Media {
			mt := domain.MediaType(m.MediaType)
			if !mt.IsValid() {
				return nil, domain.ErrInvalidMediaType
			}
			if m.MediaURL == "" {
				return nil, domain.ErrMissingMediaURL
			}
			media = append(media, &domain.PortfolioMedia{
				ID:              uuid.New(),
				PortfolioItemID: itemID,
				MediaURL:        m.MediaURL,
				MediaType:       mt,
				ThumbnailURL:    m.ThumbnailURL,
				Position:        m.Position,
				CreatedAt:       now,
			})
		}
		if len(media) > domain.MaxMediaPerItem {
			return nil, domain.ErrTooManyMedia
		}
		if err := s.portfolios.ReplaceMedia(ctx, itemID, media); err != nil {
			return nil, fmt.Errorf("replace media: %w", err)
		}
	}

	return s.portfolios.GetByID(ctx, itemID)
}

// DeleteItem deletes a portfolio item after ownership verification.
func (s *Service) DeleteItem(ctx context.Context, orgID, itemID uuid.UUID) error {
	item, err := s.portfolios.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item.OrganizationID != orgID {
		return domain.ErrNotOwner
	}
	if err := s.portfolios.Delete(ctx, itemID); err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	return nil
}

// ReorderItems updates the positions of all items for an org.
func (s *Service) ReorderItems(ctx context.Context, orgID uuid.UUID, itemIDs []uuid.UUID) error {
	return s.portfolios.ReorderItems(ctx, orgID, itemIDs)
}
