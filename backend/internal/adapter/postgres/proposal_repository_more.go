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

// Deprecated: GetLatestVersion has no app-layer callers as of the RLS
// hardening pass (2026-05-07). Under prod NOSUPERUSER NOBYPASSRLS this
// query returns ErrProposalNotFound for every caller because no tenant
// context is set. Reintroduce as GetLatestVersionForOrg(orgID) when a
// real consumer needs it.
func (r *ProposalRepository) GetLatestVersion(ctx context.Context, rootProposalID uuid.UUID) (*proposal.Proposal, error) {
	warnIfNotSystemActor(ctx, "ProposalRepository.GetLatestVersion")
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

// Deprecated: ListByConversation has no app-layer callers as of the RLS
// hardening pass (2026-05-07). Under prod NOSUPERUSER NOBYPASSRLS this
// query returns an empty slice for every caller because no tenant
// context is set. Reintroduce as ListByConversationForOrg(orgID) when
// a real consumer needs it.
func (r *ProposalRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*proposal.Proposal, error) {
	warnIfNotSystemActor(ctx, "ProposalRepository.ListByConversation")
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

	var results []*proposal.Proposal
	var nextCursor string

	doQuery := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var err error
		if cursorStr == "" {
			rows, err = runner.QueryContext(ctx, queryListActiveProjectsByOrgFirst, orgID, limit+1)
		} else {
			c, cErr := cursor.Decode(cursorStr)
			if cErr != nil {
				return fmt.Errorf("decode cursor: %w", cErr)
			}
			rows, err = runner.QueryContext(ctx, queryListActiveProjectsByOrgWithCursor,
				orgID, c.CreatedAt, c.ID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("list active projects by organization: %w", err)
		}
		defer rows.Close()

		out, nc, err := scanProposalListWithCursor(rows, limit)
		if err != nil {
			return err
		}
		results = out
		nextCursor = nc
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doQuery(tx)
		})
		if err != nil {
			return nil, "", err
		}
		return results, nextCursor, nil
	}

	if err := doQuery(r.db); err != nil {
		return nil, "", err
	}
	return results, nextCursor, nil
}

func (r *ProposalRepository) ListCompletedByOrganization(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]*proposal.Proposal, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var results []*proposal.Proposal
	var nextCursor string

	doQuery := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var err error
		if cursorStr == "" {
			rows, err = runner.QueryContext(ctx, queryListCompletedByOrgFirst, orgID, limit+1)
		} else {
			c, cErr := cursor.Decode(cursorStr)
			if cErr != nil {
				return fmt.Errorf("decode cursor: %w", cErr)
			}
			rows, err = runner.QueryContext(ctx, queryListCompletedByOrgWithCursor,
				orgID, c.CreatedAt, c.ID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("list completed by organization: %w", err)
		}
		defer rows.Close()

		out, nc, err := scanCompletedProposalListWithCursor(rows, limit)
		if err != nil {
			return err
		}
		results = out
		nextCursor = nc
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doQuery(tx)
		})
		if err != nil {
			return nil, "", err
		}
		return results, nextCursor, nil
	}

	if err := doQuery(r.db); err != nil {
		return nil, "", err
	}
	return results, nextCursor, nil
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

// IsOrgAuthorizedForProposal checks whether the given organization has
// any stake in the proposal — either as the client-side org (denormalized
// in proposals.organization_id since phase 4) or as the provider-side org
// (resolved via users.organization_id on the proposal's provider_id).
//
// CRITICAL auth check: ALWAYS runs inside RunInTxWithTenant when a
// txRunner is wired so the proposals RLS policy fires. The orgID
// argument doubles as the tenant context — if the org has no stake,
// the policy filters the row, the EXISTS sub-query returns false,
// and the caller receives "not authorized" (the safe default). A
// caller that bypasses the tenant tx and runs as
// `marketplace_scheduler` would receive `true` for any proposal
// regardless of org — but the warn guard surfaces that drift in
// the dashboard. Under `marketplace_app` the SAFE-DEFAULT holds.
func (r *ProposalRepository) IsOrgAuthorizedForProposal(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var exists bool
	doRead := func(runner sqlQuerier) error {
		if err := runner.QueryRowContext(ctx, queryIsOrgAuthorizedForProposal, proposalID, orgID).Scan(&exists); err != nil {
			return fmt.Errorf("is org authorized for proposal: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		if err := r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		}); err != nil {
			return false, err
		}
		return exists, nil
	}

	if err := doRead(r.db); err != nil {
		return false, err
	}
	return exists, nil
}

// SumPaidByClientOrganization aggregates the total amount (in cents)
// the given organization has spent as the client across paid-or-later
// proposals. See querySumPaidByClientOrg for the SQL predicate.
//
// orgID doubles as the tenant context: under prod NOSUPERUSER
// NOBYPASSRLS the proposals RLS policy admits rows where the caller's
// org is on either side, so the SUM only counts proposals the caller
// already has access to.
func (r *ProposalRepository) SumPaidByClientOrganization(ctx context.Context, orgID uuid.UUID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var total int64
	doRead := func(runner sqlQuerier) error {
		if err := runner.QueryRowContext(ctx, querySumPaidByClientOrg, orgID).Scan(&total); err != nil {
			return fmt.Errorf("sum paid by client organization: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		if err := r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		}); err != nil {
			return 0, err
		}
		return total, nil
	}

	if err := doRead(r.db); err != nil {
		return 0, err
	}
	return total, nil
}

// ListCompletedByClientOrganization returns the org's most recent
// completed deals as the client, capped at limit rows (1..100). The
// result is ordered by completed_at DESC and by id DESC as a tie-
// breaker — stable output across identical timestamps.
//
// orgID doubles as the tenant context — see SumPaidByClientOrganization
// for the rationale.
func (r *ProposalRepository) ListCompletedByClientOrganization(ctx context.Context, orgID uuid.UUID, limit int) ([]*proposal.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var results []*proposal.Proposal
	doRead := func(runner sqlQuerier) error {
		rows, err := runner.QueryContext(ctx, queryListCompletedByClientOrg, orgID, limit)
		if err != nil {
			return fmt.Errorf("list completed by client organization: %w", err)
		}
		defer rows.Close()

		out, sErr := scanProposalList(rows)
		if sErr != nil {
			return sErr
		}
		results = out
		return nil
	}

	if r.txRunner != nil {
		if err := r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		}); err != nil {
			return nil, err
		}
		return results, nil
	}

	if err := doRead(r.db); err != nil {
		return nil, err
	}
	return results, nil
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
