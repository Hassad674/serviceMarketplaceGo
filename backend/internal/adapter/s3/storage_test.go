package s3_test

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/s3"
)

// TestGetPresignedDownloadURLAsAttachment_EncodesContentDisposition
// verifies that the S3 adapter sets the
// `response-content-disposition` query param on the generated URL with
// the attachment disposition + the caller-supplied filename. R2 and
// MinIO honor this query parameter and replay it as the response
// header, which forces the browser to save the file instead of
// rendering it inline.
//
// We do not need a running S3-compatible server because the AWS SDK's
// presigner builds the URL purely from credentials + input — no
// network call is made.
func TestGetPresignedDownloadURLAsAttachment_EncodesContentDisposition(t *testing.T) {
	svc := adapter.NewStorageService(
		"localhost:9000",
		"AKIAEXAMPLE",
		"SECRETEXAMPLE",
		"test-bucket",
		"http://localhost:9000/test-bucket",
		false,
	)

	rawURL, err := svc.GetPresignedDownloadURLAsAttachment(
		context.Background(),
		"invoices/abc/FAC-000123.pdf",
		"FAC-000123.pdf",
		5*time.Minute,
	)
	require.NoError(t, err)

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	disposition := parsed.Query().Get("response-content-disposition")
	require.NotEmpty(t, disposition, "presigned URL must carry response-content-disposition")
	assert.True(t, strings.HasPrefix(disposition, "attachment;"),
		"disposition must be 'attachment', got %q", disposition)
	assert.Contains(t, disposition, `FAC-000123.pdf`,
		"disposition must include the requested filename")
}

// TestGetPresignedDownloadURL_NoAttachmentOverride sanity-checks that
// the legacy method does NOT set the override — preserving the inline
// rendering behavior for callers that legitimately want to preview a
// PDF in a browser tab (e.g. a future "View invoice" preview button).
func TestGetPresignedDownloadURL_NoAttachmentOverride(t *testing.T) {
	svc := adapter.NewStorageService(
		"localhost:9000",
		"AKIAEXAMPLE",
		"SECRETEXAMPLE",
		"test-bucket",
		"http://localhost:9000/test-bucket",
		false,
	)

	rawURL, err := svc.GetPresignedDownloadURL(
		context.Background(),
		"invoices/abc/FAC-000123.pdf",
		5*time.Minute,
	)
	require.NoError(t, err)

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	assert.Empty(t, parsed.Query().Get("response-content-disposition"),
		"legacy method must not force attachment disposition")
}
