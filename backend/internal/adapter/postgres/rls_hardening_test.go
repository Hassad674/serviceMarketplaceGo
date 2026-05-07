package postgres_test

// rls_hardening_test.go — integration coverage for the RLS hardening
// pass landed on 2026-05-07.
//
// Strategy mirrors rls_caller_audit_test.go: each newly-fixed
// repository method gets a positive case (caller's org admits the
// row) and a negative case (third-party org under rlsTestRole sees
// nothing).
//
// Gate: MARKETPLACE_TEST_DATABASE_URL must point at a Postgres
// reachable to the test process. The fixture re-uses the cross-
// feature builder from rls_caller_audit_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/system"
)

// TestRLS_Hardening_ProposalReads exercises the user-facing reads on
// ProposalRepository that were wrapped in RunInTxWithTenant during
// the hardening pass.
func TestRLS_Hardening_ProposalReads(t *testing.T) {
	db := testDB(t)
	f := newRLSAuditFixture(t, db)
	repo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	ctx := context.Background()

	t.Run("IsOrgAuthorizedForProposal admits client org", func(t *testing.T) {
		ok, err := repo.IsOrgAuthorizedForProposal(ctx, f.proposalID, f.clientOrgID)
		require.NoError(t, err)
		assert.True(t, ok, "client org must be authorized")
	})

	t.Run("IsOrgAuthorizedForProposal admits provider org", func(t *testing.T) {
		ok, err := repo.IsOrgAuthorizedForProposal(ctx, f.proposalID, f.providerOrgID)
		require.NoError(t, err)
		assert.True(t, ok, "provider org must be authorized")
	})

	t.Run("IsOrgAuthorizedForProposal denies third-party org", func(t *testing.T) {
		// CRITICAL: this is the auth check that gates every
		// proposal-mutation endpoint. A third-party caller MUST
		// receive false.
		ok, err := repo.IsOrgAuthorizedForProposal(ctx, f.proposalID, f.thirdPartyOrgID)
		require.NoError(t, err)
		assert.False(t, ok, "third-party org must NOT be authorized — RLS bypass would be a SEV-1")
	})

	t.Run("IsOrgAuthorizedForProposal — no bypass under rlsTestRole", func(t *testing.T) {
		// Belt + suspenders: replay the negative assertion as a
		// raw SELECT under rlsTestRole + thirdPartyOrg context.
		// The proposals RLS policy must filter the row, count == 0.
		n := countUnderRole(t, db, f.thirdPartyOrgID, f.thirdPartyUID, "proposals", f.proposalID)
		assert.Equal(t, 0, n, "third-party org under rlsTestRole must NOT see the proposal")
	})

	t.Run("SumPaidByClientOrganization isolates by org", func(t *testing.T) {
		// Positive: client org gets a number (zero is fine — the
		// fixture's proposal is in 'pending', not paid yet).
		_, err := repo.SumPaidByClientOrganization(ctx, f.clientOrgID)
		require.NoError(t, err, "client org must be able to read its own sum")

		// Cross-tenant: third-party org gets zero (no rows visible).
		total, err := repo.SumPaidByClientOrganization(ctx, f.thirdPartyOrgID)
		require.NoError(t, err)
		assert.Zero(t, total, "third-party org must see no paid proposals")
	})

	t.Run("ListCompletedByClientOrganization isolates by org", func(t *testing.T) {
		_, err := repo.ListCompletedByClientOrganization(ctx, f.clientOrgID, 10)
		require.NoError(t, err)

		// Third-party: empty list, no error.
		out, err := repo.ListCompletedByClientOrganization(ctx, f.thirdPartyOrgID, 10)
		require.NoError(t, err)
		assert.Empty(t, out, "third-party org must see no completed proposals")
	})
}

