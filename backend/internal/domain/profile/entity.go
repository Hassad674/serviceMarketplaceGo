// Package profile — LEGACY AGENCY-ONLY aggregate.
//
// Since the split-profile refactor (migrations 096-104) this
// package backs ONLY agency organizations. The provider_personal
// path now goes through domain/freelanceprofile and
// domain/referrerprofile. A follow-up refactor will migrate the
// agency path to its own dedicated aggregate and delete this
// package, but for now it remains in place so the agency
// handler/service/adapter chain keeps compiling.
//
// Do NOT extend this package with new fields for provider_personal
// use cases — add them to the split aggregates instead.
package profile

import (
	"time"

	"github.com/google/uuid"
)

// Profile is the organization's public-facing marketplace profile:
// the shared photo, presentation video, about text, title, location,
// languages, and availability that every team member edits
// collaboratively. Phase R2 of the team refactor moved the anchor
// from user_id to organization_id so operators invited into the same
// org see the same profile; migration 083 layered the Tier 1
// completion blocks (location / languages / availability).
//
// Pricing is NOT on this struct even though it is logically part of
// "profile Tier 1" — it lives in its own domain package
// (domain/profilepricing) because its cardinality is up to 2 rows per
// org and it persists to a dedicated table (profile_pricing) so edits
// do not churn the full profile row.
type Profile struct {
	OrganizationID       uuid.UUID
	Title                string
	About                string
	PhotoURL             string
	PresentationVideoURL string
	ReferrerAbout        string
	ReferrerVideoURL     string

	// ---- Location (migration 083) ----
	City           string
	CountryCode    string // ISO 3166-1 alpha-2, uppercase (empty = unset)
	Latitude       *float64
	Longitude      *float64
	WorkMode       []string // subset of {"remote", "on_site", "hybrid"}
	TravelRadiusKm *int     // nil when unset / remote-only

	// ---- Languages (migration 083) ----
	LanguagesProfessional   []string // ISO 639-1 codes (lowercase)
	LanguagesConversational []string

	// ---- Availability (migration 083) ----
	// AvailabilityStatus applies to the org's default offering (direct
	// freelance / agency work). Always set.
	AvailabilityStatus AvailabilityStatus
	// ReferrerAvailabilityStatus applies to the separate "apporteur"
	// offering. Nil for any org that is not a provider_personal with
	// referrer_enabled=true — the UI then hides the referrer section.
	ReferrerAvailabilityStatus *AvailabilityStatus

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewProfile seeds a fresh Profile struct with the Tier 1 fields at
// their default values: empty strings / empty slices for the arrays,
// nil for the nullable scalars, and AvailabilityNow as the default
// availability so newly-created orgs appear as "available now" on the
// marketplace until they explicitly opt out.
func NewProfile(organizationID uuid.UUID) *Profile {
	now := time.Now()
	return &Profile{
		OrganizationID:          organizationID,
		WorkMode:                []string{},
		LanguagesProfessional:   []string{},
		LanguagesConversational: []string{},
		AvailabilityStatus:      AvailabilityNow,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

// PublicProfile is the aggregated view used by search / discovery and
// by the public org page. It combines org identity (name, type, photo)
// with review metrics and a referrer flag derived from the owner.
type PublicProfile struct {
	OrganizationID  uuid.UUID
	Name            string
	OrgType         string
	Title           string
	PhotoURL        string
	ReferrerEnabled bool
	AverageRating   float64 // average of received reviews (0 when no reviews)
	ReviewCount     int     // number of non-hidden reviews received
	CreatedAt       time.Time
}
