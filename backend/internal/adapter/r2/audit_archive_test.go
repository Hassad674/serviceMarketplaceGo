package r2_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/r2"
	"marketplace-backend/internal/port/service"
)

// fakeR2 is a minimal S3-compatible PutObject endpoint. It records
// every key + body it receives so the test can decompress the JSONL
// payload and assert content. We intentionally do NOT use a MinIO
// testcontainer here — the fake covers the whole contract (auth
// headers ignored, key+body roundtrip) without docker.
type fakeR2 struct {
	mu      sync.Mutex
	objects map[string][]byte
	headers map[string]http.Header
}

func newFakeR2() *fakeR2 {
	return &fakeR2{objects: map[string][]byte{}, headers: map[string]http.Header{}}
}

func (f *fakeR2) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Path style: /<bucket>/<key>. Strip the leading bucket.
		p, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/"))
		require.NoError(t, err)
		parts := strings.SplitN(p, "/", 2)
		require.Len(t, parts, 2, "path %q must have bucket and key", p)
		key := parts[1]

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		_ = r.Body.Close()

		f.mu.Lock()
		f.objects[key] = body
		f.headers[key] = r.Header.Clone()
		f.mu.Unlock()

		w.Header().Set("ETag", `"deadbeef"`)
		w.WriteHeader(http.StatusOK)
	})
}

// newWriterPointedAt builds an AuditArchiveWriter that talks to the
// supplied test server.
func newWriterPointedAt(t *testing.T, srv *httptest.Server, bucket string) *adapter.AuditArchiveWriter {
	t.Helper()
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(srv.URL),
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
		UsePathStyle: true,
	})
	return adapter.NewAuditArchiveWriterFromClient(client, bucket)
}

func TestAuditArchiveWriter_WriteJSONL_Roundtrip(t *testing.T) {
	srv := httptest.NewServer(newFakeR2().handler(t))
	t.Cleanup(srv.Close)
	fr := newFakeR2()
	srv.Config.Handler = fr.handler(t)

	w := newWriterPointedAt(t, srv, "marketplace")

	userID := "11111111-1111-1111-1111-111111111111"
	resType := "user"
	rows := []service.AuditArchiveRow{
		{
			ID:         "aaaa1111-1111-1111-1111-111111111111",
			UserID:     &userID,
			Action:     "login_success",
			ResourceType: &resType,
			Metadata:   map[string]any{"user_agent": "test"},
			CreatedAt:  "2026-01-01T00:00:00Z",
			ArchivedAt: "2026-04-01T00:00:00Z",
		},
		{
			ID:         "bbbb2222-2222-2222-2222-222222222222",
			Action:     "logout",
			CreatedAt:  "2026-01-02T00:00:00Z",
			ArchivedAt: "2026-04-02T00:00:00Z",
		},
	}

	const key = "audit-cold/2026/05/batch.jsonl.gz"
	require.NoError(t, w.WriteJSONL(context.Background(), key, rows))

	fr.mu.Lock()
	body := fr.objects[key]
	hdr := fr.headers[key]
	fr.mu.Unlock()

	require.NotEmpty(t, body, "expected the writer to upload the bundle under %q", key)

	// Headers signal gzip + ndjson so a future cold-read can stream.
	assert.Equal(t, "application/x-ndjson", hdr.Get("Content-Type"))
	assert.Equal(t, "gzip", hdr.Get("Content-Encoding"))

	// Decompress + verify each line is a valid JSON object matching
	// the row at the same index.
	gz, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer gz.Close()
	raw, err := io.ReadAll(gz)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	require.Len(t, lines, len(rows))
	for i, line := range lines {
		var got service.AuditArchiveRow
		require.NoError(t, json.Unmarshal([]byte(line), &got), "line %d: %s", i, line)
		assert.Equal(t, rows[i].ID, got.ID)
		assert.Equal(t, rows[i].Action, got.Action)
		assert.Equal(t, rows[i].CreatedAt, got.CreatedAt)
	}
}

func TestAuditArchiveWriter_WriteJSONL_EmptySliceIsNoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not have hit the network for empty input, got %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)
	w := newWriterPointedAt(t, srv, "marketplace")
	require.NoError(t, w.WriteJSONL(context.Background(), "audit-cold/x.jsonl.gz", nil))
	require.NoError(t, w.WriteJSONL(context.Background(), "audit-cold/x.jsonl.gz", []service.AuditArchiveRow{}))
}

func TestAuditArchiveWriter_WriteJSONL_RejectsEmptyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not have reached the network for empty key")
	}))
	t.Cleanup(srv.Close)
	w := newWriterPointedAt(t, srv, "marketplace")
	err := w.WriteJSONL(context.Background(), "", []service.AuditArchiveRow{{ID: "x", Action: "y"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key required")
}

func TestAuditArchiveWriter_WriteJSONL_PropagatesUploadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	w := newWriterPointedAt(t, srv, "marketplace")
	err := w.WriteJSONL(context.Background(), "audit-cold/x.jsonl.gz", []service.AuditArchiveRow{{ID: "x", Action: "y"}})
	require.Error(t, err)
}

func TestNewAuditArchiveWriter_RequiresEndpointAndBucket(t *testing.T) {
	_, err := adapter.NewAuditArchiveWriter(adapter.Config{Bucket: "b"})
	require.Error(t, err)
	_, err = adapter.NewAuditArchiveWriter(adapter.Config{Endpoint: "localhost:9000"})
	require.Error(t, err)
	_, err = adapter.NewAuditArchiveWriter(adapter.Config{Endpoint: "localhost:9000", Bucket: "b"})
	require.NoError(t, err)
}

func TestAuditArchiveWriter_CompressionRatio(t *testing.T) {
	// JSON audit logs compress aggressively. We assert at least 2× —
	// in practice the ratio is closer to 5×–10× because the
	// metadata is repetitive. The 2× lower-bound just guards against
	// "we forgot to actually gzip the bundle".
	rows := make([]service.AuditArchiveRow, 200)
	for i := range rows {
		rows[i] = service.AuditArchiveRow{
			ID:         "11111111-1111-1111-1111-111111111111",
			Action:     "login_success",
			Metadata:   map[string]any{"user_agent": "Mozilla/5.0 (X11; Linux x86_64)", "key": "value"},
			CreatedAt:  "2026-01-01T00:00:00Z",
			ArchivedAt: "2026-04-01T00:00:00Z",
		}
	}
	srv := httptest.NewServer(newFakeR2().handler(t))
	t.Cleanup(srv.Close)
	fr := newFakeR2()
	srv.Config.Handler = fr.handler(t)

	w := newWriterPointedAt(t, srv, "b")
	require.NoError(t, w.WriteJSONL(context.Background(), "audit-cold/k", rows))

	fr.mu.Lock()
	compressedLen := len(fr.objects["audit-cold/k"])
	fr.mu.Unlock()

	// Compute raw JSONL length for comparison.
	var raw bytes.Buffer
	enc := json.NewEncoder(&raw)
	enc.SetEscapeHTML(false)
	for i := range rows {
		require.NoError(t, enc.Encode(rows[i]))
	}
	rawLen := raw.Len()

	require.Greater(t, compressedLen, 0)
	require.Greater(t, rawLen, compressedLen*2,
		"expected at least 2× compression, got raw=%d compressed=%d", rawLen, compressedLen)
	t.Logf("compression ratio: raw=%d compressed=%d ratio=%.2fx",
		rawLen, compressedLen, float64(rawLen)/float64(compressedLen))
}
