package service

import (
	"context"

	"github.com/google/uuid"
)

// MediaRecorder records uploaded media files for moderation tracking.
// Used by features that upload files (messaging, reviews) to register
// them in the media table without importing the media app package directly.
//
// The first argument is a parent context used to propagate cancellation /
// shutdown into the moderation pipeline. The implementation derives its
// own bounded sub-context (60s default), so callers may pass
// context.Background() — they just lose the SIGTERM short-circuit.
type MediaRecorder interface {
	RecordUploadRaw(
		ctx context.Context,
		uploaderID uuid.UUID,
		fileURL string,
		fileName string,
		fileType string,
		fileSize int64,
		mediaContext string,
	)
}
