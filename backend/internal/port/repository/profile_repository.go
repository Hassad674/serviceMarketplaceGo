package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// LocationInput is the write payload for ProfileRepository.UpdateLocation.
// Grouped in a struct (rather than passed as 6 positional arguments)
// so the adapter signature stays under the 4-parameter cap and the
// caller can omit fields explicitly via their zero value.
//
// Nullable fields are modeled as pointers — nil means "clear the
// column" at the database level (NULL), never "leave it untouched":
// UpdateLocation always writes the full location block atomically.
type LocationInput struct {
	City           string
	CountryCode    string
	Latitude       *float64
	Longitude      *float64
	WorkMode       []string
	TravelRadiusKm *int
}

// ProfileRepository persists the organization's public profile row.
// The Tier 1 completion (migration 083) added three targeted update
// methods (UpdateLocation / UpdateLanguages / UpdateAvailability)
// rather than bloating the existing Update(Profile) signature, so
// each block can be persisted in a single SQL UPDATE that touches
// only its own columns — cheaper in Postgres (smaller WAL per write)
// and clearer in the audit trail.
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

	// UpdateLocation writes the location block (city, country_code,
	// lat/lng, work_mode[], travel_radius_km) in a single SQL UPDATE.
	// Implementations MUST NOT touch any other column on the profiles
	// row — callers rely on the other blocks (title, about, languages,
	// availability) staying untouched after this call.
	UpdateLocation(ctx context.Context, orgID uuid.UUID, input LocationInput) error

	// UpdateLanguages replaces the two language arrays atomically.
	// Both slices are expected to be domain-normalized (ISO 639-1
	// lowercase, deduped) by the caller — the repository persists
	// them verbatim.
	UpdateLanguages(ctx context.Context, orgID uuid.UUID, professional, conversational []string) error

	// UpdateAvailability patches one or both availability columns.
	// A nil pointer means "do not touch this column" — implementations
	// MUST build the UPDATE statement dynamically so omitted columns
	// keep their current value. At least one pointer must be non-nil;
	// passing both nil is a programmer error.
	UpdateAvailability(ctx context.Context, orgID uuid.UUID, direct *profile.AvailabilityStatus, referrer *profile.AvailabilityStatus) error
}
