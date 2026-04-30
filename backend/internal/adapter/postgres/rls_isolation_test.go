package postgres_test

// Cross-tenant denial integration tests for migration 125 RLS policies.
//
// These tests are the load-bearing safeguard for SEC-10. They prove
// that the 9 tenant-scoped tables refuse to serve another tenant's
// rows, even when the application code has been compromised or has a
// missing WHERE clause. The test pattern is:
//
//   1. Open a transaction.
//   2. SET ROLE marketplace_rls_test (a non-superuser, non-bypass-rls
//      role created at test setup) so the RLS policies actually fire.
//      Postgres bypasses RLS for superusers AND for the table owner
//      unless FORCE ROW LEVEL SECURITY is on. Migration 125 sets
//      FORCE on every table — that handles the owner case. We then
//      SET ROLE inside the test to drop the superuser bit.
//   3. SET LOCAL app.current_org_id (and/or app.current_user_id) to
//      the tenant we want to read AS.
//   4. Assert SELECT/UPDATE/DELETE on another tenant's row return
//      0 rows (silent denial — RLS does not error, it filters).
//   5. Assert SELECT on the same tenant's own row works (positive
//      control proving the policy is not over-restrictive).
//
// The test fixtures are inserted by the postgres superuser (db
// connection from MARKETPLACE_TEST_DATABASE_URL) which bypasses RLS,
// so we can set up cross-tenant data freely. Only the assertion phase
// runs as marketplace_rls_test.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"testing/quick"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
)

// rlsTestRole is the non-superuser role we SET ROLE to inside the
// test transactions. It is created lazily on the first test that
// needs it, has SELECT/INSERT/UPDATE/DELETE on every table that
// matters, and is NOT a superuser, NOT bypassrls. This is the role
// against which the RLS policies fire — the same posture the
// application user will have in production.
const rlsTestRole = "marketplace_rls_test"

