// Package r2 hosts adapter code that targets Cloudflare R2's
// S3-compatible API for non-storage workloads. The general-purpose
// object storage adapter lives in adapter/s3 (it is already
// multi-cloud — MinIO in dev, R2 in prod). This package is for
// purpose-built clients that need their own bucket / prefix policy,
// notably the B.2 audit-log cold-storage writer.
//
// Why split from adapter/s3:
//   - The B.2 sweep writes very large objects (gzipped JSONL, up to
//     a few MiB) under a deterministic year/month prefix. The
//     general StorageService is tuned for user-facing files (avatars,
//     invoice PDFs) and exposes presign / public-URL helpers that
//     are irrelevant to a write-once-then-archive flow.
//   - Cost rationale: archive writes never need a presigned URL. The
//     adapter intentionally exposes only the WriteJSONL method so
//     the cold-storage code path cannot accidentally upgrade to
//     "user-facing" semantics (with caching, public URL, etc.).
//
// Configuration parity with adapter/s3: identical endpoint /
// access_key / secret_key / use_ssl / region knobs so a deployment
// can reuse the existing R2 bucket or point at a separate cold-tier
// bucket via STORAGE_AUDIT_COLD_BUCKET. The decision is wiring-time,
// not adapter-time.
package r2

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"marketplace-backend/internal/port/service"
)

// uploadTimeout caps the per-batch upload wall time. The bundle is at
// most a few MiB gzipped (10k rows × ~500 bytes raw → ~5MB raw → ~1MB
// gzipped at the typical compression ratio for JSON). Even on a
// degraded link, 60s is generous; surfacing the timeout to the sweep
// is preferable to hanging the retention pass.
const uploadTimeout = 60 * time.Second

// AuditArchiveWriter implements service.AuditArchiveWriter against an
// S3-compatible backend (Cloudflare R2 in production, MinIO in dev /
// tests). It uploads each batch as a single gzipped JSONL object in
// one PutObject call — the bundle is always small enough to fit a
// single-part write, so there is no multipart bookkeeping.
//
// Single-part is a deliberate choice: multipart adds a 3-step lifecycle
// (Initiate / UploadPart / Complete or Abort) which complicates error
// handling on a sweep that prefers "all-or-nothing" semantics. If a
// future tuning requires larger bundles, switch to multipart at that
// point — premature optimization is worse than the simple path.
type AuditArchiveWriter struct {
	client *s3.Client
	bucket string
}

// Config carries the boot-time knobs for the writer. Mirrors the
// shape of adapter/s3.NewStorageService so wiring code stays
// parallel.
type Config struct {
	// Endpoint is the host:port of the S3-compatible service
	// (e.g. "<account>.r2.cloudflarestorage.com" or "localhost:9000").
	Endpoint string
	// AccessKey + SecretKey are the static credentials for the bucket.
	AccessKey string
	SecretKey string
	// Bucket is the destination bucket. Each row's key is built as
	// audit-cold/<year>/<month>/<batch_id>.jsonl.gz so multi-month
	// retention queries can range-scan by prefix.
	Bucket string
	// UseSSL toggles https vs http. Production R2 is always https;
	// MinIO in compose uses http by default.
	UseSSL bool
	// Region is "auto" for R2; left as a knob so an AWS S3 deployment
	// can specify its real region. Defaults to "auto" when empty.
	Region string
}

// NewAuditArchiveWriter constructs a writer using static credentials
// and an S3-compatible endpoint. Returns an error when the supplied
// config is missing the bucket or endpoint — the rest is optional
// and falls back to sensible defaults (Region "auto", UseSSL false).
func NewAuditArchiveWriter(cfg Config) (*AuditArchiveWriter, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("r2 audit archive: endpoint required")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("r2 audit archive: bucket required")
	}
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	region := cfg.Region
	if region == "" {
		region = "auto"
	}
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(fmt.Sprintf("%s://%s", scheme, cfg.Endpoint)),
		Region:       region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		// Path-style addressing is required for MinIO and supported
		// by R2 for non-CNAME endpoints. Mirrors adapter/s3.
		UsePathStyle: true,
	})
	return &AuditArchiveWriter{client: client, bucket: cfg.Bucket}, nil
}

// NewAuditArchiveWriterFromClient is the seam used by tests: pass an
// already-configured *s3.Client (typically pointed at an httptest
// fake S3) and the bucket name. Production code uses
// NewAuditArchiveWriter.
func NewAuditArchiveWriterFromClient(client *s3.Client, bucket string) *AuditArchiveWriter {
	return &AuditArchiveWriter{client: client, bucket: bucket}
}

// WriteJSONL encodes rows as newline-delimited JSON, gzip-compresses
// the resulting buffer, and uploads it under `key`. Each row produces
// exactly one line in the bundle; an empty `rows` slice short-circuits
// to a no-op (no point creating an empty 20-byte gzip header object).
//
// The upload uses ContentType "application/x-ndjson" and
// ContentEncoding "gzip" so any future read path that streams the
// object directly to a client renders correctly without re-decoding.
func (w *AuditArchiveWriter) WriteJSONL(ctx context.Context, key string, rows []service.AuditArchiveRow) error {
	if len(rows) == 0 {
		return nil
	}
	if key == "" {
		return errors.New("r2 audit archive: key required")
	}
	payload, err := encodeBundle(rows)
	if err != nil {
		return fmt.Errorf("r2 audit archive: encode bundle: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()
	_, err = w.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(w.bucket),
		Key:             aws.String(key),
		Body:            bytes.NewReader(payload),
		ContentLength:   aws.Int64(int64(len(payload))),
		ContentType:     aws.String("application/x-ndjson"),
		ContentEncoding: aws.String("gzip"),
	})
	if err != nil {
		return fmt.Errorf("r2 audit archive: put %q: %w", key, err)
	}
	return nil
}

// encodeBundle is split out so the test can exercise it without a
// network. It writes one JSON object per row, separated by '\n', then
// gzips the whole buffer. We use the standard library's gzip writer
// at default compression — JSON compresses ~5×–10×, which is far
// past the cost-savings threshold the cold tier needs (R2 storage is
// ~$0.015/GB vs Postgres' ~$0.10/GB).
func encodeBundle(rows []service.AuditArchiveRow) ([]byte, error) {
	var raw bytes.Buffer
	enc := json.NewEncoder(&raw)
	enc.SetEscapeHTML(false) // audit metadata may contain HTML chars; preserve them verbatim
	for i := range rows {
		if err := enc.Encode(rows[i]); err != nil {
			return nil, fmt.Errorf("encode row %d: %w", i, err)
		}
	}
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(raw.Bytes()); err != nil {
		_ = gz.Close()
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	return compressed.Bytes(), nil
}
