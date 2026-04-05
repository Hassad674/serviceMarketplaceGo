package media

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ServiceDeps groups the dependencies required to construct a media service.
type ServiceDeps struct {
	Media               repository.MediaRepository
	Storage             service.StorageService
	Transit             service.TransitStorageService
	Moderation          service.ContentModerationService
	FlagThreshold       float64
	AutoRejectThreshold float64
}

// Service orchestrates media recording and moderation.
type Service struct {
	media               repository.MediaRepository
	storage             service.StorageService
	transit             service.TransitStorageService
	moderation          service.ContentModerationService
	flagThreshold       float64
	autoRejectThreshold float64
}

// NewService creates a new media service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		media:               deps.Media,
		storage:             deps.Storage,
		transit:             deps.Transit,
		moderation:          deps.Moderation,
		flagThreshold:       deps.FlagThreshold,
		autoRejectThreshold: deps.AutoRejectThreshold,
	}
}

// RecordUpload creates a media record and runs moderation asynchronously.
// This method should be called in a goroutine after a successful upload.
func (s *Service) RecordUpload(
	uploaderID uuid.UUID,
	fileURL string,
	fileName string,
	fileType string,
	fileSize int64,
	mediaCtx mediadomain.Context,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	m, err := mediadomain.NewMedia(mediadomain.NewMediaInput{
		UploaderID: uploaderID,
		FileURL:    fileURL,
		FileName:   fileName,
		FileType:   fileType,
		FileSize:   fileSize,
		Context:    mediaCtx,
	})
	if err != nil {
		slog.Error("media record: create entity", "error", err)
		return
	}

	if err := s.media.Create(ctx, m); err != nil {
		slog.Error("media record: persist", "error", err)
		return
	}

	switch {
	case strings.HasPrefix(fileType, "image/"):
		s.moderateImage(ctx, m)
	case strings.HasPrefix(fileType, "video/"):
		s.moderateVideo(ctx, m)
	}
}

func (s *Service) moderateImage(ctx context.Context, m *mediadomain.Media) {
	key := extractStorageKey(m.FileURL)
	if key == "" {
		slog.Warn("media moderation: cannot extract key", "url", m.FileURL)
		return
	}

	imageBytes, err := s.storage.Download(ctx, key)
	if err != nil {
		slog.Error("media moderation: download", "error", err, "key", key)
		return
	}

	result, err := s.moderation.AnalyzeImage(ctx, imageBytes)
	if err != nil {
		slog.Error("media moderation: analyze", "error", err, "media_id", m.ID)
		return
	}

	s.applyDecision(ctx, m, key, result)
}

func (s *Service) moderateVideo(ctx context.Context, m *mediadomain.Media) {
	if s.transit == nil {
		slog.Debug("media moderation: video moderation disabled (no transit storage)",
			"media_id", m.ID)
		return
	}

	srcKey := extractStorageKey(m.FileURL)
	if srcKey == "" {
		slog.Warn("media moderation: cannot extract key", "url", m.FileURL)
		return
	}

	videoBytes, err := s.storage.Download(ctx, srcKey)
	if err != nil {
		slog.Error("media moderation: download video", "error", err, "key", srcKey)
		return
	}

	transitKey := fmt.Sprintf("moderation/%s/%s", m.ID, m.FileName)
	if err := s.transit.Upload(ctx, transitKey, bytes.NewReader(videoBytes),
		m.FileType, int64(len(videoBytes))); err != nil {
		slog.Error("media moderation: upload to transit", "error", err, "media_id", m.ID)
		return
	}

	job, err := s.moderation.AnalyzeVideo(ctx, s.transit.Bucket(), transitKey)
	if err != nil {
		slog.Error("media moderation: start video job", "error", err, "media_id", m.ID)
		// Best-effort cleanup of the transit file we just uploaded.
		if delErr := s.transit.Delete(ctx, transitKey); delErr != nil {
			slog.Warn("media moderation: delete transit after job start failure",
				"error", delErr, "key", transitKey)
		}
		return
	}

	m.SetJobID(job.JobID)
	if err := s.media.Update(ctx, m); err != nil {
		slog.Error("media moderation: persist job id", "error", err, "media_id", m.ID)
	}
	slog.Info("media moderation: video job started",
		"media_id", m.ID, "job_id", job.JobID, "transit_key", transitKey)
}

// applyDecision translates a moderation result into a status change and
// enforces the auto-reject threshold by deleting the source file from storage.
func (s *Service) applyDecision(
	ctx context.Context,
	m *mediadomain.Media,
	srcKey string,
	result *service.ModerationResult,
) {
	switch {
	case result.Safe:
		m.AutoApprove(result.Score)
	case s.autoRejectThreshold > 0 && result.Score >= s.autoRejectThreshold:
		if err := s.storage.Delete(ctx, srcKey); err != nil {
			slog.Warn("media moderation: delete source after auto-reject",
				"error", err, "key", srcKey)
		}
		m.AutoReject(result.Labels, result.Score)
	case s.flagThreshold > 0 && result.Score >= s.flagThreshold:
		m.Flag(result.Labels, result.Score)
	default:
		m.Flag(result.Labels, result.Score)
	}

	if err := s.media.Update(ctx, m); err != nil {
		slog.Error("media moderation: update", "error", err, "media_id", m.ID)
	}
}

// FinalizeVideoJob is invoked by the SQS worker when Rekognition has finished
// analyzing a video. It fetches the labels, applies the decision and cleans
// up the transit file.
func (s *Service) FinalizeVideoJob(ctx context.Context, jobID string) error {
	m, err := s.media.GetByJobID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("lookup media by job id %s: %w", jobID, err)
	}

	result, err := s.moderation.GetVideoModerationResult(ctx, jobID)
	if err != nil {
		return fmt.Errorf("fetch video moderation result: %w", err)
	}

	srcKey := extractStorageKey(m.FileURL)
	s.applyDecision(ctx, m, srcKey, result)

	// Always clean up the transit object once Rekognition is done with it.
	if s.transit != nil {
		transitKey := fmt.Sprintf("moderation/%s/%s", m.ID, m.FileName)
		if err := s.transit.Delete(ctx, transitKey); err != nil {
			slog.Warn("media moderation: delete transit after job completion",
				"error", err, "key", transitKey)
		}
	}
	return nil
}

// extractStorageKey removes the public URL prefix to get the storage key.
func extractStorageKey(fileURL string) string {
	// URLs are like "http://host/bucket/profiles/uuid/photo_uuid.jpg"
	// We need the part after the last known path segment.
	parts := strings.SplitN(fileURL, "/", 4)
	if len(parts) < 4 {
		return ""
	}
	// Try to find path after domain+bucket
	idx := strings.Index(fileURL, "profiles/")
	if idx >= 0 {
		return fileURL[idx:]
	}
	idx = strings.Index(fileURL, "reviews/")
	if idx >= 0 {
		return fileURL[idx:]
	}
	idx = strings.Index(fileURL, "messages/")
	if idx >= 0 {
		return fileURL[idx:]
	}
	idx = strings.Index(fileURL, "identity/")
	if idx >= 0 {
		return fileURL[idx:]
	}
	// Fallback: return the last path component from slash 3 onwards
	return fmt.Sprintf("%s", parts[3])
}
