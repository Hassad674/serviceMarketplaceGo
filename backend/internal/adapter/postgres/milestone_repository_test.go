package postgres_test

// Integration tests for MilestoneRepository (migration 084-085 schema).
// Gated behind MARKETPLACE_TEST_DATABASE_URL — auto-skip when unset.
//
// Run against the isolated milestones DB copy:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_milestones?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestMilestoneRepository -count=1

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/milestone"
)

// newTestProposal inserts a minimal conversation + proposal pair so
// milestones have a valid FK target. Returns the proposal id.
//
// It does NOT use the ProposalRepository because that would drag in
// amount/status defaults we don't need for this suite. We insert with
// raw SQL and register a cleanup that cascades through milestones.
func newTestProposal(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()

	clientID := insertTestUserWithRole(t, db, "enterprise")
	providerID := insertTestUserWithRole(t, db, "provider")

	convID := uuid.New()
	_, err := db.Exec(`INSERT INTO conversations (id, created_at, updated_at) VALUES ($1, now(), now())`, convID)
	require.NoError(t, err, "insert conversation")

	proposalID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO proposals (
			id, conversation_id, sender_id, recipient_id, title, description,
			amount, status, parent_id, version, client_id, provider_id,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, 'Test Proposal', 'desc',
			10000, 'pending', NULL, 1, $5, $6, now(), now())`,
		proposalID, convID, clientID, providerID, clientID, providerID,
	)
	require.NoError(t, err, "insert proposal")

	t.Cleanup(func() {
		// Cleanup order: milestones -> proposal -> conversation -> users.
		_, _ = db.Exec(`DELETE FROM milestone_deliverables WHERE milestone_id IN (SELECT id FROM proposal_milestones WHERE proposal_id = $1)`, proposalID)
		_, _ = db.Exec(`DELETE FROM proposal_milestones WHERE proposal_id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	return proposalID
}

