package postgres_test

// Integration tests for the client-profile feature (migrations 114,
// 115, 116):
//
//   - ProfileRepository.UpdateClientDescription (migration 114)
//   - ProposalRepository.SumPaidByClientOrganization (migration 115)
//   - ProposalRepository.ListCompletedByClientOrganization (migration 115)
//   - New client_organization_id / provider_organization_id denorm
//     columns populated on INSERT by the updated queryInsertProposal.
//
// Gated behind MARKETPLACE_TEST_DATABASE_URL — auto-skip when unset.

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/proposal"
)

// insertTestUserWithOrg creates a user and an enterprise org + a
// member row, returning (userID, orgID). The user is bound to the
// org via users.organization_id so the denorm columns on proposals
// are populated correctly.
func insertTestUserWithOrg(t *testing.T, db *sql.DB, name string) (uuid.UUID, uuid.UUID) {
	t.Helper()

	userID := insertTestUser(t, db)
	org, err := organization.NewOrganization(userID, organization.OrgTypeEnterprise, name)
	require.NoError(t, err)
	member, err := organization.NewMember(org.ID, userID, organization.RoleOwner, "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, newOrgRepo(db).CreateWithOwnerMembership(ctx, org, member))

	// users.organization_id is the R1 source of truth; set it explicitly
	// because NewOrganization does not mutate the users row.
	_, err = db.ExecContext(ctx, `UPDATE users SET organization_id = $1 WHERE id = $2`, org.ID, userID)
	require.NoError(t, err)

	return userID, org.ID
}

// insertConversation creates a minimal conversation row so the
// proposal FK is satisfied. Returns its id.
func insertConversation(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO conversations (id, created_at, updated_at)
		VALUES ($1, NOW(), NOW())`, id)
	require.NoError(t, err, "insert conversation")
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, id) })
	return id
}

// insertProposal inserts a proposal via the real repository so the
// INSERT exercises the updated queryInsertProposal (which now
// populates client_organization_id + provider_organization_id).
func insertProposal(t *testing.T, repo *postgres.ProposalRepository, conversationID, clientID, providerID uuid.UUID, amount int64, status proposal.ProposalStatus) *proposal.Proposal {
	t.Helper()

	now := time.Now().UTC()
	deadline := now.Add(30 * 24 * time.Hour)
	p := &proposal.Proposal{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       clientID,
		RecipientID:    providerID,
		Title:          "Test deal",
		Description:    "client-profile integration test",
		Amount:         amount,
		Deadline:       &deadline,
		Status:         status,
		Version:        1,
		ClientID:       clientID,
		ProviderID:     providerID,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if status == proposal.StatusCompleted {
		p.CompletedAt = &now
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, repo.Create(ctx, p))
	return p
}

func TestProfileRepository_UpdateClientDescription_RoundTrip(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ctx := context.Background()
	require.NoError(t, repo.UpdateClientDescription(ctx, orgID, "We run an awesome client team."))

	p, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "We run an awesome client team.", p.ClientDescription)
}

func TestProfileRepository_UpdateClientDescription_DoesNotClobberOtherFields(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ctx := context.Background()
	// Seed the classic About / Title via Update so we can verify they
	// are untouched by a client description write.
	original, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	original.Title = "Signature title"
	original.About = "Provider-facing about"
	require.NoError(t, repo.Update(ctx, original))

	require.NoError(t, repo.UpdateClientDescription(ctx, orgID, "client text"))

	after, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "Signature title", after.Title)
	assert.Equal(t, "Provider-facing about", after.About)
	assert.Equal(t, "client text", after.ClientDescription)
}

func TestProposalRepository_ClientOrgDenorm_PopulatedOnInsert(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProposalRepository(db)

	clientID, clientOrgID := insertTestUserWithOrg(t, db, "ClientDenormOrg")
	providerID, providerOrgID := insertTestUserWithOrg(t, db, "ProviderDenormOrg")
	convID := insertConversation(t, db)

	p := insertProposal(t, repo, convID, clientID, providerID, 5_000, proposal.StatusPending)

	// Read back the denorm columns directly — the domain does not
	// carry them yet (Option A), so assert at the SQL level.
	var gotClientOrg, gotProviderOrg sql.NullString
	err := db.QueryRow(`SELECT client_organization_id, provider_organization_id FROM proposals WHERE id = $1`, p.ID).
		Scan(&gotClientOrg, &gotProviderOrg)
	require.NoError(t, err)
	require.True(t, gotClientOrg.Valid, "client_organization_id must be populated on insert")
	require.True(t, gotProviderOrg.Valid, "provider_organization_id must be populated on insert")
	assert.Equal(t, clientOrgID.String(), gotClientOrg.String)
	assert.Equal(t, providerOrgID.String(), gotProviderOrg.String)
}

func TestProposalRepository_SumPaidByClientOrganization(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProposalRepository(db)

	clientID, clientOrgID := insertTestUserWithOrg(t, db, "SumPaidClient")
	providerID, _ := insertTestUserWithOrg(t, db, "SumPaidProvider")
	convID := insertConversation(t, db)

	// Paid proposal — counted.
	_ = insertProposal(t, repo, convID, clientID, providerID, 10_000, proposal.StatusPaid)
	// Completed proposal — counted.
	_ = insertProposal(t, repo, convID, clientID, providerID, 2_500, proposal.StatusCompleted)
	// Pending proposal — NOT counted (money has not left the client).
	_ = insertProposal(t, repo, convID, clientID, providerID, 9_999, proposal.StatusPending)

	total, err := repo.SumPaidByClientOrganization(context.Background(), clientOrgID)
	require.NoError(t, err)
	assert.Equal(t, int64(12_500), total, "only paid-or-later proposals must count toward total spent")
}

func TestProposalRepository_SumPaidByClientOrganization_UnknownOrgReturnsZero(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProposalRepository(db)

	total, err := repo.SumPaidByClientOrganization(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
}

func TestProposalRepository_ListCompletedByClientOrganization(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProposalRepository(db)

	clientID, clientOrgID := insertTestUserWithOrg(t, db, "ListCompletedClient")
	providerID, _ := insertTestUserWithOrg(t, db, "ListCompletedProvider")
	convID := insertConversation(t, db)

	_ = insertProposal(t, repo, convID, clientID, providerID, 1_000, proposal.StatusCompleted)
	_ = insertProposal(t, repo, convID, clientID, providerID, 2_000, proposal.StatusCompleted)
	_ = insertProposal(t, repo, convID, clientID, providerID, 3_000, proposal.StatusPending)

	got, err := repo.ListCompletedByClientOrganization(context.Background(), clientOrgID, 20)
	require.NoError(t, err)
	require.Len(t, got, 2, "only completed proposals must be returned")
	for _, p := range got {
		assert.Equal(t, proposal.StatusCompleted, p.Status)
		assert.NotNil(t, p.CompletedAt)
	}
}

func TestProposalRepository_ListCompletedByClientOrganization_LimitClampsToDefault(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProposalRepository(db)

	got, err := repo.ListCompletedByClientOrganization(context.Background(), uuid.New(), 0)
	require.NoError(t, err)
	assert.Len(t, got, 0, "missing org still returns an empty slice")
}
