package postgres_test

// Integration tests for the application-credit auto-seed and lazy
// weekly refill introduced after the R12 per-user → per-org migration.
//
// These tests talk to a real PostgreSQL and are gated behind the
// MARKETPLACE_TEST_DATABASE_URL environment variable. When the variable
// is absent (the common case on CI and fresh checkouts) the whole suite
// skips — no test ever fails because Docker is not running. To run it
// against the local dev stack, point it at a disposable database:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5434/marketplace_go_test?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestJobCreditRepository -count=1
//
// The suite creates its own users and organizations with random ids,
// and cleans them up in t.Cleanup so reruns stay isolated. It never
// touches any existing rows.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/organization"
)

// testDB returns a live *sql.DB against MARKETPLACE_TEST_DATABASE_URL
// or t.Skip's the whole test when the variable is unset.
func testDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "open test database")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx), "ping test database")

	t.Cleanup(func() { _ = db.Close() })
	return db
}

// insertTestUser creates a minimal user row so the organization's
// owner_user_id FK can be satisfied. Returns the new user id and
// registers a cleanup that cascades the delete through the owned org.
func insertTestUser(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()

	id := uuid.New()
	email := fmt.Sprintf("test-%s@creditsuite.local", id.String()[:8])

	_, err := db.Exec(`
		INSERT INTO users (
			id, email, hashed_password, first_name, last_name,
			display_name, role
		)
		VALUES ($1, $2, 'x', 'Test', 'User', 'Test User', 'agency')`,
		id, email,
	)
	require.NoError(t, err, "insert test user")

	t.Cleanup(func() {
		// Delete in FK order: orgs first (they RESTRICT on owner), then
		// the user. organization_members cascades from organizations.
		_, _ = db.Exec(`DELETE FROM organizations WHERE owner_user_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, id)
	})

	return id
}

// newOrgRepo builds a repository with the real starter-credit quota
// (WeeklyQuota) so auto-seeding can be verified against production
// values. All tests in this file use WeeklyQuota as the floor.
func newOrgRepo(db *sql.DB) *postgres.OrganizationRepository {
	return postgres.NewOrganizationRepository(db, job.WeeklyQuota)
}

// newCreditRepo builds a JobCreditRepository with a custom refill
// period so tests can assert lazy-refill behavior without waiting a
// week. The starter quota always matches production.
func newCreditRepo(db *sql.DB, period time.Duration) *postgres.JobCreditRepository {
	return postgres.NewJobCreditRepository(db, job.WeeklyQuota, period)
}

// createOrg creates a brand-new organization using the real repository
// path (CreateWithOwnerMembership) so it exercises the seeding code.
// Returns the resulting org id.
func createOrg(t *testing.T, repo *postgres.OrganizationRepository, ownerID uuid.UUID) uuid.UUID {
	t.Helper()

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeAgency, "Test Org")
	require.NoError(t, err)

	member, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, repo.CreateWithOwnerMembership(ctx, org, member))

	return org.ID
}

// forceOrgCreditsState bypasses the repository to plant an arbitrary
// (balance, last_reset) pair on an org row so the refill path can be
// tested in isolation. Used only in tests.
func forceOrgCreditsState(t *testing.T, db *sql.DB, orgID uuid.UUID, balance int, lastReset time.Time) {
	t.Helper()

	_, err := db.Exec(`
		UPDATE organizations
		SET    application_credits   = $2,
		       credits_last_reset_at = $3,
		       updated_at            = now()
		WHERE  id = $1`,
		orgID, balance, lastReset,
	)
	require.NoError(t, err, "force credits state")
}

func readOrgCreditsState(t *testing.T, db *sql.DB, orgID uuid.UUID) (balance int, lastReset time.Time) {
	t.Helper()

	err := db.QueryRow(`
		SELECT application_credits, credits_last_reset_at
		FROM   organizations
		WHERE  id = $1`, orgID,
	).Scan(&balance, &lastReset)
	require.NoError(t, err, "read credits state")
	return balance, lastReset
}

// TestJobCreditRepository_AutoSeedOnOrgCreation verifies that a brand
// new organization created through CreateWithOwnerMembership is born
// with WeeklyQuota credits rather than the table DEFAULT of 0.
func TestJobCreditRepository_AutoSeedOnOrgCreation(t *testing.T) {
	db := testDB(t)
	orgRepo := newOrgRepo(db)
	creditRepo := newCreditRepo(db, job.RefillPeriod)

	ownerID := insertTestUser(t, db)
	orgID := createOrg(t, orgRepo, ownerID)

	ctx := context.Background()
	balance, err := creditRepo.GetOrCreate(ctx, orgID)
	require.NoError(t, err)

	assert.Equal(t, job.WeeklyQuota, balance,
		"new org must be seeded with the starter quota at creation time")
}

// TestJobCreditRepository_LazyRefill_TopsUpBelowQuota verifies the
// core refill flow: an org that has dropped below the quota and whose
// cursor is older than the refill period gets floor-bumped back up to
// WeeklyQuota on the next GetOrCreate call.
func TestJobCreditRepository_LazyRefill_TopsUpBelowQuota(t *testing.T) {
	db := testDB(t)
	orgRepo := newOrgRepo(db)

	// 1-second refill period so the test does not have to fake a week.
	creditRepo := newCreditRepo(db, 1*time.Second)

	ownerID := insertTestUser(t, db)
	orgID := createOrg(t, orgRepo, ownerID)

	// Force the pool below quota and age the cursor past the period.
	forceOrgCreditsState(t, db, orgID, 5, time.Now().Add(-10*time.Second))

	before := time.Now()
	balance, err := creditRepo.GetOrCreate(context.Background(), orgID)
	require.NoError(t, err)

	assert.Equal(t, job.WeeklyQuota, balance, "refill must top up to quota")

	_, lastReset := readOrgCreditsState(t, db, orgID)
	assert.WithinDuration(t, before, lastReset, 5*time.Second,
		"refill must advance credits_last_reset_at to ~now")
}

// TestJobCreditRepository_LazyRefill_PreservesBonusCredits verifies
// that an org whose balance is ABOVE the starter quota (typically
// because the proposal fraud flow awarded BonusPerMission credits)
// keeps its balance on refill. The refill is a floor, never a ceiling
// — GREATEST(current, quota) must always be the current value here.
func TestJobCreditRepository_LazyRefill_PreservesBonusCredits(t *testing.T) {
	db := testDB(t)
	orgRepo := newOrgRepo(db)
	creditRepo := newCreditRepo(db, 1*time.Second)

	ownerID := insertTestUser(t, db)
	orgID := createOrg(t, orgRepo, ownerID)

	const bonusBalance = 30
	forceOrgCreditsState(t, db, orgID, bonusBalance, time.Now().Add(-10*time.Second))

	before := time.Now()
	balance, err := creditRepo.GetOrCreate(context.Background(), orgID)
	require.NoError(t, err)

	assert.Equal(t, bonusBalance, balance,
		"refill must not clobber bonus credits accumulated from paid missions")

	_, lastReset := readOrgCreditsState(t, db, orgID)
	assert.WithinDuration(t, before, lastReset, 5*time.Second,
		"refill must still advance credits_last_reset_at to keep the period rolling")
}

// TestJobCreditRepository_LazyRefill_NoopWhenPeriodNotElapsed verifies
// that an org whose cursor is still within the refill period is left
// strictly untouched. Neither the balance nor the timestamp must move.
func TestJobCreditRepository_LazyRefill_NoopWhenPeriodNotElapsed(t *testing.T) {
	db := testDB(t)
	orgRepo := newOrgRepo(db)
	// 1h period — the test fakes a 1-hour-old cursor below.
	creditRepo := newCreditRepo(db, 1*time.Hour)

	ownerID := insertTestUser(t, db)
	orgID := createOrg(t, orgRepo, ownerID)

	// Cursor is ~1 minute old (well inside the 1h window), balance is 5.
	frozenReset := time.Now().Add(-1 * time.Minute).UTC().Truncate(time.Millisecond)
	forceOrgCreditsState(t, db, orgID, 5, frozenReset)

	balance, err := creditRepo.GetOrCreate(context.Background(), orgID)
	require.NoError(t, err)

	assert.Equal(t, 5, balance, "balance must stay untouched inside the refill window")

	newBalance, newReset := readOrgCreditsState(t, db, orgID)
	assert.Equal(t, 5, newBalance)
	assert.WithinDuration(t, frozenReset, newReset, time.Millisecond,
		"credits_last_reset_at must not be advanced inside the refill window")
}

// TestJobCreditRepository_LazyRefill_ConcurrentReadsAreSafe verifies
// that N racing GetOrCreate calls on an org due for refill never
// stack-add — the balance must end at exactly WeeklyQuota, not
// WeeklyQuota * N. The refill is a single atomic SQL statement so
// only one of the concurrent updates can satisfy the WHERE clause.
func TestJobCreditRepository_LazyRefill_ConcurrentReadsAreSafe(t *testing.T) {
	db := testDB(t)
	orgRepo := newOrgRepo(db)
	creditRepo := newCreditRepo(db, 1*time.Second)

	ownerID := insertTestUser(t, db)
	orgID := createOrg(t, orgRepo, ownerID)

	// Balance is below quota and the cursor is aged out.
	forceOrgCreditsState(t, db, orgID, 3, time.Now().Add(-10*time.Second))

	const racers = 10
	var wg sync.WaitGroup
	wg.Add(racers)
	results := make([]int, racers)
	errs := make([]error, racers)

	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx], errs[idx] = creditRepo.GetOrCreate(context.Background(), orgID)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "racer %d", i)
	}

	finalBalance, _ := readOrgCreditsState(t, db, orgID)
	assert.Equal(t, job.WeeklyQuota, finalBalance,
		"concurrent refills must converge to exactly the starter quota, not stack-add")
}
