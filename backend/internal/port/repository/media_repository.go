package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/media"
)

// AdminMediaFilters holds filters for admin media listing.
type AdminMediaFilters struct {
	Status  string
	Type    string
	Context string
	Search  string
	Sort    string
	Page    int
	Limit   int
}

// AdminMediaItem extends Media with uploader info for admin views.
type AdminMediaItem struct {
	media.Media
	UploaderDisplayName string
	UploaderEmail       string
	UploaderRole        string
}

// MediaRepository defines persistence operations for media records.
type MediaRepository interface {
	Create(ctx context.Context, m *media.Media) error
	GetByID(ctx context.Context, id uuid.UUID) (*media.Media, error)
	GetAdminByID(ctx context.Context, id uuid.UUID) (*AdminMediaItem, error)
	GetByJobID(ctx context.Context, jobID string) (*media.Media, error)
	Update(ctx context.Context, m *media.Media) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListAdmin(ctx context.Context, filters AdminMediaFilters) ([]AdminMediaItem, error)
	CountAdmin(ctx context.Context, filters AdminMediaFilters) (int, error)
	ClearSource(ctx context.Context, mediaContext string, contextID uuid.UUID) error
	CountRejectedByUploader(ctx context.Context, uploaderID uuid.UUID) (int, error)
}