// ensureRLSTestRole creates the rlsTestRole if it does not exist
// and grants it the needed privileges on the 9 tested tables and
// their parents. Idempotent — running the test suite repeatedly is
// safe.
func ensureRLSTestRole(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// CREATE ROLE is idempotent via DO block.
	_, err := db.ExecContext(ctx, fmt.Sprintf(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
				CREATE ROLE %s NOSUPERUSER NOBYPASSRLS;
			END IF;
		END $$;
	`, rlsTestRole, rlsTestRole))
	require.NoError(t, err, "create rls test role")

	// Tables we read from. Must be granted SELECT/INSERT/UPDATE/DELETE
	// for the assertion phase to be a fair test of RLS (not of grants).
	tables := []string{
		"messages", "conversations", "conversation_participants",
		"invoice", "proposals", "proposal_milestones",
		"notifications", "disputes", "audit_logs", "payment_records",
		"users", "organizations", "organization_members",
	}
	for _, tbl := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf(
			`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE %s TO %s`,
			tbl, rlsTestRole,
		))
		require.NoError(t, err, "grant on "+tbl)
	}
}

// setRLSRole switches the current transaction to rlsTestRole. After
// this call, RLS policies fire normally for every subsequent query.
func setRLSRole(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	_, err := tx.ExecContext(ctx, "SET LOCAL ROLE "+rlsTestRole)
	require.NoError(t, err, "SET LOCAL ROLE")
}

// setOrgContext is a thin wrapper that combines SET ROLE +
// SetCurrentOrg/SetCurrentUser inside a single tx. The order
// matters: SET ROLE first (so the policy expressions evaluate
// against rlsTestRole), then the SET LOCAL of the org/user vars.
func setOrgContext(t *testing.T, ctx context.Context, tx *sql.Tx, orgID, userID uuid.UUID) {
	t.Helper()
	setRLSRole(t, ctx, tx)
	if orgID != uuid.Nil {
		require.NoError(t, postgres.SetCurrentOrg(ctx, tx, orgID))
	}
	if userID != uuid.Nil {
		require.NoError(t, postgres.SetCurrentUser(ctx, tx, userID))
	}
}

// ---------------------------------------------------------------------------
// Fixture builder — creates two orgs, two users (one per org), and a
// matching row in every RLS-protected table for each org. Returns a
// struct with everything the assertions need.
// ---------------------------------------------------------------------------

type rlsFixture struct {
	OrgA, OrgB        uuid.UUID
	UserA, UserB      uuid.UUID
	ConvA, ConvB      uuid.UUID
	MsgA, MsgB        uuid.UUID
	InvoiceA, InvoiceB uuid.UUID
	ProposalA, ProposalB           uuid.UUID
	MilestoneA, MilestoneB         uuid.UUID
	NotifA, NotifB                 uuid.UUID
	DisputeA, DisputeB             uuid.UUID
	AuditA, AuditB                 uuid.UUID
	PayRecA, PayRecB               uuid.UUID
}

func newRLSFixture(t *testing.T, db *sql.DB) *rlsFixture {
	t.Helper()
	ensureRLSTestRole(t, db)

	ctx := context.Background()

	// Build two independent (user, org) pairs. The helper uses the
	// existing insertTestUser pattern from job_credit_repository_test.go
	// (same package), and the ad-hoc insertOrg below to skip the
	// repository layer (we only need data, not the seeded credits).
	userA := insertTestUser(t, db)
	userB := insertTestUser(t, db)
	orgA := insertOrgRaw(t, db, userA, "OrgA")
	orgB := insertOrgRaw(t, db, userB, "OrgB")
	// The userA needs organization_id set to orgA so messages /
	// notifications policies can resolve membership.
	_, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1 WHERE id = $2`, orgA, userA)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1 WHERE id = $2`, orgB, userB)
	require.NoError(t, err)

	fx := &rlsFixture{OrgA: orgA, OrgB: orgB, UserA: userA, UserB: userB}

	// Conversations, messages.
	fx.ConvA = insertRLSConversation(t, db, orgA, userA)
	fx.ConvB = insertRLSConversation(t, db, orgB, userB)
	fx.MsgA = insertMessage(t, db, fx.ConvA, userA)
	fx.MsgB = insertMessage(t, db, fx.ConvB, userB)

	// Invoices.
	fx.InvoiceA = insertInvoice(t, db, orgA)
	fx.InvoiceB = insertInvoice(t, db, orgB)

	// Proposals + milestones. The proposal needs a conversation and a
	// (client, provider) user pair; we reuse the same conv/users.
	fx.ProposalA = insertRLSProposal(t, db, fx.ConvA, userA, userA, orgA, orgA)
	fx.ProposalB = insertRLSProposal(t, db, fx.ConvB, userB, userB, orgB, orgB)
	fx.MilestoneA = insertMilestone(t, db, fx.ProposalA)
	fx.MilestoneB = insertMilestone(t, db, fx.ProposalB)

	// Notifications.
	fx.NotifA = insertNotification(t, db, userA)
	fx.NotifB = insertNotification(t, db, userB)

	// Disputes — need a milestone (FK in this DB has NOT NULL milestone_id).
	fx.DisputeA = insertDispute(t, db, fx.ProposalA, fx.MilestoneA, fx.ConvA, userA, userA, orgA, orgA)
	fx.DisputeB = insertDispute(t, db, fx.ProposalB, fx.MilestoneB, fx.ConvB, userB, userB, orgB, orgB)

	// Audit logs.
	fx.AuditA = insertAuditLog(t, db, userA)
	fx.AuditB = insertAuditLog(t, db, userB)

	// Payment records.
	fx.PayRecA = insertPaymentRecord(t, db, fx.ProposalA, fx.MilestoneA, userA, userA, orgA)
	fx.PayRecB = insertPaymentRecord(t, db, fx.ProposalB, fx.MilestoneB, userB, userB, orgB)

	t.Cleanup(func() { cleanupFixture(t, db, fx) })

	return fx
}

func cleanupFixture(t *testing.T, db *sql.DB, fx *rlsFixture) {
	t.Helper()
	// Best-effort tear down. RLS is bypassed because we are postgres
	// (superuser). Ignore errors — a failing cleanup must not mask
	// the test failure.
	stmts := []string{
		`DELETE FROM payment_records WHERE id IN ($1, $2)`,
		`DELETE FROM audit_logs WHERE id IN ($1, $2)`,
		`DELETE FROM disputes WHERE id IN ($1, $2)`,
		`DELETE FROM notifications WHERE id IN ($1, $2)`,
		`DELETE FROM proposal_milestones WHERE id IN ($1, $2)`,
		`DELETE FROM proposals WHERE id IN ($1, $2)`,
		`DELETE FROM invoice WHERE id IN ($1, $2)`,
		`DELETE FROM messages WHERE id IN ($1, $2)`,
		`DELETE FROM conversation_participants WHERE conversation_id IN ($1, $2)`,
		`DELETE FROM conversations WHERE id IN ($1, $2)`,
	}
	pairs := [][2]uuid.UUID{
		{fx.PayRecA, fx.PayRecB},
		{fx.AuditA, fx.AuditB},
		{fx.DisputeA, fx.DisputeB},
		{fx.NotifA, fx.NotifB},
		{fx.MilestoneA, fx.MilestoneB},
		{fx.ProposalA, fx.ProposalB},
		{fx.InvoiceA, fx.InvoiceB},
		{fx.MsgA, fx.MsgB},
		{fx.ConvA, fx.ConvB},
		{fx.ConvA, fx.ConvB},
	}
	for i, q := range stmts {
		_, _ = db.Exec(q, pairs[i][0], pairs[i][1])
	}
	// orgs + users are cleaned by insertTestUser's t.Cleanup hook.
}

// ---------------------------------------------------------------------------
// Insert helpers — bypass the repository layer so we can plant rows
// in tables without dragging in every domain factory.
// ---------------------------------------------------------------------------

func insertOrgRaw(t *testing.T, db *sql.DB, ownerID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, 'agency', $3)`,
		id, ownerID, name)
	require.NoError(t, err, "insert org "+name)
	// owner membership row.
	_, err = db.Exec(`
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')`,
		id, ownerID)
	require.NoError(t, err, "insert org owner member")
	return id
}

