package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// SocialLinkRepository defines persistence operations for org-level
// social links (website, LinkedIn, Instagram, etc.).
type SocialLinkRepository interface {
	ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]*profile.SocialLink, error)
	Upsert(ctx context.Context, link *profile.SocialLink) error
	Delete(ctx context.Context, organizationID uuid.UUID, platform string) error
}
