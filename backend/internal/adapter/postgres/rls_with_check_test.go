package postgres_test

// F.5 S1 — explicit WITH CHECK on the 8 tenant-scoped tables that had
// USING-only policies. Migration 135 adds symmetric WITH CHECK clauses
// so a future rotation to a NOBYPASSRLS application role does not
// silently break system writes (or, conversely, let a wrong-tenant
// INSERT through).
//
// Two assertions per table:
//   1. INSERT with the matching tenant context succeeds — proves the
//      WITH CHECK predicate is correctly written (mirrors USING).
//   2. INSERT with a foreign tenant context FAILS with the canonical
//      RLS error — proves the WITH CHECK is actually enforced and not
//      defaulted to (true) by accident.
//
// Run:
//   MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_f5_test?sslmode=disable \
//     go test ./internal/adapter/postgres/ -run TestRLS_WithCheck -count=1 -race

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
)

func TestRLS_WithCheck_Conversations_OwnTenantInsertSucceeds(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

	id := uuid.New()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)`,
		id, fx.OrgA)
	require.NoError(t, err, "F.5 S1: own-tenant INSERT must pass WITH CHECK")
}

func TestRLS_WithCheck_Conversations_ForeignTenantInsertRejected(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

	id := uuid.New()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)`,
		id, fx.OrgB)
	require.Error(t, err, "F.5 S1: foreign-tenant INSERT must be rejected by WITH CHECK")
	assert.ErrorContains(t, err, "row-level security",
		"rejection MUST come from RLS, not from another constraint")
}

func TestRLS_WithCheck_Invoice_ForeignTenantInsertRejected(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

	// Build a valid invoice payload but with recipient_organization_id
	// set to OrgB — must be rejected by the new WITH CHECK clause.
	id := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO invoice (
			id, number, recipient_organization_id, recipient_snapshot,
			issuer_snapshot, service_period_start, service_period_end,
			amount_excl_tax_cents, amount_incl_tax_cents, tax_regime, source_type
		) VALUES ($1, $2, $3, '{}'::jsonb, '{}'::jsonb, now(), now(),
		          1000, 1200, 'fr_franchise_base', 'subscription')`,
		id, "FAC-WC-"+id.String()[:8], fx.OrgB)
	require.Error(t, err, "F.5 S1: foreign-tenant invoice INSERT must be rejected")
	assert.ErrorContains(t, err, "row-level security")
}

func TestRLS_WithCheck_Notifications_ForeignUserInsertRejected(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	// userA's context — try to insert a notification owned by userB.
	setOrgContext(t, ctx, tx, uuid.Nil, fx.UserA)

	id := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, type, title)
		VALUES ($1, $2, 'system', 'cross-user write')`, id, fx.UserB)
	require.Error(t, err, "F.5 S1: foreign-user notification INSERT must be rejected")
	assert.ErrorContains(t, err, "row-level security")
}

func TestRLS_WithCheck_PaymentRecords_ForeignTenantInsertRejected(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

	id := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment_records (
			id, proposal_id, milestone_id, client_id, provider_id, organization_id,
			proposal_amount, platform_fee_amount, client_total_amount, provider_payout
		) VALUES ($1, $2, $3, $4, $5, $6, 1000, 100, 1100, 900)`,
		id, fx.ProposalB, fx.MilestoneB, fx.UserB, fx.UserB, fx.OrgB)
	require.Error(t, err, "F.5 S1: foreign-tenant payment_record INSERT must be rejected")
	assert.ErrorContains(t, err, "row-level security")
}

// TestRLS_WithCheck_AllNineTablesHaveWithCheck is a metadata sanity
// check: every tested table's policy MUST have an explicit WITH CHECK
// clause after migration 135. This guards against a future migration
// that drops + recreates the policy without restating WITH CHECK.
func TestRLS_WithCheck_AllNineTablesHaveWithCheck(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	tables := []string{
		"audit_logs", // already had WITH CHECK from m.129
		"conversations",
		"messages",
		"invoice",
		"proposals",
		"proposal_milestones",
		"notifications",
		"disputes",
		"payment_records",
	}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			var hasUsing, hasWithCheck sql.NullBool
			err := db.QueryRowContext(ctx, `
				SELECT polqual IS NOT NULL, polwithcheck IS NOT NULL
				FROM   pg_policy
				WHERE  polrelid = $1::regclass
				ORDER  BY polname
				LIMIT  1
			`, table).Scan(&hasUsing, &hasWithCheck)
			require.NoError(t, err, "F.5 S1: every table must have an isolation policy")
			assert.True(t, hasUsing.Bool, "USING expected")
			assert.True(t, hasWithCheck.Bool,
				"F.5 S1: %s policy must carry an explicit WITH CHECK after migration 135",
				table)
		})
	}
}

// TestRLS_WithCheck_AuditLogsAcceptsUnsetContext confirms migration 135
// did not regress migration 129's promise: audit_logs INSERT under an
// unset app.current_user_id MUST still succeed (background workers,
// system actor logs). Adding WITH CHECK to other tables must not have
// rewritten audit_logs's WITH CHECK (true) clause.
func TestRLS_WithCheck_AuditLogsAcceptsUnsetContext(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	actorID := insertTestUser(t, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	setRLSRole(t, ctx, tx)
	// app.current_user_id NOT SET — the BUG-NEW-07 bug case.

	auditID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_logs (id, user_id, action) VALUES ($1, $2, 'system_action')`,
		auditID, actorID)
	require.NoError(t, err,
		"F.5 S1 must preserve m.129: audit_logs INSERT must succeed without tenant context")

	_ = postgres.SetCurrentOrg // keep import live regardless of compile order
}
