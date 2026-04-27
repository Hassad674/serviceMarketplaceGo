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
	// GetPresignedDownloadURL returns a short-lived signed GET url for the
	// stored object. Used by handlers that gate object access on
	// application-level ownership checks (e.g. invoice PDFs) so the bucket
	// itself stays private and clients only ever see a one-shot link.
	GetPresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	// GetPresignedDownloadURLAsAttachment behaves like GetPresignedDownloadURL
	// but instructs the storage service to override the response's
	// Content-Disposition header to "attachment; filename=...". Browsers
	// honor that header and force a download dialog instead of rendering
	// the object inline (which they do by default for PDFs). Use this
	// when the user clicks an explicit "Download" link — invoice PDFs,
	// credit-note PDFs, etc.
	GetPresignedDownloadURLAsAttachment(ctx context.Context, key string, filename string, expiry time.Duration) (string, error)
	Download(ctx context.Context, key string) ([]byte, error)
}
