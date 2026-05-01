package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/pkg/cursor"
)

// auditMetadataCorruptKey is the sentinel key inserted into a returned
// Entry.Metadata when the row's metadata JSON failed to unmarshal.
// Callers can detect a corrupt row without having to peer into the
// adapter and trigger a follow-up flag (alerting, admin review).
//
// Closes BUG-20: previously `_ = json.Unmarshal(metadata, ...)` swallowed
// every failure silently — a bug in the audit log is exactly the bug we
// can no longer afford to miss, because the audit log is the very tool
// that surfaces other bugs. The new path emits a WARN with the entry
// ID and metadata size so on-call can spot a stream of corruption,
// while still returning the entry (truncated metadata is better than
// dropping the row entirely from a list endpoint).
const auditMetadataCorruptKey = "_metadata_corrupt"

// AuditRepository is the PostgreSQL adapter for the append-only
// audit_logs table created in migration 078.
//
// Repository responsibilities are deliberately narrow: insert a new
// row, list by resource, list by user. There is no Update, no Delete,
// and no aggregation — audit mutations and queries are simple on
// purpose, and any reporting built on top of this table runs through
// ad-hoc SQL in admin tooling.
//
// BUG-NEW-04 path 2/8: audit_logs is RLS-protected by migration 125
// with the policy
//
//   USING (user_id = current_setting('app.current_user_id', true)::uuid)
//
// Migration 129 added WITH CHECK (true) so INSERTs pass even when the
// tenant context is unset (BUG-NEW-07). The repository now also wraps
// reads + writes in RunInTxWithTenant when a TxRunner is wired so:
//   - Log fires under app.current_user_id = entry.UserID (or uuid.Nil
//     for system-actor paths — WITH CHECK (true) lets those through).
//   - ListByUser fires under app.current_user_id = userID parameter
//     so the rows actually return under the non-superuser role.
//   - ListByResource fires under uuid.Nil (the caller is asking for
//     ALL actors who touched a resource — admin tooling). Returns the
//     empty set under non-superuser without a manual privileged path,
//     which is the safe failure mode.
type AuditRepository struct {
	db       *sql.DB
	txRunner *TxRunner
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper. Wired
// from cmd/api/main.go so every audit_log read/write fires inside
// RunInTxWithTenant. Returns the same pointer so the wiring chain
// stays terse.
func (r *AuditRepository) WithTxRunner(runner *TxRunner) *AuditRepository {
	r.txRunner = runner
	return r
}

// Log inserts a new audit row. The caller MUST NOT propagate this
// error to the end user — audit insertion is best-effort from the
// perspective of a business flow. The service layer should wrap
// Log calls in a goroutine or an error-discarding defer so a DB
// hiccup does not break the main path.
//
// When a TxRunner is wired, the INSERT runs inside
// RunInTxWithTenant(uuid.Nil, entry.UserID, ...) so app.current_user_id
// is set to the actor before the write. WITH CHECK (true) from
// migration 129 means the INSERT would succeed even without context
// — but the explicit setter is the right defensive default and keeps
// parity with the rest of the RLS migration (path 2/8 of BUG-NEW-04).
func (r *AuditRepository) Log(ctx context.Context, entry *audit.Entry) error {
	if entry == nil {
		return fmt.Errorf("audit log: nil entry")
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("audit log: marshal metadata: %w", err)
	}

	var ipArg any
	if entry.IPAddress != nil {
		ipArg = entry.IPAddress.String()
	} else {
		ipArg = nil
	}

	var resourceTypeArg any
	if entry.ResourceType != "" {
		resourceTypeArg = string(entry.ResourceType)
	} else {
		resourceTypeArg = nil
	}

	exec := func(runner sqlExecutor) error {
		_, err := runner.ExecContext(ctx, `
			INSERT INTO audit_logs (
				id, user_id, action, resource_type, resource_id,
				metadata, ip_address, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			entry.ID,
			entry.UserID,
			string(entry.Action),
			resourceTypeArg,
			entry.ResourceID,
			metadataJSON,
			ipArg,
			entry.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("audit log: insert: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		// entry.UserID is *uuid.UUID — uuid.Nil when the audit row has
		// no actor (system worker). SetTenantContext skips the user
		// setter for uuid.Nil, leaving app.current_user_id unset. WITH
		// CHECK (true) from migration 129 lets the INSERT through; the
		// USING expression will only match this row to ITS OWN actor on
		// later reads (or to no one for system rows — admin tooling
		// reads those through a privileged path).
		actor := uuid.Nil
		if entry.UserID != nil {
			actor = *entry.UserID
		}
		return r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, actor, func(tx *sql.Tx) error {
			return exec(tx)
		})
	}
	return exec(r.db)
}

// ListByResource returns the audit entries for a given resource,
// ordered by created_at DESC, id DESC. Cursor-paginated the same
// way as other list endpoints so the admin UI can scroll through
// long histories without OFFSET.
//
// Wrapped in tenant tx with uuid.Nil for the user context when a
// runner is wired: an admin reading a resource's full audit trail
// has no specific user filter — the policy filters by user_id, so
// under non-superuser this read returns the empty set, which is the
// safe failure mode. Admin tooling that needs cross-tenant reads
// goes through a privileged DB role (out of scope for this round).
func (r *AuditRepository) ListByResource(
	ctx context.Context,
	resourceType audit.ResourceType,
	resourceID uuid.UUID,
	cursorStr string,
	limit int,
) ([]*audit.Entry, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var entries []*audit.Entry
	var nextCursor string

	exec := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var err error
		if cursorStr == "" {
			rows, err = runner.QueryContext(ctx, `
				SELECT id, user_id, action, resource_type, resource_id,
				       metadata, ip_address, created_at
				FROM audit_logs
				WHERE resource_type = $1 AND resource_id = $2
				ORDER BY created_at DESC, id DESC
				LIMIT $3`,
				string(resourceType), resourceID, limit+1)
		} else {
			c, decErr := cursor.Decode(cursorStr)
			if decErr != nil {
				return fmt.Errorf("audit list: decode cursor: %w", decErr)
			}
			rows, err = runner.QueryContext(ctx, `
				SELECT id, user_id, action, resource_type, resource_id,
				       metadata, ip_address, created_at
				FROM audit_logs
				WHERE resource_type = $1 AND resource_id = $2
				  AND (created_at, id) < ($3, $4)
				ORDER BY created_at DESC, id DESC
				LIMIT $5`,
				string(resourceType), resourceID, c.CreatedAt, c.ID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("audit list by resource: %w", err)
		}
		defer rows.Close()
		es, nc, err := r.scanAndPaginate(rows, limit)
		if err != nil {
			return err
		}
		entries = es
		nextCursor = nc
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, uuid.Nil, func(tx *sql.Tx) error {
			return exec(tx)
		})
		if err != nil {
			return nil, "", err
		}
		return entries, nextCursor, nil
	}

	if err := exec(r.db); err != nil {
		return nil, "", err
	}
	return entries, nextCursor, nil
}

// ListByUser returns the audit entries attributable to a user,
// ordered newest-first.
//
// Wrapped in tenant tx with the userID parameter as the
// app.current_user_id setter so the rows return under non-superuser.
func (r *AuditRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	cursorStr string,
	limit int,
) ([]*audit.Entry, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var entries []*audit.Entry
	var nextCursor string

	exec := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var err error
		if cursorStr == "" {
			rows, err = runner.QueryContext(ctx, `
				SELECT id, user_id, action, resource_type, resource_id,
				       metadata, ip_address, created_at
				FROM audit_logs
				WHERE user_id = $1
				ORDER BY created_at DESC, id DESC
				LIMIT $2`,
				userID, limit+1)
		} else {
			c, decErr := cursor.Decode(cursorStr)
			if decErr != nil {
				return fmt.Errorf("audit list: decode cursor: %w", decErr)
			}
			rows, err = runner.QueryContext(ctx, `
				SELECT id, user_id, action, resource_type, resource_id,
				       metadata, ip_address, created_at
				FROM audit_logs
				WHERE user_id = $1
				  AND (created_at, id) < ($2, $3)
				ORDER BY created_at DESC, id DESC
				LIMIT $4`,
				userID, c.CreatedAt, c.ID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("audit list by user: %w", err)
		}
		defer rows.Close()
		es, nc, err := r.scanAndPaginate(rows, limit)
		if err != nil {
			return err
		}
		entries = es
		nextCursor = nc
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, userID, func(tx *sql.Tx) error {
			return exec(tx)
		})
		if err != nil {
			return nil, "", err
		}
		return entries, nextCursor, nil
	}

	if err := exec(r.db); err != nil {
		return nil, "", err
	}
	return entries, nextCursor, nil
}

// scanAndPaginate walks the result set, extracts one extra row to
// detect "has more", and encodes the next cursor from the last item
// that stays in the returned slice.
func (r *AuditRepository) scanAndPaginate(rows *sql.Rows, limit int) ([]*audit.Entry, string, error) {
	var entries []*audit.Entry
	for rows.Next() {
		entry, err := scanAuditRow(rows)
		if err != nil {
			return nil, "", fmt.Errorf("audit scan row: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("audit rows iteration: %w", err)
	}

	var nextCursor string
	if len(entries) > limit {
		last := entries[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		entries = entries[:limit]
	}
	return entries, nextCursor, nil
}

// scanAuditRow turns a single row into a domain Entry. Keeps the
// scan logic in one place so ListByResource and ListByUser share
// the same column order.
func scanAuditRow(rows *sql.Rows) (*audit.Entry, error) {
	var (
		entry        audit.Entry
		userID       uuid.NullUUID
		resourceType sql.NullString
		resourceID   uuid.NullUUID
		metadata     []byte
		ipStr        sql.NullString
	)
	err := rows.Scan(
		&entry.ID,
		&userID,
		&entry.Action,
		&resourceType,
		&resourceID,
		&metadata,
		&ipStr,
		&entry.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		id := userID.UUID
		entry.UserID = &id
	}
	if resourceType.Valid {
		entry.ResourceType = audit.ResourceType(resourceType.String)
	}
	if resourceID.Valid {
		id := resourceID.UUID
		entry.ResourceID = &id
	}
	entry.Metadata = parseAuditMetadata(entry.ID, metadata)
	if ipStr.Valid {
		parsed := net.ParseIP(ipStr.String)
		if parsed != nil {
			entry.IPAddress = &parsed
		}
	}
	return &entry, nil
}

// parseAuditMetadata decodes a JSONB metadata column into a map. When
// the bytes are empty or nil, it returns an empty (non-nil) map — the
// public Entry contract guarantees Metadata is never nil so callers can
// `entry.Metadata[k]` without a guard.
//
// Closes BUG-20: previously `_ = json.Unmarshal(metadata, &entry.Metadata)`
// swallowed every failure. Corrupt metadata was returned as an empty
// map indistinguishable from a row that was logged with no metadata
// at all. We now:
//
//  1. emit a structured WARN with the row id and the byte size so on-call
//     can detect a corruption stream and confirm the row is corrupt
//     vs missing,
//  2. tag the returned map with auditMetadataCorruptKey so admin UIs
//     can flag the row to the operator without re-querying the DB,
//  3. still return a usable Entry — dropping the row would leave a
//     hole in the audit timeline, which is exactly what we are
//     trying to prevent.
func parseAuditMetadata(entryID uuid.UUID, raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		slog.Warn("audit: metadata unmarshal failed",
			"audit_entry_id", entryID,
			"metadata_size", len(raw),
			"error", err.Error(),
		)
		return map[string]any{
			auditMetadataCorruptKey: err.Error(),
		}
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}

// Silence unused import warning for pq — kept for consistency with
// the rest of the postgres package which uses pq.Error to map unique
// constraint violations. Audit log has no unique constraints today,
// but leaving the import documents the intent if one is added later.
var _ = pq.Error{}
