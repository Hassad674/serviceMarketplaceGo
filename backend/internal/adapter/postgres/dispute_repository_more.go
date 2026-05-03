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
// AI chat messages
// ---------------------------------------------------------------------------

func (r *DisputeRepository) CreateChatMessage(ctx context.Context, msg *dispute.ChatMessage) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	_, err := r.db.ExecContext(ctx, queryInsertChatMessage,
		msg.ID, msg.DisputeID, string(msg.Role), msg.Content,
		msg.InputTokens, msg.OutputTokens, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert chat message: %w", err)
	}
	return nil
}

func (r *DisputeRepository) ListChatMessages(ctx context.Context, disputeID uuid.UUID) ([]*dispute.ChatMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListChatMessages, disputeID)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()

	var results []*dispute.ChatMessage
	for rows.Next() {
		m := &dispute.ChatMessage{}
		var role string
		if err := rows.Scan(
			&m.ID, &m.DisputeID, &role, &m.Content,
			&m.InputTokens, &m.OutputTokens, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		m.Role = dispute.ChatMessageRole(role)
		results = append(results, m)
	}
	if results == nil {
		results = []*dispute.ChatMessage{}
	}
	return results, rows.Err()
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
		&d.ID, &d.ProposalID, &d.MilestoneID, &d.ConversationID, &d.InitiatorID, &d.RespondentID,
		&d.ClientID, &d.ProviderID, &d.ClientOrganizationID, &d.ProviderOrganizationID,
		&reason, &d.Description,
		&d.RequestedAmount, &d.ProposalAmount, &status,
		&resType, &d.ResolutionAmountClient, &d.ResolutionAmountProvider,
		&d.ResolvedBy, &d.ResolutionNote, &d.AISummary,
		&d.EscalatedAt, &d.ResolvedAt, &d.CancelledAt,
		&d.LastActivityAt, &d.RespondentFirstReplyAt,
		&d.CancellationRequestedBy, &d.CancellationRequestedAt,
		&d.AISummaryInputTokens, &d.AISummaryOutputTokens,
		&d.AIChatInputTokens, &d.AIChatOutputTokens,
		&d.AIBudgetBonusTokens,
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
		&d.ID, &d.ProposalID, &d.MilestoneID, &d.ConversationID, &d.InitiatorID, &d.RespondentID,
		&d.ClientID, &d.ProviderID, &d.ClientOrganizationID, &d.ProviderOrganizationID,
		&reason, &d.Description,
		&d.RequestedAmount, &d.ProposalAmount, &status,
		&resType, &d.ResolutionAmountClient, &d.ResolutionAmountProvider,
		&d.ResolvedBy, &d.ResolutionNote, &d.AISummary,
		&d.EscalatedAt, &d.ResolvedAt, &d.CancelledAt,
		&d.LastActivityAt, &d.RespondentFirstReplyAt,
		&d.CancellationRequestedBy, &d.CancellationRequestedAt,
		&d.AISummaryInputTokens, &d.AISummaryOutputTokens,
		&d.AIChatInputTokens, &d.AIChatOutputTokens,
		&d.AIBudgetBonusTokens,
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
