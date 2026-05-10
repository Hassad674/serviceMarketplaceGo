package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"marketplace-backend/internal/domain/consent"
)

// ConsentLogRepository is the PostgreSQL adapter for the consent_log
// table created in migration 139. Append-only — no Update / Delete.
type ConsentLogRepository struct {
	db *sql.DB
}

// NewConsentLogRepository wires the adapter to a sql.DB handle. The
// caller (cmd/api wire layer) owns the lifecycle of the underlying
// pool.
func NewConsentLogRepository(db *sql.DB) *ConsentLogRepository {
	return &ConsentLogRepository{db: db}
}

// Create inserts a new consent_log row. Uses the entry's ID and
// CreatedAt verbatim — the domain layer owns identity + timestamp so
// the persisted row matches what the service returned to the handler.
func (r *ConsentLogRepository) Create(ctx context.Context, entry *consent.Entry) error {
	if entry == nil {
		return fmt.Errorf("consent_log: nil entry")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `
		INSERT INTO consent_log (
			id, user_id, session_id, categories, action,
			ip_anonymized, user_agent_hash, created_at
		) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8)
	`

	var userID interface{}
	if entry.UserID != nil {
		userID = *entry.UserID
	}

	if _, err := r.db.ExecContext(
		ctx,
		stmt,
		entry.ID,
		userID,
		entry.SessionID,
		pq.Array(entry.Categories),
		string(entry.Action),
		entry.IPAnonymized,
		entry.UserAgentHash,
		entry.CreatedAt,
	); err != nil {
		return fmt.Errorf("consent_log: insert: %w", err)
	}
	return nil
}
