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

// VideoJob represents an async video moderation job.
type VideoJob struct {
	JobID string
}

// ContentModerationService analyzes media content for policy violations.
type ContentModerationService interface {
	AnalyzeImage(ctx context.Context, imageBytes []byte) (*ModerationResult, error)
	// AnalyzeVideo starts an async video moderation job.
	// The result will be published to SNS when complete.
	// Returns a jobID that can be used to retrieve results later.
	AnalyzeVideo(ctx context.Context, s3Bucket, s3Key string) (*VideoJob, error)
	// GetVideoModerationResult retrieves the result of a completed video moderation job.
	GetVideoModerationResult(ctx context.Context, jobID string) (*ModerationResult, error)
}
