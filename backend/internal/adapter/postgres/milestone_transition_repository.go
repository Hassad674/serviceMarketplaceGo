package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
)

// MilestoneTransitionRepository is the postgres implementation of
// the append-only audit trail for milestone state transitions.
//
// The migration 088 grants the application DB user INSERT and
// SELECT only on this table — there is no Update or Delete method.
// Every successful milestone state change writes exactly one row
// here so the admin dashboard, dispute arbitration, and incident
// review can replay the timeline.
type MilestoneTransitionRepository struct {
	db *sql.DB
}

// NewMilestoneTransitionRepository wires the adapter against the
// shared sql.DB pool.
func NewMilestoneTransitionRepository(db *sql.DB) *MilestoneTransitionRepository {
	return &MilestoneTransitionRepository{db: db}
}

const queryInsertMilestoneTransition = `
INSERT INTO milestone_transitions (
    id, milestone_id, proposal_id,
    from_status, to_status,
    actor_id, actor_org_id,
    reason, metadata, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`

// Insert persists one transition. Caller is responsible for treating
// errors as non-fatal: the milestone state has already been committed
// by the time we get here, so a transient INSERT failure is logged
// but does not roll back business state.
func (r *MilestoneTransitionRepository) Insert(ctx context.Context, t *milestone.Transition) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertMilestoneTransition,
		t.ID, t.MilestoneID, t.ProposalID,
		string(t.FromStatus), string(t.ToStatus),
		t.ActorID, t.ActorOrgID,
		nullableString(t.Reason), t.Metadata,
		t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert milestone transition: %w", err)
	}
	return nil
}

const queryListMilestoneTransitionsByMilestone = `
SELECT id, milestone_id, proposal_id,
       from_status, to_status,
       actor_id, actor_org_id,
       reason, metadata, created_at
FROM milestone_transitions
WHERE milestone_id = $1
ORDER BY created_at ASC
`

// ListByMilestone returns the full chronological history of
// transitions on a single milestone. Used by the admin timeline
// view and the dispute arbitration evidence pack.
func (r *MilestoneTransitionRepository) ListByMilestone(ctx context.Context, milestoneID uuid.UUID) ([]*milestone.Transition, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListMilestoneTransitionsByMilestone, milestoneID)
	if err != nil {
		return nil, fmt.Errorf("list milestone transitions: %w", err)
	}
	defer rows.Close()

	var out []*milestone.Transition
	for rows.Next() {
		t, err := scanMilestoneTransition(rows)
		if err != nil {
			return nil, fmt.Errorf("scan milestone transition: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return out, nil
}

// scanMilestoneTransition materialises a row into a domain entity.
// Status enums are converted from TEXT, and the optional reason
// column is read via sql.NullString.
func scanMilestoneTransition(s scanner) (*milestone.Transition, error) {
	var (
		t          milestone.Transition
		fromStatus string
		toStatus   string
		reason     sql.NullString
	)
	if err := s.Scan(
		&t.ID, &t.MilestoneID, &t.ProposalID,
		&fromStatus, &toStatus,
		&t.ActorID, &t.ActorOrgID,
		&reason, &t.Metadata,
		&t.CreatedAt,
	); err != nil {
		return nil, err
	}
	t.FromStatus = milestone.MilestoneStatus(fromStatus)
	t.ToStatus = milestone.MilestoneStatus(toStatus)
	if reason.Valid {
		t.Reason = reason.String
	}
	return &t, nil
}

// nullableString returns sql.NullString from a Go string so empty
// strings become NULL in the column instead of empty strings.
func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
