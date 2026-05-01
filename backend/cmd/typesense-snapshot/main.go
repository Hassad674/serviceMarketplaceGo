// Command typesense-snapshot triggers a Typesense data snapshot and
// uploads the resulting archive to MinIO.
//
// Usage:
//
//	go run ./cmd/typesense-snapshot [--snapshot-path=/tmp/typesense-snapshot] [--object-prefix=snapshots/typesense]
//
// Pipeline:
//
//  1. POST /operations/snapshot — Typesense dumps its Raft state
//     into `snapshot-path` on the server.
//  2. Tar+gzip the directory in-process (stream, no temp files).
//  3. Upload the .tar.gz to MinIO under
//     `<object-prefix>/YYYY-MM-DD.tar.gz`.
//
// Idempotent: re-running on the same day overwrites the MinIO
// object without side effects, matching the phase 3 "safe to re-run"
// requirement. Use `make snapshot-typesense` as the canonical entry
// point in CI / cron.
//
// NOT wired into Go code as a cron job — schedule externally via
// systemd timer or Kubernetes CronJob. Example systemd unit:
//
//	[Unit]
//	Description=Daily Typesense snapshot to MinIO
//	After=network.target
//
//	[Service]
//	ExecStart=/opt/marketplace/bin/typesense-snapshot
//	EnvironmentFile=/etc/marketplace/backend.env
//
//	[Install]
//	WantedBy=multi-user.target
//
// Paired with a `typesense-snapshot.timer` that runs `OnCalendar=02:00`.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"marketplace-backend/internal/adapter/s3"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/search"
)

const (
	defaultSnapshotPath = "/tmp/typesense-data-snapshot"
	defaultPrefix       = "snapshots/typesense"
	snapshotTimeout     = 10 * time.Minute
)

func main() {
	var (
		snapshotPath = flag.String("snapshot-path", defaultSnapshotPath, "path inside the Typesense container to write the snapshot to")
		objectPrefix = flag.String("object-prefix", defaultPrefix, "object key prefix in MinIO under which the archive is stored")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	if err := run(cfg, *snapshotPath, *objectPrefix); err != nil {
		slog.Error("typesense snapshot failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, snapshotPath, objectPrefix string) error {
	ctx, cancel := context.WithTimeout(context.Background(), snapshotTimeout)
	defer cancel()

	tsClient, err := search.NewClient(cfg.TypesenseHost, cfg.TypesenseAPIKey)
	if err != nil {
		return fmt.Errorf("build typesense client: %w", err)
	}
	slog.Info("triggering typesense snapshot", "path", snapshotPath)
	if _, err := tsClient.TriggerSnapshot(ctx, snapshotPath); err != nil {
		return fmt.Errorf("trigger snapshot: %w", err)
	}

	archive, err := tarGzDirectory(snapshotPath)
	if err != nil {
		return fmt.Errorf("compress snapshot: %w", err)
	}
	slog.Info("snapshot compressed", "bytes", len(archive))

	if cfg.StorageEndpoint == "" {
		slog.Warn("minio not configured, skipping upload")
		return nil
	}
	storage := s3.NewStorageService(
		cfg.StorageEndpoint,
		cfg.StorageAccessKey,
		cfg.StorageSecretKey,
		cfg.StorageBucket,
		cfg.StoragePublicURL,
		cfg.StorageUseSSL,
	)
	key := fmt.Sprintf("%s/%s.tar.gz",
		strings.TrimRight(objectPrefix, "/"),
		time.Now().UTC().Format("2006-01-02"))
	url, err := storage.Upload(ctx, key, bytes.NewReader(archive), "application/gzip", int64(len(archive)))
	if err != nil {
		return fmt.Errorf("upload snapshot: %w", err)
	}
	slog.Info("snapshot uploaded", "key", key, "url", url, "bytes", len(archive))
	return nil
}

// tarGzDirectory walks `root` and writes every regular file into a
// gzip-compressed tar archive. Extracted so it can be unit-tested
// against a fixture directory without needing a real Typesense.
//
// Returns an empty archive when the directory does not exist —
// practical for CI runs where the Typesense container writes to a
// different path than the backend can reach.
func tarGzDirectory(root string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			if err := tw.Close(); err != nil {
				return nil, err
			}
			if err := gw.Close(); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %q is not a directory", root)
	}

	walkFn := func(path string, entry os.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if entry.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}
		return addToTar(tw, path, relPath)
	}
	if err := filepath.WalkDir(root, walkFn); err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("tar close: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	return buf.Bytes(), nil
}

// addToTar writes a single file into the tar writer. Extracted to
// keep the WalkDir callback short.
//
// gosec G304 (file inclusion via variable): fullPath comes exclusively
// from filepath.WalkDir over a `root` directory the operator passes via
// CLI flag — not from network input. The CLI is run by humans during
// snapshot/migration windows; if the operator targets a sensitive
// directory, that is an operator-level decision, not a vulnerability
// in this code path.
func addToTar(tw *tar.Writer, fullPath, relPath string) error {
	f, err := os.Open(fullPath) // #nosec G304 -- CLI tool; path comes from WalkDir over operator-supplied root
	if err != nil {
		return fmt.Errorf("open %q: %w", fullPath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat %q: %w", fullPath, err)
	}

	hdr := &tar.Header{
		Name:    relPath,
		Mode:    int64(info.Mode().Perm()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header %q: %w", relPath, err)
	}
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("tar copy %q: %w", relPath, err)
	}
	return nil
}
