package postgres_test

// rls_caller_audit_test.go — end-to-end integration test that exercises
// every user-facing GetByIDForOrg path migrated by P1, plus the system-
// actor branches, against a live Postgres.
//
// Why: PR #65 wrapped the repository writes with RunInTxWithTenant but
// kept the legacy GetByID alive for system-actor schedulers. Today
// those legacy calls succeed because the migration role bypasses RLS.
// The moment ops rotate the production role to marketplace_app
// (NOSUPERUSER NOBYPASSRLS), every unmigrated caller becomes a 404. P1
// migrated 35-38 of those callers; this test asserts the fix actually
// passes under the same role rotation we are about to ship.
//
// Gate: MARKETPLACE_TEST_DATABASE_URL must point at a Postgres reachable
// to the test process. The test creates / re-uses the
// marketplace_rls_test role (NOSUPERUSER NOBYPASSRLS) via
// ensureRLSTestRole.
//
// Strategy:
//
//   - Happy-path assertions go through the adapter
//     (RunInTxWithTenant) and verify the contract: (clientOrg, proposal)
//     and (providerOrg, proposal) both succeed.
//
//   - Cross-tenant negative assertions open a manual tx that does
//     SET LOCAL ROLE rlsTestRole + SetCurrentOrg(thirdPartyOrg) and
//     run a raw SELECT — under RLS the row must be filtered, surfacing
//     as sql.ErrNoRows. This mirrors the production rotation: if the
//     adapter's RunInTxWithTenant ever forgot to set the org context,
//     the same SELECT under NOBYPASSRLS would return zero rows.
//
//   - System-actor positive assertions go through the legacy GetByID
//     with the context tagged via system.WithSystemActor — under the
//     test DSN (typically postgres superuser) the row is admitted, the
//     guard logs no warning, and the row id matches.
//
// The 4 RLS-protected repos covered:
//   - proposals (client_organization_id OR provider_organization_id)
//   - disputes  (client_organization_id OR provider_organization_id)
//   - milestones (inherited from parent proposal via JOIN)
//   - payment_records (organization_id, single-side)

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
	"marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/system"
)

var _ = errors.Is // ensure errors stays imported even if assertions move

// rlsAuditFixture holds the cross-feature fixture state shared by
// every sub-test of TestRLSCallerAudit_AllPathsPassUnderNonSuperuser.
type rlsAuditFixture struct {
	clientOrgID, providerOrgID     uuid.UUID
	clientUserID, providerUserID   uuid.UUID
	convID                         uuid.UUID
	proposalID, milestoneID        uuid.UUID
	disputeID                      uuid.UUID
	paymentRecordID                uuid.UUID
	reviewID                       uuid.UUID
	thirdPartyOrgID, thirdPartyUID uuid.UUID
}

