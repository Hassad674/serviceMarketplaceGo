package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/user"
)

const queryTimeout = 5 * time.Second

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, referrer_enabled, is_admin, organization_id, linkedin_id, google_id, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.HashedPassword, u.FirstName, u.LastName, u.DisplayName,
		string(u.Role), u.ReferrerEnabled, u.IsAdmin, u.OrganizationID,
		u.LinkedInID, u.GoogleID, u.EmailVerified, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return user.ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, email, hashed_password, first_name, last_name, display_name, role, referrer_enabled, is_admin, organization_id, linkedin_id, google_id, email_verified, created_at, updated_at
		FROM users WHERE id = $1`

	u := &user.User{}
	var role string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
		&role, &u.ReferrerEnabled, &u.IsAdmin, &u.OrganizationID,
		&u.LinkedInID, &u.GoogleID, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	u.Role = user.Role(role)
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, email, hashed_password, first_name, last_name, display_name, role, referrer_enabled, is_admin, organization_id, linkedin_id, google_id, email_verified, created_at, updated_at
		FROM users WHERE email = $1`

	u := &user.User{}
	var role string
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
		&role, &u.ReferrerEnabled, &u.IsAdmin, &u.OrganizationID,
		&u.LinkedInID, &u.GoogleID, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	u.Role = user.Role(role)
	return u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		UPDATE users SET email = $2, hashed_password = $3, first_name = $4, last_name = $5, display_name = $6, role = $7, referrer_enabled = $8, is_admin = $9, organization_id = $10, linkedin_id = $11, google_id = $12, email_verified = $13
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.HashedPassword, u.FirstName, u.LastName, u.DisplayName,
		string(u.Role), u.ReferrerEnabled, u.IsAdmin, u.OrganizationID,
		u.LinkedInID, u.GoogleID, u.EmailVerified,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return user.ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}
