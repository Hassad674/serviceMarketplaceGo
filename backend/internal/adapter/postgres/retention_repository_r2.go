package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/retention"
	"marketplace-backend/internal/port/service"
)

// errAuditArchiveWriterNotWired is returned when StrategyArchiveToR2
// is invoked on a repository that was not configured with an
// AuditArchiveWriter. The retention service catches the error and
// logs it; the sweep skips that policy on this tick. Returning a
// distinct sentinel rather than a generic error so the operator-side
// log filters can alert on it specifically.
var errAuditArchiveWriterNotWired = errors.New("retention: archive_to_r2 strategy requires WithAuditArchiveWriter")

// sweepArchiveAuditLogsToR2 implements the B.2 cold-tier sweep on
// audit_logs_archive. The flow runs in two phases per row:
//
//  1. UPLOAD phase (this method, when r2_key IS NULL):
//     - SELECT a batch of rows whose archived_at is older than the
//       cold-tier cutoff and that have not yet been uploaded.
//     - Build the JSONL bundle, gzip + PutObject to R2 under
//       audit-cold/<year>/<month>/<batch_id>.jsonl.gz.
//     - UPDATE the same rows with r2_key = '<key>'. The UPDATE is in
//       its own transaction; if it fails after a successful upload,
//       the next tick will simply re-upload (R2 keys are deterministic
//       per-batch — see batchID below — so re-uploads overwrite the
//       same object idempotently).
//
//  2. DELETE phase (this method, when r2_key IS NOT NULL):
//     - SELECT a batch of rows that have an r2_key set AND are still
//       past the cutoff.
//     - DELETE them. The R2 object is the canonical copy.
//
// One Sweep call only does ONE of the two phases. The dispatcher
// alternates: if any rows still need uploading on this tick, do the
// upload; otherwise, hard-delete a batch. This keeps each Sweep call
// short and bounded — the retention service's loop drives both phases
// to completion across multiple ticks (see Service.runPolicy).
//
// Year/month bucketing in the key (`<year>/<month>/`) is intentional:
// a future cold-tier deletion (when we eventually want to drop data
// past 4 years) can range-delete an entire monthly prefix instead of
// listing all keys. R2 charges per Class A operation; prefix deletes
// keep the cost cap predictable.
func (r *RetentionRepository) sweepArchiveAuditLogsToR2(
	ctx context.Context,
	policy retention.Policy,
	now time.Time,
) (int, error) {
	if r.archiveWriter == nil {
		return 0, errAuditArchiveWriterNotWired
	}
	if policy.Table != "audit_logs_archive" {
		return 0, fmt.Errorf("retention: archive_to_r2 only valid for audit_logs_archive, got %q", policy.Table)
	}

	cutoff := policy.Cutoff(now)
	batch := policy.EffectiveBatchSize()

	// Phase decision: prefer the UPLOAD phase when there is at least
	// one pending row, otherwise advance to DELETE. Two cheap COUNTs
	// (each backed by a partial index from migration 149) are far
	// cheaper than running both phases unconditionally.
	pendingUpload, err := r.countPendingUploads(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("retention archive_to_r2: count pending uploads: %w", err)
	}
	if pendingUpload > 0 {
		return r.uploadAuditArchiveBatch(ctx, cutoff, batch, now)
	}
	return r.deleteUploadedAuditArchiveBatch(ctx, cutoff, batch)
}

