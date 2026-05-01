package postgres_test

// Tests for the RLS tenant-context wrap on PaymentRecordRepository.
//
// BUG-NEW-04 path 7/8 — payment_records. The payment_records table is
// RLS-protected by migration 125 with the policy
//
//   USING (organization_id = current_setting('app.current_org_id', true)::uuid)
//
// Single-side ownership: the row's organization_id (the client org)
// is the access boundary. Provider-side reads go through the proposal
// path (already isolated). Under prod NOSUPERUSER NOBYPASSRLS the
// SELECT/INSERT/UPDATE all need app.current_org_id set to the row's
// owning org.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/payment"
)

func TestPaymentRecordRepository_WithTxRunner_ReturnsSameRepo(t *testing.T) {
	repo := postgres.NewPaymentRecordRepository(nil)
	runner := postgres.NewTxRunner(nil)
	got := repo.WithTxRunner(runner)
	assert.Same(t, repo, got)
}

func TestPaymentRecordRepository_WithTxRunner_NilRunner_NoPanic(t *testing.T) {
	repo := postgres.NewPaymentRecordRepository(nil)
	got := repo.WithTxRunner(nil)
	assert.NotNil(t, got)
}

// TestPaymentRecordRepository_Create_UnderRLS verifies the INSERT lands
// under the tenant tx with the client's org as app.current_org_id.
// The payment_records.organization_id is auto-resolved at INSERT time
// from organization_members, so the tenant context must match what
// the sub-select will populate.
func TestPaymentRecordRepository_Create_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientOrgID, providerOrgID, clientUserID, providerUserID, convID := newRLSProposalFixture(t)
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, providerOrgID, providerUserID)
	require.NoError(t, err)

	// Need a parent proposal + milestone for the FK constraints.
	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	p := makeRLSProposal(clientOrgID, providerOrgID, clientUserID, providerUserID, convID)
	require.NoError(t, proposalRepo.Create(ctx, p))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, p.ID) })

	milestoneID := insertMilestone(t, db, p.ID)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, milestoneID) })

	repo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	now := time.Now()
	rec := &payment.PaymentRecord{
		ID:                uuid.New(),
		ProposalID:        p.ID,
		MilestoneID:       milestoneID,
		ClientID:          clientUserID,
		ProviderID:        providerUserID,
		ProposalAmount:    5000,
		StripeFeeAmount:   145,
		PlatformFeeAmount: 500,
		ClientTotalAmount: 5645,
		ProviderPayout:    4500,
		Currency:          "EUR",
		Status:            payment.PaymentRecordStatus("pending"),
		TransferStatus:    payment.TransferStatus("pending"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	require.NoError(t, repo.Create(ctx, rec),
		"Create with tenant context derived from client's org membership must succeed under RLS")

	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM payment_records WHERE id = $1`, rec.ID) })

	// Read back via tenant-aware path.
	got, err := repo.ListByOrganization(ctx, clientOrgID)
	require.NoError(t, err)
	require.NotEmpty(t, got, "ListByOrganization under tenant context must return the row")
}

// TestPaymentRecordRepository_GetByIDForOrg_UnderRLS asserts the read
// path passes the policy when wrapped with the caller's org.
func TestPaymentRecordRepository_GetByIDForOrg_UnderRLS(t *testing.T) {
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

	repo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	now := time.Now()
	rec := &payment.PaymentRecord{
		ID:                uuid.New(),
		ProposalID:        p.ID,
		MilestoneID:       milestoneID,
		ClientID:          clientUserID,
		ProviderID:        providerUserID,
		ProposalAmount:    5000,
		StripeFeeAmount:   145,
		PlatformFeeAmount: 500,
		ClientTotalAmount: 5645,
		ProviderPayout:    4500,
		Currency:          "EUR",
		Status:            payment.PaymentRecordStatus("pending"),
		TransferStatus:    payment.TransferStatus("pending"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, repo.Create(ctx, rec))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM payment_records WHERE id = $1`, rec.ID) })

	// Client org reads.
	got, err := repo.GetByIDForOrg(ctx, rec.ID, clientOrgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, rec.ID, got.ID)
}

// TestPaymentRecordRepository_Legacy_NoTxRunner_StillWorks confirms
// the *sql.DB-only path stays valid.
func TestPaymentRecordRepository_Legacy_NoTxRunner_StillWorks(t *testing.T) {
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

	repo := postgres.NewPaymentRecordRepository(db) // legacy
	now := time.Now()
	rec := &payment.PaymentRecord{
		ID:                uuid.New(),
		ProposalID:        p.ID,
		MilestoneID:       milestoneID,
		ClientID:          clientUserID,
		ProviderID:        providerUserID,
		ProposalAmount:    5000,
		StripeFeeAmount:   145,
		PlatformFeeAmount: 500,
		ClientTotalAmount: 5645,
		ProviderPayout:    4500,
		Currency:          "EUR",
		Status:            payment.PaymentRecordStatus("pending"),
		TransferStatus:    payment.TransferStatus("pending"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, repo.Create(ctx, rec),
		"legacy path must keep working for unit tests with only *sql.DB")
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM payment_records WHERE id = $1`, rec.ID) })
}
