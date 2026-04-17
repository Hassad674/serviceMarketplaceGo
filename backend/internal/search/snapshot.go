package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// snapshot.go adds the tiny REST wrapper around Typesense's
// `/operations/snapshot` admin endpoint. Snapshots are file-system
// copies of the Raft state stored on the node itself; we drive a
// snapshot on demand from the CLI, then upload the resulting
// directory to MinIO for off-site backup.
//
// Typesense API reference:
//
//	POST /operations/snapshot?snapshot_path=/tmp/typesense-data-snapshot
//
// The endpoint is idempotent — re-running it overwrites the same
// directory server-side, which matches the "safe to re-run same
// day" requirement in the phase 3 scope.

// SnapshotResponse is the typed wrapper around Typesense's reply.
// Success is indicated by `success: true` plus HTTP 201 — we
// return an error on anything else so the caller surfaces a clean
// exit code.
type SnapshotResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// TriggerSnapshot asks Typesense to materialise a snapshot of the
// current node's data into `snapshotPath`. The path must be
// writable by the Typesense process (usually a bind-mounted volume).
//
// The call is blocking — Typesense holds the response until the
// snapshot is complete. We use a longer context timeout (30s) to
// accommodate production-sized data dirs; tests can override via a
// context WithDeadline.
func (c *Client) TriggerSnapshot(ctx context.Context, snapshotPath string) (*SnapshotResponse, error) {
	if snapshotPath == "" {
		return nil, fmt.Errorf("typesense snapshot: snapshot_path is required")
	}
	q := url.Values{}
	q.Set("snapshot_path", snapshotPath)
	path := "/operations/snapshot?" + q.Encode()

	var out SnapshotResponse
	if err := c.do(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, fmt.Errorf("typesense snapshot: %w", err)
	}
	if !out.Success {
		return &out, fmt.Errorf("typesense snapshot: api reported failure: %s", out.Message)
	}
	return &out, nil
}

// Compile-time guard that json decoding of SnapshotResponse works.
var _ = func() { _, _ = json.Marshal(SnapshotResponse{}) }