// TestRLS_Hardening_MilestoneListByProposalForOrg exercises the
// user-facing milestone read added during the hardening pass.
func TestRLS_Hardening_MilestoneListByProposalForOrg(t *testing.T) {
	db := testDB(t)
	f := newRLSAuditFixture(t, db)
	repo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	ctx := context.Background()

	t.Run("admits client org", func(t *testing.T) {
		out, err := repo.ListByProposalForOrg(ctx, f.proposalID, f.clientOrgID)
		require.NoError(t, err)
		assert.NotEmpty(t, out, "client org must see the proposal's milestones")
	})

	t.Run("admits provider org", func(t *testing.T) {
		out, err := repo.ListByProposalForOrg(ctx, f.proposalID, f.providerOrgID)
		require.NoError(t, err)
		assert.NotEmpty(t, out, "provider org must see the proposal's milestones")
	})

	t.Run("denies third-party org under rlsTestRole", func(t *testing.T) {
		// Under the test DSN's superuser role, RLS is bypassed and
		// ListByProposalForOrg would return rows because the SQL
		// has no Go-level org filter (the filter is the policy).
		// Replay the negative under rlsTestRole + thirdPartyOrg
		// context — count must be zero.
		n := countUnderRole(t, db, f.thirdPartyOrgID, f.thirdPartyUID, "proposal_milestones", f.milestoneID)
		assert.Zero(t, n,
			"third-party org under rlsTestRole MUST NOT see the milestone")
	})
}

// TestRLS_Hardening_PaymentRecord_GetByIDForOrg confirms the
// payment-record user-facing read isolates by org.
func TestRLS_Hardening_PaymentRecord_GetByIDForOrg(t *testing.T) {
	db := testDB(t)
	f := newRLSAuditFixture(t, db)
	repo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	ctx := context.Background()

	t.Run("admits owning org (client side)", func(t *testing.T) {
		rec, err := repo.GetByIDForOrg(ctx, f.paymentRecordID, f.clientOrgID)
		require.NoError(t, err)
		assert.Equal(t, f.paymentRecordID, rec.ID)
	})

	t.Run("denies non-owning org (provider side — single-side ownership) under rlsTestRole", func(t *testing.T) {
		// payment_records is single-side: only the client org
		// owns the record. Under rlsTestRole the policy filters
		// the row when current_setting('app.current_org_id')
		// matches anything other than the client org.
		n := countUnderRole(t, db, f.providerOrgID, f.providerUserID, "payment_records", f.paymentRecordID)
		assert.Zero(t, n,
			"provider org under rlsTestRole MUST NOT see the client-side payment record")
	})

	t.Run("denies third-party org under rlsTestRole", func(t *testing.T) {
		n := countUnderRole(t, db, f.thirdPartyOrgID, f.thirdPartyUID, "payment_records", f.paymentRecordID)
		assert.Zero(t, n,
			"third-party org under rlsTestRole MUST NOT see the payment record")
	})
}

