package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/proposal"
	"marketplace-backend/pkg/cursor"
)

// ProposalRepository implements repository.ProposalRepository against
// Postgres.
//
// BUG-NEW-04 path 4/8: the proposals table is RLS-protected by
// migration 125 with the policy
//
//   USING (
//       client_organization_id = current_setting('app.current_org_id', true)::uuid
//       OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
//   )
//
// Both the client and the provider org are stakeholders. Under prod
// NOSUPERUSER NOBYPASSRLS the SELECT/INSERT/UPDATE all need
// app.current_org_id to be set to one of the two orgs.
//
// Strategy:
//   - Create / CreateWithDocuments[AndMilestones] derive the tenant
//     context from the proposal payload. The client_organization_id
//     and provider_organization_id columns are auto-resolved from the
//     users table at INSERT time (queryInsertProposal). We set
//     app.current_org_id to the client side's users.organization_id
//     (or fall back to the provider side) BEFORE the insert so RLS
//     accepts the new row.
//   - GetByID is read-only; without org context it returns ErrNotFound
//     under non-superuser. The new GetByIDForOrg accepts a callerOrg
//     and runs inside the tenant tx.
//   - Update does a two-step under tenant tx (read orgs first, then
//     update inside a tenant tx with the row's client org).
//   - List* methods take orgID directly and use it as the tenant ctx.
type ProposalRepository struct {
	db       *sql.DB
	txRunner *TxRunner
}

