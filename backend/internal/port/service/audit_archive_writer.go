package service

import "context"

// AuditArchiveRow is the dehydrated representation of a single
// audit_logs_archive row that the B.2 cold-tier sweep dumps to R2.
//
// We keep this on the port (not the adapter) because the postgres
// retention repository builds the slice and hands it to whichever
// AuditArchiveWriter is wired in main. Using a flat map[string]any
// would compile but lose the type-safety: the writer would have to
// runtime-assert every field.
//
// Field names match the source table columns 1:1 and are emitted to
// JSONL with the same names — keeping the on-disk shape stable so a
// future re-import script (or a compliance read) does not have to
// chase a translation table. Nullable columns are surfaced as a
// pointer-to-string (or pointer-to-time.Time) so JSON marshalling
// produces a real `null` instead of a zero value.
type AuditArchiveRow struct {
	ID           string         `json:"id"`
	UserID       *string        `json:"user_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType *string        `json:"resource_type,omitempty"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	IPAddress    *string        `json:"ip_address,omitempty"`
	CreatedAt    string         `json:"created_at"`
	ArchivedAt   string         `json:"archived_at"`
}

// AuditArchiveWriter is the cold-storage port: it accepts a batch of
// archived audit-log rows and writes them as a single gzipped JSONL
// bundle under the supplied key.
//
// The writer MUST:
//   - encode each row as a single JSON object on its own line
//     (newline-delimited JSON);
//   - gzip the whole bundle;
//   - upload it to the configured cold-storage backend;
//   - return only after the upload has fully durably committed
//     (i.e., the storage backend has acknowledged the write).
//
// The writer MUST NOT:
//   - retry indefinitely — surface transport errors so the sweep can
//     log + skip the batch and try again next tick;
//   - mutate the rows;
//   - leak partial uploads — multipart writes must be aborted on
//     error so a half-written object is not visible to a future read.
//
// The interface is single-method on purpose (ISP): the only client is
// the retention repository's cold-tier sweep. A future "rehydrate from
// R2" path would live on a different port (read-side, not write-side).
type AuditArchiveWriter interface {
	// WriteJSONL serialises rows to gzipped JSONL and uploads under
	// the supplied key. The key is fully qualified — the adapter does
	// not prepend a bucket prefix beyond what the storage backend
	// requires.
	WriteJSONL(ctx context.Context, key string, rows []AuditArchiveRow) error
}