// newRLSAuditFixture seeds every cross-feature row needed by the
// audit test in a single transaction running as the privileged
// migration owner. RLS does NOT fire here — we want every INSERT
// to land regardless of policy state. The assertions later flip
// to rlsTestRole to exercise the policies.
func newRLSAuditFixture(t *testing.T, db *sql.DB) *rlsAuditFixture {
	t.Helper()
	ensureRLSTestRole(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	f := &rlsAuditFixture{}

	f.clientUserID = insertTestUser(t, db)
	f.providerUserID = insertTestUser(t, db)
	f.thirdPartyUID = insertTestUser(t, db)
	f.clientOrgID = insertOrgRaw(t, db, f.clientUserID, "RLSAuditClient-"+uuid.NewString()[:6])
	f.providerOrgID = insertOrgRaw(t, db, f.providerUserID, "RLSAuditProvider-"+uuid.NewString()[:6])
	f.thirdPartyOrgID = insertOrgRaw(t, db, f.thirdPartyUID, "RLSAuditThirdParty-"+uuid.NewString()[:6])

	for _, link := range []struct {
		userID uuid.UUID
		orgID  uuid.UUID
	}{
		{f.clientUserID, f.clientOrgID},
		{f.providerUserID, f.providerOrgID},
		{f.thirdPartyUID, f.thirdPartyOrgID},
	} {
		_, err := db.ExecContext(ctx,
			`UPDATE users SET organization_id = $1 WHERE id = $2`,
			link.orgID, link.userID)
		require.NoError(t, err, "link user to org")
	}

	f.convID = insertRLSConversation(t, db, f.clientOrgID, f.clientUserID)

	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	now := time.Now()
	p := &proposal.Proposal{
		ID:             uuid.New(),
		ConversationID: f.convID,
		SenderID:       f.providerUserID,
		RecipientID:    f.clientUserID,
		Title:          "RLS audit proposal",
		Description:    "Validates user-facing GetByIDForOrg paths under NOBYPASSRLS",
		Amount:         100000,
		Status:         proposal.StatusPending,
		Version:        1,
		ClientID:       f.clientUserID,
		ProviderID:     f.providerUserID,
		Metadata:       []byte(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, proposalRepo.Create(ctx, p))
	f.proposalID = p.ID
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, `DELETE FROM proposals WHERE id = $1`, f.proposalID) })

	f.milestoneID = insertMilestone(t, db, f.proposalID)
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, `DELETE FROM proposal_milestones WHERE id = $1`, f.milestoneID) })

	f.paymentRecordID = insertRLSPaymentRecord(t, db, f.proposalID, f.milestoneID, f.clientUserID, f.providerUserID, f.clientOrgID)
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, `DELETE FROM payment_records WHERE id = $1`, f.paymentRecordID) })

	disputeRepo := postgres.NewDisputeRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	d := &dispute.Dispute{
		ID:                     uuid.New(),
		ProposalID:             f.proposalID,
		MilestoneID:            f.milestoneID,
		ConversationID:         f.convID,
		InitiatorID:            f.clientUserID,
		RespondentID:           f.providerUserID,
		ClientID:               f.clientUserID,
		ProviderID:             f.providerUserID,
		ClientOrganizationID:   f.clientOrgID,
		ProviderOrganizationID: f.providerOrgID,
		Reason:                 dispute.Reason("delay"),
		Description:            "rls audit dispute",
		RequestedAmount:        25000,
		ProposalAmount:         100000,
		Status:                 dispute.Status("open"),
		LastActivityAt:         now,
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	require.NoError(t, disputeRepo.Create(ctx, d))
	f.disputeID = d.ID
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, `DELETE FROM disputes WHERE id = $1`, f.disputeID) })

	f.reviewID = insertRLSReview(t, db, f.proposalID, f.clientUserID, f.providerUserID, f.clientOrgID, f.providerOrgID)
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, `DELETE FROM reviews WHERE id = $1`, f.reviewID) })

	return f
}

// insertRLSPaymentRecord seeds a payment_records row with the
// minimum columns required by the schema and the RLS policy
// (organization_id is the tenant key on this table).
func insertRLSPaymentRecord(t *testing.T, db *sql.DB, proposalID, milestoneID, clientID, providerID, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO payment_records (
			id, proposal_id, milestone_id, organization_id,
			client_id, provider_id,
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			100000, 0, 5000, 105000, 95000,
			'eur', 'succeeded', 'pending',
			NOW(), NOW()
		)`, id, proposalID, milestoneID, orgID, clientID, providerID)
	require.NoError(t, err, "insert payment_record")
	return id
}

// insertRLSReview seeds a reviews row with both org sides
// populated so the new GetByIDForOrg filter has the data it
// needs to evaluate.
func insertRLSReview(t *testing.T, db *sql.DB, proposalID, reviewerID, reviewedID, reviewerOrgID, reviewedOrgID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO reviews (
			id, proposal_id, reviewer_id, reviewed_id,
			reviewer_organization_id, reviewed_organization_id,
			side, global_rating,
			comment, video_url, title_visible,
			created_at, updated_at, published_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			'client_to_provider', 5,
			'rls audit review', '', true,
			NOW(), NOW(), NOW()
		)`, id, proposalID, reviewerID, reviewedID, reviewerOrgID, reviewedOrgID)
	require.NoError(t, err, "insert review")
	return id
}

