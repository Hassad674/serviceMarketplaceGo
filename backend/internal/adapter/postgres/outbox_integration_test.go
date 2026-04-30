package postgres_test

// Integration tests for the BUG-05 outbox guarantee: a failed
// pending_events INSERT inside the same transaction as a profile
// UPDATE must roll back BOTH writes. The pre-fix code did the
// schedule hors-tx and could leave the profile updated in Postgres
// while the search.reindex event vanished — producing permanent
// drift between Postgres and Typesense.
//
// Gated behind MARKETPLACE_TEST_DATABASE_URL like every other
// integration test. Skips on a fresh checkout.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/domain/profile"
)

// freshFreelanceOrg returns an organization id that already has a
// freelance_profiles row — the integration test can then call the
// tx-aware UpdateCoreTx and assert on commit / rollback semantics.
func freshFreelanceOrg(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	// Create a user → org → freelance profile chain. We rely on the
	// freelance repo's lazy GetOrCreateByOrgID path so the test does
	// not need to know the freelance_profiles defaults inline.
	userID := insertTestUser(t, db)
	orgID := insertTestOrgForUser(t, db, userID)

	repo := postgres.NewFreelanceProfileRepository(db)
	_, err := repo.GetOrCreateByOrgID(ctx, orgID)
	require.NoError(t, err)
	return orgID
}

// insertTestOrgForUser creates a minimal organization row owned by
// the given user. Cleanup cascades through the foreign keys.
func insertTestOrgForUser(t *testing.T, db *sql.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, 'provider_personal', 'Test Freelance Org')`,
		orgID, userID,
	)
	require.NoError(t, err, "insert test organization")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM freelance_profiles WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
	})
	return orgID
}

// fetchTitle reads the freelance_profiles.title column out-of-band
// so the assertion sees the committed (or pre-tx) value, not whatever
// the tx tried to write.
func fetchTitle(t *testing.T, db *sql.DB, orgID uuid.UUID) string {
	t.Helper()
	var got string
	err := db.QueryRow(
		`SELECT title FROM freelance_profiles WHERE organization_id = $1`,
		orgID,
	).Scan(&got)
	require.NoError(t, err)
	return got
}

// countPendingForOrg counts search.reindex rows for an org. A
// committed tx adds one; a rolled-back tx adds zero.
func countPendingForOrg(t *testing.T, db *sql.DB, orgID uuid.UUID) int {
	t.Helper()
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM pending_events
		 WHERE event_type = 'search.reindex'
		   AND payload::text LIKE '%' || $1::text || '%'`,
		orgID.String(),
	).Scan(&count)
	require.NoError(t, err)
	return count
}

func TestOutbox_FreelanceUpdateCore_CommitsBothRowsAtomically(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	orgID := freshFreelanceOrg(t, db)

	repo := postgres.NewFreelanceProfileRepository(db)
	eventsRepo := postgres.NewPendingEventRepository(db)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	pendingBefore := countPendingForOrg(t, db, orgID)

	err := runner.RunInTx(ctx, func(tx *sql.Tx) error {
		if err := repo.UpdateCoreTx(ctx, tx, orgID, "Atomic Title", "About", ""); err != nil {
			return err
		}
		// Insert a sibling search.reindex event in the same tx.
		ev, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
			EventType: pendingevent.TypeSearchReindex,
			Payload:   []byte(`{"organization_id":"` + orgID.String() + `","persona":"freelance"}`),
			FiresAt:   time.Now(),
		})
		if err != nil {
			return err
		}
		return eventsRepo.ScheduleTx(ctx, tx, ev)
	})
	require.NoError(t, err)

	// BOTH writes must be visible.
	assert.Equal(t, "Atomic Title", fetchTitle(t, db, orgID),
		"the freelance UPDATE must have committed")
	assert.Equal(t, pendingBefore+1, countPendingForOrg(t, db, orgID),
		"the search.reindex event must have committed")
}

func TestOutbox_FreelanceUpdateCore_RollsBackBothOnEventError(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	orgID := freshFreelanceOrg(t, db)

	repo := postgres.NewFreelanceProfileRepository(db)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	titleBefore := fetchTitle(t, db, orgID)
	pendingBefore := countPendingForOrg(t, db, orgID)

	err := runner.RunInTx(ctx, func(tx *sql.Tx) error {
		if err := repo.UpdateCoreTx(ctx, tx, orgID, "Should Not Persist", "X", ""); err != nil {
			return err
		}
		// Simulate the publisher failing to schedule — must roll
		// back the profile UPDATE too.
		return errors.New("simulated typesense outbox failure")
	})
	require.Error(t, err)

	// Neither side wrote — this is the BUG-05 fix.
	assert.Equal(t, titleBefore, fetchTitle(t, db, orgID),
		"a rolled-back tx must NOT mutate freelance_profiles.title")
	assert.Equal(t, pendingBefore, countPendingForOrg(t, db, orgID),
		"a rolled-back tx must NOT add a pending_events row")
}

func TestOutbox_FreelanceUpdateAvailability_RollsBackBothOnEventError(t *testing.T) {
	db := testDB(t)
	cleanupPendingEvents(t)
	orgID := freshFreelanceOrg(t, db)

	repo := postgres.NewFreelanceProfileRepository(db)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	view, err := repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	availabilityBefore := view.Profile.AvailabilityStatus
	pendingBefore := countPendingForOrg(t, db, orgID)

	err = runner.RunInTx(ctx, func(tx *sql.Tx) error {
		if err := repo.UpdateAvailabilityTx(ctx, tx, orgID, profile.AvailabilityNot); err != nil {
			return err
		}
		return errors.New("forced rollback")
	})
	require.Error(t, err)

	view, err = repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, availabilityBefore, view.Profile.AvailabilityStatus,
		"availability must NOT change when the outbox tx is rolled back")
	assert.Equal(t, pendingBefore, countPendingForOrg(t, db, orgID))
}

// TestOutbox_FreelanceProfileNotFoundSurfaces verifies the tx-aware
// repo method still propagates the domain ErrProfileNotFound when
// the org has no row. This protects callers that rely on errors.Is
// to detect the not-found case.
func TestOutbox_FreelanceProfileNotFoundSurfaces(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewFreelanceProfileRepository(db)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	err := runner.RunInTx(ctx, func(tx *sql.Tx) error {
		return repo.UpdateCoreTx(ctx, tx, uuid.New(), "X", "Y", "")
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)
}
