package service

import (
	"context"
	"io"
	"time"
)

// BulkDeleteResult captures the outcome of a single key inside a
// best-effort BulkDelete batch. The slice returned by
// StorageService.BulkDelete contains one entry per requested key, with
// Err non-nil when the key failed. The implementation MUST NOT abort
// the batch on the first failure — every key gets its own row so the
// caller can persist a per-key audit (used by the GDPR purge for
// right-to-erasure compliance evidence).
type BulkDeleteResult struct {
	Key string
	Err error
}

type StorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	Delete(ctx context.Context, key string) error
	// BulkDelete deletes every key in `keys` in a best-effort manner.
	// The return slice contains one BulkDeleteResult per requested
	// key, in the same order, with Err non-nil for failures. The
	// method itself returns a non-nil error only when the batch
	// could not be issued at all (e.g. transport failure on every
	// retry); per-object errors do NOT bubble up so callers can log
	// + audit the failures without aborting the broader purge tx.
	BulkDelete(ctx context.Context, keys []string) ([]BulkDeleteResult, error)
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
