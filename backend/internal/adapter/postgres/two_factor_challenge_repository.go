package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/twofactor"
	"marketplace-backend/internal/port/repository"
)

// TwoFactorChallengeRepository persists 2FA challenge rows in
// Postgres. Implements repository.TwoFactorChallengeRepository.
//
// All queries go through the slow_query helpers (Exec / QueryRow) so
// the slow-query observability layer applies uniformly. Every method
// uses the package-wide queryTimeout (5s) — challenges are tiny rows
// with a single indexed lookup so even the cold-cache path stays well
// under that budget.
type TwoFactorChallengeRepository struct {
	db *sql.DB
}

// NewTwoFactorChallengeRepository wires the postgres adapter. main.go
// passes the same *sql.DB used for every other repository.
func NewTwoFactorChallengeRepository(db *sql.DB) *TwoFactorChallengeRepository {
	return &TwoFactorChallengeRepository{db: db}
}

// Create persists a fresh challenge. The row is INSERTed with the
// fields the domain prepared (id, user_id, code_hash, attempts_left,
// expires_at, used_at — always nil here, created_at, client_ip,
// user_agent_hash). A uniqueness violation on id is treated as a
// driver bug rather than a domain error because the domain generates
// id with uuid.New().
func (r *TwoFactorChallengeRepository) Create(ctx context.Context, c *twofactor.Challenge) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var clientIP any
	if c.ClientIP != nil {
		clientIP = c.ClientIP.String()
	}
	var userAgentHash any
	if c.UserAgentHash != "" {
		userAgentHash = c.UserAgentHash
	}

	const query = `
        INSERT INTO two_factor_challenges (
            id, user_id, code_hash, attempts_left, expires_at,
            used_at, created_at, client_ip, user_agent_hash
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.UserID, c.CodeHash, c.AttemptsLeft, c.ExpiresAt,
		c.UsedAt, c.CreatedAt, clientIP, userAgentHash,
	)
	if err != nil {
		return fmt.Errorf("two_factor_challenge: create: %w", err)
	}
	return nil
}

// FindLatestPendingForUser returns the most recent NOT-USED challenge
// for the given user. Backed by the partial index
// idx_2fa_user_pending so this is a single-row index lookup even on a
// large table. The query also filters expires_at > NOW() to avoid
// surfacing stale rows the caller would have to discard anyway —
// keeps the app layer free of an additional IsExpired check on the
// happy path.
func (r *TwoFactorChallengeRepository) FindLatestPendingForUser(ctx context.Context, userID uuid.UUID) (*twofactor.Challenge, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
        SELECT id, user_id, code_hash, attempts_left, expires_at,
               used_at, created_at, client_ip, user_agent_hash
        FROM two_factor_challenges
        WHERE user_id = $1
          AND used_at IS NULL
          AND expires_at > NOW()
        ORDER BY created_at DESC
        LIMIT 1`

	row := QueryRow(ctx, r.db, query, userID)
	c, err := scanTwoFactorChallenge(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrTwoFactorChallengeNotFound
		}
		return nil, fmt.Errorf("two_factor_challenge: find latest pending: %w", err)
	}
	return c, nil
}

// MarkUsed sets used_at to NOW() for the row. Idempotent — if used_at
// is already non-NULL the UPDATE is a no-op (row count stays at 1).
func (r *TwoFactorChallengeRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `UPDATE two_factor_challenges SET used_at = NOW() WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("two_factor_challenge: mark used: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("two_factor_challenge: mark used rows affected: %w", err)
	}
	if rows == 0 {
		return repository.ErrTwoFactorChallengeNotFound
	}
	return nil
}

// DecrementAttempts subtracts one from attempts_left, floored at 0
// via GREATEST so a buggy caller cannot drive the counter negative.
func (r *TwoFactorChallengeRepository) DecrementAttempts(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
        UPDATE two_factor_challenges
        SET attempts_left = GREATEST(attempts_left - 1, 0)
        WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("two_factor_challenge: decrement attempts: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("two_factor_challenge: decrement attempts rows affected: %w", err)
	}
	if rows == 0 {
		return repository.ErrTwoFactorChallengeNotFound
	}
	return nil
}

// scanTwoFactorChallenge maps a row into a domain Challenge. The
// helper hides the nullable column dance (used_at, client_ip,
// user_agent_hash) behind a single signature so both single-row and
// future multi-row scans reuse it.
func scanTwoFactorChallenge(scanner interface{ Scan(...any) error }) (*twofactor.Challenge, error) {
	var (
		c             twofactor.Challenge
		usedAt        sql.NullTime
		clientIP      sql.NullString
		userAgentHash sql.NullString
		expiresAt     time.Time
		createdAt     time.Time
	)
	if err := scanner.Scan(
		&c.ID, &c.UserID, &c.CodeHash, &c.AttemptsLeft, &expiresAt,
		&usedAt, &createdAt, &clientIP, &userAgentHash,
	); err != nil {
		return nil, err
	}
	c.ExpiresAt = expiresAt
	c.CreatedAt = createdAt
	if usedAt.Valid {
		t := usedAt.Time
		c.UsedAt = &t
	}
	if clientIP.Valid {
		parsed := net.ParseIP(clientIP.String)
		if parsed != nil {
			c.ClientIP = &parsed
		}
	}
	if userAgentHash.Valid {
		c.UserAgentHash = userAgentHash.String
	}
	return &c, nil
}
