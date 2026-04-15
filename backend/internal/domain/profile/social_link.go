package profile

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SocialLink represents a social network link displayed on the
// organization's public profile. Phase R2 anchors social links on the
// org rather than on an individual user. A single organization can
// hold multiple independent sets of links — one per persona — to
// support users that operate as both freelance and referrer.
type SocialLink struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Persona        SocialLinkPersona
	Platform       string
	URL            string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SocialLinkPersona is the identity scope under which a social link
// is displayed. Agencies keep their legacy single set under the
// "agency" persona. Provider_personal users get two independent sets:
// "freelance" for their freelance marketplace identity and
// "referrer" for their apporteur d'affaires identity.
type SocialLinkPersona string

const (
	PersonaFreelance SocialLinkPersona = "freelance"
	PersonaReferrer  SocialLinkPersona = "referrer"
	PersonaAgency    SocialLinkPersona = "agency"
)

// ValidPlatforms lists all accepted social link platform identifiers.
var ValidPlatforms = []string{
	"linkedin",
	"instagram",
	"youtube",
	"twitter",
	"github",
	"website",
}

// ValidPersonas lists all accepted persona identifiers. Kept
// alongside the platform allowlist so both validators live in one
// place and reviews stay concise.
var ValidPersonas = []SocialLinkPersona{
	PersonaFreelance,
	PersonaReferrer,
	PersonaAgency,
}

var (
	ErrInvalidPlatform = errors.New("invalid social link platform")
	ErrInvalidURL      = errors.New("invalid social link URL")
	ErrInvalidPersona  = errors.New("invalid social link persona")
)

// IsValidPlatform checks whether the given platform string is supported.
func IsValidPlatform(platform string) bool {
	lower := strings.ToLower(platform)
	for _, p := range ValidPlatforms {
		if p == lower {
			return true
		}
	}
	return false
}

// IsValidPersona checks whether the given persona string is one of
// the recognised identity scopes.
func IsValidPersona(persona SocialLinkPersona) bool {
	for _, p := range ValidPersonas {
		if p == persona {
			return true
		}
	}
	return false
}

// ValidateSocialURL checks that the URL is well-formed and uses http(s).
func ValidateSocialURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	if parsed.Host == "" {
		return ErrInvalidURL
	}
	return nil
}
