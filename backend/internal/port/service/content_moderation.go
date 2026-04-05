package service

import (
	"context"

	"marketplace-backend/internal/domain/media"
)

// ModerationResult holds the outcome of content moderation analysis.
type ModerationResult struct {
	Safe   bool
	Labels []media.ModerationLabel
	Score  float64
}

// ContentModerationService analyzes media content for policy violations.
type ContentModerationService interface {
	AnalyzeImage(ctx context.Context, imageBytes []byte) (*ModerationResult, error)
}
