package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/session"
)

// UserSessionRepository is the PostgreSQL adapter for user_sessions
// (migration 147 / B.4). Implements port.UserSessionRepository.
//
// Every query gets a 5-second timeout. The table is read on every
// /auth/refresh and /auth/logout call, so the adapter sticks to
// single-row INDEXed lookups (jti, user_id) — no scan-based query
// shape.
type UserSessionRepository struct {
	db *sql.DB
}

// NewUserSessionRepository wires the adapter onto a sql.DB handle.
func NewUserSessionRepository(db *sql.DB) *UserSessionRepository {
	return &UserSessionRepository{db: db}
}

// listActiveLimit caps the rows returned by ListActiveByUser. The
// realistic number of concurrent sessions per user is a single-digit
// figure (web + mobile + maybe a tablet), so 50 is two orders of
// magnitude over the upper bound — it absorbs misuse without ever
// becoming a memory hazard.
const listActiveLimit = 50

// Create inserts a new session row. Returns the underlying pq error
// if the JTI uniqueness constraint trips.
func (r *UserSessionRepository) Create(ctx context.Context, s *session.Session) error {
	if s == nil {
		return fmt.Errorf("user_sessions: nil session")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `
		INSERT INTO user_sessions (
			id, user_id, jti, parent_jti, user_agent_hash,
			ip_anonymized, login_method, created_at, last_used_at,
			expires_at, revoked_at
		) VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10, $11)
	`
	var revoked any
	if s.RevokedAt != nil {
		revoked = *s.RevokedAt
	}
	_, err := r.db.ExecContext(ctx, stmt,
		s.ID,
		s.UserID,
		s.JTI,
		s.ParentJTI,
		s.UserAgentHash,
		s.IPAnonymized,
		string(s.LoginMethod),
		s.CreatedAt,
		s.LastUsedAt,
		s.ExpiresAt,
		revoked,
	)
	if err != nil {
		return fmt.Errorf("user_sessions: insert: %w", err)
	}
	return nil
}

// FindByJTI fetches the row whose jti column equals the argument.
// Returns session.ErrNotFound when no row matches.
func (r *UserSessionRepository) FindByJTI(ctx context.Context, jti string) (*session.Session, error) {
	if jti == "" {
		return nil, session.ErrJTIRequired
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `
		SELECT id, user_id, jti, COALESCE(parent_jti, ''), user_agent_hash,
		       text(ip_anonymized), login_method, created_at, last_used_at,
		       expires_at, revoked_at
		FROM user_sessions
		WHERE jti = $1
	`
	row := r.db.QueryRowContext(ctx, stmt, jti)
	out, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user_sessions: find_by_jti: %w", err)
	}
	return out, nil
}

// Touch bumps last_used_at to NOW() for the matching session row.
// Idempotent: a missing row is not an error from the caller's POV
// (the auth flow may have raced with a logout).
func (r *UserSessionRepository) Touch(ctx context.Context, jti string) error {
	if jti == "" {
		return session.ErrJTIRequired
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `UPDATE user_sessions SET last_used_at = NOW() WHERE jti = $1`
	if _, err := r.db.ExecContext(ctx, stmt, jti); err != nil {
		return fmt.Errorf("user_sessions: touch: %w", err)
	}
	return nil
}

// Revoke sets revoked_at to NOW() for the matching session — only
// when it was previously NULL, so a second revoke does not overwrite
// the original timestamp.
func (r *UserSessionRepository) Revoke(ctx context.Context, jti string) error {
	if jti == "" {
		return session.ErrJTIRequired
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `UPDATE user_sessions SET revoked_at = NOW() WHERE jti = $1 AND revoked_at IS NULL`
	if _, err := r.db.ExecContext(ctx, stmt, jti); err != nil {
		return fmt.Errorf("user_sessions: revoke: %w", err)
	}
	return nil
}

// RevokeAllForUser stamps revoked_at on every still-active session
// belonging to userID. Used on token-reuse detection (assume the
// account is compromised) and on the future "logout from every
// device" flow.
func (r *UserSessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return session.ErrUserIDRequired
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := r.db.ExecContext(ctx, stmt, userID); err != nil {
		return fmt.Errorf("user_sessions: revoke_all_for_user: %w", err)
	}
	return nil
}

// ListActiveByUser returns every still-usable session for the user,
// newest expiry first, capped at listActiveLimit.
func (r *UserSessionRepository) ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]*session.Session, error) {
	if userID == uuid.Nil {
		return nil, session.ErrUserIDRequired
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const stmt = `
		SELECT id, user_id, jti, COALESCE(parent_jti, ''), user_agent_hash,
		       text(ip_anonymized), login_method, created_at, last_used_at,
		       expires_at, revoked_at
		FROM user_sessions
		WHERE user_id = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		ORDER BY expires_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, stmt, userID, listActiveLimit)
	if err != nil {
		return nil, fmt.Errorf("user_sessions: list_active: %w", err)
	}
	defer rows.Close()

	out := make([]*session.Session, 0, 8)
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("user_sessions: list_active scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("user_sessions: list_active iterate: %w", err)
	}
	return out, nil
}

// scanSession decodes a row into a Session value object. Reuses the
// package-level rowScanner type (declared next to scanModerationResult)
// so the column order stays in lock-step between FindByJTI and
// ListActiveByUser — drift here would silently corrupt the audit
// trail.
func scanSession(row rowScanner) (*session.Session, error) {
	var (
		s         session.Session
		parent    string
		ipText    string
		method    string
		revokedAt sql.NullTime
	)
	if err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.JTI,
		&parent,
		&s.UserAgentHash,
		&ipText,
		&method,
		&s.CreatedAt,
		&s.LastUsedAt,
		&s.ExpiresAt,
		&revokedAt,
	); err != nil {
		return nil, err
	}
	s.ParentJTI = parent
	s.LoginMethod = session.LoginMethod(method)
	s.IPAnonymized = ipText
	if revokedAt.Valid {
		t := revokedAt.Time
		s.RevokedAt = &t
	}
	return &s, nil
}
