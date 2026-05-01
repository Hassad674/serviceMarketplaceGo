package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/milestone"
)

// MilestoneRepository is the postgres implementation of the milestone port.
//
// All mutations go through an optimistic-locked UPDATE (matching id AND
// version) and return milestone.ErrConcurrentUpdate on a zero-row result.
// This prevents two concurrent transitions from silently clobbering each
// other — essential for the "client approves" vs "client opens dispute"
// race that can happen on a submitted milestone.
//
// BUG-NEW-04 path 5/8: proposal_milestones is RLS-protected by migration
// 125 with the policy
//
//   USING (EXISTS (
//     SELECT 1 FROM proposals p
//     WHERE p.id = proposal_milestones.proposal_id
//       AND (p.client_organization_id   = current_setting('app.current_org_id', true)::uuid
//         OR p.provider_organization_id = current_setting('app.current_org_id', true)::uuid)
//   ))
//
// Milestones inherit security from the parent proposal. Strategy:
//   - Mutations look up the parent proposal's stakeholder orgs via the
//     legacy db connection (single SELECT — works because callers always
//     have the proposal id), then open a tenant tx with the client side
//     org and run the SQL.
//   - Reads (GetByID / ListByProposal) similarly resolve the parent's
//     org first. New ForOrg variants take the caller's org explicitly
//     for paths where the caller is an authenticated org member.
type MilestoneRepository struct {
	db       *sql.DB
	txRunner *TxRunner
}

// NewMilestoneRepository wires a milestone repository against the given
// database handle. The handle is expected to be a pool (sql.DB), not a
// single connection, so the adapter can serve concurrent callers.
func NewMilestoneRepository(db *sql.DB) *MilestoneRepository {
	return &MilestoneRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper. Wired
// from cmd/api/main.go. Returns the same pointer for fluent chaining.
func (r *MilestoneRepository) WithTxRunner(runner *TxRunner) *MilestoneRepository {
	r.txRunner = runner
	return r
}

// resolveParentProposalOrg returns one of the parent proposal's
// stakeholder orgs (preferring client side) so we can install
// app.current_org_id on the tenant tx. Single SELECT — no policy check
// at this layer because we read on the legacy db connection.
func (r *MilestoneRepository) resolveParentProposalOrg(ctx context.Context, proposalID uuid.UUID) (uuid.UUID, error) {
	var clientOrg, providerOrg uuid.NullUUID
	err := r.db.QueryRowContext(ctx,
		`SELECT client_organization_id, provider_organization_id
		 FROM proposals WHERE id = $1`, proposalID,
	).Scan(&clientOrg, &providerOrg)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, milestone.ErrMilestoneNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve parent proposal org: %w", err)
	}
	if clientOrg.Valid {
		return clientOrg.UUID, nil
	}
	if providerOrg.Valid {
		return providerOrg.UUID, nil
	}
	return uuid.Nil, nil
}

// CreateBatch inserts every milestone of a proposal in a SINGLE round
// trip — the function name is now accurate. Pre PERF-B-04 the
// implementation looped N sequential INSERTs inside a transaction
// (10–40 ms wasted across an AZ for a 5-jalon proposal). The new
// path builds a multi-row VALUES tuple ($1..$19), ($20..$38), … so
// Postgres takes one parse + one network round trip.
//
// The slice must come from milestone.NewMilestoneBatch so sequences are
// consecutive and the 20-milestone cap is enforced (the cap also
// keeps the parameter count well under Postgres's 65535-arg limit:
// 20*19 = 380).
func (r *MilestoneRepository) CreateBatch(ctx context.Context, milestones []*milestone.Milestone) error {
	if len(milestones) == 0 {
		return milestone.ErrEmptyBatch
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const colsPerRow = 19
	args := make([]any, 0, len(milestones)*colsPerRow)
	placeholders := make([]string, 0, len(milestones))

	for i, m := range milestones {
		// Build the placeholder tuple ($N..$N+18) for this row.
		base := i*colsPerRow + 1
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base, base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8,
			base+9, base+10, base+11, base+12, base+13, base+14, base+15, base+16, base+17, base+18,
		))
		args = append(args,
			m.ID, m.ProposalID, m.Sequence, m.Title, m.Description, m.Amount, m.Deadline,
			string(m.Status), m.Version,
			m.FundedAt, m.SubmittedAt, m.ApprovedAt, m.ReleasedAt,
			m.DisputedAt, m.CancelledAt,
			m.ActiveDisputeID, m.LastDisputeID,
			m.CreatedAt, m.UpdatedAt,
		)
	}

	// gosec G201: the variable parts of the formatted SQL are static
	// — `placeholders` is a slice of generated `($N, ..., $N+18)`
	// tuples whose only inputs are loop counters. Every value reaches
	// Postgres via the `args` slice and parameterised $N placeholders.
	query := "INSERT INTO proposal_milestones " + queryInsertMilestoneColumns +
		" VALUES " + strings.Join(placeholders, ", ") // #nosec G201

	doInsert := func(runner sqlExecutor) error {
		if _, err := runner.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("insert milestones batch: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		// All milestones in the batch share a single proposal_id (the
		// caller's domain factory enforces this — milestone.NewMilestoneBatch).
		// Resolve the parent's stakeholder org for the tenant context.
		orgID, err := r.resolveParentProposalOrg(ctx, milestones[0].ProposalID)
		if err != nil {
			return err
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doInsert(tx)
		})
	}

	return doInsert(r.db)
}