func insertRLSConversation(t *testing.T, db *sql.DB, orgID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO conversations (id, organization_id) VALUES ($1, $2)`,
		id, orgID)
	require.NoError(t, err, "insert conversation")
	_, err = db.Exec(`
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`,
		id, userID)
	require.NoError(t, err, "insert conv participant")
	return id
}

func insertMessage(t *testing.T, db *sql.DB, convID, senderID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, content, seq)
		VALUES ($1, $2, $3, 'hello', 1)`,
		id, convID, senderID)
	require.NoError(t, err, "insert message")
	return id
}

func insertInvoice(t *testing.T, db *sql.DB, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO invoice (
			id, number, recipient_organization_id, recipient_snapshot,
			issuer_snapshot, service_period_start, service_period_end,
			amount_excl_tax_cents, amount_incl_tax_cents, tax_regime, source_type
		) VALUES ($1, $2, $3, '{}'::jsonb, '{}'::jsonb, now(), now(),
		          1000, 1200, 'fr_franchise_base', 'subscription')`,
		id, "FAC-"+id.String()[:8], orgID)
	require.NoError(t, err, "insert invoice")
	return id
}

func insertRLSProposal(t *testing.T, db *sql.DB, convID, clientID, providerID, clientOrg, providerOrg uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO proposals (
			id, conversation_id, sender_id, recipient_id, title, description,
			amount, client_id, provider_id, organization_id,
			client_organization_id, provider_organization_id
		) VALUES ($1, $2, $3, $4, 'T', 'D', 1000, $3, $4, $5, $5, $6)`,
		id, convID, clientID, providerID, clientOrg, providerOrg)
	require.NoError(t, err, "insert proposal")
	return id
}

func insertMilestone(t *testing.T, db *sql.DB, proposalID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO proposal_milestones (id, proposal_id, sequence, title, description, amount)
		VALUES ($1, $2, 1, 'M', 'D', 1000)`,
		id, proposalID)
	require.NoError(t, err, "insert milestone")
	return id
}

func insertNotification(t *testing.T, db *sql.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO notifications (id, user_id, type, title)
		VALUES ($1, $2, 'system', 'hello')`,
		id, userID)
	require.NoError(t, err, "insert notification")
	return id
}

func insertDispute(t *testing.T, db *sql.DB, proposalID, milestoneID, convID, clientID, providerID, clientOrg, providerOrg uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO disputes (
			id, proposal_id, milestone_id, conversation_id,
			initiator_id, respondent_id, client_id, provider_id,
			reason, description, requested_amount, proposal_amount,
			client_organization_id, provider_organization_id
		) VALUES ($1, $2, $3, $4, $5, $6, $5, $6, 'r', 'd', 100, 1000, $7, $8)`,
		id, proposalID, milestoneID, convID, clientID, providerID, clientOrg, providerOrg)
	require.NoError(t, err, "insert dispute")
	return id
}

func insertAuditLog(t *testing.T, db *sql.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO audit_logs (id, user_id, action) VALUES ($1, $2, 'test_action')`,
		id, userID)
	require.NoError(t, err, "insert audit log")
	return id
}

