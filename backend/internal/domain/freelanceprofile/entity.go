// Package freelanceprofile owns the domain model for the freelance
// persona of a provider_personal organization (migration 097). It is
// one half of the split profile aggregate introduced by the split
// refactor — the other half is domain/referrerprofile, and the
// shared fields (photo, location, languages) live directly on the
// organizations table and are JOINed at read time.
//
// The package is fully independent: it does NOT import any other
// feature package. Where a persona-specific concept overlaps with the
// legacy profile domain — most notably the AvailabilityStatus enum —
// the legacy profile package remains the single source of truth and
// this package imports its enum helpers. That reuse is acceptable
// because profile.AvailabilityStatus is a simple string-backed enum
// with no behaviour that conflicts with the split model.
package freelanceprofile

import (
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// Profile is one row of the freelance_profiles table. Every field is
// persona-specific — the shared fields (photo, languages, location,
// travel radius) are owned by the organization entity and joined by
// the handler response DTO. Keeping the domain struct lean preserves
// the single-responsibility principle: this type is ONLY about the
// freelance offering of an org.
//
// The ID is a surrogate UUID (not the organization_id). It exists so
// other feature tables that want to reference a freelance profile
// have a stable handle distinct from the org itself, and so the
// freelance_pricing table can carry a clean foreign key.
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

// New builds a fresh Profile with safe defaults. The repository layer
// uses this factory when inserting a brand-new row (e.g. the split
// migration on every provider_personal org). Input values are ignored
// because the freelance profile is seeded empty — the user fills it
// in via subsequent UpdateCore / UpdateAvailability / UpdateExpertise
// calls.
//
// Timestamps are set to time.Now() so both Created/Updated reflect
// the in-process creation moment. The database triggers will re-bump
// UpdatedAt on subsequent writes.
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

// UpdateCore applies the "classic" text edits to the profile. It
// never touches availability, expertise, or the shared fields — each
// of those has its own focused update method so a bug in one save
// flow cannot clobber the other slots.
//
// Empty strings are written verbatim. The handler layer is
// responsible for deciding whether to pass empty values through
// (e.g. clearing a video URL) or pre-filling them with the current
// value (e.g. keeping a previously-set title).
func (p *Profile) UpdateCore(title, about, videoURL string) {
	p.Title = title
	p.About = about
	p.VideoURL = videoURL
	p.UpdatedAt = time.Now()
}

// UpdateAvailability swaps the availability slot. The caller must
// pass a validated value — the method re-checks via IsValid to
// protect the domain invariant but never promotes zero values.
func (p *Profile) UpdateAvailability(status profile.AvailabilityStatus) error {
	if !status.IsValid() {
		return profile.ErrInvalidAvailabilityStatus
	}
	p.AvailabilityStatus = status
	p.UpdatedAt = time.Now()
	return nil
}

// UpdateExpertiseDomains replaces the full expertise list atomically.
// The caller is expected to have normalized and deduplicated the
// input — this method stores it verbatim. A nil input is coerced to
// an empty slice so downstream JSON marshaling yields `[]` rather
// than null.
func (p *Profile) UpdateExpertiseDomains(domains []string) {
	if domains == nil {
		domains = []string{}
	}
	p.ExpertiseDomains = domains
	p.UpdatedAt = time.Now()
}
