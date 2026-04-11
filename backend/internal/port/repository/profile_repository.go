package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

type ProfileRepository interface {
	Create(ctx context.Context, p *profile.Profile) error
	GetByOrganizationID(ctx context.Context, organizationID uuid.UUID) (*profile.Profile, error)
	Update(ctx context.Context, p *profile.Profile) error
	SearchPublic(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error)
	GetPublicProfilesByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) ([]*profile.PublicProfile, error)

	// OrgProfilesByUserIDs returns the org public profile for each
	// given user, keyed by user_id. Used by flows that anchor on a
	// user (job applications, reviews) but need to display that
	// user's team identity. The mapping happens via users.organization_id.
	OrgProfilesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error)
}
