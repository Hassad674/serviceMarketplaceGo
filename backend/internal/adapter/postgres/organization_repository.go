package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/organization"
)

// OrganizationRepository persists Organization entities.
//
// All queries use a 5-second context timeout (inherited from the shared
// queryTimeout constant) and go through parameterized statements. On
// unique-constraint violations (owner already has an org, duplicate id)
// the repository returns the appropriate domain sentinel.
type OrganizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create inserts a new organization row. Returns ErrOrgAlreadyExists-style
// error (ErrOwnerAlreadyExists) when the owner already has an organization,
// enforced by the UNIQUE(owner_user_id) constraint in migration 053.
func (r *OrganizationRepository) Create(ctx context.Context, org *organization.Organization) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO organizations (
			id, owner_user_id, type, name,
			pending_transfer_to_user_id, pending_transfer_initiated_at, pending_transfer_expires_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		org.ID, org.OwnerUserID, string(org.Type), org.Name,
		org.PendingTransferToUserID, org.PendingTransferInitiatedAt, org.PendingTransferExpiresAt,
		org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return organization.ErrOwnerAlreadyExists
		}
		return fmt.Errorf("insert organization: %w", err)
	}
	return nil
}

// FindByID returns the organization with the given id.
func (r *OrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, owner_user_id, type, name,
		       pending_transfer_to_user_id, pending_transfer_initiated_at, pending_transfer_expires_at,
		       created_at, updated_at
		FROM organizations WHERE id = $1`

	return r.scanOne(r.db.QueryRowContext(ctx, query, id))
}

// FindByOwnerUserID returns the organization owned by the given user, or
// ErrOrgNotFound if no such org exists. Used at JWT issuance to resolve
// a user's org context.
func (r *OrganizationRepository) FindByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*organization.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, owner_user_id, type, name,
		       pending_transfer_to_user_id, pending_transfer_initiated_at, pending_transfer_expires_at,
		       created_at, updated_at
		FROM organizations WHERE owner_user_id = $1`

	return r.scanOne(r.db.QueryRowContext(ctx, query, ownerUserID))
}

// Update persists changes to the organization. The primary use is to
// record an ownership transfer (pending_transfer_* fields) or to commit
// a transfer (OwnerUserID change + clear pending_transfer).
func (r *OrganizationRepository) Update(ctx context.Context, org *organization.Organization) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		UPDATE organizations
		SET owner_user_id                 = $2,
		    type                          = $3,
		    name                          = $4,
		    pending_transfer_to_user_id   = $5,
		    pending_transfer_initiated_at = $6,
		    pending_transfer_expires_at   = $7
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		org.ID, org.OwnerUserID, string(org.Type), org.Name,
		org.PendingTransferToUserID, org.PendingTransferInitiatedAt, org.PendingTransferExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("update organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrOrgNotFound
	}
	return nil
}

// Delete removes an organization row. CASCADE deletes wipe members and
// invitations. V1 blocks this from the UI — exposed only for admin use
// and the /remove-feature tooling.
func (r *OrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, "DELETE FROM organizations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrOrgNotFound
	}
	return nil
}

// CountAll returns the total number of organizations on the platform.
// Used by the admin dashboard to surface an "Organisations" stat tile.
// Single aggregate query — no filters.
func (r *OrganizationRepository) CountAll(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count organizations: %w", err)
	}
	return count, nil
}

// CreateWithOwnerMembership is the atomic convenience method used at
// account registration. It creates the organization row and the
// corresponding Owner membership in a single transaction so both sides
// are always consistent.
//
// This method exists on OrganizationRepository (not on the member repo)
// because the member insert is tightly coupled to the org insert — you
// never create one without the other at registration.
func (r *OrganizationRepository) CreateWithOwnerMembership(
	ctx context.Context,
	org *organization.Organization,
	member *organization.Member,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organizations (
			id, owner_user_id, type, name,
			pending_transfer_to_user_id, pending_transfer_initiated_at, pending_transfer_expires_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		org.ID, org.OwnerUserID, string(org.Type), org.Name,
		org.PendingTransferToUserID, org.PendingTransferInitiatedAt, org.PendingTransferExpiresAt,
		org.CreatedAt, org.UpdatedAt,
	); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return organization.ErrOwnerAlreadyExists
		}
		return fmt.Errorf("insert organization: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organization_members (
			id, organization_id, user_id, role, title, joined_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		member.ID, member.OrganizationID, member.UserID, string(member.Role),
		member.Title, member.JoinedAt, member.CreatedAt, member.UpdatedAt,
	); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return organization.ErrAlreadyMember
		}
		return fmt.Errorf("insert owner membership: %w", err)
	}

	// Denormalize the org onto users.organization_id so single-row lookups
	// (JWT refresh, /me, resource backfills) can read it without joining
	// organization_members. organization_members remains the source of
	// truth for membership state.
	if _, err := tx.ExecContext(ctx, `
		UPDATE users SET organization_id = $1, updated_at = now() WHERE id = $2`,
		org.ID, member.UserID,
	); err != nil {
		return fmt.Errorf("set user organization_id: %w", err)
	}

	return tx.Commit()
}

// scanOne turns a sql.Row into a domain Organization, mapping common
// errors to domain sentinels.
func (r *OrganizationRepository) scanOne(row *sql.Row) (*organization.Organization, error) {
	var (
		org     organization.Organization
		orgType string
	)
	err := row.Scan(
		&org.ID, &org.OwnerUserID, &orgType, &org.Name,
		&org.PendingTransferToUserID, &org.PendingTransferInitiatedAt, &org.PendingTransferExpiresAt,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, organization.ErrOrgNotFound
		}
		return nil, fmt.Errorf("scan organization: %w", err)
	}
	org.Type = organization.OrgType(orgType)
	return &org, nil
}
