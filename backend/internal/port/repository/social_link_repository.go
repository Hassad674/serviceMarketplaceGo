package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// SocialLinkRepository defines persistence operations for org-level
// social links scoped by persona (agency, freelance, referrer).
//
// Every method takes a persona so that a single organization can
// expose multiple independent sets — the freelance persona of a
// provider_personal user holds its LinkedIn/GitHub/portfolio, while
// its referrer persona keeps a separate set for the apporteur
// d'affaires identity.
type SocialLinkRepository interface {
	ListByOrganizationPersona(
		ctx context.Context,
		organizationID uuid.UUID,
		persona profile.SocialLinkPersona,
	) ([]*profile.SocialLink, error)

	Upsert(ctx context.Context, link *profile.SocialLink) error

	Delete(
		ctx context.Context,
		organizationID uuid.UUID,
		persona profile.SocialLinkPersona,
		platform string,
	) error
}