// GetByID fetches a milestone without taking a lock. Suitable for read-only
// queries (listings, projections, UI detail views).
func (r *MilestoneRepository) GetByID(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, queryGetMilestoneByID, id)
	m, err := scanMilestone(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, milestone.ErrMilestoneNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get milestone by id: %w", err)
	}
	return m, nil
}

// GetByIDWithVersion fetches a milestone and returns its current Version
// for optimistic-concurrency control by the caller.
//
// BUG-11 (renamed from GetByIDForUpdate): the previous name suggested
// a pessimistic SELECT FOR UPDATE lock was taken, but the
// implementation opened a transaction, ran SELECT FOR UPDATE, and
// committed immediately — which RELEASES the lock. The race protection
// always came from the WHERE id = $1 AND version = $2 clause in
// Update, which returns milestone.ErrConcurrentUpdate when zero rows
// match (the version bumped between fetch and write).
//
// The new implementation drops the no-op transaction + FOR UPDATE
// dance and uses a plain QueryRowContext on the same SELECT statement
// the read path uses — clearer semantics, identical concurrency
// behaviour. Two concurrent callers that both fetch version V will
// both reach Update; the one that lands first bumps the version, the
// loser gets ErrConcurrentUpdate.
func (r *MilestoneRepository) GetByIDWithVersion(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, queryGetMilestoneByID, id)
	m, err := scanMilestone(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, milestone.ErrMilestoneNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get milestone with version: %w", err)
	}
	return m, nil
}

// ListByProposal returns every milestone of a proposal, ordered by
// ascending sequence. Used to render the milestone tracker, compute the
// macro status, and recompute the proposal.amount cache.
func (r *MilestoneRepository) ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*milestone.Milestone, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListMilestonesByProposal, proposalID)
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}
	defer rows.Close()

	var milestones []*milestone.Milestone
	for rows.Next() {
		m, err := scanMilestone(rows)
		if err != nil {
			return nil, fmt.Errorf("scan milestone: %w", err)
		}
		milestones = append(milestones, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return milestones, nil
}

// GetCurrentActive returns the first non-terminal milestone of the proposal
// by ascending sequence. Returns milestone.ErrMilestoneNotFound if every
// milestone is terminal — the caller uses that as a signal that the
// proposal has no work left to do.
func (r *MilestoneRepository) GetCurrentActive(ctx context.Context, proposalID uuid.UUID) (*milestone.Milestone, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, queryGetCurrentActiveMilestone, proposalID)
	m, err := scanMilestone(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, milestone.ErrMilestoneNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get current active milestone: %w", err)
	}
	return m, nil
}

// Update persists a domain transition with optimistic concurrency.
//
// The WHERE clause matches id AND the pre-transition version; on success
// the version column is bumped by the SQL. If zero rows are affected, we
// return milestone.ErrConcurrentUpdate. The caller's in-memory copy is
// then stale and must be refetched before retrying.
//
// Note: the Go struct's Version field is incremented to match the DB on
// successful return, so subsequent updates within the same call chain
// see the new value.
func (r *MilestoneRepository) Update(ctx context.Context, m *milestone.Milestone) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	doUpdate := func(runner sqlExecutor) error {
		result, err := runner.ExecContext(ctx, queryUpdateMilestone,
			m.ID, m.Version, string(m.Status),
			m.FundedAt, m.SubmittedAt, m.ApprovedAt, m.ReleasedAt,
			m.DisputedAt, m.CancelledAt,
			m.ActiveDisputeID, m.LastDisputeID,
			m.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("update milestone: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected: %w", err)
		}
		if affected == 0 {
			return milestone.ErrConcurrentUpdate
		}
		m.Version++
		return nil
	}

	if r.txRunner != nil {
		orgID, err := r.resolveParentProposalOrg(ctx, m.ProposalID)
		if err != nil {
			return err
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doUpdate(tx)
		})
	}

	return doUpdate(r.db)
}

