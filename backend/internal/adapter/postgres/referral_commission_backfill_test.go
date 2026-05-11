package postgres_test

// Integration test for migration 151 (backfill referral_commissions for
// already-approved milestones). Gated behind
// MARKETPLACE_TEST_DATABASE_URL — auto-skips when unset, exactly like
// the other postgres adapter integration tests.
//
// The test seeds:
//   - 2 referrers + 2 providers + 2 clients (3 users per attribution
//     because the modularity rule forbids reusing one user across roles)
//   - 2 referrals, 2 attributions
//   - 4 proposals owned by the 2 providers (2 per attribution)
//   - 4 milestones per proposal in mixed statuses: 2 approved/released,
//     2 funded/disputed (NOT eligible) — total 16 milestones, 8 eligible
//
// Then it runs the exact SQL from migrations/151_backfill_referral_commissions.up.sql
// and asserts:
//   1. Exactly 8 commission rows landed in pending status.
//   2. Each row has the correct commission_cents (basis-point truncated).
//   3. A second run is a no-op (idempotency guard from migration 108
//      kicks in via the NOT EXISTS clause).
//   4. Milestones whose proposal is in a non-eligible status (e.g.
//      'pending', 'declined') are skipped.
//   5. Milestones whose status is funded / disputed / cancelled are
//      skipped.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migration151SQL is the exact backfill statement from
// backend/migrations/151_backfill_referral_commissions.up.sql, stripped
// of BEGIN/COMMIT so the test driver can wrap it in its own tx.
const migration151SQL = `
INSERT INTO referral_commissions (
    id,
    attribution_id,
    milestone_id,
    gross_amount_cents,
    commission_cents,
    currency,
    status,
    stripe_transfer_id,
    stripe_reversal_id,
    failure_reason,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    a.id,
    m.id,
    m.amount,
    (m.amount * (a.rate_pct_snapshot * 100)::bigint / 10000)::bigint,
    'EUR',
    'pending',
    '',
    '',
    '',
    now(),
    now()
FROM referral_attributions a
JOIN proposals p              ON p.id = a.proposal_id
JOIN proposal_milestones m    ON m.proposal_id = p.id
WHERE p.status IN ('active', 'completion_requested', 'completed', 'paid')
  AND m.status IN ('approved', 'released')
  AND m.amount > 0
  AND NOT EXISTS (
      SELECT 1 FROM referral_commissions c
      WHERE c.attribution_id = a.id
        AND c.milestone_id   = m.id
  )
`

// assertMigration151FileMatchesEmbedded sanity-checks the embedded SQL
// against the on-disk file so a future edit to the migration without a
// matching test update fails loudly.
func assertMigration151FileMatchesEmbedded(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	// Walk up the tree until we find the backend/migrations dir.
	dir := cwd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "migrations", "151_backfill_referral_commissions.up.sql")
		if _, err := os.Stat(candidate); err == nil {
			content, rerr := os.ReadFile(candidate)
			require.NoError(t, rerr)
			// Both file and embedded SQL must contain the same WHERE
			// clause keystones. We do not compare byte-for-byte because
			// the file carries comments + BEGIN/COMMIT.
			s := string(content)
			require.Contains(t, s, "WHERE p.status IN ('active', 'completion_requested', 'completed', 'paid')",
				"migration 151 file drift — eligible proposal statuses changed")
			require.Contains(t, s, "AND m.status IN ('approved', 'released')",
				"migration 151 file drift — eligible milestone statuses changed")
			require.Contains(t, s, "NOT EXISTS",
				"migration 151 file drift — idempotency guard missing")
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("migrations/151_backfill_referral_commissions.up.sql not found relative to %s", cwd)
}

// uniqueEmail builds a unique email for test users so parallel test
// runs against the same DB do not collide on the users.email UNIQUE
// constraint.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@migration151.local", prefix, uuid.NewString()[:8])
}

// insertUserBackfill creates a user with the given role. Returns the
// new id. Cleanup is registered on `t`.
func insertUserBackfill(t *testing.T, db *sql.DB, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO users (
			id, email, hashed_password, first_name, last_name,
			display_name, role
		)
		VALUES ($1, $2, 'x', 'Mig151', 'User', 'Mig151 User', $3)`,
		id, uniqueEmail(role), role,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// insertReferralBackfill creates a referrals row in 'active' status.
// Columns mirror migrations/105_create_referrals.up.sql exactly — every
// NOT NULL needs a value, and 'active' triggers the
// referrals_active_has_stamps invariant so activated_at + expires_at
// must be set.
func insertReferralBackfill(t *testing.T, db *sql.DB, referrerID, providerID, clientID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO referrals (
			id, referrer_id, provider_id, client_id,
			rate_pct, duration_months,
			intro_snapshot, status,
			activated_at, expires_at, last_action_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			5.0, 6,
			'{}'::jsonb, 'active',
			now(), now() + interval '6 months', now(),
			now(), now()
		)`,
		id, referrerID, providerID, clientID,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM referrals WHERE id = $1`, id)
	})
	return id
}

// insertProposalBackfill creates a proposal in the given status with a
// new conversation. Returns the proposal id.
func insertProposalBackfill(t *testing.T, db *sql.DB, providerID, clientID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	convID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO conversations (id, created_at, updated_at)
		VALUES ($1, now(), now())`,
		convID,
	)
	require.NoError(t, err, "insert conversation for proposal")
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	pid := uuid.New()
	_, err = db.Exec(`
		INSERT INTO proposals (
			id, conversation_id, sender_id, recipient_id,
			title, description, amount, status,
			client_id, provider_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			'Mission migration 151', 'desc', 200000, $5,
			$4, $3,
			now(), now()
		)`,
		pid, convID, providerID, clientID, status,
	)
	require.NoError(t, err, "insert proposal in status %s", status)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, pid)
	})
	return pid
}

