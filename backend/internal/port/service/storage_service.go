package service

import (
	"context"
	"io"
)

type StorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	Delete(ctx context.Context, key string) error
	GetPublicURL(key string) string
}
