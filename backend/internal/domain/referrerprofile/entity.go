// Package referrerprofile owns the domain model for the apporteur
// d'affaires ("business referrer") persona of a provider_personal
// organization with the referrer toggle enabled (migration 098). It
// is the sibling of domain/freelanceprofile and, like that package,
// stays fully independent at the Go-package level — the only shared
// dependency is the profile.AvailabilityStatus enum, reused to keep a
// single wire value for both personas.
//
// Shared fields (photo, location, languages, travel radius) live on
// the organizations table — the handler response DTO JOINs them at
// read time so the client sees a consistent public profile shape.
package referrerprofile

import (
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// Profile is one row of the referrer_profiles table. Every field is
// persona-specific — the shared fields (photo, languages, location,
// travel radius) are owned by the organization entity.
//
// The ID is a surrogate UUID (not the organization_id). It exists so
// other feature tables can reference a referrer profile with a stable
// handle distinct from the org, and so the referrer_pricing table
// carries a clean foreign key.
type Profile struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	Title              string
	About              string
	VideoURL           string
	AvailabilityStatus profile.AvailabilityStatus
	ExpertiseDomains   []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// New builds a fresh Profile with safe defaults. The app layer uses
// this factory when it needs to lazily create a referrer profile on
// first GET (the referrer service auto-creates an empty row when one
// does not exist yet, unlike the freelance service which relies on
// the bulk migration having seeded every provider_personal org).
func New(organizationID uuid.UUID) *Profile {
	now := time.Now()
	return &Profile{
		ID:                 uuid.New(),
		OrganizationID:     organizationID,
		AvailabilityStatus: profile.AvailabilityNow,
		ExpertiseDomains:   []string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// UpdateCore applies the "classic" text edits (title, about, video)
// to the profile. See freelanceprofile.Profile.UpdateCore for the
// mirrored semantics.
func (p *Profile) UpdateCore(title, about, videoURL string) {
	p.Title = title
	p.About = about
	p.VideoURL = videoURL
	p.UpdatedAt = time.Now()
}

// UpdateAvailability swaps the availability slot. Re-checks the
// enum via IsValid so the domain invariant holds even if a bug in a
// higher layer lets a zero value through.
func (p *Profile) UpdateAvailability(status profile.AvailabilityStatus) error {
	if !status.IsValid() {
		return profile.ErrInvalidAvailabilityStatus
	}
	p.AvailabilityStatus = status
	p.UpdatedAt = time.Now()
	return nil
}

// UpdateExpertiseDomains replaces the full expertise list atomically.
// Nil input is coerced to an empty slice so downstream JSON serialization
// yields `[]` rather than null.
func (p *Profile) UpdateExpertiseDomains(domains []string) {
	if domains == nil {
		domains = []string{}
	}
	p.ExpertiseDomains = domains
	p.UpdatedAt = time.Now()
}
