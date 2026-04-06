package noop

import (
	"context"

	portservice "marketplace-backend/internal/port/service"
)

// ModerationService is a no-op implementation that always returns safe.
type ModerationService struct{}

// NewModerationService creates a no-op moderation service for local development.
func NewModerationService() *ModerationService {
	return &ModerationService{}
}

// AnalyzeImage always returns a safe result with no labels.
func (s *ModerationService) AnalyzeImage(
	_ context.Context,
	_ []byte,
) (*portservice.ModerationResult, error) {
	return &portservice.ModerationResult{
		Safe:   true,
		Labels: nil,
		Score:  0,
	}, nil
}

// AnalyzeVideo returns an empty job id — no async work is scheduled.
func (s *ModerationService) AnalyzeVideo(
	_ context.Context,
	_ string,
	_ string,
) (*portservice.VideoJob, error) {
	return &portservice.VideoJob{JobID: ""}, nil
}

// GetVideoModerationResult always returns a safe result with no labels.
func (s *ModerationService) GetVideoModerationResult(
	_ context.Context,
	_ string,
) (*portservice.ModerationResult, error) {
	return &portservice.ModerationResult{
		Safe:   true,
		Labels: nil,
		Score:  0,
	}, nil
}