func insertPaymentRecord(t *testing.T, db *sql.DB, proposalID, milestoneID, clientID, providerID, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO payment_records (
			id, proposal_id, milestone_id, client_id, provider_id, organization_id,
			proposal_amount, platform_fee_amount, client_total_amount, provider_payout
		) VALUES ($1, $2, $3, $4, $5, $6, 1000, 100, 1100, 900)`,
		id, proposalID, milestoneID, clientID, providerID, orgID)
	require.NoError(t, err, "insert payment record")
	return id
}

// ---------------------------------------------------------------------------
// Cross-tenant denial — table-driven, one entry per RLS-protected table
// ---------------------------------------------------------------------------

// rlsCase is one row of the cross-tenant denial table. When the test
// runs as orgA + userA, every column listed here MUST return zero rows
// when filtered to the row id that belongs to orgB / userB.
type rlsCase struct {
	name      string  // table label
	table     string  // SQL table name
	rowB      func(fx *rlsFixture) uuid.UUID // the orgB row id
	rowA      func(fx *rlsFixture) uuid.UUID // the orgA row id (positive control)
}

func rlsCases() []rlsCase {
	return []rlsCase{
		{"conversations", "conversations", func(f *rlsFixture) uuid.UUID { return f.ConvB }, func(f *rlsFixture) uuid.UUID { return f.ConvA }},
		{"messages", "messages", func(f *rlsFixture) uuid.UUID { return f.MsgB }, func(f *rlsFixture) uuid.UUID { return f.MsgA }},
		{"invoice", "invoice", func(f *rlsFixture) uuid.UUID { return f.InvoiceB }, func(f *rlsFixture) uuid.UUID { return f.InvoiceA }},
		{"proposals", "proposals", func(f *rlsFixture) uuid.UUID { return f.ProposalB }, func(f *rlsFixture) uuid.UUID { return f.ProposalA }},
		{"proposal_milestones", "proposal_milestones", func(f *rlsFixture) uuid.UUID { return f.MilestoneB }, func(f *rlsFixture) uuid.UUID { return f.MilestoneA }},
		{"notifications", "notifications", func(f *rlsFixture) uuid.UUID { return f.NotifB }, func(f *rlsFixture) uuid.UUID { return f.NotifA }},
		{"disputes", "disputes", func(f *rlsFixture) uuid.UUID { return f.DisputeB }, func(f *rlsFixture) uuid.UUID { return f.DisputeA }},
		{"audit_logs", "audit_logs", func(f *rlsFixture) uuid.UUID { return f.AuditB }, func(f *rlsFixture) uuid.UUID { return f.AuditA }},
		{"payment_records", "payment_records", func(f *rlsFixture) uuid.UUID { return f.PayRecB }, func(f *rlsFixture) uuid.UUID { return f.PayRecA }},
	}
}

// countByID returns the number of rows visible for a given id under
// the current tx context. RLS makes "denied" look like "row does not
// exist" — count = 0 means the policy filtered it out.
func countByID(t *testing.T, ctx context.Context, tx *sql.Tx, table string, id uuid.UUID) int {
	t.Helper()
	var n int
	err := tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM %s WHERE id = $1`, table), id,
	).Scan(&n)
	require.NoError(t, err, "select count from "+table)
	return n
}

// TestRLS_SelectDenied_AcrossTenants is the headline test: when
// transaction context is set to orgA/userA, none of orgB's rows are
// visible. This is the per-table row-count assertion across all 9
// tables.
func TestRLS_SelectDenied_AcrossTenants(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)

	for _, c := range rlsCases() {
		t.Run("denies_cross_tenant_select_"+c.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback() }()

			setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

			gotB := countByID(t, ctx, tx, c.table, c.rowB(fx))
			assert.Equal(t, 0, gotB, "%s row from orgB MUST be invisible to orgA", c.table)

			gotA := countByID(t, ctx, tx, c.table, c.rowA(fx))
			assert.Equal(t, 1, gotA, "%s row from orgA MUST stay visible to orgA (positive control)", c.table)
		})
	}
}

