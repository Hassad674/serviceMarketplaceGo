package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ExpertiseRepository is the PostgreSQL-backed implementation of
// repository.ExpertiseRepository. It owns exactly one table,
// organization_expertise_domains (migration 080), and performs all
// writes inside an explicit transaction so that concurrent readers
// never observe a partial Replace.
//
// The repository is stateless — the only field is the shared *sql.DB
// handle injected from cmd/api/main.go. It can be constructed once
// and shared across all handlers.
type ExpertiseRepository struct {
	db *sql.DB
}

// NewExpertiseRepository returns a repository ready to talk to the
// given *sql.DB. The caller is responsible for ensuring the DB handle
// is already tuned (SetMaxOpenConns, SetMaxIdleConns, …).
func NewExpertiseRepository(db *sql.DB) *ExpertiseRepository {
	return &ExpertiseRepository{db: db}
}

// ListByOrganization returns the organization's declared expertise
// keys in display order. Always returns a non-nil slice so callers
// can marshal the result directly to a JSON array without a nil
// check — an org with no declared expertise receives an empty []string.
func (r *ExpertiseRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT domain_key
		   FROM organization_expertise_domains
		  WHERE organization_id = $1
		  ORDER BY position ASC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list expertise by organization: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0, 8)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan expertise row: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expertise rows: %w", err)
	}
	return keys, nil
}

// ListByOrganizationIDs batch-loads expertise for multiple orgs in a
// single round trip. Returns a map keyed by organization id; orgs
// with no declared expertise are simply absent from the map.
//
// Uses pq.Array with []string rather than the typed UUID slice so
// that the statement still works against drivers that do not ship a
// uuid array encoder — the TEXT values are coerced to UUID by the
// server side thanks to the column type. This mirrors the pattern
// already used by GetPublicProfilesByOrgIDs in profile_repository.go.
func (r *ExpertiseRepository) ListByOrganizationIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	out := make(map[uuid.UUID][]string, len(orgIDs))
	if len(orgIDs) == 0 {
		return out, nil
	}

	ids := make([]string, len(orgIDs))
	for i, id := range orgIDs {
		ids[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT organization_id, domain_key
		   FROM organization_expertise_domains
		  WHERE organization_id = ANY($1)
		  ORDER BY organization_id, position ASC`,
		pq.Array(ids),
	)
	if err != nil {
		return nil, fmt.Errorf("list expertise by organization ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orgID uuid.UUID
		var key string
		if err := rows.Scan(&orgID, &key); err != nil {
			return nil, fmt.Errorf("scan expertise batch row: %w", err)
		}
		out[orgID] = append(out[orgID], key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expertise batch rows: %w", err)
	}
	return out, nil
}

// Replace atomically swaps the declared expertise for an organization.
// Implementation notes:
//
//   - DELETE + INSERT inside a single transaction. A concurrent reader
//     on READ COMMITTED (the PostgreSQL default) will either see the
//     old list entirely or the new list entirely — never a half-written
//     state.
//   - defer tx.Rollback() runs on every error path. Commit at the end
//     makes the rollback a no-op. This is the standard Go database/sql
//     transactional idiom.
//   - An empty domainKeys slice is a valid input and simply clears the
//     organization's list.
//   - The incoming order is preserved: index 0 becomes position 0,
//     index 1 becomes position 1, and so on.
func (r *ExpertiseRepository) Replace(ctx context.Context, orgID uuid.UUID, domainKeys []string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin expertise replace tx: %w", err)
	}
	// Rollback is a no-op after a successful Commit; safe to always defer.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM organization_expertise_domains WHERE organization_id = $1`,
		orgID,
	); err != nil {
		return fmt.Errorf("delete existing expertise: %w", err)
	}

	if len(domainKeys) > 0 {
		if err := insertExpertiseRows(ctx, tx, orgID, domainKeys); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit expertise replace: %w", err)
	}
	return nil
}

// insertExpertiseRows inserts the ordered slice of keys via a single
// multi-row INSERT. Extracted into a helper so Replace stays under the
// 50-line / 3-nesting limit and reads as a pipeline.
//
// Uses an $1, $2, … parameter list built programmatically — never
// string concatenation on user input. The orgID param is placed first
// and each subsequent domain key gets its own placeholder.
func insertExpertiseRows(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, domainKeys []string) error {
	// Build the VALUES clause: ($1, $2, 0), ($1, $3, 1), ($1, $4, 2), …
	// Each row shares the same orgID placeholder ($1); each key gets
	// its own placeholder ($2, $3, …); the position is a literal int
	// because positions are derived from the caller's slice index and
	// are never user input.
	args := make([]any, 0, len(domainKeys)+1)
	args = append(args, orgID)

	valueClauses := make([]string, 0, len(domainKeys))
	for i, key := range domainKeys {
		args = append(args, key)
		// $1 = orgID, key is args[i+1] → placeholder $(i+2)
		valueClauses = append(valueClauses, fmt.Sprintf("($1, $%d, %d)", i+2, i))
	}

	query := "INSERT INTO organization_expertise_domains (organization_id, domain_key, position) VALUES "
	for i, clause := range valueClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("insert expertise rows: %w", err)
	}
	return nil
}