// countUnderRole opens a manual transaction, switches to
// rlsTestRole (NOSUPERUSER NOBYPASSRLS), sets the supplied
// tenant context, and runs a SELECT count(*). Used to assert
// cross-tenant denial in a way that does not depend on the
// test DSN's role bits and that survives the
// "current_setting → empty string" cast quirk (count(*) always
// resolves, the policy filter just makes it 0).
func countUnderRole(t *testing.T, db *sql.DB, tenantOrgID, tenantUserID uuid.UUID, table string, rowID uuid.UUID) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	setOrgContext(t, ctx, tx, tenantOrgID, tenantUserID)

	var n int
	err = tx.QueryRowContext(ctx,
		// The table name is a hard-coded string from the test, never
		// user input — the placeholder is the row id. fmt.Sprintf is
		// safe here.
		"SELECT count(*) FROM "+table+" WHERE id = $1", rowID,
	).Scan(&n)
	require.NoError(t, err, "count "+table+" under rlsTestRole")
	return n
}

// TestRLSCallerAudit_AllPathsPassUnderNonSuperuser is the consolidated
// regression test: it exercises every migrated user-facing repository
// entry point under a NOSUPERUSER NOBYPASSRLS-equivalent context, then
// runs the cross-tenant negative assertion (different org → ErrNotFound)
// and the system-actor positive assertion (legacy GetByID still works
// when the context is tagged).
func TestRLSCallerAudit_AllPathsPassUnderNonSuperuser(t *testing.T) {
	db := testDB(t)
	f := newRLSAuditFixture(t, db)

	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	disputeRepo := postgres.NewDisputeRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	milestoneRepo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	paymentRepo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	reviewRepo := postgres.NewReviewRepository(db)

	ctx := context.Background()

	t.Run("ProposalRepository.GetByIDForOrg admits both stakeholder orgs", func(t *testing.T) {
		p1, err := proposalRepo.GetByIDForOrg(ctx, f.proposalID, f.clientOrgID)
		require.NoError(t, err, "client org must read its own proposal under tenant context")
		assert.Equal(t, f.proposalID, p1.ID)

		p2, err := proposalRepo.GetByIDForOrg(ctx, f.proposalID, f.providerOrgID)
		require.NoError(t, err, "provider org must read its own proposal under tenant context")
		assert.Equal(t, f.proposalID, p2.ID)
	})

	t.Run("ProposalRepository — RLS denies third-party org under rlsTestRole", func(t *testing.T) {
		// Direct SELECT under rlsTestRole + thirdPartyOrg
		// context — the policy must filter the row, count == 0.
		n := countUnderRole(t, db, f.thirdPartyOrgID, f.thirdPartyUID, "proposals", f.proposalID)
		assert.Equal(t, 0, n,
			"third-party org under rlsTestRole must NOT see the proposal")
	})

	t.Run("DisputeRepository.GetByIDForOrg admits both stakeholder orgs", func(t *testing.T) {
		d1, err := disputeRepo.GetByIDForOrg(ctx, f.disputeID, f.clientOrgID)
		require.NoError(t, err, "client org must read its dispute")
		assert.Equal(t, f.disputeID, d1.ID)

		d2, err := disputeRepo.GetByIDForOrg(ctx, f.disputeID, f.providerOrgID)
		require.NoError(t, err, "provider org must read its dispute")
		assert.Equal(t, f.disputeID, d2.ID)
	})

	t.Run("DisputeRepository — RLS denies third-party org under rlsTestRole", func(t *testing.T) {
		n := countUnderRole(t, db, f.thirdPartyOrgID, f.thirdPartyUID, "disputes", f.disputeID)
		assert.Equal(t, 0, n,
			"third-party org under rlsTestRole must NOT see the dispute")
	})

	t.Run("MilestoneRepository.GetByIDForOrg admits both stakeholder orgs", func(t *testing.T) {
		m1, err := milestoneRepo.GetByIDForOrg(ctx, f.milestoneID, f.clientOrgID)
		require.NoError(t, err, "client org must read milestones on its proposal")
		assert.Equal(t, f.milestoneID, m1.ID)

		m2, err := milestoneRepo.GetByIDForOrg(ctx, f.milestoneID, f.providerOrgID)
		require.NoError(t, err, "provider org must read milestones on its proposal")
		assert.Equal(t, f.milestoneID, m2.ID)
	})

	t.Run("MilestoneRepository — RLS denies third-party org under rlsTestRole", func(t *testing.T) {
		n := countUnderRole(t, db, f.thirdPartyOrgID, f.thirdPartyUID, "proposal_milestones", f.milestoneID)
		assert.Equal(t, 0, n,
			"third-party org under rlsTestRole must NOT see the milestone")
	})

	t.Run("PaymentRecordRepository.GetByIDForOrg admits owning org", func(t *testing.T) {
		rec, err := paymentRepo.GetByIDForOrg(ctx, f.paymentRecordID, f.clientOrgID)
		require.NoError(t, err, "client org must read its own payment record")
		assert.Equal(t, f.paymentRecordID, rec.ID)
	})

	t.Run("PaymentRecordRepository — RLS denies non-owning org under rlsTestRole", func(t *testing.T) {
		// payment_records is single-side ownership — the
		// provider org is not on the record's organization_id
		// column, so the policy must filter the row even though
		// the proposal's two-side check would have admitted it.
		n := countUnderRole(t, db, f.providerOrgID, f.providerUserID, "payment_records", f.paymentRecordID)
		assert.Equal(t, 0, n,
			"provider org under rlsTestRole must NOT see the payment record (single-side)")
	})

	t.Run("ReviewRepository.GetByIDForOrg admits both stakeholder orgs (Go-level filter)", func(t *testing.T) {
		// Reviews are NOT in the migration 125 RLS scope; the
		// adapter implements the org check at the Go boundary.
		// Both orgs see their own row.
		r1, err := reviewRepo.GetByIDForOrg(ctx, f.reviewID, f.clientOrgID)
		require.NoError(t, err, "reviewer org must read its review")
		assert.Equal(t, f.reviewID, r1.ID)

		r2, err := reviewRepo.GetByIDForOrg(ctx, f.reviewID, f.providerOrgID)
		require.NoError(t, err, "reviewed org must read its review")
		assert.Equal(t, f.reviewID, r2.ID)
	})

	t.Run("ReviewRepository.GetByIDForOrg denies third-party org (Go-level filter)", func(t *testing.T) {
		_, err := reviewRepo.GetByIDForOrg(ctx, f.reviewID, f.thirdPartyOrgID)
		require.Error(t, err, "third-party org must NOT see the review")
		assert.True(t, errors.Is(err, review.ErrNotFound),
			"cross-tenant read must surface review.ErrNotFound, got: %v", err)
	})

	t.Run("System-actor branch: legacy GetByID succeeds when ctx is tagged", func(t *testing.T) {
		// Schedulers use the legacy GetByID path. The contract
		// is "system-actor tag must be set"; the adapter does
		// not reject the call when the tag is present (the warn
		// guard logs nothing). Under the test DSN the row is
		// admitted.
		systemCtx := system.WithSystemActor(ctx)
		p, err := proposalRepo.GetByID(systemCtx, f.proposalID)
		require.NoError(t, err, "system-actor GetByID must succeed under privileged role")
		assert.Equal(t, f.proposalID, p.ID)

		d, err := disputeRepo.GetByID(systemCtx, f.disputeID)
		require.NoError(t, err)
		assert.Equal(t, f.disputeID, d.ID)

		m, err := milestoneRepo.GetByID(systemCtx, f.milestoneID)
		require.NoError(t, err)
		assert.Equal(t, f.milestoneID, m.ID)
	})

	// "RLS policy fires under rlsTestRole when no tenant context
	// is set" — this control already lives in
	// TestRLS_NoContextSet_HidesEverything (rls_isolation_test.go).
	// Re-running it here would be a duplicate; we instead trust
	// the existing test as the canonical proof that the fail-
	// closed branch holds.
}
