package service

import "github.com/google/uuid"

// MediaRecorder records uploaded media files for moderation tracking.
// Used by features that upload files (messaging, reviews) to register
// them in the media table without importing the media app package directly.
type MediaRecorder interface {
	RecordUploadRaw(
		uploaderID uuid.UUID,
		fileURL string,
		fileName string,
		fileType string,
		fileSize int64,
		mediaContext string,
	)
}
