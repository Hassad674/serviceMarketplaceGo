package noop

import (
	"context"

	portservice "marketplace-backend/internal/port/service"
)

// TextModerationService is a no-op implementation that always returns safe.
type TextModerationService struct{}

// NewTextModerationService creates a no-op text moderation service for local development.
func NewTextModerationService() *TextModerationService {
	return &TextModerationService{}
}

// AnalyzeText always returns a safe result with no labels.
func (s *TextModerationService) AnalyzeText(
	_ context.Context,
	_ string,
) (*portservice.TextModerationResult, error) {
	return &portservice.TextModerationResult{
		Labels:   nil,
		MaxScore: 0,
		IsSafe:   true,
	}, nil
}
