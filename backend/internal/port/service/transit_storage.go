package service

import (
	"context"
	"io"
)

// TransitStorageService is a temporary staging area used to hand off files
// to external analyzers (e.g. AWS Rekognition requires files in an S3 bucket
// within the same account/region).
type TransitStorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error
	Delete(ctx context.Context, key string) error
	Bucket() string
}
