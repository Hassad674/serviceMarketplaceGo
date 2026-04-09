package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/dispute"
	"marketplace-backend/pkg/cursor"
)

type DisputeRepository struct {
	db *sql.DB
}

func NewDisputeRepository(db *sql.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

// ---------------------------------------------------------------------------
// Core CRUD
// ---------------------------------------------------------------------------

func (r *DisputeRepository) Create(ctx context.Context, d *dispute.Dispute) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertDispute,
		d.ID, d.ProposalID, d.ConversationID, d.InitiatorID, d.RespondentID,
		d.ClientID, d.ProviderID, string(d.Reason), d.Description,
		d.RequestedAmount, d.ProposalAmount, string(d.Status),
		d.ResolutionType, d.ResolutionAmountClient, d.ResolutionAmountProvider,
		d.ResolvedBy, d.ResolutionNote, d.AISummary,
		d.EscalatedAt, d.ResolvedAt, d.CancelledAt,
		d.LastActivityAt, d.RespondentFirstReplyAt,
		d.CancellationRequestedBy, d.CancellationRequestedAt,
		d.Version, d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert dispute: %w", err)
	}
	return nil
}

func (r *DisputeRepository) GetByID(ctx context.Context, id uuid.UUID) (*dispute.Dispute, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	d, err := scanDispute(r.db.QueryRowContext(ctx, queryGetDisputeByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, dispute.ErrDisputeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dispute by id: %w", err)
	}
	return d, nil
}

func (r *DisputeRepository) GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*dispute.Dispute, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	d, err := scanDispute(r.db.QueryRowContext(ctx, queryGetDisputeByProposalID, proposalID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // no active dispute — not an error
	}
	if err != nil {
		return nil, fmt.Errorf("get dispute by proposal: %w", err)
	}
	return d, nil
}

func (r *DisputeRepository) Update(ctx context.Context, d *dispute.Dispute) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryUpdateDispute,
		d.ID, string(d.Status),
		d.ResolutionType, d.ResolutionAmountClient, d.ResolutionAmountProvider,
		d.ResolvedBy, d.ResolutionNote, d.AISummary,
		d.EscalatedAt, d.ResolvedAt, d.CancelledAt,
		d.LastActivityAt, d.RespondentFirstReplyAt,
		d.CancellationRequestedBy, d.CancellationRequestedAt,
		d.Version, // WHERE version = $16 for optimistic concurrency
	)
	if err != nil {
		return fmt.Errorf("update dispute: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("dispute not found or version conflict")
	}
	d.Version++ // reflect the DB increment
	return nil
}

// ---------------------------------------------------------------------------
// Listings
// ---------------------------------------------------------------------------

func (r *DisputeRepository) ListByUserID(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*dispute.Dispute, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error
	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListDisputesByUserFirst, userID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListDisputesByUserWithCursor, userID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list disputes by user: %w", err)
	}
	defer rows.Close()

	return scanDisputeListWithCursor(rows, limit)
}

func (r *DisputeRepository) ListPendingForScheduler(ctx context.Context) ([]*dispute.Dispute, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListDisputesPendingScheduler)
	if err != nil {
		return nil, fmt.Errorf("list pending disputes: %w", err)
	}
	defer rows.Close()

	return scanDisputeList(rows)
}

func (r *DisputeRepository) ListAll(ctx context.Context, cursorStr string, limit int, statusFilter string) ([]*dispute.Dispute, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error
	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListAllDisputesFirst, statusFilter, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListAllDisputesWithCursor, statusFilter, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list all disputes: %w", err)
	}
	defer rows.Close()

	return scanDisputeListWithCursor(rows, limit)
}

// ---------------------------------------------------------------------------
// Evidence
// ---------------------------------------------------------------------------

func (r *DisputeRepository) CreateEvidence(ctx context.Context, e *dispute.Evidence) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	_, err := r.db.ExecContext(ctx, queryInsertEvidence,
		e.ID, e.DisputeID, e.CounterProposalID, e.UploaderID,
		e.Filename, e.URL, e.Size, e.MimeType, e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}
	return nil
}

func (r *DisputeRepository) ListEvidence(ctx context.Context, disputeID uuid.UUID) ([]*dispute.Evidence, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListEvidence, disputeID)
	if err != nil {
		return nil, fmt.Errorf("list evidence: %w", err)
	}
	defer rows.Close()

	var results []*dispute.Evidence
	for rows.Next() {
		e := &dispute.Evidence{}
		if err := rows.Scan(&e.ID, &e.DisputeID, &e.CounterProposalID, &e.UploaderID, &e.Filename, &e.URL, &e.Size, &e.MimeType, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan evidence: %w", err)
		}
		results = append(results, e)
	}
	if results == nil {
		results = []*dispute.Evidence{}
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Counter-proposals
// ---------------------------------------------------------------------------

func (r *DisputeRepository) CreateCounterProposal(ctx context.Context, cp *dispute.CounterProposal) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertCounterProposal,
		cp.ID, cp.DisputeID, cp.ProposerID,
		cp.AmountClient, cp.AmountProvider, cp.Message,
		string(cp.Status), cp.RespondedAt, cp.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert counter-proposal: %w", err)
	}
	return nil
}

