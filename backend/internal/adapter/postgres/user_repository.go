package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
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
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, referrer_enabled, is_admin, status, suspended_at, suspension_reason, suspension_expires_at, banned_at, ban_reason, organization_id, linkedin_id, google_id, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.HashedPassword, u.FirstName, u.LastName, u.DisplayName,
		string(u.Role), u.ReferrerEnabled, u.IsAdmin, string(u.Status),
		u.SuspendedAt, u.SuspensionReason, u.SuspensionExpiresAt, u.BannedAt, u.BanReason,
		u.OrganizationID, u.LinkedInID, u.GoogleID, u.EmailVerified, u.CreatedAt, u.UpdatedAt,
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
		SELECT id, email, hashed_password, first_name, last_name, display_name, role, referrer_enabled, is_admin, status, suspended_at, suspension_reason, suspension_expires_at, banned_at, ban_reason, organization_id, linkedin_id, google_id, email_verified, stripe_account_id, kyc_first_earning_at, created_at, updated_at
		FROM users WHERE id = $1`

	u := &user.User{}
	var role, status string
	var stripeAcctID sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
		&role, &u.ReferrerEnabled, &u.IsAdmin, &status,
		&u.SuspendedAt, &u.SuspensionReason, &u.SuspensionExpiresAt, &u.BannedAt, &u.BanReason,
		&u.OrganizationID, &u.LinkedInID, &u.GoogleID, &u.EmailVerified,
		&stripeAcctID, &u.KYCFirstEarningAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	u.Role = user.Role(role)
	u.Status = user.UserStatus(status)
	if stripeAcctID.Valid {
		u.StripeAccountID = &stripeAcctID.String
	}
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, email, hashed_password, first_name, last_name, display_name, role, referrer_enabled, is_admin, status, suspended_at, suspension_reason, suspension_expires_at, banned_at, ban_reason, organization_id, linkedin_id, google_id, email_verified, stripe_account_id, kyc_first_earning_at, created_at, updated_at
		FROM users WHERE email = $1`

	u := &user.User{}
	var role, status string
	var stripeAcctID sql.NullString
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
		&role, &u.ReferrerEnabled, &u.IsAdmin, &status,
		&u.SuspendedAt, &u.SuspensionReason, &u.SuspensionExpiresAt, &u.BannedAt, &u.BanReason,
		&u.OrganizationID, &u.LinkedInID, &u.GoogleID, &u.EmailVerified,
		&stripeAcctID, &u.KYCFirstEarningAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	u.Role = user.Role(role)
	u.Status = user.UserStatus(status)
	if stripeAcctID.Valid {
		u.StripeAccountID = &stripeAcctID.String
	}
	return u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		UPDATE users SET email = $2, hashed_password = $3, first_name = $4, last_name = $5, display_name = $6, role = $7, referrer_enabled = $8, is_admin = $9, status = $10, suspended_at = $11, suspension_reason = $12, suspension_expires_at = $13, banned_at = $14, ban_reason = $15, organization_id = $16, linkedin_id = $17, google_id = $18, email_verified = $19
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.HashedPassword, u.FirstName, u.LastName, u.DisplayName,
		string(u.Role), u.ReferrerEnabled, u.IsAdmin, string(u.Status),
		u.SuspendedAt, u.SuspensionReason, u.SuspensionExpiresAt, u.BannedAt, u.BanReason,
		u.OrganizationID, u.LinkedInID, u.GoogleID, u.EmailVerified,
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