// countPendingUploads returns the number of rows past the cold-tier
// cutoff that are still waiting for their R2 dump. Uses the partial
// index idx_audit_logs_archive_pending_upload (migration 149) so the
// count is O(matching rows), not O(table size).
func (r *RetentionRepository) countPendingUploads(ctx context.Context, cutoff time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var n int
	err := r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM audit_logs_archive
         WHERE r2_key IS NULL
           AND archived_at < $1`, cutoff).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// uploadAuditArchiveBatch implements phase 1: SELECT a batch of
// pending rows, build the JSONL bundle, write it to R2, then UPDATE
// the rows with the resulting r2_key. Returns the number of rows that
// successfully transitioned to "uploaded".
//
// The SELECT and the UPDATE are in separate transactions on purpose:
// the upload happens between them, and we explicitly DO NOT want the
// SELECT lock held across a multi-second network call. SKIP LOCKED on
// the SELECT prevents two parallel sweeps from picking the same row.
func (r *RetentionRepository) uploadAuditArchiveBatch(
	ctx context.Context,
	cutoff time.Time,
	batch int,
	now time.Time,
) (int, error) {
	rows, ids, err := r.fetchPendingArchiveBatch(ctx, cutoff, batch)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	key := buildAuditColdKey(now)
	if err := r.archiveWriter.WriteJSONL(ctx, key, rows); err != nil {
		return 0, fmt.Errorf("retention archive_to_r2: upload %q: %w", key, err)
	}
	affected, err := r.markBatchUploaded(ctx, ids, key)
	if err != nil {
		return 0, fmt.Errorf("retention archive_to_r2: mark uploaded: %w", err)
	}
	return affected, nil
}

// fetchPendingArchiveBatch SELECTs at most `batch` rows past the
// cold-tier cutoff that are not yet uploaded. Returns the
// service-layer rows (ready to hand to the writer) plus the matching
// id slice for the follow-up UPDATE.
//
// FOR UPDATE SKIP LOCKED keeps concurrent schedulers cooperating
// without coordination. The transaction is short — we COMMIT before
// going to the network — so the held locks live only for the duration
// of the SELECT.
func (r *RetentionRepository) fetchPendingArchiveBatch(
	ctx context.Context,
	cutoff time.Time,
	batch int,
) ([]service.AuditArchiveRow, []uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// #nosec G201 -- batch is a validated int from policy.EffectiveBatchSize.
	q := fmt.Sprintf(`
        SELECT id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at, archived_at
          FROM audit_logs_archive
         WHERE r2_key IS NULL
           AND archived_at < $1
         ORDER BY archived_at ASC
         LIMIT %d
         FOR UPDATE SKIP LOCKED`, batch)
	sqlRows, err := tx.QueryContext(ctx, q, cutoff)
	if err != nil {
		return nil, nil, fmt.Errorf("select batch: %w", err)
	}
	defer sqlRows.Close()

	out := make([]service.AuditArchiveRow, 0, batch)
	ids := make([]uuid.UUID, 0, batch)
	for sqlRows.Next() {
		row, id, err := scanArchiveRow(sqlRows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, row)
		ids = append(ids, id)
	}
	if err := sqlRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate batch: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit select tx: %w", err)
	}
	return out, ids, nil
}

// scanArchiveRow converts one DB row into the port-shaped struct the
// writer consumes. Nullable columns become *string / map[string]any
// so the JSON payload preserves real `null`s instead of empty
// strings.
func scanArchiveRow(rs *sql.Rows) (service.AuditArchiveRow, uuid.UUID, error) {
	var (
		id           uuid.UUID
		userID       sql.NullString
		action       string
		resourceType sql.NullString
		resourceID   sql.NullString
		metadataRaw  []byte
		ipRaw        sql.NullString
		createdAt    time.Time
		archivedAt   time.Time
	)
	if err := rs.Scan(&id, &userID, &action, &resourceType, &resourceID, &metadataRaw, &ipRaw, &createdAt, &archivedAt); err != nil {
		return service.AuditArchiveRow{}, uuid.Nil, fmt.Errorf("scan archive row: %w", err)
	}
	row := service.AuditArchiveRow{
		ID:         id.String(),
		Action:     action,
		CreatedAt:  createdAt.UTC().Format(time.RFC3339Nano),
		ArchivedAt: archivedAt.UTC().Format(time.RFC3339Nano),
	}
	if userID.Valid {
		v := userID.String
		row.UserID = &v
	}
	if resourceType.Valid {
		v := resourceType.String
		row.ResourceType = &v
	}
	if resourceID.Valid {
		v := resourceID.String
		row.ResourceID = &v
	}
	if ipRaw.Valid {
		// Strip the netmask Postgres adds to inet/cidr scans
		// ("203.0.113.7/32") to keep the JSONL payload free of
		// trivia that would surprise a downstream reader.
		ipStr := ipRaw.String
		if ip, _, err := net.ParseCIDR(ipStr); err == nil {
			ipStr = ip.String()
		}
		row.IPAddress = &ipStr
	}
	if len(metadataRaw) > 0 {
		var m map[string]any
		if err := json.Unmarshal(metadataRaw, &m); err == nil && len(m) > 0 {
			row.Metadata = m
		}
	}
	return row, id, nil
}

// markBatchUploaded sets r2_key on the rows whose ids were just
// successfully uploaded. Runs in its own short transaction so the
// UPDATE is the boundary of "this batch is durably in R2".
//
// We re-check r2_key IS NULL in the WHERE clause: if a parallel
// sweeper raced and already stamped the row, we leave their key
// alone (R2 stores both versions; the older one is overwritten on
// the next bucket lifecycle pass). This keeps the operation
// idempotent without coordination.
func (r *RetentionRepository) markBatchUploaded(
	ctx context.Context,
	ids []uuid.UUID,
	key string,
) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := r.db.ExecContext(ctx, `
        UPDATE audit_logs_archive
           SET r2_key = $1
         WHERE id = ANY($2::uuid[])
           AND r2_key IS NULL`, key, pgUUIDArrayLiteral(ids))
	if err != nil {
		return 0, fmt.Errorf("update r2_key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return int(n), nil
}

// deleteUploadedAuditArchiveBatch implements phase 2: hard-delete
// rows whose payload is already in R2 and that are past the cold
// cutoff. The materialised CTE shape mirrors the other strategies'
// "DELETE with LIMIT" pattern so the affected count is exactly batch
// size at most — never the per-outer-row pitfall of a naive subquery.
func (r *RetentionRepository) deleteUploadedAuditArchiveBatch(
	ctx context.Context,
	cutoff time.Time,
	batch int,
) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// #nosec G201 -- table hard-coded, batch validated.
	q := fmt.Sprintf(`
        WITH eligible AS MATERIALIZED (
            SELECT id FROM audit_logs_archive
             WHERE r2_key IS NOT NULL
               AND archived_at < $1
             ORDER BY archived_at ASC
             LIMIT %d
             FOR UPDATE SKIP LOCKED
        )
        DELETE FROM audit_logs_archive
         WHERE id IN (SELECT id FROM eligible)`, batch)
	res, err := r.db.ExecContext(ctx, q, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete uploaded batch: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return int(n), nil
}

// buildAuditColdKey produces the R2 object key for a single batch:
// `audit-cold/<year>/<month>/<batch_id>.jsonl.gz` with year/month
// derived from `now` (the sweep tick clock). batch_id is a UUID v4
// so two parallel sweeps in the same minute never collide on a key.
func buildAuditColdKey(now time.Time) string {
	t := now.UTC()
	return fmt.Sprintf("audit-cold/%04d/%02d/%s.jsonl.gz",
		t.Year(), int(t.Month()), uuid.NewString())
}

// pgUUIDArrayLiteral formats a slice of UUIDs as the literal Postgres
// expects for `= ANY($1::uuid[])`. We avoid importing lib/pq's Array
// helper here because the rest of this package already builds these
// literals manually (see the integration test helper) and consistency
// matters more than a few saved bytes.
func pgUUIDArrayLiteral(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return "{}"
	}
	out := make([]byte, 0, len(ids)*40)
	out = append(out, '{')
	for i, id := range ids {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, id.String()...)
	}
	out = append(out, '}')
	return string(out)
}