func (r *DisputeRepository) GetCounterProposalByID(ctx context.Context, id uuid.UUID) (*dispute.CounterProposal, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cp := &dispute.CounterProposal{}
	var status string
	err := r.db.QueryRowContext(ctx, queryGetCounterProposalByID, id).Scan(
		&cp.ID, &cp.DisputeID, &cp.ProposerID,
		&cp.AmountClient, &cp.AmountProvider, &cp.Message,
		&status, &cp.RespondedAt, &cp.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, dispute.ErrCounterProposalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get counter-proposal: %w", err)
	}
	cp.Status = dispute.CounterProposalStatus(status)
	return cp, nil
}

func (r *DisputeRepository) UpdateCounterProposal(ctx context.Context, cp *dispute.CounterProposal) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpdateCounterProposal,
		cp.ID, string(cp.Status), cp.RespondedAt,
	)
	if err != nil {
		return fmt.Errorf("update counter-proposal: %w", err)
	}
	return nil
}

func (r *DisputeRepository) ListCounterProposals(ctx context.Context, disputeID uuid.UUID) ([]*dispute.CounterProposal, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListCounterProposals, disputeID)
	if err != nil {
		return nil, fmt.Errorf("list counter-proposals: %w", err)
	}
	defer rows.Close()

	var results []*dispute.CounterProposal
	for rows.Next() {
		cp := &dispute.CounterProposal{}
		var status string
		if err := rows.Scan(
			&cp.ID, &cp.DisputeID, &cp.ProposerID,
			&cp.AmountClient, &cp.AmountProvider, &cp.Message,
			&status, &cp.RespondedAt, &cp.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan counter-proposal: %w", err)
		}
		cp.Status = dispute.CounterProposalStatus(status)
		results = append(results, cp)
	}
	if results == nil {
		results = []*dispute.CounterProposal{}
	}
	return results, rows.Err()
}

func (r *DisputeRepository) SupersedeAllPending(ctx context.Context, disputeID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, querySupersedeAllPending, disputeID)
	if err != nil {
		return fmt.Errorf("supersede pending counter-proposals: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func (r *DisputeRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx, queryCountDisputesByUser, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count disputes by user: %w", err)
	}
	return count, nil
}

func (r *DisputeRepository) CountAll(ctx context.Context) (total int, open int, escalated int, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = r.db.QueryRowContext(ctx, queryCountAllDisputes).Scan(&total, &open, &escalated)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("count all disputes: %w", err)
	}
	return total, open, escalated, nil
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

func scanDispute(row *sql.Row) (*dispute.Dispute, error) {
	d := &dispute.Dispute{}
	var reason, status string
	var resType sql.NullString

	err := row.Scan(
		&d.ID, &d.ProposalID, &d.ConversationID, &d.InitiatorID, &d.RespondentID,
		&d.ClientID, &d.ProviderID, &reason, &d.Description,
		&d.RequestedAmount, &d.ProposalAmount, &status,
		&resType, &d.ResolutionAmountClient, &d.ResolutionAmountProvider,
		&d.ResolvedBy, &d.ResolutionNote, &d.AISummary,
		&d.EscalatedAt, &d.ResolvedAt, &d.CancelledAt,
		&d.LastActivityAt, &d.RespondentFirstReplyAt,
		&d.CancellationRequestedBy, &d.CancellationRequestedAt,
		&d.Version, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.Reason = dispute.Reason(reason)
	d.Status = dispute.Status(status)
	if resType.Valid {
		rt := dispute.ResolutionType(resType.String)
		d.ResolutionType = &rt
	}
	return d, nil
}

func scanDisputeFromRows(rows *sql.Rows) (*dispute.Dispute, error) {
	d := &dispute.Dispute{}
	var reason, status string
	var resType sql.NullString

	err := rows.Scan(
		&d.ID, &d.ProposalID, &d.ConversationID, &d.InitiatorID, &d.RespondentID,
		&d.ClientID, &d.ProviderID, &reason, &d.Description,
		&d.RequestedAmount, &d.ProposalAmount, &status,
		&resType, &d.ResolutionAmountClient, &d.ResolutionAmountProvider,
		&d.ResolvedBy, &d.ResolutionNote, &d.AISummary,
		&d.EscalatedAt, &d.ResolvedAt, &d.CancelledAt,
		&d.LastActivityAt, &d.RespondentFirstReplyAt,
		&d.CancellationRequestedBy, &d.CancellationRequestedAt,
		&d.Version, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.Reason = dispute.Reason(reason)
	d.Status = dispute.Status(status)
	if resType.Valid {
		rt := dispute.ResolutionType(resType.String)
		d.ResolutionType = &rt
	}
	return d, nil
}

func scanDisputeList(rows *sql.Rows) ([]*dispute.Dispute, error) {
	var results []*dispute.Dispute
	for rows.Next() {
		d, err := scanDisputeFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dispute: %w", err)
		}
		results = append(results, d)
	}
	if results == nil {
		results = []*dispute.Dispute{}
	}
	return results, rows.Err()
}

func scanDisputeListWithCursor(rows *sql.Rows, limit int) ([]*dispute.Dispute, string, error) {
	results, err := scanDisputeList(rows)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}
	return results, nextCursor, nil
}
