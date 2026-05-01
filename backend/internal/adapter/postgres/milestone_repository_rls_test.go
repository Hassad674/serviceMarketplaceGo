package postgres_test

// Tests for the RLS tenant-context wrap on MilestoneRepository.
//
// BUG-NEW-04 path 5/8 — proposal_milestones. The milestones inherit
// security from the parent proposal via the policy
//
//   USING (EXISTS (
//     SELECT 1 FROM proposals p
//     WHERE p.id = proposal_milestones.proposal_id
//       AND (p.client_organization_id   = current_setting('app.current_org_id', true)::uuid
//         OR p.provider_organization_id = current_setting('app.current_org_id', true)::uuid)
//   ))
//
// Strategy mirrors the proposal repo: every milestone operation looks
// up the parent proposal's stakeholder orgs via the legacy db
// connection (single-row SELECT — works because we always have the
// proposal id from a prior read), then opens the tenant tx with the
// client side org and runs the SQL.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/milestone"
)

func TestMilestoneRepository_WithTxRunner_ReturnsSameRepo(t *testing.T) {
	repo := postgres.NewMilestoneRepository(nil)
	runner := postgres.NewTxRunner(nil)
	got := repo.WithTxRunner(runner)
	assert.Same(t, repo, got)
}

func TestMilestoneRepository_WithTxRunner_NilRunner_NoPanic(t *testing.T) {
	repo := postgres.NewMilestoneRepository(nil)
	got := repo.WithTxRunner(nil)
	assert.NotNil(t, got)
}

// TestMilestoneRepository_CreateBatch_UnderRLS_Succeeds: the milestone
// rows must land under the non-superuser role + RLS, with the parent
// proposal's stakeholder org installed as app.current_org_id.
func TestMilestoneRepository_CreateBatch_UnderRLS_Succeeds(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	milestoneRepo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	now := time.Now()
	ms := []*milestone.Milestone{
		{
			ID:          uuid.New(),
			ProposalID:  p.ID,
			Sequence:    1,
			Title:       "Phase 1",
			Description: "First phase",
			Amount:      5000,
			Status:      milestone.StatusPendingFunding,
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			ProposalID:  p.ID,
			Sequence:    2,
			Title:       "Phase 2",
			Description: "Second phase",
			Amount:      5000,
			Status:      milestone.StatusPendingFunding,
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	require.NoError(t, milestoneRepo.CreateBatch(ctx, ms),
		"CreateBatch with tenant context derived from parent proposal must succeed under RLS")

	t.Cleanup(func() {
		for _, m := range ms {
			_, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, m.ID)
		}
	})

	// Read back via tenant-aware path.
	got, err := milestoneRepo.ListByProposalForOrg(ctx, p.ID, clientOrgID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

// TestMilestoneRepository_GetByIDForOrg_UnderRLS asserts the read path
// passes the policy when wrapped with the caller's org.
func TestMilestoneRepository_GetByIDForOrg_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	milestoneRepo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	now := time.Now()
	m := &milestone.Milestone{
		ID:          uuid.New(),
		ProposalID:  p.ID,
		Sequence:    1,
		Title:       "Phase 1",
		Description: "First phase",
		Amount:      5000,
		Status:      milestone.StatusPendingFunding,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, milestoneRepo.CreateBatch(ctx, []*milestone.Milestone{m}))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, m.ID) })

	got, err := milestoneRepo.GetByIDForOrg(ctx, m.ID, clientOrgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, m.ID, got.ID)

	// Provider side also sees the row.
	gotProv, err := milestoneRepo.GetByIDForOrg(ctx, m.ID, providerOrgID)
	require.NoError(t, err)
	require.NotNil(t, gotProv)
}

// TestMilestoneRepository_Legacy_NoTxRunner_StillWorks confirms backward
// compat via the *sql.DB-only constructor.
func TestMilestoneRepository_Legacy_NoTxRunner_StillWorks(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	proposalRepo := postgres.NewProposalRepository(db) // legacy
	milestoneRepo := postgres.NewMilestoneRepository(db) // legacy

	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	now := time.Now()
	m := &milestone.Milestone{
		ID:          uuid.New(),
		ProposalID:  p.ID,
		Sequence:    1,
		Title:       "Phase 1",
		Description: "First phase",
		Amount:      5000,
		Status:      milestone.StatusPendingFunding,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, milestoneRepo.CreateBatch(ctx, []*milestone.Milestone{m}),
		"legacy path must keep working for unit tests with only *sql.DB")
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, m.ID) })
}
