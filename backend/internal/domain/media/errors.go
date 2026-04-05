package media

import "errors"

var (
	ErrMediaNotFound   = errors.New("media not found")
	ErrMissingUploader = errors.New("uploader ID is required")
	ErrMissingFileURL  = errors.New("file URL is required")
	ErrMissingFileName = errors.New("file name is required")
	ErrMissingFileType = errors.New("file type is required")
	ErrInvalidContext  = errors.New("invalid media context")
	ErrInvalidStatus   = errors.New("invalid moderation status")
)
