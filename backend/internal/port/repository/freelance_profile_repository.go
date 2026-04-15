package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
)

// FreelanceProfileView is the hydrated read shape returned by the
// "get by org" path. It bundles the persona-specific columns from
// freelance_profiles with the shared columns fetched from the
// organizations row via a single JOIN — no N+1 between the two
// stores at read time.
//
// This type lives in the port layer (not on the domain) because it
// is a transport-only bundle. The domain model stays lean and
// persona-focused; the view adds the joined context only where the
// handler needs it.
type FreelanceProfileView struct {
	Profile *freelanceprofile.Profile
	Shared  OrganizationSharedProfile
}

// FreelanceProfileRepository persists the freelance persona row of
// a provider_personal organization.
//
// The interface is intentionally small. Updates are split by slot
// (core text, availability, expertise) so a bug in one save flow
// cannot clobber the other slots — mirrors the pattern already used
// by the legacy ProfileRepository's UpdateLocation / UpdateLanguages
// / UpdateAvailability triplet. A lazy GetOrCreateByOrgID path
// exists for the owner-side read so new provider_personal accounts
// registered after the split migration transparently get a row on
// first access instead of hitting a 404.
type FreelanceProfileRepository interface {
	// GetByOrgID returns the freelance profile for the org, JOINed
	// with the organizations shared-profile block. Callers receive a
	// FreelanceProfileView ready for direct DTO mapping. Returns
	// freelanceprofile.ErrProfileNotFound when no freelance row
	// exists — used by the public endpoint (strict read).
	GetByOrgID(ctx context.Context, orgID uuid.UUID) (*FreelanceProfileView, error)

	// GetOrCreateByOrgID is the lazy variant used by the owner-side
	// read path: if no row exists it inserts a fresh default and
	// re-fetches. Never returns ErrProfileNotFound.
	GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*FreelanceProfileView, error)

	// UpdateCore writes the title / about / video_url triplet in a
	// single SQL UPDATE. Callers supply all three values — empty
	// strings clear the corresponding column, so a delete-video flow
	// is a regular update with video_url=''. The repository must NOT
	// touch any other column on the row.
	UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error

	// UpdateAvailability writes a single availability value. Takes
	// the validated enum from the domain layer — the repository does
	// not re-validate.
	UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error

	// UpdateExpertiseDomains replaces the full expertise list
	// atomically. The caller is expected to have normalized /
	// deduplicated the slice — the repository persists it verbatim.
	UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error

	// UpdateVideo writes the video_url slot in isolation. Used by
	// the per-persona video upload handler so the upload flow never
	// touches title/about — a race with an in-flight core edit cannot
	// clobber the presentation text. Passing an empty string clears
	// the column (used by the DELETE path). Returns
	// freelanceprofile.ErrProfileNotFound when no row exists.
	UpdateVideo(ctx context.Context, orgID uuid.UUID, videoURL string) error

	// GetVideoURL returns the currently stored video_url for the
	// org, used by the upload handler to delete the previous MinIO
	// object before overwriting it. Returns
	// freelanceprofile.ErrProfileNotFound when no row exists.
	GetVideoURL(ctx context.Context, orgID uuid.UUID) (string, error)
}
