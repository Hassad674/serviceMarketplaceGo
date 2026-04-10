package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

// OrganizationMemberRepository persists organization_members rows.
// The single-Owner invariant is enforced at the DB level by the partial
// unique index idx_org_members_unique_owner.
type OrganizationMemberRepository struct {
	db *sql.DB
}

func NewOrganizationMemberRepository(db *sql.DB) *OrganizationMemberRepository {
	return &OrganizationMemberRepository{db: db}
}

const orgMemberCols = `id, organization_id, user_id, role, title, joined_at, created_at, updated_at`

func (r *OrganizationMemberRepository) Create(ctx context.Context, member *organization.Member) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO organization_members (`+orgMemberCols+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		member.ID, member.OrganizationID, member.UserID, string(member.Role),
		member.Title, member.JoinedAt, member.CreatedAt, member.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// Either (org_id, user_id) duplicate or a second Owner attempt.
			// Callers can disambiguate by checking CountByRole first if the
			// exact reason matters, but for most flows "already a member"
			// covers both.
			return organization.ErrAlreadyMember
		}
		return fmt.Errorf("insert organization member: %w", err)
	}
	return nil
}

func (r *OrganizationMemberRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.Member, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgMemberCols+` FROM organization_members WHERE id = $1`, id)
	return r.scanOne(row)
}

func (r *OrganizationMemberRepository) FindByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*organization.Member, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgMemberCols+` FROM organization_members WHERE organization_id = $1 AND user_id = $2`,
		orgID, userID)
	return r.scanOne(row)
}

func (r *OrganizationMemberRepository) FindOwner(ctx context.Context, orgID uuid.UUID) (*organization.Member, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgMemberCols+` FROM organization_members WHERE organization_id = $1 AND role = 'owner'`,
		orgID)
	return r.scanOne(row)
}

// FindUserPrimaryOrg returns the user's single organization membership.
// V1 assumption: one user belongs to at most one organization, so we
// select the first match (ordered by joined_at DESC for determinism in
// case the invariant is ever violated).
func (r *OrganizationMemberRepository) FindUserPrimaryOrg(ctx context.Context, userID uuid.UUID) (*organization.Member, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgMemberCols+`
		 FROM organization_members
		 WHERE user_id = $1
		 ORDER BY joined_at DESC
		 LIMIT 1`,
		userID)
	return r.scanOne(row)
}

func (r *OrganizationMemberRepository) List(ctx context.Context, params repository.ListMembersParams) ([]*organization.Member, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{params.OrganizationID}
	where := "WHERE organization_id = $1"
	argIdx := 2

	if params.Cursor != "" {
		c, err := cursor.Decode(params.Cursor)
		if err == nil {
			where += fmt.Sprintf(" AND (joined_at, id) < ($%d, $%d)", argIdx, argIdx+1)
			args = append(args, c.CreatedAt, c.ID)
			argIdx += 2
		}
	}

	query := fmt.Sprintf(
		`SELECT `+orgMemberCols+`
		 FROM organization_members %s
		 ORDER BY joined_at DESC, id DESC
		 LIMIT $%d`, where, argIdx)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list organization members: %w", err)
	}
	defer rows.Close()

	var members []*organization.Member
	for rows.Next() {
		m, err := scanMemberRow(rows)
		if err != nil {
			return nil, "", err
		}
		members = append(members, m)
	}

	var nextCursor string
	if len(members) > limit {
		last := members[limit-1]
		nextCursor = cursor.Encode(last.JoinedAt, last.ID)
		members = members[:limit]
	}
	return members, nextCursor, nil
}

func (r *OrganizationMemberRepository) CountByRole(ctx context.Context, orgID uuid.UUID) (map[organization.Role]int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT role, COUNT(*) FROM organization_members WHERE organization_id = $1 GROUP BY role`,
		orgID)
	if err != nil {
		return nil, fmt.Errorf("count members by role: %w", err)
	}
	defer rows.Close()

	result := make(map[organization.Role]int)
	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return nil, fmt.Errorf("scan role count: %w", err)
		}
		result[organization.Role(role)] = count
	}
	return result, rows.Err()
}

func (r *OrganizationMemberRepository) Update(ctx context.Context, member *organization.Member) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE organization_members
		SET role = $2, title = $3
		WHERE id = $1`,
		member.ID, string(member.Role), member.Title,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// Attempt to promote to Owner when another Owner already exists.
			return organization.ErrOwnerAlreadyExists
		}
		return fmt.Errorf("update organization member: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrMemberNotFound
	}
	return nil
}

func (r *OrganizationMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Delete the membership row and clear the denormalized
	// users.organization_id in one transaction. The WHERE guard on the
	// UPDATE prevents clobbering a user who has already been re-assigned
	// to a different org concurrently.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var (
		deletedUserID uuid.UUID
		deletedOrgID  uuid.UUID
	)
	if err := tx.QueryRowContext(ctx,
		`DELETE FROM organization_members WHERE id = $1 RETURNING user_id, organization_id`,
		id,
	).Scan(&deletedUserID, &deletedOrgID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return organization.ErrMemberNotFound
		}
		return fmt.Errorf("delete organization member: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET organization_id = NULL, updated_at = now()
		 WHERE id = $1 AND organization_id = $2`,
		deletedUserID, deletedOrgID,
	); err != nil {
		return fmt.Errorf("clear user organization_id: %w", err)
	}

	return tx.Commit()
}

// scanOne scans a single row (from QueryRowContext) into a Member.
func (r *OrganizationMemberRepository) scanOne(row *sql.Row) (*organization.Member, error) {
	var (
		m    organization.Member
		role string
	)
	err := row.Scan(
		&m.ID, &m.OrganizationID, &m.UserID, &role, &m.Title,
		&m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, organization.ErrMemberNotFound
		}
		return nil, fmt.Errorf("scan organization member: %w", err)
	}
	m.Role = organization.Role(role)
	return &m, nil
}

// scanMemberRow scans a single row from a multi-row Rows iterator.
func scanMemberRow(rows *sql.Rows) (*organization.Member, error) {
	var (
		m    organization.Member
		role string
	)
	if err := rows.Scan(
		&m.ID, &m.OrganizationID, &m.UserID, &role, &m.Title,
		&m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan organization member row: %w", err)
	}
	m.Role = organization.Role(role)
	return &m, nil
}
