package profileapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// SocialLinkService handles social link use cases.
type SocialLinkService struct {
	links repository.SocialLinkRepository
}

// NewSocialLinkService creates a service for social link operations.
func NewSocialLinkService(links repository.SocialLinkRepository) *SocialLinkService {
	return &SocialLinkService{links: links}
}

// ListByOrganization returns all social links for the given organization.
func (s *SocialLinkService) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*profile.SocialLink, error) {
	links, err := s.links.ListByOrganization(ctx, orgID)
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

// Upsert creates or updates a social link for the given organization.
func (s *SocialLinkService) Upsert(ctx context.Context, orgID uuid.UUID, input UpsertInput) error {
	platform := strings.ToLower(input.Platform)
	if !profile.IsValidPlatform(platform) {
		return profile.ErrInvalidPlatform
	}
	if err := profile.ValidateSocialURL(input.URL); err != nil {
		return err
	}

	link := &profile.SocialLink{
		OrganizationID: orgID,
		Platform:       platform,
		URL:            input.URL,
	}
	if err := s.links.Upsert(ctx, link); err != nil {
		return fmt.Errorf("upsert social link: %w", err)
	}
	return nil
}

// Delete removes a social link for the given organization.
func (s *SocialLinkService) Delete(ctx context.Context, orgID uuid.UUID, platform string) error {
	platform = strings.ToLower(platform)
	if !profile.IsValidPlatform(platform) {
		return profile.ErrInvalidPlatform
	}
	if err := s.links.Delete(ctx, orgID, platform); err != nil {
		return fmt.Errorf("delete social link: %w", err)
	}
	return nil
}
