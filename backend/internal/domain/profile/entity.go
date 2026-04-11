package profile

import (
	"time"

	"github.com/google/uuid"
)

// Profile is the organization's public-facing marketplace profile:
// the shared photo, presentation video, about text, and title that
// every team member edits collaboratively. Phase R2 of the team
// refactor moves the anchor from user_id to organization_id so
// operators invited into the same org see the same profile.
type Profile struct {
	OrganizationID       uuid.UUID
	Title                string
	About                string
	PhotoURL             string
	PresentationVideoURL string
	ReferrerAbout        string
	ReferrerVideoURL     string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewProfile(organizationID uuid.UUID) *Profile {
	now := time.Now()
	return &Profile{
		OrganizationID: organizationID,
		CreatedAt:      now,
		UpdatedAt:      now,
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