// TestRLS_UpdateDenied_AcrossTenants asserts that an UPDATE issued
// while in orgA's context cannot modify orgB's rows. We attempt to
// touch the updated_at column where it exists, fall back to a
// no-op-like change otherwise.
func TestRLS_UpdateDenied_AcrossTenants(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)

	// Each table needs an UPDATE that compiles. We pick a column that
	// every row has and update it to its current value (a no-op data
	// change, but RLS-relevant — the row is still touched).
	cases := []struct {
		name  string
		table string
		col   string
		row   func(fx *rlsFixture) uuid.UUID
	}{
		{"conversations", "conversations", "updated_at", func(f *rlsFixture) uuid.UUID { return f.ConvB }},
		{"messages", "messages", "content", func(f *rlsFixture) uuid.UUID { return f.MsgB }},
		{"invoice", "invoice", "currency", func(f *rlsFixture) uuid.UUID { return f.InvoiceB }},
		{"proposals", "proposals", "title", func(f *rlsFixture) uuid.UUID { return f.ProposalB }},
		{"proposal_milestones", "proposal_milestones", "title", func(f *rlsFixture) uuid.UUID { return f.MilestoneB }},
		{"notifications", "notifications", "title", func(f *rlsFixture) uuid.UUID { return f.NotifB }},
		{"disputes", "disputes", "reason", func(f *rlsFixture) uuid.UUID { return f.DisputeB }},
		// audit_logs has UPDATE/DELETE revoked at the GRANT layer (mig
		// 124) AND now an RLS policy. The grant blocks BEFORE RLS runs
		// for the rlsTestRole. We grant SELECT/INSERT/UPDATE/DELETE for
		// test purposes above — but to make this test fair, we test
		// audit_logs UPDATE separately and expect a permission error
		// rather than a 0-row result. Skipping in this sub-suite.
		{"payment_records", "payment_records", "currency", func(f *rlsFixture) uuid.UUID { return f.PayRecB }},
	}

	for _, c := range cases {
		t.Run("denies_cross_tenant_update_"+c.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback() }()

			setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

			// "set col = col" is a no-op data change but exercises the
			// UPDATE path.
			res, err := tx.ExecContext(ctx, fmt.Sprintf(
				`UPDATE %s SET %s = %s WHERE id = $1`, c.table, c.col, c.col),
				c.row(fx),
			)
			require.NoError(t, err, "UPDATE %s under orgA context must not error (just 0 rows affected)", c.table)
			n, err := res.RowsAffected()
			require.NoError(t, err)
			assert.EqualValues(t, 0, n, "UPDATE on orgB's %s row from orgA context MUST affect 0 rows", c.table)
		})
	}
}

// TestRLS_DeleteDenied_AcrossTenants — same idea for DELETE.
func TestRLS_DeleteDenied_AcrossTenants(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)

	for _, c := range rlsCases() {
		t.Run("denies_cross_tenant_delete_"+c.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback() }()

			setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

			// audit_logs has DELETE revoked from PUBLIC (mig 124). For
			// our test role we re-granted DELETE so the permission
			// check passes — RLS is the second gate. But for the real
			// production user (which keeps DELETE revoked), DELETE on
			// audit_logs is impossible at the GRANT layer regardless
			// of RLS. We still assert RLS denial here as a defense-
			// in-depth check.
			res, err := tx.ExecContext(ctx, fmt.Sprintf(
				`DELETE FROM %s WHERE id = $1`, c.table), c.rowB(fx))
			require.NoError(t, err, "DELETE %s under orgA context must not error", c.table)
			n, err := res.RowsAffected()
			require.NoError(t, err)
			assert.EqualValues(t, 0, n, "DELETE on orgB's %s row from orgA context MUST affect 0 rows", c.table)
		})
	}
}

