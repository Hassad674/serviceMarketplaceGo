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

// OrganizationInvitationRepository persists organization_invitations rows.
//
// In Phase 1 this adapter provides basic CRUD + token lookup. The full
// invitation flow (send email, resend, acceptance with user creation)
// lives in the invitation app service in Phase 2.
type OrganizationInvitationRepository struct {
	db *sql.DB
}

func NewOrganizationInvitationRepository(db *sql.DB) *OrganizationInvitationRepository {
	return &OrganizationInvitationRepository{db: db}
}

const orgInvitationCols = `id, organization_id, email, first_name, last_name, title, role, token, invited_by_user_id, status, expires_at, accepted_at, cancelled_at, created_at, updated_at`

func (r *OrganizationInvitationRepository) Create(ctx context.Context, inv *organization.Invitation) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO organization_invitations (`+orgInvitationCols+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		inv.ID, inv.OrganizationID, inv.Email, inv.FirstName, inv.LastName,
		inv.Title, string(inv.Role), inv.Token, inv.InvitedByUserID, string(inv.Status),
		inv.ExpiresAt, inv.AcceptedAt, inv.CancelledAt, inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// Either the token collided (cosmic improbability) or a pending
			// invitation already exists for this (org, email) combination.
			return organization.ErrAlreadyInvited
		}
		return fmt.Errorf("insert organization invitation: %w", err)
	}
	return nil
}

func (r *OrganizationInvitationRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.Invitation, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgInvitationCols+` FROM organization_invitations WHERE id = $1`, id)
	return r.scanOne(row)
}

func (r *OrganizationInvitationRepository) FindByToken(ctx context.Context, token string) (*organization.Invitation, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgInvitationCols+` FROM organization_invitations WHERE token = $1`, token)
	return r.scanOne(row)
}

func (r *OrganizationInvitationRepository) FindPendingByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*organization.Invitation, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgInvitationCols+`
		 FROM organization_invitations
		 WHERE organization_id = $1 AND lower(email) = lower($2) AND status = 'pending'`,
		orgID, email)
	return r.scanOne(row)
}

func (r *OrganizationInvitationRepository) List(ctx context.Context, params repository.ListInvitationsParams) ([]*organization.Invitation, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{params.OrganizationID}
	where := "WHERE organization_id = $1"
	argIdx := 2

	if params.StatusFilter != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(params.StatusFilter))
		argIdx++
	}

	if params.Cursor != "" {
		c, err := cursor.Decode(params.Cursor)
		if err == nil {
			where += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argIdx, argIdx+1)
			args = append(args, c.CreatedAt, c.ID)
			argIdx += 2
		}
	}

	query := fmt.Sprintf(
		`SELECT `+orgInvitationCols+`
		 FROM organization_invitations %s
		 ORDER BY created_at DESC, id DESC
		 LIMIT $%d`, where, argIdx)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list organization invitations: %w", err)
	}
	defer rows.Close()

	var invs []*organization.Invitation
	for rows.Next() {
		inv, err := scanInvitationRow(rows)
		if err != nil {
			return nil, "", err
		}
		invs = append(invs, inv)
	}

	var nextCursor string
	if len(invs) > limit {
		last := invs[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		invs = invs[:limit]
	}
	return invs, nextCursor, nil
}

func (r *OrganizationInvitationRepository) Update(ctx context.Context, inv *organization.Invitation) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE organization_invitations
		SET email        = $2,
		    first_name   = $3,
		    last_name    = $4,
		    title        = $5,
		    role         = $6,
		    token        = $7,
		    status       = $8,
		    expires_at   = $9,
		    accepted_at  = $10,
		    cancelled_at = $11
		WHERE id = $1`,
		inv.ID, inv.Email, inv.FirstName, inv.LastName, inv.Title,
		string(inv.Role), inv.Token, string(inv.Status),
		inv.ExpiresAt, inv.AcceptedAt, inv.CancelledAt,
	)
	if err != nil {
		return fmt.Errorf("update organization invitation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrInvitationNotFound
	}
	return nil
}

func (r *OrganizationInvitationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, "DELETE FROM organization_invitations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete organization invitation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrInvitationNotFound
	}
	return nil
}

// ExpireStale marks pending invitations with expires_at < now as expired.
// Runs as a bulk UPDATE so a background sweeper can call it periodically
// without iterating row-by-row.
func (r *OrganizationInvitationRepository) ExpireStale(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx,
		`UPDATE organization_invitations
		 SET status = 'expired'
		 WHERE status = 'pending' AND expires_at < now()`)
	if err != nil {
		return 0, fmt.Errorf("expire stale invitations: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("check rows affected: %w", err)
	}
	return int(rows), nil
}

// scanOne scans a single row (from QueryRowContext) into an Invitation.
func (r *OrganizationInvitationRepository) scanOne(row *sql.Row) (*organization.Invitation, error) {
	var (
		inv         organization.Invitation
		role        string
		status      string
	)
	err := row.Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.FirstName, &inv.LastName,
		&inv.Title, &role, &inv.Token, &inv.InvitedByUserID, &status,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CancelledAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, organization.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("scan organization invitation: %w", err)
	}
	inv.Role = organization.Role(role)
	inv.Status = organization.InvitationStatus(status)
	return &inv, nil
}

// scanInvitationRow scans a row from a multi-row Rows iterator.
func scanInvitationRow(rows *sql.Rows) (*organization.Invitation, error) {
	var (
		inv    organization.Invitation
		role   string
		status string
	)
	if err := rows.Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.FirstName, &inv.LastName,
		&inv.Title, &role, &inv.Token, &inv.InvitedByUserID, &status,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CancelledAt, &inv.CreatedAt, &inv.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan invitation row: %w", err)
	}
	inv.Role = organization.Role(role)
	inv.Status = organization.InvitationStatus(status)
	return &inv, nil
}
