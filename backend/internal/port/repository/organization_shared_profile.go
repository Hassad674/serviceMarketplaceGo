package repository

import (
	"context"

	"github.com/google/uuid"
)

// OrganizationSharedProfile is the read shape of the shared-profile
// columns that live on the organizations row (migration 096). Since
// the split refactor, the photo, location, languages, and travel
// radius columns belong to the organization itself rather than to
// the legacy profiles row — that way both the freelance and referrer
// personas of the same org see a single source of truth for the
// fields they share.
//
// This type intentionally lives in port/repository (not in domain)
// because it is a transport-only bundle: the handler JOINs it into
// the freelance or referrer response DTO at read time. If a future
// refactor needs rich behaviour on these fields, they can be moved
// into a proper domain value object without changing the interface
// surface.
type OrganizationSharedProfile struct {
	PhotoURL                string
	City                    string
	CountryCode             string
	Latitude                *float64
	Longitude               *float64
	WorkMode                []string
	TravelRadiusKm          *int
	LanguagesProfessional   []string
	LanguagesConversational []string
}

// SharedProfileLocationInput is the write payload for the location
// block. A dedicated struct (rather than six positional parameters)
// keeps the repository signature under the 4-parameter cap and
// makes the nullable semantics explicit: nil pointers clear the
// column to NULL on the database side.
type SharedProfileLocationInput struct {
	City           string
	CountryCode    string
	Latitude       *float64
	Longitude      *float64
	WorkMode       []string
	TravelRadiusKm *int
}

// OrganizationSharedProfileWriter isolates the write operations for
// the shared-profile columns on organizations. Defining it as a
// separate small interface (instead of extending the main
// OrganizationRepository) keeps the interface segregation principle
// intact — services that only need to write location / languages /
// photo do not have to depend on Stripe / KYC / transfer methods.
//
// In production the postgres OrganizationRepository satisfies this
// interface directly, so no separate adapter is needed — the wiring
// layer just passes the same concrete value behind two interface
// handles.
type OrganizationSharedProfileWriter interface {
	// UpdateSharedLocation rewrites the entire location block
	// atomically. Empty strings clear city / country_code, nil
	// pointers clear lat / lng / travel_radius_km, and a nil slice
	// clears work_mode to an empty array. Never partial.
	UpdateSharedLocation(ctx context.Context, orgID uuid.UUID, input SharedProfileLocationInput) error

	// UpdateSharedLanguages replaces the two language arrays.
	UpdateSharedLanguages(ctx context.Context, orgID uuid.UUID, professional, conversational []string) error

	// UpdateSharedPhotoURL writes a single photo_url value. Empty
	// string clears the photo.
	UpdateSharedPhotoURL(ctx context.Context, orgID uuid.UUID, photoURL string) error

	// GetSharedProfile returns the full shared-profile block for an
	// org. Used by read paths that need the shared fields without
	// joining a persona (e.g. the /organization/shared endpoint).
	GetSharedProfile(ctx context.Context, orgID uuid.UUID) (*OrganizationSharedProfile, error)
}
