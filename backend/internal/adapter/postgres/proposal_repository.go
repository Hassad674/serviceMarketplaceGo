package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/proposal"
	"marketplace-backend/pkg/cursor"
)

type ProposalRepository struct {
	db *sql.DB
}

func NewProposalRepository(db *sql.DB) *ProposalRepository {
	return &ProposalRepository{db: db}
}

func (r *ProposalRepository) Create(ctx context.Context, p *proposal.Proposal) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertProposal,
		p.ID, p.ConversationID, p.SenderID, p.RecipientID,
		p.Title, p.Description, p.Amount, p.Deadline,
		string(p.Status), p.ParentID, p.Version,
		p.ClientID, p.ProviderID, p.Metadata,
		p.ActiveDisputeID, p.LastDisputeID,
		p.AcceptedAt, p.DeclinedAt, p.PaidAt, p.CompletedAt,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert proposal: %w", err)
	}

	return nil
}

func (r *ProposalRepository) CreateWithDocuments(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, queryInsertProposal,
		p.ID, p.ConversationID, p.SenderID, p.RecipientID,
		p.Title, p.Description, p.Amount, p.Deadline,
		string(p.Status), p.ParentID, p.Version,
		p.ClientID, p.ProviderID, p.Metadata,
		p.ActiveDisputeID, p.LastDisputeID,
		p.AcceptedAt, p.DeclinedAt, p.PaidAt, p.CompletedAt,
		p.CreatedAt, p.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert proposal: %w", err)
	}

	for _, doc := range docs {
		if _, err := tx.ExecContext(ctx, queryInsertProposalDocument,
			doc.ID, doc.ProposalID, doc.Filename, doc.URL, doc.Size, doc.MimeType, doc.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert document: %w", err)
		}
	}

	return tx.Commit()
}

func (r *ProposalRepository) GetByID(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p, err := scanProposal(r.db.QueryRowContext(ctx, queryGetProposalByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, proposal.ErrProposalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get proposal by id: %w", err)
	}

	return p, nil
}

func (r *ProposalRepository) Update(ctx context.Context, p *proposal.Proposal) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryUpdateProposal,
		p.ID, string(p.Status),
		p.AcceptedAt, p.DeclinedAt, p.PaidAt, p.CompletedAt,
		p.Metadata, p.ActiveDisputeID, p.LastDisputeID, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return proposal.ErrProposalNotFound
	}

	return nil
}

func (r *ProposalRepository) GetLatestVersion(ctx context.Context, rootProposalID uuid.UUID) (*proposal.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p, err := scanProposal(r.db.QueryRowContext(ctx, queryGetLatestVersion, rootProposalID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, proposal.ErrProposalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get latest version: %w", err)
	}

	return p, nil
}

func (r *ProposalRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*proposal.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListByConversation, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list by conversation: %w", err)
	}
	defer rows.Close()

	return scanProposalList(rows)
}

func (r *ProposalRepository) ListActiveProjectsByOrganization(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]*proposal.Proposal, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListActiveProjectsByOrgFirst, orgID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListActiveProjectsByOrgWithCursor,
			orgID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list active projects by organization: %w", err)
	}
	defer rows.Close()

	return scanProposalListWithCursor(rows, limit)
}

func (r *ProposalRepository) ListCompletedByOrganization(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]*proposal.Proposal, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListCompletedByOrgFirst, orgID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListCompletedByOrgWithCursor,
			orgID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list completed by organization: %w", err)
	}
	defer rows.Close()

	return scanCompletedProposalListWithCursor(rows, limit)
}

func (r *ProposalRepository) GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*proposal.ProposalDocument, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryGetProposalDocuments, proposalID)
	if err != nil {
		return nil, fmt.Errorf("get documents: %w", err)
	}
	defer rows.Close()

	var docs []*proposal.ProposalDocument
	for rows.Next() {
		doc := &proposal.ProposalDocument{}
		if err := rows.Scan(
			&doc.ID, &doc.ProposalID, &doc.Filename, &doc.URL,
			&doc.Size, &doc.MimeType, &doc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if docs == nil {
		docs = []*proposal.ProposalDocument{}
	}

	return docs, nil
}

func (r *ProposalRepository) CreateDocument(ctx context.Context, doc *proposal.ProposalDocument) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertProposalDocument,
		doc.ID, doc.ProposalID, doc.Filename, doc.URL,
		doc.Size, doc.MimeType, doc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	return nil
}

func (r *ProposalRepository) CountAll(ctx context.Context) (total int, active int, err error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM proposals").Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("count total proposals: %w", err)
	}

	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM proposals WHERE status IN ('paid', 'active', 'completion_requested')",
	).Scan(&active)
	if err != nil {
		return 0, 0, fmt.Errorf("count active proposals: %w", err)
	}
	return total, active, nil
}

// scanProposal scans a single proposal from a QueryRow result.
func scanProposal(row *sql.Row) (*proposal.Proposal, error) {
	p := &proposal.Proposal{}
	var status string
	var metadata []byte

	err := row.Scan(
		&p.ID, &p.ConversationID, &p.SenderID, &p.RecipientID,
		&p.Title, &p.Description, &p.Amount, &p.Deadline,
		&status, &p.ParentID, &p.Version,
		&p.ClientID, &p.ProviderID, &metadata,
		&p.ActiveDisputeID, &p.LastDisputeID,
		&p.AcceptedAt, &p.DeclinedAt, &p.PaidAt, &p.CompletedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.Status = proposal.ProposalStatus(status)
	if len(metadata) > 0 {
		p.Metadata = json.RawMessage(metadata)
	}

	return p, nil
}

// scanProposalFromRows scans a single proposal from a Rows iterator.
func scanProposalFromRows(rows *sql.Rows) (*proposal.Proposal, error) {
	p := &proposal.Proposal{}
	var status string
	var metadata []byte

	err := rows.Scan(
		&p.ID, &p.ConversationID, &p.SenderID, &p.RecipientID,
		&p.Title, &p.Description, &p.Amount, &p.Deadline,
		&status, &p.ParentID, &p.Version,
		&p.ClientID, &p.ProviderID, &metadata,
		&p.ActiveDisputeID, &p.LastDisputeID,
		&p.AcceptedAt, &p.DeclinedAt, &p.PaidAt, &p.CompletedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.Status = proposal.ProposalStatus(status)
	if len(metadata) > 0 {
		p.Metadata = json.RawMessage(metadata)
	}

	return p, nil
}

func scanProposalList(rows *sql.Rows) ([]*proposal.Proposal, error) {
	var results []*proposal.Proposal
	for rows.Next() {
		p, err := scanProposalFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		results = append(results, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if results == nil {
		results = []*proposal.Proposal{}
	}
	return results, nil
}

func scanProposalListWithCursor(rows *sql.Rows, limit int) ([]*proposal.Proposal, string, error) {
	results, err := scanProposalList(rows)
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

// scanCompletedProposalListWithCursor is like scanProposalListWithCursor but
// uses CompletedAt as the cursor timestamp (for ListCompletedByProvider).
func scanCompletedProposalListWithCursor(rows *sql.Rows, limit int) ([]*proposal.Proposal, string, error) {
	results, err := scanProposalList(rows)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		if last.CompletedAt != nil {
			nextCursor = cursor.Encode(*last.CompletedAt, last.ID)
		}
		results = results[:limit]
	}

	return results, nextCursor, nil
}
