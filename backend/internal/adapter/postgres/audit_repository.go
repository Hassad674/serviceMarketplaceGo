package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/pkg/cursor"
)

// AuditRepository is the PostgreSQL adapter for the append-only
// audit_logs table created in migration 078.
//
// Repository responsibilities are deliberately narrow: insert a new
// row, list by resource, list by user. There is no Update, no Delete,
// and no aggregation — audit mutations and queries are simple on
// purpose, and any reporting built on top of this table runs through
// ad-hoc SQL in admin tooling.
type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Log inserts a new audit row. The caller MUST NOT propagate this
// error to the end user — audit insertion is best-effort from the
// perspective of a business flow. The service layer should wrap
// Log calls in a goroutine or an error-discarding defer so a DB
// hiccup does not break the main path.
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

	_, err = r.db.ExecContext(ctx, `
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

// ListByResource returns the audit entries for a given resource,
// ordered by created_at DESC, id DESC. Cursor-paginated the same
// way as other list endpoints so the admin UI can scroll through
// long histories without OFFSET.
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

	var rows *sql.Rows
	var err error
	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, `
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
			return nil, "", fmt.Errorf("audit list: decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, `
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
		return nil, "", fmt.Errorf("audit list by resource: %w", err)
	}
	defer rows.Close()

	return r.scanAndPaginate(rows, limit)
}

// ListByUser returns the audit entries attributable to a user,
// ordered newest-first.
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

	var rows *sql.Rows
	var err error
	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, `
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
			return nil, "", fmt.Errorf("audit list: decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, `
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
		return nil, "", fmt.Errorf("audit list by user: %w", err)
	}
	defer rows.Close()

	return r.scanAndPaginate(rows, limit)
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
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &entry.Metadata)
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}
	if ipStr.Valid {
		parsed := net.ParseIP(ipStr.String)
		if parsed != nil {
			entry.IPAddress = &parsed
		}
	}
	return &entry, nil
}

// Silence unused import warning for pq — kept for consistency with
// the rest of the postgres package which uses pq.Error to map unique
// constraint violations. Audit log has no unique constraints today,
// but leaving the import documents the intent if one is added later.
var _ = pq.Error{}
