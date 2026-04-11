package postgres

import (
	"context"
	"database/sql"
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

	accountType := u.AccountType
	if accountType == "" {
		accountType = user.AccountTypeMarketplaceOwner
	}

	query := `
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type, referrer_enabled, is_admin, status, suspended_at, suspension_reason, suspension_expires_at, banned_at, ban_reason, organization_id, linkedin_id, google_id, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.HashedPassword, u.FirstName, u.LastName, u.DisplayName,
		string(u.Role), string(accountType), u.ReferrerEnabled, u.IsAdmin, string(u.Status),
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

const userColumns = `id, email, hashed_password, first_name, last_name, display_name,
		role, account_type, session_version, referrer_enabled, is_admin, status,
		suspended_at, suspension_reason, suspension_expires_at, banned_at, ban_reason,
		organization_id, linkedin_id, google_id, email_verified, created_at, updated_at`

func (r *UserRepository) scanUserRow(scanner interface{ Scan(...any) error }) (*user.User, error) {
	u := &user.User{}
	var role, accountType, status string
	err := scanner.Scan(
		&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
		&role, &accountType, &u.SessionVersion, &u.ReferrerEnabled, &u.IsAdmin, &status,
		&u.SuspendedAt, &u.SuspensionReason, &u.SuspensionExpiresAt, &u.BannedAt, &u.BanReason,
		&u.OrganizationID, &u.LinkedInID, &u.GoogleID, &u.EmailVerified,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Role = user.Role(role)
	u.AccountType = user.AccountType(accountType)
	u.Status = user.UserStatus(status)
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	u, err := r.scanUserRow(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return u, nil
}

// GetByIDs batch-fetches users for a set of ids in a single query.
// Used by features that join a secondary dataset (e.g. organization
// members, reviews) against users without running an N+1 loop.
//
// Unknown ids are silently dropped from the result — the caller is
// expected to handle "user was deleted" gracefully (typically by
// leaving that row without an identity block).
func (r *UserRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*user.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	query := `SELECT ` + userColumns + ` FROM users WHERE id = ANY($1)`
	rows, err := r.db.QueryContext(ctx, query, pq.Array(idStrings))
	if err != nil {
		return nil, fmt.Errorf("failed to get users by ids: %w", err)
	}
	defer rows.Close()

	users := make([]*user.User, 0, len(ids))
	for rows.Next() {
		u, scanErr := r.scanUserRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", scanErr)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users rows: %w", err)
	}
	return users, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	u, err := r.scanUserRow(r.db.QueryRowContext(ctx, query, email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	accountType := u.AccountType
	if accountType == "" {
		accountType = user.AccountTypeMarketplaceOwner
	}

	query := `
		UPDATE users SET email = $2, hashed_password = $3, first_name = $4, last_name = $5, display_name = $6, role = $7, account_type = $8, referrer_enabled = $9, is_admin = $10, status = $11, suspended_at = $12, suspension_reason = $13, suspension_expires_at = $14, banned_at = $15, ban_reason = $16, organization_id = $17, linkedin_id = $18, google_id = $19, email_verified = $20
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.HashedPassword, u.FirstName, u.LastName, u.DisplayName,
		string(u.Role), string(accountType), u.ReferrerEnabled, u.IsAdmin, string(u.Status),
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

// BumpSessionVersion atomically increments the user's session_version
// counter by one and returns the new value. Every mutation that changes
// the user's effective permissions (role change, membership removal,
// suspension, password change, token theft recovery) MUST call this
// method so the auth middleware knows to reject any in-flight JWT that
// was issued with the previous version.
//
// The operation is idempotent with respect to external caches: the
// middleware always compares the JWT's session_version against the
// fresh value returned by this method (cached for a short TTL in
// Redis), so a bump takes effect on the next request.
func (r *UserRepository) BumpSessionVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var newVersion int
	err := r.db.QueryRowContext(ctx,
		`UPDATE users
		 SET session_version = session_version + 1, updated_at = now()
		 WHERE id = $1
		 RETURNING session_version`,
		userID,
	).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, user.ErrUserNotFound
		}
		return 0, fmt.Errorf("bump session version: %w", err)
	}
	return newVersion, nil
}

// GetSessionVersion reads the current session_version for a user. Used
// by the auth middleware's revocation check — typically cached in Redis
// with a short TTL so it doesn't add a DB round-trip to every request.
func (r *UserRepository) GetSessionVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var version int
	err := r.db.QueryRowContext(ctx,
		`SELECT session_version FROM users WHERE id = $1`,
		userID,
	).Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, user.ErrUserNotFound
		}
		return 0, fmt.Errorf("get session version: %w", err)
	}
	return version, nil
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
		SELECT id, email, hashed_password, first_name, last_name, display_name, role, account_type, referrer_enabled, is_admin, status, suspended_at, suspension_reason, suspension_expires_at, banned_at, ban_reason, organization_id, linkedin_id, google_id, email_verified, created_at, updated_at
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
		var role, accountType, status string
		if err := rows.Scan(
			&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
			&role, &accountType, &u.ReferrerEnabled, &u.IsAdmin, &status,
			&u.SuspendedAt, &u.SuspensionReason, &u.SuspensionExpiresAt, &u.BannedAt, &u.BanReason,
			&u.OrganizationID, &u.LinkedInID, &u.GoogleID, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan admin user: %w", err)
		}
		u.Role = user.Role(role)
		u.AccountType = user.AccountType(accountType)
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
		       role, account_type, referrer_enabled, is_admin, status,
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
		var role, accountType, status string
		if err := rows.Scan(
			&u.ID, &u.Email, &u.HashedPassword, &u.FirstName, &u.LastName, &u.DisplayName,
			&role, &accountType, &u.ReferrerEnabled, &u.IsAdmin, &status,
			&u.SuspendedAt, &u.SuspensionReason, &u.SuspensionExpiresAt,
			&u.BannedAt, &u.BanReason, &u.OrganizationID, &u.LinkedInID, &u.GoogleID,
			&u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recent signup: %w", err)
		}
		u.Role = user.Role(role)
		u.AccountType = user.AccountType(accountType)
		u.Status = user.UserStatus(status)
		users = append(users, u)
	}
	return users, rows.Err()
}

// Stripe Connect and KYC enforcement used to live on UserRepository via
// the stripe_* and kyc_* columns (migrations 040 / 044). Phase R5 moved
// them onto the organization row — the merchant of record is the team,
// not an individual user — so those methods have been removed from this
// file and their equivalents live in OrganizationRepository.