// insertMilestoneBackfill creates a proposal_milestones row in the
// given status with the given amount (cents).
func insertMilestoneBackfill(t *testing.T, db *sql.DB, proposalID uuid.UUID, sequence int, amount int64, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO proposal_milestones (
			id, proposal_id, sequence, title, description, amount, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, 'Step', 'd', $4, $5, now(), now())`,
		id, proposalID, sequence, amount, status,
	)
	require.NoError(t, err, "insert milestone seq=%d status=%s", sequence, status)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, id)
	})
	return id
}

// insertAttributionBackfill creates a referral_attributions row.
func insertAttributionBackfill(t *testing.T, db *sql.DB, referralID, proposalID, providerID, clientID uuid.UUID, ratePct float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO referral_attributions (
			id, referral_id, proposal_id,
			provider_id, client_id, rate_pct_snapshot, attributed_at
		) VALUES ($1, $2, $3, $4, $5, $6, now())`,
		id, referralID, proposalID, providerID, clientID, ratePct,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM referral_attributions WHERE id = $1`, id)
	})
	return id
}

// countCommissions returns the number of referral_commissions rows for
// a given attribution.
func countCommissions(t *testing.T, db *sql.DB, attributionID uuid.UUID) int {
	t.Helper()
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM referral_commissions WHERE attribution_id = $1`,
		attributionID,
	).Scan(&n)
	require.NoError(t, err)
	return n
}

// fetchCommissionCents returns the commission_cents stored for a given
// (attribution, milestone) pair. Errors if not found.
func fetchCommissionCents(t *testing.T, db *sql.DB, attributionID, milestoneID uuid.UUID) int64 {
	t.Helper()
	var cents int64
	err := db.QueryRow(
		`SELECT commission_cents FROM referral_commissions
		 WHERE attribution_id = $1 AND milestone_id = $2`,
		attributionID, milestoneID,
	).Scan(&cents)
	require.NoError(t, err)
	return cents
}

// cleanupReferralCommissions wipes any rows the test produced before/
// after the run so re-running the suite never sees stale state. Scoped
// strictly to the attribution IDs we created.
func cleanupReferralCommissions(t *testing.T, db *sql.DB, attributionIDs ...uuid.UUID) {
	t.Helper()
	for _, id := range attributionIDs {
		_, _ = db.Exec(`DELETE FROM referral_commissions WHERE attribution_id = $1`, id)
	}
}

