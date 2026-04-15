package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		want     bool
	}{
		{"linkedin lowercase", "linkedin", true},
		{"LinkedIn mixed case", "LinkedIn", true},
		{"instagram", "instagram", true},
		{"youtube", "youtube", true},
		{"twitter", "twitter", true},
		{"github", "github", true},
		{"website", "website", true},
		{"WEBSITE uppercase", "WEBSITE", true},
		{"empty string", "", false},
		{"unknown platform", "tiktok", false},
		{"facebook", "facebook", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidPlatform(tt.platform)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidPersona(t *testing.T) {
	tests := []struct {
		name    string
		persona SocialLinkPersona
		want    bool
	}{
		{"freelance", PersonaFreelance, true},
		{"referrer", PersonaReferrer, true},
		{"agency", PersonaAgency, true},
		{"empty string", SocialLinkPersona(""), false},
		{"unknown", SocialLinkPersona("admin"), false},
		{"case sensitive", SocialLinkPersona("FREELANCE"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidPersona(tt.persona))
		})
	}
}

func TestValidateSocialURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"valid https url", "https://linkedin.com/in/user", nil},
		{"valid http url", "http://example.com", nil},
		{"empty string", "", ErrInvalidURL},
		{"no scheme", "linkedin.com/in/user", ErrInvalidURL},
		{"javascript scheme", "javascript:alert(1)", ErrInvalidURL},
		{"ftp scheme", "ftp://files.example.com", ErrInvalidURL},
		{"data uri", "data:text/html,<h1>hi</h1>", ErrInvalidURL},
		{"no host", "https://", ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSocialURL(tt.url)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
