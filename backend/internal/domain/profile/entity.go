package profile

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	UserID               uuid.UUID
	Title                string
	About                string
	PhotoURL             string
	PresentationVideoURL string
	ReferrerAbout        string
	ReferrerVideoURL     string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewProfile(userID uuid.UUID) *Profile {
	now := time.Now()
	return &Profile{
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// PublicProfile combines user and profile data for search/discovery.
type PublicProfile struct {
	UserID          uuid.UUID
	DisplayName     string
	FirstName       string
	LastName        string
	Role            string
	Title           string
	PhotoURL        string
	ReferrerEnabled bool
	CreatedAt       time.Time
}
