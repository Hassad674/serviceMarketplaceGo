package media

import (
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

// Service orchestrates media recording and moderation.
type Service struct {
	media      repository.MediaRepository
	storage    service.StorageService
	moderation service.ContentModerationService
}

// NewService creates a new media service.
func NewService(
	media repository.MediaRepository,
	storage service.StorageService,
	moderation service.ContentModerationService,
) *Service {
	return &Service{
		media:      media,
		storage:    storage,
		moderation: moderation,
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	if !strings.HasPrefix(fileType, "image/") {
		return
	}

	s.moderateImage(ctx, m)
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

	if result.Safe {
		m.AutoApprove(result.Score)
	} else {
		m.Flag(result.Labels, result.Score)
	}

	if err := s.media.Update(ctx, m); err != nil {
		slog.Error("media moderation: update", "error", err, "media_id", m.ID)
	}
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