func (r *UserRepository) ListAdmin(ctx context.Context, filters repository.AdminUserFilters) ([]*user.User, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var conditions []string
	var args []any
	argIdx := 1

	if filters.Role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, filters.Role)
		argIdx++
	}
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d OR display_name ILIKE $%d)",
			argIdx, argIdx, argIdx, argIdx,
		))
		args = append(args, searchPattern)
		argIdx++
	}
	if filters.Reported {
		conditions = append(conditions,
			"EXISTS (SELECT 1 FROM reports r WHERE r.target_type = 'user' AND r.target_id = users.id AND r.status = 'pending')",
		)
	}
	useOffset := filters.Page > 0 && filters.Cursor == ""

	if !useOffset && filters.Cursor != "" {
		c, err := cursor.Decode(filters.Cursor)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argIdx, argIdx+1))
			args = append(args, c.CreatedAt, c.ID)
			argIdx += 2
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var offsetClause string
	if useOffset {
		offsetClause = fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, (filters.Page-1)*limit)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, email, hashed_password, first_name, last_name, display_name, role, referrer_enabled, is_admin, status, suspended_at, suspension_reason, suspension_expires_at, banned_at, ban_reason, organization_id, linkedin_id, google_id, email_verified, created_at, updated_at
		FROM users %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d%s`, where, argIdx, offsetClause)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u := &user.User{}
		var role, status string
		if err := rows.Scan(
			&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
			&role, &u.ReferrerEnabled, &u.IsAdmin, &status,
			&u.SuspendedAt, &u.SuspensionReason, &u.SuspensionExpiresAt, &u.BannedAt, &u.BanReason,
			&u.OrganizationID, &u.LinkedInID, &u.GoogleID, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan admin user: %w", err)
		}
		u.Role = user.Role(role)
		u.Status = user.UserStatus(status)
		users = append(users, u)
	}

	var nextCursor string
	if len(users) > limit {
		last := users[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		users = users[:limit]
	}

	return users, nextCursor, nil
}

func (r *UserRepository) CountAdmin(ctx context.Context, filters repository.AdminUserFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var conditions []string
	var args []any
	argIdx := 1

	if filters.Role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, filters.Role)
		argIdx++
	}
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d OR display_name ILIKE $%d)",
			argIdx, argIdx, argIdx, argIdx,
		))
		args = append(args, searchPattern)
		argIdx++
	}
	if filters.Reported {
		conditions = append(conditions,
			"EXISTS (SELECT 1 FROM reports r WHERE r.target_type = 'user' AND r.target_id = users.id AND r.status = 'pending')",
		)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM users %s", where)

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count admin users: %w", err)
	}

	return count, nil
}

func (r *UserRepository) CountByRole(ctx context.Context) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, "SELECT role, COUNT(*) FROM users GROUP BY role")
	if err != nil {
		return nil, fmt.Errorf("count users by role: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return nil, fmt.Errorf("scan role count: %w", err)
		}
		result[role] = count
	}
	return result, rows.Err()
}

func (r *UserRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, "SELECT status, COUNT(*) FROM users GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("count users by status: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}
		result[status] = count
	}
	return result, rows.Err()
}

func (r *UserRepository) RecentSignups(ctx context.Context, limit int) ([]*user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, email, hashed_password, first_name, last_name, display_name,
		       role, referrer_enabled, is_admin, status,
		       suspended_at, suspension_reason, suspension_expires_at,
		       banned_at, ban_reason, organization_id, linkedin_id, google_id,
		       email_verified, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("recent signups: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u := &user.User{}
		var role, status string
		if err := rows.Scan(
			&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
			&role, &u.ReferrerEnabled, &u.IsAdmin, &status,
			&u.SuspendedAt, &u.SuspensionReason, &u.SuspensionExpiresAt,
			&u.BannedAt, &u.BanReason, &u.OrganizationID, &u.LinkedInID, &u.GoogleID,
			&u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recent signup: %w", err)
		}
		u.Role = user.Role(role)
		u.Status = user.UserStatus(status)
		users = append(users, u)
	}
	return users, rows.Err()
}

// ---------------------------------------------------------------------------
// Stripe account operations (migration 040)
//
// These methods manipulate the stripe_* columns on users added in
// migration 040. They live on UserRepository because the Stripe account
// is a 1-1 attribute of the user, not a separate entity.
// ---------------------------------------------------------------------------

// GetStripeAccount returns the Stripe account_id + country for a user.
// Returns empty strings + sql.ErrNoRows if the user has no Stripe account yet.
func (r *UserRepository) GetStripeAccount(ctx context.Context, userID uuid.UUID) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var accountID sql.NullString
	var country sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT stripe_account_id, stripe_account_country FROM users WHERE id = $1`,
		userID,
	).Scan(&accountID, &country)
	if err != nil {
		return "", "", err
	}
	if !accountID.Valid {
		return "", "", sql.ErrNoRows
	}
	return accountID.String, country.String, nil
}

