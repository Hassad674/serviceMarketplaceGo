package profileapp

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

type Service struct {
	profiles repository.ProfileRepository
}

func NewService(profiles repository.ProfileRepository) *Service {
	return &Service{profiles: profiles}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*profile.Profile, error) {
	p, err := s.profiles.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return p, nil
}

type UpdateProfileInput struct {
	Title                string
	About                string
	PhotoURL             string
	PresentationVideoURL string
	ReferrerAbout        string
	ReferrerVideoURL     string
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*profile.Profile, error) {
	p, err := s.profiles.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	applyUpdates(p, input)

	if err := s.profiles.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return p, nil
}

func applyUpdates(p *profile.Profile, input UpdateProfileInput) {
	if input.Title != "" {
		p.Title = input.Title
	}
	if input.About != "" {
		p.About = input.About
	}
	if input.PhotoURL != "" {
		p.PhotoURL = input.PhotoURL
	}
	if input.PresentationVideoURL != "" {
		p.PresentationVideoURL = input.PresentationVideoURL
	}
	if input.ReferrerAbout != "" {
		p.ReferrerAbout = input.ReferrerAbout
	}
	if input.ReferrerVideoURL != "" {
		p.ReferrerVideoURL = input.ReferrerVideoURL
	}
}