func NewProposalRepository(db *sql.DB) *ProposalRepository {
	return &ProposalRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper. Wired
// from cmd/api/main.go so RLS-protected proposal reads/writes pass
// under prod NOSUPERUSER NOBYPASSRLS. Returns the same pointer for
// fluent chaining.
func (r *ProposalRepository) WithTxRunner(runner *TxRunner) *ProposalRepository {
	r.txRunner = runner
	return r
}

// resolveProposalOrgs reads the client_organization_id and
// provider_organization_id from the users table for the given
// client/provider user ids. Returns either side as the tenant org —
// preferring the client side because that's the org that "owns" the
// proposal lifecycle. Falls back to provider when client has no org
// (solo provider client — uncommon but legal).
//
// Used by Create paths to install app.current_org_id BEFORE the
// INSERT fires (the schema's auto-resolution sub-selects from users
// only fill the columns AFTER the policy is evaluated).
func (r *ProposalRepository) resolveProposalOrgs(ctx context.Context, clientUserID, providerUserID uuid.UUID) (uuid.UUID, error) {
	var clientOrg, providerOrg uuid.NullUUID
	if err := r.db.QueryRowContext(ctx,
		`SELECT organization_id FROM users WHERE id = $1`, clientUserID,
	).Scan(&clientOrg); err != nil {
		return uuid.Nil, fmt.Errorf("resolve client org: %w", err)
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT organization_id FROM users WHERE id = $1`, providerUserID,
	).Scan(&providerOrg); err != nil {
		return uuid.Nil, fmt.Errorf("resolve provider org: %w", err)
	}
	if clientOrg.Valid {
		return clientOrg.UUID, nil
	}
	if providerOrg.Valid {
		return providerOrg.UUID, nil
	}
	return uuid.Nil, nil
}

func (r *ProposalRepository) Create(ctx context.Context, p *proposal.Proposal) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	doInsert := func(runner sqlExecutor) error {
		_, err := runner.ExecContext(ctx, queryInsertProposal,
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

	if r.txRunner != nil {
		// Resolve the proposal's stakeholder orgs from users so the
		// tenant tx has app.current_org_id set BEFORE the INSERT —
		// otherwise the policy rejects the row even though the
		// auto-resolved client_organization_id matches the org.
		orgID, err := r.resolveProposalOrgs(ctx, p.ClientID, p.ProviderID)
		if err != nil {
			return err
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doInsert(tx)
		})
	}

	return doInsert(r.db)
}

func (r *ProposalRepository) CreateWithDocuments(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument) error {
	return r.CreateWithDocumentsAndMilestones(ctx, p, docs, nil)
}

// CreateWithDocumentsAndMilestones persists the proposal, its
// documents, and its milestone batch in a single transaction so all
// three land together or not at all. Passing nil for milestones keeps
// backward compatibility with the legacy CreateWithDocuments path —
// used by phase 4 refactor paths that don't yet supply milestones
// (e.g. legacy tests) and safe because the caller will create the
// synthetic milestone in a follow-up step. Prefer the full form
// whenever possible.
func (r *ProposalRepository) CreateWithDocumentsAndMilestones(
	ctx context.Context,
	p *proposal.Proposal,
	docs []*proposal.ProposalDocument,
	milestones []*milestone.Milestone,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	insertAll := func(tx *sql.Tx) error {
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

		for _, m := range milestones {
			if _, err := tx.ExecContext(ctx, queryInsertMilestone,
				m.ID, m.ProposalID, m.Sequence, m.Title, m.Description, m.Amount, m.Deadline,
				string(m.Status), m.Version,
				m.FundedAt, m.SubmittedAt, m.ApprovedAt, m.ReleasedAt,
				m.DisputedAt, m.CancelledAt,
				m.ActiveDisputeID, m.LastDisputeID,
				m.CreatedAt, m.UpdatedAt,
			); err != nil {
				return fmt.Errorf("insert milestone: %w", err)
			}
		}
		return nil
	}

	if r.txRunner != nil {
		orgID, err := r.resolveProposalOrgs(ctx, p.ClientID, p.ProviderID)
		if err != nil {
			return err
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, insertAll)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertAll(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// GetByID returns a proposal by id WITHOUT installing tenant context.
// Under prod NOSUPERUSER NOBYPASSRLS this returns ErrProposalNotFound
// for any caller that didn't pre-set app.current_org_id (which the
// repo cannot do here without knowing the caller's org).
//
// New callers should use GetByIDForOrg(id, callerOrgID) instead. The
// legacy GetByID is preserved for system-actor scheduler paths
// (AutoApproveMilestone, AutoCloseProposal) which run with a
// privileged DB connection in production OR via a two-step approach
// (read row then re-read inside tenant tx).
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

// GetByIDForOrg returns a proposal by id under the caller's org tenant
// context. Use this entry point whenever an authenticated org member
// (client or provider side) reads a proposal — RLS will admit the row
// only if the caller's org is one of the proposal's two stakeholder
// orgs.
func (r *ProposalRepository) GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*proposal.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var p *proposal.Proposal
	doRead := func(runner sqlQuerier) error {
		got, err := scanProposal(runner.QueryRowContext(ctx, queryGetProposalByID, id))
		if errors.Is(err, sql.ErrNoRows) {
			return proposal.ErrProposalNotFound
		}
		if err != nil {
			return fmt.Errorf("get proposal by id: %w", err)
		}
		p = got
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, callerOrgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		})
		if err != nil {
			return nil, err
		}
		return p, nil
	}

	if err := doRead(r.db); err != nil {
		return nil, err
	}
	return p, nil
}

// GetByIDs batch-loads proposals for the given ids in a single query.
// Unknown ids are silently dropped — the primary caller (apporteur
// reputation) joins this against a referrer-scoped attribution list,
// so a missing proposal simply means the row was archived or deleted
// after the attribution was recorded.
func (r *ProposalRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*proposal.Proposal, error) {
	if len(ids) == 0 {
		return []*proposal.Proposal{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx, queryGetProposalsByIDs, pq.Array(idStrings))
	if err != nil {
		return nil, fmt.Errorf("get proposals by ids: %w", err)
	}
	defer rows.Close()

	return scanProposalList(rows)
}

func (r *ProposalRepository) Update(ctx context.Context, p *proposal.Proposal) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	doUpdate := func(runner sqlExecutor) error {
		result, err := runner.ExecContext(ctx, queryUpdateProposal,
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

	if r.txRunner != nil {
		// Two-step: first SELECT the row's stakeholder orgs via the
		// legacy db connection (this works because every caller of
		// Update already holds the proposal's id from a prior read), then
		// open a tenant tx with the client side org and run the UPDATE.
		var clientOrg, providerOrg uuid.NullUUID
		err := r.db.QueryRowContext(ctx,
			`SELECT client_organization_id, provider_organization_id
			 FROM proposals WHERE id = $1`, p.ID,
		).Scan(&clientOrg, &providerOrg)
		if errors.Is(err, sql.ErrNoRows) {
			return proposal.ErrProposalNotFound
		}
		if err != nil {
			return fmt.Errorf("update proposal: lookup orgs: %w", err)
		}
		orgID := uuid.Nil
		if clientOrg.Valid {
			orgID = clientOrg.UUID
		} else if providerOrg.Valid {
			orgID = providerOrg.UUID
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doUpdate(tx)
		})
	}

	return doUpdate(r.db)
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
func (r *ProposalRepository) IsOrgAuthorizedForProposal(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var exists bool
	if err := r.db.QueryRowContext(ctx, queryIsOrgAuthorizedForProposal, proposalID, orgID).Scan(&exists); err != nil {
		return false, fmt.Errorf("is org authorized for proposal: %w", err)
	}
	return exists, nil
}

// SumPaidByClientOrganization aggregates the total amount (in cents)
// the given organization has spent as the client across paid-or-later
// proposals. See querySumPaidByClientOrg for the SQL predicate.
func (r *ProposalRepository) SumPaidByClientOrganization(ctx context.Context, orgID uuid.UUID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var total int64
	if err := r.db.QueryRowContext(ctx, querySumPaidByClientOrg, orgID).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum paid by client organization: %w", err)
	}
	return total, nil
}

// ListCompletedByClientOrganization returns the org's most recent
// completed deals as the client, capped at limit rows (1..100). The
// result is ordered by completed_at DESC and by id DESC as a tie-
// breaker — stable output across identical timestamps.
func (r *ProposalRepository) ListCompletedByClientOrganization(ctx context.Context, orgID uuid.UUID, limit int) ([]*proposal.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, queryListCompletedByClientOrg, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list completed by client organization: %w", err)
	}
	defer rows.Close()

	return scanProposalList(rows)
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
