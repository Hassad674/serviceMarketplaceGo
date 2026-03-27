package messaging

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
)

type GetPresignedURLInput struct {
	UserID      uuid.UUID
	Filename    string
	ContentType string
}

type PresignedUploadResult struct {
	UploadURL string
	FileKey   string
	PublicURL string
}

// allowedUploadExtensions is the allowlist of file extensions accepted for messaging uploads.
var allowedUploadExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true,
	".txt": true, ".csv": true, ".rtf": true, ".md": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
	".mp4": true, ".mp3": true, ".wav": true, ".ogg": true, ".webm": true, ".m4a": true,
}

func (s *Service) GetPresignedUploadURL(ctx context.Context, input GetPresignedURLInput) (PresignedUploadResult, error) {
	ext := strings.ToLower(filepath.Ext(input.Filename))
	if !allowedUploadExtensions[ext] {
		return PresignedUploadResult{}, message.ErrInvalidFileType
	}

	// Use a random UUID as the filename to prevent path traversal and info leakage
	safeFilename := uuid.New().String() + ext
	key := fmt.Sprintf("messaging/%s/%d_%s", input.UserID.String(), time.Now().UnixMilli(), safeFilename)

	uploadURL, err := s.storage.GetPresignedUploadURL(ctx, key, input.ContentType, 15*time.Minute)
	if err != nil {
		return PresignedUploadResult{}, fmt.Errorf("get presigned url: %w", err)
	}

	publicURL := s.storage.GetPublicURL(key)

	return PresignedUploadResult{
		UploadURL: uploadURL,
		FileKey:   key,
		PublicURL: publicURL,
	}, nil
}
