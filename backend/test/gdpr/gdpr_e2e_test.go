package gdpr_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	gdprapp "marketplace-backend/internal/app/gdpr"
	domaingdpr "marketplace-backend/internal/domain/gdpr"
)

// gdprTestDB returns a DB connection for the GDPR integration tests.
// Mirrors the searchTestDB pattern: gated behind
// MARKETPLACE_TEST_DATABASE_URL, auto-skips when unset.
//
// The tests assume migration 132 has been applied. Run
// `make migrate-up` against the test DB before invoking.
func gdprTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping gdpr integration test")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedUser writes a user row + a unique email so the tests are
// self-contained. Returns the user id and the email so the caller
// can verify anonymization later.
func seedUser(t *testing.T, db *sql.DB, suffix string) (uuid.UUID, string) {
	t.Helper()
	uid := uuid.New()
	email := fmt.Sprintf("gdpr-%s-%s@e2e.test", suffix, uid.String()[:8])
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
		VALUES ($1, $2, 'hashed', 'Test', $3, $4, 'provider', 'marketplace_owner')`,
		uid, email, suffix, "GDPR Test "+suffix)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, uid)
	})
	return uid, email
}

func seedAuditLog(t *testing.T, db *sql.DB, userID uuid.UUID) {
	t.Helper()
	meta := map[string]any{
		"email":      "real-actor@example.com",
		"actor_name": "Real Name",
	}
	mb, _ := json.Marshal(meta)
	_, err := db.Exec(`
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at)
		VALUES ($1, $2, 'auth.login_success', 'user', $2, $3, '203.0.113.42'::inet, NOW())`,
		uuid.New(), userID, mb)
	require.NoError(t, err)
}

// TestGDPR_Lifecycle_HappyPath exercises the full happy-path flow:
// soft-delete → cron skip until day 30 → cron purge → audit anonymized.
//
// Time travel is simulated by writing the deleted_at timestamp far in
// the past so the cron's WHERE deleted_at < NOW() - INTERVAL '30 days'
// matches without us having to wait.
func TestGDPR_Lifecycle_HappyPath(t *testing.T) {
	db := gdprTestDB(t)

	uid, email := seedUser(t, db, "happy")
	seedAuditLog(t, db, uid)

	repo := postgres.NewGDPRRepository(db)

	// 1. Soft-delete the user.
	now := time.Now().UTC()
	deletedAt, err := repo.SoftDelete(context.Background(), uid, now)
	require.NoError(t, err)
	assert.WithinDuration(t, now, deletedAt, time.Second)

	// 2. Cron run BEFORE the 30-day window: nothing should purge.
	cutoffEarly := now.Add(-29 * 24 * time.Hour)
	ids, err := repo.ListPurgeable(context.Background(), cutoffEarly, 100)
	require.NoError(t, err)
	for _, id := range ids {
		assert.NotEqual(t, uid, id, "user must not be purgeable before T+30")
	}

	// 3. Time travel: set deleted_at 31 days in the past so the cron
	//    sees the row as ripe.
	old := now.Add(-31 * 24 * time.Hour)
	_, err = db.Exec(`UPDATE users SET deleted_at = $1 WHERE id = $2`, old, uid)
	require.NoError(t, err)

	// 4. Cron lists purgeable rows with cutoff = now - 30d.
	cutoff := now.Add(-30 * 24 * time.Hour)
	ids, err = repo.ListPurgeable(context.Background(), cutoff, 100)
	require.NoError(t, err)
	found := false
	for _, id := range ids {
		if id == uid {
			found = true
			break
		}
	}
	assert.True(t, found, "user should be in the purgeable list at T+31")

	// 5. Purge.
	salt := "test-gdpr-salt-stable-1"
	ok, err := repo.PurgeUser(context.Background(), uid, cutoff, salt)
	require.NoError(t, err)
	assert.True(t, ok, "purge should report success")

	// 6. Verify the user row is anonymized in place + email replaced.
	var (
		dbEmail    string
		dbFirst    string
		dbLast     string
		hashed     string
		dbDeleted  sql.NullTime
	)
	err = db.QueryRow(`SELECT email, first_name, last_name, hashed_password, deleted_at FROM users WHERE id = $1`, uid).
		Scan(&dbEmail, &dbFirst, &dbLast, &hashed, &dbDeleted)
	require.NoError(t, err)
	assert.NotEqual(t, email, dbEmail, "email must be anonymized")
	assert.Contains(t, dbEmail, "anonymized+", "email anonymized form expected")
	assert.Equal(t, "anonymized", dbFirst)
	assert.Equal(t, "user", dbLast)
	assert.Equal(t, "!ANONYMIZED!", hashed)

	// 7. Verify audit_log metadata is anonymized.
	var meta []byte
	var ip sql.NullString
	err = db.QueryRow(`SELECT metadata, ip_address::text FROM audit_logs WHERE user_id = $1 LIMIT 1`, uid).
		Scan(&meta, &ip)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(meta, &parsed))
	assert.Nil(t, parsed["email"], "email must be removed from metadata")
	assert.Nil(t, parsed["actor_email"], "actor_email must be removed")
	assert.Nil(t, parsed["actor_name"], "actor_name must be removed")
	assert.NotNil(t, parsed["actor_email_hash"], "actor_email_hash must be present")
	assert.IsType(t, "", parsed["actor_email_hash"])
	assert.Len(t, parsed["actor_email_hash"], 64, "sha256 hex is 64 chars")
	assert.NotNil(t, parsed["anonymized_at"])
	if ip.Valid {
		// network() of 203.0.113.42/16 → 203.0.0.0/16. The test
		// checks the host portion is zeroed without depending on
		// PostgreSQL's exact serialization of inet_to_text.
		assert.Contains(t, ip.String, "203.0.0.0/16", "IP should be masked to /16")
	}
}

// TestGDPR_Lifecycle_CancelRace verifies that a CancelDeletion that
// lands between ListPurgeable and PurgeUser is honored: the purge
// re-checks deleted_at IS NOT NULL inside the tx and skips when it
// finds NULL.
func TestGDPR_Lifecycle_CancelRace(t *testing.T) {
	db := gdprTestDB(t)
	uid, _ := seedUser(t, db, "cancel-race")

	repo := postgres.NewGDPRRepository(db)
	salt := "test-gdpr-salt-stable-2"
	old := time.Now().Add(-31 * 24 * time.Hour)
	_, err := db.Exec(`UPDATE users SET deleted_at = $1 WHERE id = $2`, old, uid)
	require.NoError(t, err)

	// Simulate cancel by clearing deleted_at right before purge.
	cancelled, err := repo.CancelDeletion(context.Background(), uid)
	require.NoError(t, err)
	assert.True(t, cancelled)

	// Purge should now skip — deleted_at is NULL again.
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	ok, err := repo.PurgeUser(context.Background(), uid, cutoff, salt)
	require.NoError(t, err)
	assert.False(t, ok, "purge must skip when cancel won the race")

	// Verify user row still has the original email (NOT anonymized).
	var deletedAt sql.NullTime
	err = db.QueryRow(`SELECT deleted_at FROM users WHERE id = $1`, uid).Scan(&deletedAt)
	require.NoError(t, err)
	assert.False(t, deletedAt.Valid, "deleted_at must remain NULL after cancel")
}

// TestGDPR_Lifecycle_OwnerBlocked covers Decision 6: a user who owns
// an organization with active members cannot delete their account.
func TestGDPR_Lifecycle_OwnerBlocked(t *testing.T) {
	db := gdprTestDB(t)

	owner, _ := seedUser(t, db, "owner")
	member, _ := seedUser(t, db, "member")

	// Build an organization with the seeded owner + a second member.
	orgID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, 'agency', 'Acme Test')`, orgID, owner)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id IN ($2, $3)`, orgID, owner, member)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO organization_members (id, organization_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, 'owner', NOW()),
		       ($4, $2, $5, 'admin', NOW())`,
		uuid.New(), orgID, owner, uuid.New(), member)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM organization_members WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`UPDATE users SET organization_id = NULL WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := postgres.NewGDPRRepository(db)
	blocked, err := repo.FindOwnedOrgsBlockingDeletion(context.Background(), owner)
	require.NoError(t, err)
	require.NotEmpty(t, blocked, "owner with member should be blocked")
	assert.Equal(t, "Acme Test", blocked[0].OrgName)
	assert.Equal(t, 2, blocked[0].MemberCount, "two members in the org")
	assert.Contains(t, blocked[0].Actions, domaingdpr.ActionTransferOwnership)
	assert.Contains(t, blocked[0].Actions, domaingdpr.ActionDissolveOrg)
}

// TestGDPR_Lifecycle_ExportLoadsProfile verifies LoadExport returns a
// non-empty Profile section for a fresh user. Other sections may be
// empty (the user has no proposals yet) but profile MUST have one row.
func TestGDPR_Lifecycle_ExportLoadsProfile(t *testing.T) {
	db := gdprTestDB(t)
	uid, email := seedUser(t, db, "export")

	repo := postgres.NewGDPRRepository(db)
	exp, err := repo.LoadExport(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, exp.Profile, 1, "profile section must have one row")
	assert.Equal(t, email, exp.Email)
	assert.Equal(t, email, exp.Profile[0]["email"])
}

// TestGDPR_PurgeOnce_BatchHappyPath verifies the service-level
// PurgeOnce: list batch, purge each, return counts. Uses two seeded
// users to assert batching works.
func TestGDPR_PurgeOnce_BatchHappyPath(t *testing.T) {
	db := gdprTestDB(t)
	repo := postgres.NewGDPRRepository(db)

	u1, _ := seedUser(t, db, "batch-1")
	u2, _ := seedUser(t, db, "batch-2")

	old := time.Now().Add(-31 * 24 * time.Hour)
	_, err := db.Exec(`UPDATE users SET deleted_at = $1 WHERE id IN ($2, $3)`, old, u1, u2)
	require.NoError(t, err)

	svc := gdprapp.NewService(gdprapp.ServiceDeps{
		Repo:        repo,
		Users:       nil, // PurgeOnce only uses repo
		Hasher:      nil,
		Email:       nil,
		Signer:      nil,
		FrontendURL: "https://app.test",
	})

	res, err := svc.PurgeOnce(context.Background(), "test-salt-batch", 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, res.Examined, 2)
	assert.GreaterOrEqual(t, res.Purged, 2)
	assert.Empty(t, res.Errors)
}
