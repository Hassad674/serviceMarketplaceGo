package postgres_test

// Tests for the RLS tenant-context wrap on ProposalRepository.
//
// BUG-NEW-04 path 4/8 — proposals. The proposals table is RLS-
// protected by migration 125 with the policy
//
//   USING (
//       client_organization_id = current_setting('app.current_org_id', true)::uuid
//       OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
//   )
//
// Both the client-side org and the provider-side org are stakeholders
// — either can read/update the proposal. Under prod NOSUPERUSER
// NOBYPASSRLS, app.current_org_id MUST be set to either side's org
// before the SELECT/INSERT/UPDATE fires, otherwise the row is filtered
// or the write rejected.
//
// Strategy:
//   - Create / CreateWithDocumentsAndMilestones use the proposal's
//     client_organization_id (or provider_organization_id when client
//     is nil) as the tenant context — the data carries it.
//   - GetByID / Update do a two-step under tenant tx: first SELECT
//     the row's client + provider orgs via a privileged db.QueryRow
//     (legacy path — defensive for callers that only have the id),
//     then open a tenant tx with one of the orgs and run the operation.
//   - List* methods take orgID directly and use it as the tenant
//     context.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/proposal"
)

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

func TestProposalRepository_WithTxRunner_ReturnsSameRepo(t *testing.T) {
	repo := postgres.NewProposalRepository(nil)
	runner := postgres.NewTxRunner(nil)
	got := repo.WithTxRunner(runner)
	assert.Same(t, repo, got)
}

func TestProposalRepository_WithTxRunner_NilRunner_NoPanic(t *testing.T) {
	repo := postgres.NewProposalRepository(nil)
	got := repo.WithTxRunner(nil)
	assert.NotNil(t, got)
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func newRLSProposalFixture(t *testing.T) (clientOrgID, providerOrgID, clientUserID, providerUserID, convID uuid.UUID) {
	t.Helper()
	db := testDB(t)
	ensureRLSTestRole(t, db)

	clientUserID = insertTestUser(t, db)
	providerUserID = insertTestUser(t, db)
	clientOrgID = insertOrgRaw(t, db, clientUserID, "ProposalClient-"+uuid.NewString()[:6])
	providerOrgID = insertOrgRaw(t, db, providerUserID, "ProposalProvider-"+uuid.NewString()[:6])
	convID = insertRLSConversation(t, db, clientOrgID, clientUserID)
	return
}

func makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID uuid.UUID) *proposal.Proposal {
	now := time.Now()
	return &proposal.Proposal{
		ID:             uuid.New(),
		ConversationID: convID,
		SenderID:       providerUserID,
		RecipientID:    clientUserID,
		Title:          "Test proposal — BUG-NEW-04 path 4/8",
		Description:    "Validates RLS tenant wrap on proposals path",
		Amount:         10000,
		Status:         proposal.StatusPending,
		ParentID:       nil,
		Version:        1,
		ClientID:       clientUserID,
		ProviderID:     providerUserID,
		// In the schema: client_organization_id + provider_organization_id
		// are stored as part of the proposal row alongside ClientID/ProviderID
		// — so the repo's INSERT plumbs them via Metadata. To wire this we
		// have to resolve org from users table during the test setup.
		Metadata:  []byte(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TestProposalRepository_Create_UnderRLS verifies the parent INSERT
// goes through under non-superuser when the proposal carries its
// org information. The repo derives app.current_org_id from the row.
func TestProposalRepository_Create_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)

	// The schema requires client_organization_id and provider_organization_id
	// at INSERT time — these come from the users.organization_id column,
	// so we make sure both users have their orgs set.
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	repo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, repo.Create(ctx, p),
		"Create with tenant context derived from proposal must succeed under RLS")

	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	// Read back via tenant-aware path using the client org.
	got, err := repo.GetByIDForOrg(ctx, p.ID, clientOrgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p.ID, got.ID)
}

// TestProposalRepository_GetByIDForOrg_UnderRLS asserts that the
// tenant-aware single-row read passes the policy and returns the row
// to either side (client or provider org).
func TestProposalRepository_GetByIDForOrg_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	repo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, repo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	// Provider org reads.
	gotForProvider, err := repo.GetByIDForOrg(ctx, p.ID, providerOrgID)
	require.NoError(t, err)
	require.NotNil(t, gotForProvider)

	// Client org reads — same row, both sides allowed.
	gotForClient, err := repo.GetByIDForOrg(ctx, p.ID, clientOrgID)
	require.NoError(t, err)
	require.NotNil(t, gotForClient)
}

// TestProposalRepository_ListActiveProjectsByOrganization_UnderRLS
// verifies the org-scoped list pagination passes the policy.
func TestProposalRepository_ListActiveProjectsByOrganization_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	repo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	p.Status = proposal.StatusPaid
	now := time.Now()
	p.PaidAt = &now
	require.NoError(t, repo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	got, _, err := repo.ListActiveProjectsByOrganization(ctx, clientOrgID, "", 10)
	require.NoError(t, err)
	require.NotEmpty(t, got, "ListActiveProjectsByOrganization under tenant context must return the row")
}

// TestProposalRepository_Legacy_NoTxRunner_StillWorks confirms backward
// compat via the *sql.DB-only constructor.
func TestProposalRepository_Legacy_NoTxRunner_StillWorks(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	repo := postgres.NewProposalRepository(db) // no txRunner
	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, repo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })
}