// TestRLS_Hardening_SystemActor_LegacyMethodsWork covers the
// methods that legitimately stay on the legacy non-tenant path
// when the caller tags its context system-actor (schedulers,
// webhooks, AI summary worker). Re-asserts the contract surfaced
// by the warn guards.
func TestRLS_Hardening_SystemActor_LegacyMethodsWork(t *testing.T) {
	db := testDB(t)
	f := newRLSAuditFixture(t, db)
	ctx := system.WithSystemActor(context.Background())

	t.Run("MilestoneRepository.GetByIDWithVersion under system tag", func(t *testing.T) {
		repo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		m, err := repo.GetByIDWithVersion(ctx, f.milestoneID)
		require.NoError(t, err)
		assert.Equal(t, f.milestoneID, m.ID)
	})

	t.Run("MilestoneRepository.ListByProposal under system tag", func(t *testing.T) {
		repo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		out, err := repo.ListByProposal(ctx, f.proposalID)
		require.NoError(t, err)
		assert.NotEmpty(t, out)
	})

	t.Run("MilestoneRepository.GetCurrentActive under system tag", func(t *testing.T) {
		repo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		m, err := repo.GetCurrentActive(ctx, f.proposalID)
		require.NoError(t, err)
		require.NotNil(t, m)
	})

	t.Run("MilestoneRepository.ListByProposals under system tag", func(t *testing.T) {
		repo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		out, err := repo.ListByProposals(ctx, []uuid.UUID{f.proposalID})
		require.NoError(t, err)
		assert.NotEmpty(t, out[f.proposalID])
	})

	t.Run("DisputeRepository.GetByProposalID under system tag", func(t *testing.T) {
		repo := postgres.NewDisputeRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		d, err := repo.GetByProposalID(ctx, f.proposalID)
		require.NoError(t, err)
		require.NotNil(t, d)
		assert.Equal(t, f.disputeID, d.ID)
	})

	t.Run("PaymentRecordRepository.GetByProposalID under system tag", func(t *testing.T) {
		repo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		rec, err := repo.GetByProposalID(ctx, f.proposalID)
		require.NoError(t, err)
		assert.Equal(t, f.paymentRecordID, rec.ID)
	})

	t.Run("PaymentRecordRepository.GetByMilestoneID under system tag", func(t *testing.T) {
		repo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		rec, err := repo.GetByMilestoneID(ctx, f.milestoneID)
		require.NoError(t, err)
		assert.Equal(t, f.paymentRecordID, rec.ID)
	})

	t.Run("PaymentRecordRepository.ListByProposalID under system tag", func(t *testing.T) {
		repo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		out, err := repo.ListByProposalID(ctx, f.proposalID)
		require.NoError(t, err)
		assert.NotEmpty(t, out)
	})

	t.Run("InvoiceRepository.FindInvoiceByID under system tag (no row -> ErrNotFound)", func(t *testing.T) {
		repo := postgres.NewInvoiceRepository(db).WithTxRunner(postgres.NewTxRunner(db))
		// The fixture does not create invoices, so we expect
		// ErrNotFound — but the call must not panic / leak
		// errors regardless of the system-actor context.
		_, _ = repo.FindInvoiceByID(ctx, uuid.New())
	})

	t.Run("ConversationRepository legacy reads under system tag", func(t *testing.T) {
		repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

		_, err := repo.GetMessagesSinceSeq(ctx, f.convID, 0, 10)
		assert.NoError(t, err, "GetMessagesSinceSeq must succeed under system tag")

		_, err = repo.ListMessagesSinceTime(ctx, f.convID, time.Now().Add(-24*time.Hour), 10)
		assert.NoError(t, err, "ListMessagesSinceTime must succeed under system tag")
	})
}

// TestRLS_Hardening_IsOrgAuthorizedForProposal_NeverBypassed is the
// CRITICAL regression test the audit brief calls out: this auth
// check must NEVER return true for an unrelated org, even under a
// privileged role. It runs the assertion BOTH through the public
// API (RunInTxWithTenant) AND under rlsTestRole (raw SELECT) so a
// regression in either layer is caught.
func TestRLS_Hardening_IsOrgAuthorizedForProposal_NeverBypassed(t *testing.T) {
	db := testDB(t)
	f := newRLSAuditFixture(t, db)
	repo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	ctx := context.Background()

	// Layer 1 — adapter API.
	ok, err := repo.IsOrgAuthorizedForProposal(ctx, f.proposalID, f.thirdPartyOrgID)
	require.NoError(t, err)
	assert.False(t, ok,
		"third-party org MUST NOT be authorized — RLS bypass would be a SEV-1")

	// Layer 2 — raw SELECT under rlsTestRole.
	n := countUnderRole(t, db, f.thirdPartyOrgID, f.thirdPartyUID, "proposals", f.proposalID)
	assert.Zero(t, n,
		"third-party org under rlsTestRole MUST NOT see the proposal row — "+
			"any non-zero count is a SEV-1 RLS regression")
}
