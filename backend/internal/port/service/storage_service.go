package service

import (
	"context"
	"io"
	"time"
)

type StorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	Delete(ctx context.Context, key string) error
	GetPublicURL(key string) string
	GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error)
	Download(ctx context.Context, key string) ([]byte, error)
}
