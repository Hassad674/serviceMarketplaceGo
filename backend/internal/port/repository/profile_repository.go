package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

type ProfileRepository interface {
	Create(ctx context.Context, p *profile.Profile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*profile.Profile, error)
	Update(ctx context.Context, p *profile.Profile) error
	SearchPublic(ctx context.Context, roleFilter string, referrerOnly bool, limit int) ([]*profile.PublicProfile, error)
	GetPublicProfilesByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]*profile.PublicProfile, error)
}