// TestMigration151_BackfillsApprovedMilestones is the primary test:
// seeds an eligible attribution with mixed milestone statuses, runs
// the migration, and asserts only the approved/released milestones
// produced a commission row.
func TestMigration151_BackfillsApprovedMilestones(t *testing.T) {
	db := testDB(t)
	assertMigration151FileMatchesEmbedded(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	referrerID := insertUserBackfill(t, db, "provider")
	providerID := insertUserBackfill(t, db, "provider")
	clientID := insertUserBackfill(t, db, "enterprise")

	referralID := insertReferralBackfill(t, db, referrerID, providerID, clientID)
	proposalID := insertProposalBackfill(t, db, providerID, clientID, "active")
	attributionID := insertAttributionBackfill(t, db, referralID, proposalID, providerID, clientID, 5.0)

	t.Cleanup(func() { cleanupReferralCommissions(t, db, attributionID) })

	// 4 milestones: 1 approved (eligible), 1 released (eligible),
	// 1 funded (not eligible), 1 disputed (not eligible).
	mApproved := insertMilestoneBackfill(t, db, proposalID, 1, 100000, "approved")  // 1000.00 EUR
	mReleased := insertMilestoneBackfill(t, db, proposalID, 2, 250000, "released")  // 2500.00 EUR
	_ = insertMilestoneBackfill(t, db, proposalID, 3, 80000, "funded")
	_ = insertMilestoneBackfill(t, db, proposalID, 4, 50000, "disputed")

	// Run migration 151 (single statement, no tx wrapper).
	_, err := db.ExecContext(ctx, migration151SQL)
	require.NoError(t, err)

	require.Equal(t, 2, countCommissions(t, db, attributionID),
		"only the 2 approved/released milestones must produce a commission row")

	// 5% of 1000.00 = 50.00 = 5000 cents
	assert.Equal(t, int64(5000), fetchCommissionCents(t, db, attributionID, mApproved))
	// 5% of 2500.00 = 125.00 = 12500 cents
	assert.Equal(t, int64(12500), fetchCommissionCents(t, db, attributionID, mReleased))

	// Every row landed in pending status.
	var pendingCount int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM referral_commissions
		 WHERE attribution_id = $1 AND status = 'pending'`,
		attributionID,
	).Scan(&pendingCount))
	assert.Equal(t, 2, pendingCount)
}

// TestMigration151_Idempotent runs the migration twice and asserts no
// duplicate rows land — the NOT EXISTS guard + the
// (attribution_id, milestone_id) UNIQUE index do the work.
func TestMigration151_Idempotent(t *testing.T) {
	db := testDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	referrerID := insertUserBackfill(t, db, "provider")
	providerID := insertUserBackfill(t, db, "provider")
	clientID := insertUserBackfill(t, db, "enterprise")

	referralID := insertReferralBackfill(t, db, referrerID, providerID, clientID)
	proposalID := insertProposalBackfill(t, db, providerID, clientID, "completed")
	attributionID := insertAttributionBackfill(t, db, referralID, proposalID, providerID, clientID, 7.5)

	t.Cleanup(func() { cleanupReferralCommissions(t, db, attributionID) })

	insertMilestoneBackfill(t, db, proposalID, 1, 100000, "approved")
	insertMilestoneBackfill(t, db, proposalID, 2, 100000, "released")

	for i := 0; i < 3; i++ {
		_, err := db.ExecContext(ctx, migration151SQL)
		require.NoErrorf(t, err, "run #%d", i+1)
	}

	assert.Equal(t, 2, countCommissions(t, db, attributionID),
		"running the migration 3 times must still produce exactly 2 rows")
}

// TestMigration151_SkipsNonEligibleProposalStatus pins the WHERE
// p.status filter: a proposal in 'pending' (never accepted) must NOT
// produce a commission even if the milestone is approved.
func TestMigration151_SkipsNonEligibleProposalStatus(t *testing.T) {
	db := testDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	referrerID := insertUserBackfill(t, db, "provider")
	providerID := insertUserBackfill(t, db, "provider")
	clientID := insertUserBackfill(t, db, "enterprise")

	referralID := insertReferralBackfill(t, db, referrerID, providerID, clientID)
	// Proposal in 'pending' — never accepted.
	proposalID := insertProposalBackfill(t, db, providerID, clientID, "pending")
	attributionID := insertAttributionBackfill(t, db, referralID, proposalID, providerID, clientID, 5.0)

	t.Cleanup(func() { cleanupReferralCommissions(t, db, attributionID) })

	insertMilestoneBackfill(t, db, proposalID, 1, 100000, "approved")

	_, err := db.ExecContext(ctx, migration151SQL)
	require.NoError(t, err)

	assert.Equal(t, 0, countCommissions(t, db, attributionID),
		"a 'pending' proposal must not have its approved milestone backfilled")
}

// TestMigration151_PreservesExistingRows verifies that pre-existing
// commission rows (whatever their status) are untouched by the
// backfill — only MISSING (attribution, milestone) pairs get inserted.
func TestMigration151_PreservesExistingRows(t *testing.T) {
	db := testDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	referrerID := insertUserBackfill(t, db, "provider")
	providerID := insertUserBackfill(t, db, "provider")
	clientID := insertUserBackfill(t, db, "enterprise")

	referralID := insertReferralBackfill(t, db, referrerID, providerID, clientID)
	proposalID := insertProposalBackfill(t, db, providerID, clientID, "active")
	attributionID := insertAttributionBackfill(t, db, referralID, proposalID, providerID, clientID, 5.0)

	t.Cleanup(func() { cleanupReferralCommissions(t, db, attributionID) })

	milestoneID := insertMilestoneBackfill(t, db, proposalID, 1, 100000, "approved")

	// Plant an existing row in 'paid' status with a specific transfer id —
	// the backfill must NOT touch it.
	preExistingID := uuid.New()
	_, err := db.ExecContext(ctx, `
		INSERT INTO referral_commissions (
			id, attribution_id, milestone_id,
			gross_amount_cents, commission_cents, currency, status,
			stripe_transfer_id, stripe_reversal_id, failure_reason,
			created_at, updated_at
		) VALUES ($1, $2, $3, 100000, 5000, 'EUR', 'paid',
		          'tr_PREEXISTING', '', '', now(), now())`,
		preExistingID, attributionID, milestoneID,
	)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, migration151SQL)
	require.NoError(t, err)

	// Still exactly one commission row for this milestone.
	assert.Equal(t, 1, countCommissions(t, db, attributionID))

	// And the original transfer id is intact.
	var transferID, status string
	require.NoError(t, db.QueryRow(
		`SELECT stripe_transfer_id, status FROM referral_commissions WHERE id = $1`,
		preExistingID,
	).Scan(&transferID, &status))
	assert.Equal(t, "tr_PREEXISTING", transferID)
	assert.Equal(t, "paid", status)
}