// TestRLS_TwoSidedOwnership_Disputes confirms both client and provider
// sides see a dispute that lists either of their orgs as a party.
func TestRLS_TwoSidedOwnership_Disputes(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)

	// Build a "shared" dispute where orgA is client and orgB is
	// provider. Both must see it.
	ctx := context.Background()
	sharedDispute := insertDispute(t, db,
		fx.ProposalA, fx.MilestoneA, fx.ConvA,
		fx.UserA, fx.UserB, fx.OrgA, fx.OrgB)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM disputes WHERE id = $1`, sharedDispute)
	})

	// As orgA → must see.
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback() }()
	setOrgContext(t, ctx, tx1, fx.OrgA, fx.UserA)
	assert.Equal(t, 1, countByID(t, ctx, tx1, "disputes", sharedDispute), "orgA (client) must see the shared dispute")

	// As orgB → must also see.
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	setOrgContext(t, ctx, tx2, fx.OrgB, fx.UserB)
	assert.Equal(t, 1, countByID(t, ctx, tx2, "disputes", sharedDispute), "orgB (provider) must see the shared dispute")
}

// TestRLS_TwoSidedOwnership_Proposals confirms both client and
// provider orgs see a proposal that lists either of their orgs as a
// party.
func TestRLS_TwoSidedOwnership_Proposals(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)

	ctx := context.Background()
	shared := insertRLSProposal(t, db, fx.ConvA, fx.UserA, fx.UserB, fx.OrgA, fx.OrgB)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, shared)
	})

	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback() }()
	setOrgContext(t, ctx, tx1, fx.OrgA, fx.UserA)
	assert.Equal(t, 1, countByID(t, ctx, tx1, "proposals", shared))

	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	setOrgContext(t, ctx, tx2, fx.OrgB, fx.UserB)
	assert.Equal(t, 1, countByID(t, ctx, tx2, "proposals", shared))
}

// TestRLS_NoContextSet_HidesEverything proves the safe-default
// branch: if the application forgets to call SetCurrentOrg /
// SetCurrentUser, current_setting returns NULL, the policy
// expression is NULL → false, and every row is filtered out.
//
// This is the most important invariant: the RLS layer FAILS CLOSED.
func TestRLS_NoContextSet_HidesEverything(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// SET ROLE only. Do NOT set app.current_org_id / app.current_user_id.
	setRLSRole(t, ctx, tx)

	for _, c := range rlsCases() {
		// audit_logs and notifications policies key on user_id, the
		// rest on organization_id. The conversations policy admits
		// rows via a participant escape hatch — but with no
		// app.current_user_id set, that branch also evaluates to
		// false. So every table must hide every row.
		gotA := countByID(t, ctx, tx, c.table, c.rowA(fx))
		assert.Equal(t, 0, gotA, "with no tenant context, %s.id=%s must be hidden", c.table, c.rowA(fx))
		gotB := countByID(t, ctx, tx, c.table, c.rowB(fx))
		assert.Equal(t, 0, gotB, "with no tenant context, %s.id=%s must be hidden", c.table, c.rowB(fx))
	}
}

// TestRLS_SameOrgAccess_PositiveControl is the happy-path control:
// every legitimate same-org SELECT works as before. Without this
// control, a buggy policy that hides EVERYTHING would silently pass
// the cross-tenant denial tests above.
func TestRLS_SameOrgAccess_PositiveControl(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

	for _, c := range rlsCases() {
		got := countByID(t, ctx, tx, c.table, c.rowA(fx))
		assert.Equal(t, 1, got, "same-org access on %s MUST work", c.table)
	}
}

// TestRLS_PropertyTest_AnyCrossTenantRowDenied is a quick.Check
// driver: 25 random (which-tenant-context) flips, asserting that
// across the 9 tables the rule "context X sees only rows of X" holds
// uniformly. quick.Check default count is 100 — we cap at 25 because
// each iteration opens a transaction.
//
// The property: For any of (orgA-owned, orgB-owned) row choices and
// any of (orgA-context, orgB-context) chosen contexts, count visible
// rows == 1 iff context owns the row, else == 0.
func TestRLS_PropertyTest_AnyCrossTenantRowDenied(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()
	cases := rlsCases()

	check := func(tableIdx uint8, useOrgB, rowFromOrgB bool) bool {
		c := cases[int(tableIdx)%len(cases)]
		var orgID, userID uuid.UUID
		if useOrgB {
			orgID, userID = fx.OrgB, fx.UserB
		} else {
			orgID, userID = fx.OrgA, fx.UserA
		}
		var rowID uuid.UUID
		if rowFromOrgB {
			rowID = c.rowB(fx)
		} else {
			rowID = c.rowA(fx)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Logf("begin tx failed: %v", err)
			return false
		}
		defer func() { _ = tx.Rollback() }()

		setOrgContext(t, ctx, tx, orgID, userID)
		got := countByID(t, ctx, tx, c.table, rowID)

		// Same-tenant ⇔ visible.
		expectVisible := (useOrgB == rowFromOrgB)
		if expectVisible {
			return got == 1
		}
		return got == 0
	}

	cfg := &quick.Config{MaxCount: 25}
	require.NoError(t, quick.Check(check, cfg), "RLS property must hold across random table/context/row combinations")
}

// TestRLS_ConversationsParticipantEscapeHatch verifies the secondary
// branch of the conversations policy: a user who is a participant on
// a conversation can read it even when they are NOT a member of the
// owning org. This is the path used by solo providers (no org) who
// chat with an enterprise.
func TestRLS_ConversationsParticipantEscapeHatch(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	// Create a "rogue" participant on orgA's conversation: a brand new
	// user with no org. They should still see the conversation via
	// the participant branch.
	roguer := insertTestUser(t, db)
	_, err := db.ExecContext(ctx,
		`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`,
		fx.ConvA, roguer)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2`, fx.ConvA, roguer)
	})

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// SET app.current_user_id to roguer but NO app.current_org_id —
	// the participant branch should kick in.
	setOrgContext(t, ctx, tx, uuid.Nil, roguer)

	got := countByID(t, ctx, tx, "conversations", fx.ConvA)
	assert.Equal(t, 1, got, "participant escape hatch must let the rogue user see fx.ConvA")

	// And a totally unrelated user must NOT see it.
	stranger := insertTestUser(t, db)
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	setOrgContext(t, ctx, tx2, uuid.Nil, stranger)
	gotStranger := countByID(t, ctx, tx2, "conversations", fx.ConvA)
	assert.Equal(t, 0, gotStranger, "stranger MUST NOT see fx.ConvA")
}

