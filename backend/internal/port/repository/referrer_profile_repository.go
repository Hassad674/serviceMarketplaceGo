package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/referrerprofile"
)

// ReferrerProfileView is the hydrated read shape returned by the
// "get or create by org" path. Bundles the persona-specific columns
// from referrer_profiles with the shared columns fetched from the
// organizations row via a single JOIN.
type ReferrerProfileView struct {
	Profile *referrerprofile.Profile
	Shared  OrganizationSharedProfile
}

// ReferrerProfileRepository persists the referrer persona row of a
// provider_personal organization whose owner has referrer_enabled=
// true. Mirrors the shape of FreelanceProfileRepository with one
// addition: GetOrCreateByOrgID exists because referrer profiles are
// NOT bulk-seeded — they are lazily created the first time an org
// reads its referrer profile after toggling the apporteur flag on.
type ReferrerProfileRepository interface {
	// GetOrCreateByOrgID returns the referrer profile for the org,
	// JOINed with the shared-profile block. If no row exists the
	// implementation MUST insert a fresh default row and return it.
	// Callers (the service layer) never see
	// referrerprofile.ErrProfileNotFound from this method.
	GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*ReferrerProfileView, error)

	// UpdateCore writes the title / about / video_url triplet.
	// See FreelanceProfileRepository.UpdateCore for semantics.
	UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error

	// UpdateAvailability writes a single availability value.
	UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error

	// UpdateExpertiseDomains replaces the expertise list atomically.
	UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error

	// UpdateVideo writes the video_url slot in isolation. Used by
	// the per-persona video upload handler. See
	// FreelanceProfileRepository.UpdateVideo for semantics. Returns
	// referrerprofile.ErrProfileNotFound when no row exists.
	UpdateVideo(ctx context.Context, orgID uuid.UUID, videoURL string) error

	// GetVideoURL returns the currently stored video_url for the
	// org. Returns referrerprofile.ErrProfileNotFound when no row
	// exists.
	GetVideoURL(ctx context.Context, orgID uuid.UUID) (string, error)
}
