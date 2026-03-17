package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

type PasswordResetRepository struct {
	db *sql.DB
}

func NewPasswordResetRepository(db *sql.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(ctx context.Context, pr *repository.PasswordReset) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO password_resets (id, user_id, token, expires_at, used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		pr.ID, pr.UserID, pr.Token, pr.ExpiresAt, pr.Used, pr.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create password reset: %w", err)
	}

	return nil
}

func (r *PasswordResetRepository) GetByToken(ctx context.Context, token string) (*repository.PasswordReset, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, user_id, token, expires_at, used, created_at
		FROM password_resets
		WHERE token = $1 AND used = false AND expires_at > now()`

	pr := &repository.PasswordReset{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&pr.ID, &pr.UserID, &pr.Token, &pr.ExpiresAt, &pr.Used, &pr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUnauthorized
		}
		return nil, fmt.Errorf("failed to get password reset by token: %w", err)
	}

	return pr, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `UPDATE password_resets SET used = true WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark password reset as used: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("password reset not found")
	}

	return nil
}

func (r *PasswordResetRepository) DeleteExpired(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `DELETE FROM password_resets WHERE expires_at < $1 OR used = true`

	_, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired password resets: %w", err)
	}

	return nil
}