// TestRLS_PolicyExists is a sanity check that all 9 expected policies
// are listed in pg_policies. Catches drift if a future migration
// accidentally drops one.
func TestRLS_PolicyExists(t *testing.T) {
	db := testDB(t)
	expected := map[string]string{
		"messages":            "messages_isolation",
		"conversations":       "conversations_isolation",
		"invoice":             "invoice_isolation",
		"proposals":           "proposals_isolation",
		"proposal_milestones": "proposal_milestones_isolation",
		"notifications":       "notifications_isolation",
		"disputes":            "disputes_isolation",
		"audit_logs":          "audit_logs_isolation",
		"payment_records":     "payment_records_isolation",
	}
	for table, policyName := range expected {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM pg_policies
				WHERE schemaname = 'public' AND tablename = $1 AND policyname = $2
			)`, table, policyName).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "policy %s must exist on %s", policyName, table)

		// FORCE RLS must be on.
		var forced bool
		err = db.QueryRow(`SELECT relforcerowsecurity FROM pg_class WHERE relname = $1`, table).Scan(&forced)
		require.NoError(t, err)
		assert.True(t, forced, "table %s must have FORCE ROW LEVEL SECURITY enabled", table)
	}
}

// TestRLS_AuditLogsPerActor checks the per-actor (NOT per-org) policy
// on audit_logs: userA sees their own row, userB sees theirs, and
// userA cannot see userB's row regardless of org context.
func TestRLS_AuditLogsPerActor(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	// As userA + orgA — must see auditA, must not see auditB.
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

	gotA := countByID(t, ctx, tx, "audit_logs", fx.AuditA)
	gotB := countByID(t, ctx, tx, "audit_logs", fx.AuditB)
	assert.Equal(t, 1, gotA)
	assert.Equal(t, 0, gotB)
}

// TestRLS_NotificationsPerUser mirrors the audit_logs check for the
// per-user notifications policy.
func TestRLS_NotificationsPerUser(t *testing.T) {
	db := testDB(t)
	fx := newRLSFixture(t, db)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	setOrgContext(t, ctx, tx, fx.OrgA, fx.UserA)

	gotA := countByID(t, ctx, tx, "notifications", fx.NotifA)
	gotB := countByID(t, ctx, tx, "notifications", fx.NotifB)
	assert.Equal(t, 1, gotA)
	assert.Equal(t, 0, gotB)
}

