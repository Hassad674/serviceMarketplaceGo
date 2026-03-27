package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// SocialLinkRepository defines persistence operations for social links.
type SocialLinkRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*profile.SocialLink, error)
	Upsert(ctx context.Context, link *profile.SocialLink) error
	Delete(ctx context.Context, userID uuid.UUID, platform string) error
}
