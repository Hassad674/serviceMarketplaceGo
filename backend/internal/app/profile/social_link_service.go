package profileapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// SocialLinkService handles social link use cases for a single
// persona. Each persona (freelance / referrer / agency) gets its own
// service instance bound to its persona at construction time — this
// keeps callers unaware of the persona dimension and makes a service
// value a complete, self-contained "own social links" API.
type SocialLinkService struct {
	links   repository.SocialLinkRepository
	persona profile.SocialLinkPersona
}

// NewSocialLinkService creates a service for social link operations
// scoped to the given persona. Returns an error if the persona is
// not one of the recognised identity scopes — this is a programmer
// error at wiring time, not a runtime concern.
func NewSocialLinkService(
	links repository.SocialLinkRepository,
	persona profile.SocialLinkPersona,
) (*SocialLinkService, error) {
	if !profile.IsValidPersona(persona) {
		return nil, profile.ErrInvalidPersona
	}
	return &SocialLinkService{links: links, persona: persona}, nil
}

// Persona returns the identity scope this service is bound to. Useful
// for diagnostics and defensive checks in adjacent layers.
func (s *SocialLinkService) Persona() profile.SocialLinkPersona {
	return s.persona
}

// ListByOrganization returns all social links for the given
// organization under this service's persona.
func (s *SocialLinkService) ListByOrganization(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*profile.SocialLink, error) {
	links, err := s.links.ListByOrganizationPersona(ctx, orgID, s.persona)
	if err != nil {
		return nil, fmt.Errorf("list social links: %w", err)
	}
	return links, nil
}

// UpsertInput carries validated data for creating or updating a social link.
type UpsertInput struct {
	Platform string
	URL      string
}

// Upsert creates or updates a social link for the given organization
// under this service's persona.
func (s *SocialLinkService) Upsert(
	ctx context.Context,
	orgID uuid.UUID,
	input UpsertInput,
) error {
	platform := strings.ToLower(input.Platform)
	if !profile.IsValidPlatform(platform) {
		return profile.ErrInvalidPlatform
	}
	if err := profile.ValidateSocialURL(input.URL); err != nil {
		return err
	}

	link := &profile.SocialLink{
		OrganizationID: orgID,
		Persona:        s.persona,
		Platform:       platform,
		URL:            input.URL,
	}
	if err := s.links.Upsert(ctx, link); err != nil {
		return fmt.Errorf("upsert social link: %w", err)
	}
	return nil
}

// Delete removes a social link for the given organization under this
// service's persona.
func (s *SocialLinkService) Delete(
	ctx context.Context,
	orgID uuid.UUID,
	platform string,
) error {
	platform = strings.ToLower(platform)
	if !profile.IsValidPlatform(platform) {
		return profile.ErrInvalidPlatform
	}
	if err := s.links.Delete(ctx, orgID, s.persona, platform); err != nil {
		return fmt.Errorf("delete social link: %w", err)
	}
	return nil
}
