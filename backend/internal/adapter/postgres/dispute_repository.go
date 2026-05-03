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

// DisputeRepository is the postgres implementation of the dispute port.
//
// BUG-NEW-04 path 6/8: the disputes table is RLS-protected by migration
// 125 with the policy
//
//	USING (
//	  client_organization_id   = current_setting('app.current_org_id', true)::uuid
//	  OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
//	)
//
// Two-sided ownership — either the client or the provider org can
// read/update the dispute. Mirrors the proposals path 4/8 wrap.
//
// Sub-tables (dispute_evidence, dispute_counter_proposals,
// dispute_ai_chat_messages) are NOT directly in the migration 125
// scope, so they stay on the legacy direct-db path. The application
// authorization layer enforces access control at the parent dispute
// level.
type DisputeRepository struct {
	db       *sql.DB
	txRunner *TxRunner
}

func NewDisputeRepository(db *sql.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper.
func (r *DisputeRepository) WithTxRunner(runner *TxRunner) *DisputeRepository {
	r.txRunner = runner
	return r
}

// ---------------------------------------------------------------------------
// Core CRUD
// ---------------------------------------------------------------------------

func (r *DisputeRepository) Create(ctx context.Context, d *dispute.Dispute) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	doInsert := func(runner sqlExecutor) error {
		_, err := runner.ExecContext(ctx, queryInsertDispute,
			d.ID, d.ProposalID, d.MilestoneID, d.ConversationID, d.InitiatorID, d.RespondentID,
			d.ClientID, d.ProviderID, d.ClientOrganizationID, d.ProviderOrganizationID,
			string(d.Reason), d.Description,
			d.RequestedAmount, d.ProposalAmount, string(d.Status),
			d.ResolutionType, d.ResolutionAmountClient, d.ResolutionAmountProvider,
			d.ResolvedBy, d.ResolutionNote, d.AISummary,
			d.EscalatedAt, d.ResolvedAt, d.CancelledAt,
			d.LastActivityAt, d.RespondentFirstReplyAt,
			d.CancellationRequestedBy, d.CancellationRequestedAt,
			d.AISummaryInputTokens, d.AISummaryOutputTokens,
			d.AIChatInputTokens, d.AIChatOutputTokens,
			d.AIBudgetBonusTokens,
			d.Version, d.CreatedAt, d.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert dispute: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		// The dispute payload carries both stakeholder org ids — pick the
		// client side as the tenant context (RLS admits the row when
		// EITHER side matches, but a single setter is enough). Falls back
		// to provider when client is nil.
		orgID := d.ClientOrganizationID
		if orgID == uuid.Nil {
			orgID = d.ProviderOrganizationID
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doInsert(tx)
		})
	}

	return doInsert(r.db)
}

// GetByID returns a dispute WITHOUT installing tenant context. Under
// prod NOSUPERUSER NOBYPASSRLS this returns ErrDisputeNotFound for any
// caller that didn't pre-set app.current_org_id.
//
// New callers should use GetByIDForOrg(id, callerOrgID) instead. The
// legacy GetByID is preserved for system-actor scheduler paths
// (dispute scheduler, AI summary worker) which run with a privileged
// DB connection in production.
//
// SYSTEM-ACTOR: same contract as ProposalRepository.GetByID —
// callers MUST tag their context with system.WithSystemActor.
// The dispute scheduler (cmd/api/wire_dispute.go) and the
// loadDisputeForActor system-actor branch (app/dispute) already
// do this; an untagged caller surfaces a WARN log.
func (r *DisputeRepository) GetByID(ctx context.Context, id uuid.UUID) (*dispute.Dispute, error) {
	warnIfNotSystemActor(ctx, "DisputeRepository.GetByID")
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

// GetByIDForOrg returns a dispute by id under the caller's org tenant
// context. RLS admits the row only when callerOrgID matches one of the
// dispute's two stakeholder orgs.
func (r *DisputeRepository) GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*dispute.Dispute, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var d *dispute.Dispute
	doRead := func(runner sqlQuerier) error {
		got, err := scanDispute(runner.QueryRowContext(ctx, queryGetDisputeByID, id))
		if errors.Is(err, sql.ErrNoRows) {
			return dispute.ErrDisputeNotFound
		}
		if err != nil {
			return fmt.Errorf("get dispute by id for org: %w", err)
		}
		d = got
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, callerOrgID, uuid.Nil, func(tx *sql.Tx) error {
			return doRead(tx)
		})
		if err != nil {
			return nil, err
		}
		return d, nil
	}

	if err := doRead(r.db); err != nil {
		return nil, err
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

	doUpdate := func(runner sqlExecutor) error {
		result, err := runner.ExecContext(ctx, queryUpdateDispute,
			d.ID, string(d.Status),
			d.ResolutionType, d.ResolutionAmountClient, d.ResolutionAmountProvider,
			d.ResolvedBy, d.ResolutionNote, d.AISummary,
			d.EscalatedAt, d.ResolvedAt, d.CancelledAt,
			d.LastActivityAt, d.RespondentFirstReplyAt,
			d.CancellationRequestedBy, d.CancellationRequestedAt,
			d.AISummaryInputTokens, d.AISummaryOutputTokens,
			d.AIChatInputTokens, d.AIChatOutputTokens,
			d.AIBudgetBonusTokens,
			d.Version, // WHERE version = $21 for optimistic concurrency
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

	if r.txRunner != nil {
		// The in-memory dispute carries both stakeholder org ids; use the
		// client side (or provider when client is nil) as the tenant ctx.
		orgID := d.ClientOrganizationID
		if orgID == uuid.Nil {
			orgID = d.ProviderOrganizationID
		}
		return r.txRunner.RunInTxWithTenant(ctx, orgID, uuid.Nil, func(tx *sql.Tx) error {
			return doUpdate(tx)
		})
	}

	return doUpdate(r.db)
}

// ---------------------------------------------------------------------------
// Listings
// ---------------------------------------------------------------------------

func (r *DisputeRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]*dispute.Dispute, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var results []*dispute.Dispute
	var nextCursor string

	doQuery := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var err error
		if cursorStr == "" {
			rows, err = runner.QueryContext(ctx, queryListDisputesByOrgFirst, orgID, limit+1)
		} else {
			c, cErr := cursor.Decode(cursorStr)
			if cErr != nil {
				return fmt.Errorf("decode cursor: %w", cErr)
			}
			rows, err = runner.QueryContext(ctx, queryListDisputesByOrgWithCursor, orgID, c.CreatedAt, c.ID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("list disputes by organization: %w", err)
		}
		defer rows.Close()

		out, nc, err := scanDisputeListWithCursor(rows, limit)
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