// FindUserIDByStripeAccount reverse-lookup: Stripe account_id → user_id.
// Used by the embedded Notifier to route webhooks to the right user.
func (r *UserRepository) FindUserIDByStripeAccount(ctx context.Context, accountID string) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM users WHERE stripe_account_id = $1`,
		accountID,
	).Scan(&userID)
	return userID, err
}

// SetStripeAccount persists the Stripe account_id + country for a user.
// Idempotent — safe to call on every /account-session to refresh.
func (r *UserRepository) SetStripeAccount(ctx context.Context, userID uuid.UUID, accountID, country string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE users
		 SET stripe_account_id      = $2,
		     stripe_account_country = $3,
		     updated_at             = now()
		 WHERE id = $1`,
		userID, accountID, country,
	)
	return err
}

// ClearStripeAccount wipes the Stripe account mapping for a user.
// Used by the test reset flow to allow recreating with a different country.
func (r *UserRepository) ClearStripeAccount(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE users
		 SET stripe_account_id      = NULL,
		     stripe_account_country = NULL,
		     stripe_last_state      = NULL,
		     updated_at             = now()
		 WHERE id = $1`,
		userID,
	)
	return err
}

// GetStripeLastState returns the raw JSONB snapshot the Notifier uses
// to diff incoming webhooks. Returns (nil, nil) when no state is stored.
func (r *UserRepository) GetStripeLastState(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var raw sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT stripe_last_state FROM users WHERE id = $1`,
		userID,
	).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	return []byte(raw.String), nil
}

// SaveStripeLastState persists the Notifier's last-seen state for a user.
func (r *UserRepository) SaveStripeLastState(ctx context.Context, userID uuid.UUID, state []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE users
		 SET stripe_last_state = $2::jsonb,
		     updated_at        = now()
		 WHERE id = $1`,
		userID, string(state),
	)
	return err
}

// ---------------------------------------------------------------------------
// KYC enforcement (migration 044)
// ---------------------------------------------------------------------------

// SetKYCFirstEarning records when the user first had funds available for
// withdrawal. Idempotent: only writes if kyc_first_earning_at is NULL.
func (r *UserRepository) SetKYCFirstEarning(ctx context.Context, userID uuid.UUID, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE users
		 SET kyc_first_earning_at = $2, updated_at = now()
		 WHERE id = $1 AND kyc_first_earning_at IS NULL`,
		userID, at,
	)
	return err
}

// GetKYCPendingUsers returns all users who have earned money but have NOT
// completed KYC (no stripe_account_id). Used by the KYC scheduler.
func (r *UserRepository) GetKYCPendingUsers(ctx context.Context) ([]*user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, display_name, role, stripe_account_id,
		        kyc_first_earning_at, kyc_restriction_notified_at
		 FROM users
		 WHERE kyc_first_earning_at IS NOT NULL
		   AND stripe_account_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("get kyc pending users: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u := &user.User{}
		var role string
		var stripeID sql.NullString
		var notifiedRaw sql.NullString
		if err := rows.Scan(
			&u.ID, &u.Email, &u.DisplayName, &role, &stripeID,
			&u.KYCFirstEarningAt, &notifiedRaw,
		); err != nil {
			return nil, fmt.Errorf("scan kyc pending user: %w", err)
		}
		u.Role = user.Role(role)
		if stripeID.Valid {
			u.StripeAccountID = &stripeID.String
		}
		if notifiedRaw.Valid && notifiedRaw.String != "" {
			_ = json.Unmarshal([]byte(notifiedRaw.String), &u.KYCRestrictionNotifiedAt)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SaveKYCNotificationState persists the notification tier state JSONB.
func (r *UserRepository) SaveKYCNotificationState(ctx context.Context, userID uuid.UUID, state map[string]time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal kyc notification state: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE users SET kyc_restriction_notified_at = $2::jsonb, updated_at = now() WHERE id = $1`,
		userID, string(raw),
	)
	return err
}
