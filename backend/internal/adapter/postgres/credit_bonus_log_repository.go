package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
)

// CreditBonusLogRepository implements repository.CreditBonusLogRepository.
type CreditBonusLogRepository struct {
	db *sql.DB
}

func NewCreditBonusLogRepository(db *sql.DB) *CreditBonusLogRepository {
	return &CreditBonusLogRepository{db: db}
}

func (r *CreditBonusLogRepository) Insert(ctx context.Context, entry *repository.CreditBonusLogEntry) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO credit_bonus_log
		 (id, provider_id, client_id, proposal_id, client_card_fingerprint,
		  credits_awarded, status, block_reason, proposal_created_at, proposal_paid_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.ID, entry.ProviderID, entry.ClientID, entry.ProposalID,
		optionalString(entry.ClientCardFingerprint),
		entry.CreditsAwarded, entry.Status,
		optionalString(entry.BlockReason),
		optionalTime(entry.ProposalCreatedAt), entry.ProposalPaidAt,
	)
	if err != nil {
		return fmt.Errorf("insert credit bonus log: %w", err)
	}
	return nil
}

func (r *CreditBonusLogRepository) CountByProviderAndClient(
	ctx context.Context, providerID, clientID uuid.UUID, since time.Time,
) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM credit_bonus_log
		 WHERE provider_id = $1 AND client_id = $2 AND created_at >= $3`,
		providerID, clientID, since,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count credit bonus log: %w", err)
	}
	return count, nil
}

func (r *CreditBonusLogRepository) ListPendingReview(
	ctx context.Context, cursor string, limit int,
) ([]*repository.CreditBonusLogEntry, string, error) {
	return r.listByFilter(ctx, "pending_review", cursor, limit)
}

func (r *CreditBonusLogRepository) ListAll(
	ctx context.Context, cursor string, limit int,
) ([]*repository.CreditBonusLogEntry, string, error) {
	return r.listByFilter(ctx, "", cursor, limit)
}

func (r *CreditBonusLogRepository) GetByID(
	ctx context.Context, id uuid.UUID,
) (*repository.CreditBonusLogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT id, provider_id, client_id, proposal_id,
		        COALESCE(client_card_fingerprint, ''), credits_awarded,
		        status, COALESCE(block_reason, ''),
		        COALESCE(proposal_created_at, '0001-01-01'::timestamptz),
		        proposal_paid_at, created_at
		 FROM credit_bonus_log WHERE id = $1`, id)

	entry, err := scanBonusLogEntry(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credit bonus log entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get credit bonus log: %w", err)
	}
	return entry, nil
}

func (r *CreditBonusLogRepository) UpdateStatus(
	ctx context.Context, id uuid.UUID, status string, creditsAwarded int,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx,
		`UPDATE credit_bonus_log
		 SET status = $2, credits_awarded = $3
		 WHERE id = $1`,
		id, status, creditsAwarded,
	)
	if err != nil {
		return fmt.Errorf("update credit bonus log status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("credit bonus log entry not found")
	}
	return nil
}

func (r *CreditBonusLogRepository) listByFilter(
	ctx context.Context, statusFilter string, cursor string, limit int,
) ([]*repository.CreditBonusLogEntry, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	baseQuery := `SELECT id, provider_id, client_id, proposal_id,
	        COALESCE(client_card_fingerprint, ''), credits_awarded,
	        status, COALESCE(block_reason, ''),
	        COALESCE(proposal_created_at, '0001-01-01'::timestamptz),
	        proposal_paid_at, created_at
	 FROM credit_bonus_log`

	if statusFilter != "" && cursor == "" {
		rows, err = r.db.QueryContext(ctx,
			baseQuery+` WHERE status = $1 ORDER BY created_at DESC, id DESC LIMIT $2`,
			statusFilter, limit+1)
	} else if statusFilter != "" && cursor != "" {
		cursorTime, cursorID := decodeBonusCursor(cursor)
		rows, err = r.db.QueryContext(ctx,
			baseQuery+` WHERE status = $1 AND (created_at, id) < ($2, $3)
			ORDER BY created_at DESC, id DESC LIMIT $4`,
			statusFilter, cursorTime, cursorID, limit+1)
	} else if cursor == "" {
		rows, err = r.db.QueryContext(ctx,
			baseQuery+` ORDER BY created_at DESC, id DESC LIMIT $1`,
			limit+1)
	} else {
		cursorTime, cursorID := decodeBonusCursor(cursor)
		rows, err = r.db.QueryContext(ctx,
			baseQuery+` WHERE (created_at, id) < ($1, $2)
			ORDER BY created_at DESC, id DESC LIMIT $3`,
			cursorTime, cursorID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list credit bonus log: %w", err)
	}
	defer rows.Close()

	var entries []*repository.CreditBonusLogEntry
	for rows.Next() {
		e, scanErr := scanBonusLogRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("scan credit bonus log: %w", scanErr)
		}
		entries = append(entries, e)
	}

	var nextCursor string
	if len(entries) > limit {
		entries = entries[:limit]
		last := entries[limit-1]
		nextCursor = encodeBonusCursor(last.CreatedAt, last.ID)
	}

	return entries, nextCursor, nil
}

func scanBonusLogEntry(row *sql.Row) (*repository.CreditBonusLogEntry, error) {
	var e repository.CreditBonusLogEntry
	err := row.Scan(
		&e.ID, &e.ProviderID, &e.ClientID, &e.ProposalID,
		&e.ClientCardFingerprint, &e.CreditsAwarded,
		&e.Status, &e.BlockReason,
		&e.ProposalCreatedAt, &e.ProposalPaidAt, &e.CreatedAt,
	)
	return &e, err
}

func scanBonusLogRow(rows *sql.Rows) (*repository.CreditBonusLogEntry, error) {
	var e repository.CreditBonusLogEntry
	err := rows.Scan(
		&e.ID, &e.ProviderID, &e.ClientID, &e.ProposalID,
		&e.ClientCardFingerprint, &e.CreditsAwarded,
		&e.Status, &e.BlockReason,
		&e.ProposalCreatedAt, &e.ProposalPaidAt, &e.CreatedAt,
	)
	return &e, err
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func encodeBonusCursor(t time.Time, id uuid.UUID) string {
	data, _ := json.Marshal(map[string]string{
		"created_at": t.Format(time.RFC3339Nano),
		"id":         id.String(),
	})
	return base64.StdEncoding.EncodeToString(data)
}

func decodeBonusCursor(cursor string) (time.Time, uuid.UUID) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return time.Time{}, uuid.Nil
	}
	t, _ := time.Parse(time.RFC3339Nano, m["created_at"])
	id, _ := uuid.Parse(m["id"])
	return t, id
}