// insertTestUserWithRole is a milestone-suite local helper that inserts a
// user with a specific role (enterprise/provider/agency). We can't reuse
// the shared insertTestUser because it hardcodes 'agency'.
func insertTestUserWithRole(t *testing.T, db *sql.DB, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := id.String()[:8] + "@milestones.local"
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role)
		VALUES ($1, $2, 'x', 'Test', 'User', 'Test User', $3)`,
		id, email, role,
	)
	require.NoError(t, err, "insert user")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM organizations WHERE owner_user_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func TestMilestoneRepository_CreateBatchAndList(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	inputs := []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "Design", Description: "wireframes", Amount: 30000},
		{Sequence: 2, Title: "Build", Description: "dev", Amount: 80000},
		{Sequence: 3, Title: "Launch", Description: "deploy", Amount: 20000},
	}
	batch, err := milestone.NewMilestoneBatch(proposalID, inputs)
	require.NoError(t, err)

	require.NoError(t, repo.CreateBatch(ctx, batch))

	got, err := repo.ListByProposal(ctx, proposalID)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// Ordered by sequence ASC.
	assert.Equal(t, 1, got[0].Sequence)
	assert.Equal(t, "Design", got[0].Title)
	assert.EqualValues(t, 30000, got[0].Amount)
	assert.Equal(t, milestone.StatusPendingFunding, got[0].Status)
	assert.Equal(t, 0, got[0].Version)

	assert.Equal(t, 2, got[1].Sequence)
	assert.Equal(t, 3, got[2].Sequence)

	// Sum matches.
	assert.EqualValues(t, 130000, milestone.SumAmount(got))
}

func TestMilestoneRepository_CreateBatch_SequenceUnique(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	first, err := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "A", Description: "a", Amount: 1000},
	})
	require.NoError(t, err)
	require.NoError(t, repo.CreateBatch(ctx, first))

	// Trying to insert another milestone with sequence=1 on the same
	// proposal must fail on the UNIQUE(proposal_id, sequence) constraint.
	second, err := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "B", Description: "b", Amount: 2000},
	})
	require.NoError(t, err)
	err = repo.CreateBatch(ctx, second)
	require.Error(t, err)
}

func TestMilestoneRepository_GetByID(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	batch, _ := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "Only", Description: "d", Amount: 5000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batch))

	got, err := repo.GetByID(ctx, batch[0].ID)
	require.NoError(t, err)
	assert.Equal(t, batch[0].ID, got.ID)
	assert.Equal(t, milestone.StatusPendingFunding, got.Status)
	assert.Equal(t, 0, got.Version)
}

func TestMilestoneRepository_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, milestone.ErrMilestoneNotFound)
}

func TestMilestoneRepository_Update_HappyPath(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	batch, _ := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "Phase 1", Description: "d", Amount: 20000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batch))

	// Fetch for update, transition to funded, update.
	m, err := repo.GetByIDForUpdate(ctx, batch[0].ID)
	require.NoError(t, err)
	require.NoError(t, m.Fund())
	require.NoError(t, repo.Update(ctx, m))

	// In-memory version should now be 1 (bumped by Update).
	assert.Equal(t, 1, m.Version)

	// Re-read and assert persistent state.
	reloaded, err := repo.GetByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, milestone.StatusFunded, reloaded.Status)
	assert.Equal(t, 1, reloaded.Version)
	assert.NotNil(t, reloaded.FundedAt)
}

func TestMilestoneRepository_Update_OptimisticConflict(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	batch, _ := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "Phase 1", Description: "d", Amount: 20000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batch))

	// Two concurrent readers observe version=0.
	a, err := repo.GetByID(ctx, batch[0].ID)
	require.NoError(t, err)
	b, err := repo.GetByID(ctx, batch[0].ID)
	require.NoError(t, err)

	// Writer A commits a Fund() transition first.
	require.NoError(t, a.Fund())
	require.NoError(t, repo.Update(ctx, a))

	// Writer B now tries to commit with its stale version=0. This must
	// fail with ErrConcurrentUpdate — B's view of the world is obsolete
	// and the app layer should retry after refetching.
	require.NoError(t, b.Fund())
	err = repo.Update(ctx, b)
	require.Error(t, err)
	assert.True(t, errors.Is(err, milestone.ErrConcurrentUpdate),
		"expected ErrConcurrentUpdate, got %v", err)
}

func TestMilestoneRepository_GetCurrentActive(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	batch, _ := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "M1", Description: "d", Amount: 10000},
		{Sequence: 2, Title: "M2", Description: "d", Amount: 20000},
		{Sequence: 3, Title: "M3", Description: "d", Amount: 30000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batch))

	// Initially the first milestone is active.
	current, err := repo.GetCurrentActive(ctx, proposalID)
	require.NoError(t, err)
	assert.Equal(t, 1, current.Sequence)

	// Walk M1 to released, then M2 must become the current active.
	m1 := batch[0]
	m1State, err := repo.GetByID(ctx, m1.ID)
	require.NoError(t, err)
	require.NoError(t, m1State.Fund())
	require.NoError(t, repo.Update(ctx, m1State))
	require.NoError(t, m1State.Submit())
	require.NoError(t, repo.Update(ctx, m1State))
	require.NoError(t, m1State.Approve())
	require.NoError(t, repo.Update(ctx, m1State))
	require.NoError(t, m1State.Release())
	require.NoError(t, repo.Update(ctx, m1State))

	current, err = repo.GetCurrentActive(ctx, proposalID)
	require.NoError(t, err)
	assert.Equal(t, 2, current.Sequence, "after releasing M1, M2 becomes the current active milestone")

	// Cancel M2, M3 must become active.
	m2State, err := repo.GetByID(ctx, batch[1].ID)
	require.NoError(t, err)
	require.NoError(t, m2State.Cancel())
	require.NoError(t, repo.Update(ctx, m2State))

	current, err = repo.GetCurrentActive(ctx, proposalID)
	require.NoError(t, err)
	assert.Equal(t, 3, current.Sequence, "M3 becomes current after M1 released and M2 cancelled")
}

func TestMilestoneRepository_GetCurrentActive_AllTerminal(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	batch, _ := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "M1", Description: "d", Amount: 10000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batch))

	// Cancel the only milestone.
	m, err := repo.GetByID(ctx, batch[0].ID)
	require.NoError(t, err)
	require.NoError(t, m.Cancel())
	require.NoError(t, repo.Update(ctx, m))

	_, err = repo.GetCurrentActive(ctx, proposalID)
	require.ErrorIs(t, err, milestone.ErrMilestoneNotFound,
		"proposal with no active milestones must return ErrMilestoneNotFound")
}

func TestMilestoneRepository_ListByProposals_Batch(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()

	pA := newTestProposal(t, db)
	pB := newTestProposal(t, db)

	batchA, _ := milestone.NewMilestoneBatch(pA, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "A1", Description: "d", Amount: 1000},
		{Sequence: 2, Title: "A2", Description: "d", Amount: 2000},
	})
	batchB, _ := milestone.NewMilestoneBatch(pB, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "B1", Description: "d", Amount: 5000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batchA))
	require.NoError(t, repo.CreateBatch(ctx, batchB))

	grouped, err := repo.ListByProposals(ctx, []uuid.UUID{pA, pB})
	require.NoError(t, err)
	require.Len(t, grouped, 2)
	assert.Len(t, grouped[pA], 2)
	assert.Len(t, grouped[pB], 1)
	assert.Equal(t, "A1", grouped[pA][0].Title)
	assert.Equal(t, "A2", grouped[pA][1].Title)
	assert.Equal(t, "B1", grouped[pB][0].Title)
}

func TestMilestoneRepository_ListByProposals_Empty(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	got, err := repo.ListByProposals(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestMilestoneRepository_Deliverables_CRUD(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)
	uploaderID := insertTestUserWithRole(t, db, "agency")

	batch, _ := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "P1", Description: "d", Amount: 5000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batch))
	mID := batch[0].ID

	// Add two deliverables.
	d1, err := milestone.NewDeliverable(milestone.NewDeliverableInput{
		MilestoneID: mID,
		Filename:    "brief.pdf",
		URL:         "https://cdn.example.com/brief.pdf",
		Size:        4096,
		MimeType:    "application/pdf",
		UploadedBy:  uploaderID,
	})
	require.NoError(t, err)
	require.NoError(t, repo.CreateDeliverable(ctx, d1))

	// Small artificial delay so created_at ordering is stable on fast machines.
	time.Sleep(2 * time.Millisecond)

	d2, err := milestone.NewDeliverable(milestone.NewDeliverableInput{
		MilestoneID: mID,
		Filename:    "reference.png",
		URL:         "https://cdn.example.com/ref.png",
		Size:        8192,
		MimeType:    "image/png",
		UploadedBy:  uploaderID,
	})
	require.NoError(t, err)
	require.NoError(t, repo.CreateDeliverable(ctx, d2))

	got, err := repo.ListDeliverables(ctx, mID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Ordered by created_at ASC.
	assert.Equal(t, "brief.pdf", got[0].Filename)
	assert.Equal(t, "reference.png", got[1].Filename)
	assert.Equal(t, uploaderID, got[0].UploadedBy)

	// Delete one, list should now return one.
	require.NoError(t, repo.DeleteDeliverable(ctx, d1.ID))
	got, err = repo.ListDeliverables(ctx, mID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "reference.png", got[0].Filename)

	// Delete the same one twice -> ErrDeliverableNotFound.
	err = repo.DeleteDeliverable(ctx, d1.ID)
	require.ErrorIs(t, err, milestone.ErrDeliverableNotFound)
}

func TestMilestoneRepository_DisputeLifecycle(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()
	proposalID := newTestProposal(t, db)

	batch, _ := milestone.NewMilestoneBatch(proposalID, []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "P1", Description: "d", Amount: 5000},
	})
	require.NoError(t, repo.CreateBatch(ctx, batch))

	// Walk to funded state.
	m, err := repo.GetByID(ctx, batch[0].ID)
	require.NoError(t, err)
	require.NoError(t, m.Fund())
	require.NoError(t, repo.Update(ctx, m))

	// Open a dispute.
	disputeID := uuid.New()
	require.NoError(t, m.OpenDispute(disputeID))
	require.NoError(t, repo.Update(ctx, m))

	reloaded, err := repo.GetByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, milestone.StatusDisputed, reloaded.Status)
	assert.NotNil(t, reloaded.ActiveDisputeID)
	assert.Equal(t, disputeID, *reloaded.ActiveDisputeID)
	assert.NotNil(t, reloaded.LastDisputeID)

	// Restore the dispute: back to funded.
	require.NoError(t, reloaded.RestoreFromDispute(milestone.StatusFunded))
	require.NoError(t, repo.Update(ctx, reloaded))

	final, err := repo.GetByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, milestone.StatusFunded, final.Status)
	assert.Nil(t, final.ActiveDisputeID, "ActiveDisputeID must be cleared after restore")
	assert.NotNil(t, final.LastDisputeID, "LastDisputeID must be preserved for history")
}
