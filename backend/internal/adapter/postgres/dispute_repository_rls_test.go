package postgres_test

// Tests for the RLS tenant-context wrap on DisputeRepository.
//
// BUG-NEW-04 path 6/8 — disputes. The disputes table is RLS-protected
// by migration 125 with the policy
//
//   USING (
//     client_organization_id   = current_setting('app.current_org_id', true)::uuid
//     OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
//   )
//
// Two-sided ownership — same pattern as proposals. Sub-tables
// (dispute_evidence, dispute_counter_proposals, dispute_ai_chat_messages)
// are NOT directly in the migration 125 RLS scope, so this commit
// wraps only the disputes table reads/writes.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/dispute"
)

func TestDisputeRepository_WithTxRunner_ReturnsSameRepo(t *testing.T) {
	repo := postgres.NewDisputeRepository(nil)
	runner := postgres.NewTxRunner(nil)
	got := repo.WithTxRunner(runner)
	assert.Same(t, repo, got)
}

func TestDisputeRepository_WithTxRunner_NilRunner_NoPanic(t *testing.T) {
	repo := postgres.NewDisputeRepository(nil)
	got := repo.WithTxRunner(nil)
	assert.NotNil(t, got)
}

// TestDisputeRepository_Create_UnderRLS verifies the INSERT lands
// when wrapped in tenant tx using one of the dispute's stakeholder
// orgs.
func TestDisputeRepository_Create_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	// Need a parent proposal + milestone for FK constraints.
	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	milestoneID := insertMilestone(t, db, p.ID)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, milestoneID) })

	disputeRepo := postgres.NewDisputeRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	now := time.Now()
	d := &dispute.Dispute{
		ID:                     uuid.New(),
		ProposalID:             p.ID,
		MilestoneID:            milestoneID,
		ConversationID:         convID,
		InitiatorID:            clientUserID,
		RespondentID:           providerUserID,
		ClientID:               clientUserID,
		ProviderID:             providerUserID,
		ClientOrganizationID:   clientOrgID,
		ProviderOrganizationID: providerOrgID,
		Reason:                 dispute.Reason("quality_issue"),
		Description:            "test dispute for BUG-NEW-04 path 6/8",
		RequestedAmount:        2500,
		ProposalAmount:         5000,
		Status:                 dispute.Status("open"),
		LastActivityAt:         now,
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	require.NoError(t, disputeRepo.Create(ctx, d),
		"Create with tenant context derived from dispute payload must succeed under RLS")

	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM disputes WHERE id = $1`, d.ID) })

	// Read back via tenant-aware path.
	got, err := disputeRepo.GetByIDForOrg(ctx, d.ID, clientOrgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, d.ID, got.ID)
}

// TestDisputeRepository_GetByIDForOrg_UnderRLS confirms both stakeholder
// orgs (client + provider) can read the dispute under the tenant wrap.
func TestDisputeRepository_GetByIDForOrg_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	milestoneID := insertMilestone(t, db, p.ID)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, milestoneID) })

	disputeRepo := postgres.NewDisputeRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	now := time.Now()
	d := &dispute.Dispute{
		ID:                     uuid.New(),
		ProposalID:             p.ID,
		MilestoneID:            milestoneID,
		ConversationID:         convID,
		InitiatorID:            clientUserID,
		RespondentID:           providerUserID,
		ClientID:               clientUserID,
		ProviderID:             providerUserID,
		ClientOrganizationID:   clientOrgID,
		ProviderOrganizationID: providerOrgID,
		Reason:                 dispute.Reason("delay"),
		Description:            "two-sided read test",
		RequestedAmount:        500,
		ProposalAmount:         5000,
		Status:                 dispute.Status("open"),
		LastActivityAt:         now,
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	require.NoError(t, disputeRepo.Create(ctx, d))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM disputes WHERE id = $1`, d.ID) })

	// Client side reads.
	gotClient, err := disputeRepo.GetByIDForOrg(ctx, d.ID, clientOrgID)
	require.NoError(t, err)
	require.NotNil(t, gotClient)

	// Provider side also sees the dispute.
	gotProvider, err := disputeRepo.GetByIDForOrg(ctx, d.ID, providerOrgID)
	require.NoError(t, err)
	require.NotNil(t, gotProvider)
}

// TestDisputeRepository_ListByOrganization_UnderRLS verifies the list
// pagination passes the policy.
func TestDisputeRepository_ListByOrganization_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	milestoneID := insertMilestone(t, db, p.ID)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, milestoneID) })

	disputeRepo := postgres.NewDisputeRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	now := time.Now()
	d := &dispute.Dispute{
		ID:                     uuid.New(),
		ProposalID:             p.ID,
		MilestoneID:            milestoneID,
		ConversationID:         convID,
		InitiatorID:            clientUserID,
		RespondentID:           providerUserID,
		ClientID:               clientUserID,
		ProviderID:             providerUserID,
		ClientOrganizationID:   clientOrgID,
		ProviderOrganizationID: providerOrgID,
		Reason:                 dispute.Reason("delay"),
		Description:            "list test",
		RequestedAmount:        100,
		ProposalAmount:         5000,
		Status:                 dispute.Status("open"),
		LastActivityAt:         now,
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	require.NoError(t, disputeRepo.Create(ctx, d))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM disputes WHERE id = $1`, d.ID) })

	got, _, err := disputeRepo.ListByOrganization(ctx, clientOrgID, "", 10)
	require.NoError(t, err)
	require.NotEmpty(t, got, "ListByOrganization under tenant context must return the row")
}

// TestDisputeRepository_Legacy_NoTxRunner_StillWorks confirms the
// *sql.DB-only path stays valid.
func TestDisputeRepository_Legacy_NoTxRunner_StillWorks(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	proposalRepo := postgres.NewProposalRepository(db) // legacy
	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	milestoneID := insertMilestone(t, db, p.ID)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, milestoneID) })

	disputeRepo := postgres.NewDisputeRepository(db) // legacy
	now := time.Now()
	d := &dispute.Dispute{
		ID:                     uuid.New(),
		ProposalID:             p.ID,
		MilestoneID:            milestoneID,
		ConversationID:         convID,
		InitiatorID:            clientUserID,
		RespondentID:           providerUserID,
		ClientID:               clientUserID,
		ProviderID:             providerUserID,
		ClientOrganizationID:   clientOrgID,
		ProviderOrganizationID: providerOrgID,
		Reason:                 dispute.Reason("delay"),
		Description:            "legacy test",
		RequestedAmount:        100,
		ProposalAmount:         5000,
		Status:                 dispute.Status("open"),
		LastActivityAt:         now,
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	require.NoError(t, disputeRepo.Create(ctx, d),
		"legacy path must keep working for unit tests with only *sql.DB")
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM disputes WHERE id = $1`, d.ID) })
}
