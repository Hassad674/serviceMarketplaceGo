package profile

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SocialLink represents a social network link associated with a user profile.
type SocialLink struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Platform  string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ValidPlatforms lists all accepted social link platform identifiers.
var ValidPlatforms = []string{
	"linkedin",
	"instagram",
	"youtube",
	"twitter",
	"github",
	"website",
}

var (
	ErrInvalidPlatform = errors.New("invalid social link platform")
	ErrInvalidURL      = errors.New("invalid social link URL")
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