// GetByIDForOrg returns a milestone by id under the caller's org tenant
// context. RLS admits the row only when the caller's org matches one
// of the parent proposal's stakeholder orgs.
func (r *MilestoneRepository) GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*milestone.Milestone, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var m *milestone.Milestone
	doRead := func(runner sqlQuerier) error {
		row := runner.QueryRowContext(ctx, queryGetMilestoneByID, id)
		got, err := scanMilestone(row)
		if errors.Is(err, sql.ErrNoRows) {
			return milestone.ErrMilestoneNotFound
		}
		if err != nil {
			return fmt.Errorf("get milestone by id for org: %w", err)
		}
		m = got
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, callerOrgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		})
		if err != nil {
			return nil, err
		}
		return m, nil
	}

	if err := doRead(r.db); err != nil {
		return nil, err
	}
	return m, nil
}

// ListByProposalForOrg returns every milestone of a proposal under the
// caller's org tenant context.
func (r *MilestoneRepository) ListByProposalForOrg(ctx context.Context, proposalID, callerOrgID uuid.UUID) ([]*milestone.Milestone, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var milestones []*milestone.Milestone
	doRead := func(runner sqlQuerier) error {
		rows, err := runner.QueryContext(ctx, queryListMilestonesByProposal, proposalID)
		if err != nil {
			return fmt.Errorf("list milestones for org: %w", err)
		}
		defer rows.Close()

		milestones = nil
		for rows.Next() {
			m, err := scanMilestone(rows)
			if err != nil {
				return fmt.Errorf("scan milestone: %w", err)
			}
			milestones = append(milestones, m)
		}
		return rows.Err()
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, callerOrgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		})
		if err != nil {
			return nil, err
		}
		return milestones, nil
	}

	if err := doRead(r.db); err != nil {
		return nil, err
	}
	return milestones, nil
}

// CreateDeliverable stores a file attached to a milestone.
func (r *MilestoneRepository) CreateDeliverable(ctx context.Context, d *milestone.Deliverable) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertMilestoneDeliverable,
		d.ID, d.MilestoneID, d.Filename, d.URL, d.Size, d.MimeType, d.UploadedBy, d.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert deliverable: %w", err)
	}
	return nil
}

// ListDeliverables returns every deliverable for a milestone ordered by created_at ASC.
func (r *MilestoneRepository) ListDeliverables(ctx context.Context, milestoneID uuid.UUID) ([]*milestone.Deliverable, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListMilestoneDeliverables, milestoneID)
	if err != nil {
		return nil, fmt.Errorf("list deliverables: %w", err)
	}
	defer rows.Close()

	var out []*milestone.Deliverable
	for rows.Next() {
		var d milestone.Deliverable
		if err := rows.Scan(
			&d.ID, &d.MilestoneID, &d.Filename, &d.URL, &d.Size, &d.MimeType, &d.UploadedBy, &d.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan deliverable: %w", err)
		}
		out = append(out, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return out, nil
}

// DeleteDeliverable removes a deliverable by ID. Mutability enforcement
// (status must be pending_funding or funded) is the caller's job.
func (r *MilestoneRepository) DeleteDeliverable(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryDeleteMilestoneDeliverable, id)
	if err != nil {
		return fmt.Errorf("delete deliverable: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return milestone.ErrDeliverableNotFound
	}
	return nil
}

// ListByProposals resolves a batch of proposals in one round trip and
// groups the results by proposal_id. Used by list endpoints to fan out
// milestone summaries without generating N+1 queries.
func (r *MilestoneRepository) ListByProposals(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error) {
	if len(proposalIDs) == 0 {
		return map[uuid.UUID][]*milestone.Milestone{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// pq.Array marshals the Go slice into a postgres uuid[] that
	// matches the WHERE proposal_id = ANY($1::uuid[]) clause.
	rows, err := r.db.QueryContext(ctx, queryListMilestonesByProposals, pq.Array(proposalIDs))
	if err != nil {
		return nil, fmt.Errorf("list milestones by proposals: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]*milestone.Milestone, len(proposalIDs))
	for rows.Next() {
		m, err := scanMilestone(rows)
		if err != nil {
			return nil, fmt.Errorf("scan milestone: %w", err)
		}
		result[m.ProposalID] = append(result[m.ProposalID], m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return result, nil
}

// scanner abstracts *sql.Row and *sql.Rows so scanMilestone can serve both.
type scanner interface {
	Scan(dest ...any) error
}

// scanMilestone materialises a milestone row into a domain struct. Status
// is converted from TEXT to the typed enum so callers never see a raw string.
func scanMilestone(s scanner) (*milestone.Milestone, error) {
	var m milestone.Milestone
	var status string
	if err := s.Scan(
		&m.ID, &m.ProposalID, &m.Sequence, &m.Title, &m.Description, &m.Amount, &m.Deadline,
		&status, &m.Version,
		&m.FundedAt, &m.SubmittedAt, &m.ApprovedAt, &m.ReleasedAt,
		&m.DisputedAt, &m.CancelledAt,
		&m.ActiveDisputeID, &m.LastDisputeID,
		&m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return nil, err
	}
	m.Status = milestone.MilestoneStatus(status)
	return &m, nil
}
