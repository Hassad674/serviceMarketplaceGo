package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/portfolio"
)

// PortfolioRepository defines persistence operations for portfolio items.
type PortfolioRepository interface {
	// Create inserts a portfolio item with all its media in a transaction.
	Create(ctx context.Context, item *portfolio.PortfolioItem) error

	// GetByID returns a single item with all its media loaded.
	GetByID(ctx context.Context, id uuid.UUID) (*portfolio.PortfolioItem, error)

	// ListByOrganization returns items ordered by position ASC with
	// cursor pagination. Each item includes its media.
	ListByOrganization(ctx context.Context, organizationID uuid.UUID, cursor string, limit int) ([]*portfolio.PortfolioItem, string, error)

	// Update updates title, description, link_url and updated_at.
	Update(ctx context.Context, item *portfolio.PortfolioItem) error

	// Delete removes an item. CASCADE deletes its media.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByOrganization returns the number of portfolio items for an org.
	CountByOrganization(ctx context.Context, organizationID uuid.UUID) (int, error)

	// ReorderItems batch-updates positions. Index in the slice = new position.
	ReorderItems(ctx context.Context, organizationID uuid.UUID, itemIDs []uuid.UUID) error

	// ReplaceMedia deletes all existing media for an item and inserts new ones.
	ReplaceMedia(ctx context.Context, itemID uuid.UUID, media []*portfolio.PortfolioMedia) error
}
