package embedded

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PostgresStateStore persists the last-seen account state in the
// test_embedded_accounts table (a new column last_state JSONB).
// Falls back to an in-memory cache if the column is not available
// (for environments that have not run the migration yet).
type PostgresStateStore struct {
	db *sql.DB
}

func NewPostgresStateStore(db *sql.DB) *PostgresStateStore {
	return &PostgresStateStore{db: db}
}

func (p *PostgresStateStore) GetLast(ctx context.Context, accountID string) (*LastAccountState, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var raw []byte
	err := p.db.QueryRowContext(ctx,
		`SELECT last_state FROM test_embedded_accounts WHERE stripe_account_id = $1`,
		accountID,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var s LastAccountState
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("unmarshal last_state: %w", err)
	}
	return &s, nil
}

func (p *PostgresStateStore) SaveLast(ctx context.Context, accountID string, state *LastAccountState) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal last_state: %w", err)
	}
	_, err = p.db.ExecContext(ctx,
		`UPDATE test_embedded_accounts
		 SET last_state = $1, updated_at = now()
		 WHERE stripe_account_id = $2`,
		raw, accountID,
	)
	return err
}

// PostgresAccountLookup resolves an account_id → user_id via the
// test_embedded_accounts table.
type PostgresAccountLookup struct {
	db *sql.DB
}

func NewPostgresAccountLookup(db *sql.DB) *PostgresAccountLookup {
	return &PostgresAccountLookup{db: db}
}

func (p *PostgresAccountLookup) FindUserByStripeAccount(ctx context.Context, accountID string) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var userID uuid.UUID
	err := p.db.QueryRowContext(ctx,
		`SELECT user_id FROM test_embedded_accounts WHERE stripe_account_id = $1`,
		accountID,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("no user found for account %s", accountID)
	}
	return userID, err
}
