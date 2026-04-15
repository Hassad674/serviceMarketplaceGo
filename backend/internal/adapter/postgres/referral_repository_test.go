package postgres_test

// Integration tests for the postgres-backed ReferralRepository (migrations
// 105-108). Gated by MARKETPLACE_TEST_DATABASE_URL — auto-skips when unset.
//
// To run:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_feat_referral?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestReferralRepository -count=1
//
// The suite creates fresh users with random IDs and cleans them up via
// t.Cleanup, so reruns stay isolated. It uses insertTestUser from
// job_credit_repository_test.go (same package).

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
)

// newReferralFixture returns a valid referral targeting three fresh users
// inserted into the test DB. Cleanup happens via insertTestUser's hook.
func newReferralFixture(t *testing.T, db *sql.DB) (*referral.Referral, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	referrerID := insertTestUser(t, db)
	providerID := insertTestUser(t, db)
	clientID := insertTestUser(t, db)

	r, err := referral.NewReferral(referral.NewReferralInput{
		ReferrerID:           referrerID,
		ProviderID:           providerID,
		ClientID:             clientID,
		RatePct:              5,
		DurationMonths:       6,
		IntroSnapshot:        referral.IntroSnapshot{Provider: referral.ProviderSnapshot{Region: "IDF"}},
		IntroMessageProvider: "test pitch provider",
		IntroMessageClient:   "test pitch client",
	})
	require.NoError(t, err)
	return r, referrerID, providerID, clientID
}

// cleanupReferral deletes referrals owned by the given users to keep reruns
// isolated. Negotiations and attributions cascade or are restricted; we drop
// them explicitly for safety.
func cleanupReferral(t *testing.T, db *sql.DB, ids ...uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = db.Exec(`DELETE FROM referral_commissions WHERE attribution_id IN
				(SELECT id FROM referral_attributions WHERE provider_id = $1 OR client_id = $1)`, id)
			_, _ = db.Exec(`DELETE FROM referral_attributions WHERE provider_id = $1 OR client_id = $1`, id)
			_, _ = db.Exec(`DELETE FROM referral_negotiations WHERE actor_id = $1`, id)
			_, _ = db.Exec(`DELETE FROM referrals WHERE referrer_id = $1 OR provider_id = $1 OR client_id = $1`, id)
		}
	})
}

func TestReferralRepository_CreateGetRoundTrip(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)

	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, ref))

	got, err := repo.GetByID(ctx, ref.ID)
	require.NoError(t, err)
	assert.Equal(t, ref.ID, got.ID)
	assert.Equal(t, ref.ReferrerID, got.ReferrerID)
	assert.Equal(t, ref.ProviderID, got.ProviderID)
	assert.Equal(t, ref.ClientID, got.ClientID)
	assert.Equal(t, ref.RatePct, got.RatePct)
	assert.Equal(t, referral.StatusPendingProvider, got.Status)
	assert.Equal(t, "IDF", got.IntroSnapshot.Provider.Region)
}

func TestReferralRepository_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, referral.ErrNotFound)
}

func TestReferralRepository_Update_PersistsTransition(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, ref))
	require.NoError(t, ref.AcceptByProvider(provID))
	require.NoError(t, repo.Update(ctx, ref))

	got, err := repo.GetByID(ctx, ref.ID)
	require.NoError(t, err)
	assert.Equal(t, referral.StatusPendingClient, got.Status)
}

func TestReferralRepository_Update_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref := &referral.Referral{
		ID:           uuid.New(),
		Status:       referral.StatusPendingProvider,
		Version:      1,
		LastActionAt: time.Now().UTC(),
	}
	err := repo.Update(context.Background(), ref)
	require.ErrorIs(t, err, referral.ErrNotFound)
}

func TestReferralRepository_FindActiveByCouple(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()

	// Before insert: not found
	_, err := repo.FindActiveByCouple(ctx, provID, cliID)
	require.ErrorIs(t, err, referral.ErrNotFound)

	// After insert: found
	require.NoError(t, repo.Create(ctx, ref))
	got, err := repo.FindActiveByCouple(ctx, provID, cliID)
	require.NoError(t, err)
	assert.Equal(t, ref.ID, got.ID)

	// After rejection: no longer found
	require.NoError(t, ref.RejectByProvider(provID, "no thanks"))
	require.NoError(t, repo.Update(ctx, ref))
	_, err = repo.FindActiveByCouple(ctx, provID, cliID)
	require.ErrorIs(t, err, referral.ErrNotFound)
}

func TestReferralRepository_CoupleLocked(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	otherReferrer := insertTestUser(t, db)
	cleanupReferral(t, db, otherReferrer)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, ref))

	// Second referral by a DIFFERENT referrer on the SAME couple → locked.
	dup, err := referral.NewReferral(referral.NewReferralInput{
		ReferrerID:           otherReferrer,
		ProviderID:           provID,
		ClientID:             cliID,
		RatePct:              7,
		DurationMonths:       6,
		IntroSnapshot:        referral.IntroSnapshot{},
		IntroMessageProvider: "x",
		IntroMessageClient:   "y",
	})
	require.NoError(t, err)
	err = repo.Create(ctx, dup)
	require.ErrorIs(t, err, referral.ErrCoupleLocked)
}

func TestReferralRepository_CoupleAvailableAfterRejection(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	otherReferrer := insertTestUser(t, db)
	cleanupReferral(t, db, otherReferrer)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, ref))
	require.NoError(t, ref.RejectByProvider(provID, "no"))
	require.NoError(t, repo.Update(ctx, ref))

	// Now another apporteur can create one on the same couple.
	dup, _ := referral.NewReferral(referral.NewReferralInput{
		ReferrerID:           otherReferrer,
		ProviderID:           provID,
		ClientID:             cliID,
		RatePct:              7,
		DurationMonths:       6,
		IntroSnapshot:        referral.IntroSnapshot{},
		IntroMessageProvider: "x",
		IntroMessageClient:   "y",
	})
	require.NoError(t, repo.Create(ctx, dup))
}

func TestReferralRepository_ListByReferrer_FilterAndPagination(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	referrerID := insertTestUser(t, db)
	cleanupReferral(t, db, referrerID)
	ctx := context.Background()

	// Create 3 referrals with distinct couples.
	for i := 0; i < 3; i++ {
		provID := insertTestUser(t, db)
		cliID := insertTestUser(t, db)
		cleanupReferral(t, db, provID, cliID)
		r, err := referral.NewReferral(referral.NewReferralInput{
			ReferrerID:           referrerID,
			ProviderID:           provID,
			ClientID:             cliID,
			RatePct:              float64(i + 1),
			DurationMonths:       6,
			IntroSnapshot:        referral.IntroSnapshot{},
			IntroMessageProvider: "p",
			IntroMessageClient:   "c",
		})
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, r))
		time.Sleep(2 * time.Millisecond) // ensure distinct created_at for stable ordering
	}

	// All three with no filter, limit 2 → first page returns 2 + cursor.
	page, cursor, err := repo.ListByReferrer(ctx, referrerID, repository.ReferralListFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page, 2)
	assert.NotEmpty(t, cursor)

	// Second page returns the 3rd, no more cursor.
	page, cursor, err = repo.ListByReferrer(ctx, referrerID, repository.ReferralListFilter{Cursor: cursor, Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page, 1)
	assert.Empty(t, cursor)

	// Status filter → only pending_provider matches all 3.
	page, _, err = repo.ListByReferrer(ctx, referrerID, repository.ReferralListFilter{
		Statuses: []referral.Status{referral.StatusPendingProvider},
	})
	require.NoError(t, err)
	assert.Len(t, page, 3)

	// Filter that matches none.
	page, _, err = repo.ListByReferrer(ctx, referrerID, repository.ReferralListFilter{
		Statuses: []referral.Status{referral.StatusActive},
	})
	require.NoError(t, err)
	assert.Empty(t, page)
}

func TestReferralRepository_AppendAndListNegotiations(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, ref))

	n1, err := referral.NewNegotiation(referral.NewNegotiationInput{
		ReferralID: ref.ID,
		Version:    1,
		ActorID:    refID,
		ActorRole:  referral.ActorReferrer,
		Action:     referral.NegoActionProposed,
		RatePct:    5,
		Message:    "init",
	})
	require.NoError(t, err)
	require.NoError(t, repo.AppendNegotiation(ctx, n1))

	n2, err := referral.NewNegotiation(referral.NewNegotiationInput{
		ReferralID: ref.ID,
		Version:    2,
		ActorID:    provID,
		ActorRole:  referral.ActorProvider,
		Action:     referral.NegoActionCountered,
		RatePct:    3,
		Message:    "lower please",
	})
	require.NoError(t, err)
	require.NoError(t, repo.AppendNegotiation(ctx, n2))

	negos, err := repo.ListNegotiations(ctx, ref.ID)
	require.NoError(t, err)
	require.Len(t, negos, 2)
	assert.Equal(t, referral.NegoActionProposed, negos[0].Action)
	assert.Equal(t, referral.NegoActionCountered, negos[1].Action)
}

func TestReferralRepository_AttributionIdempotent(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, ref))

	proposalID := uuid.New()
	a1, _ := referral.NewAttribution(referral.NewAttributionInput{
		ReferralID:      ref.ID,
		ProposalID:      proposalID,
		ProviderID:      provID,
		ClientID:        cliID,
		RatePctSnapshot: 5,
	})
	require.NoError(t, repo.CreateAttribution(ctx, a1))

	// Second insert with same proposal_id → no error (ON CONFLICT DO NOTHING).
	a2, _ := referral.NewAttribution(referral.NewAttributionInput{
		ReferralID:      ref.ID,
		ProposalID:      proposalID,
		ProviderID:      provID,
		ClientID:        cliID,
		RatePctSnapshot: 5,
	})
	require.NoError(t, repo.CreateAttribution(ctx, a2))

	// Lookup returns the original.
	got, err := repo.FindAttributionByProposal(ctx, proposalID)
	require.NoError(t, err)
	assert.Equal(t, a1.ID, got.ID)
}

func TestReferralRepository_CommissionLifecycle(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, ref))

	a, _ := referral.NewAttribution(referral.NewAttributionInput{
		ReferralID:      ref.ID,
		ProposalID:      uuid.New(),
		ProviderID:      provID,
		ClientID:        cliID,
		RatePctSnapshot: 5,
	})
	require.NoError(t, repo.CreateAttribution(ctx, a))

	c, _ := referral.NewCommission(referral.NewCommissionInput{
		AttributionID:    a.ID,
		MilestoneID:      uuid.New(),
		GrossAmountCents: 1000_00,
		RatePct:          5,
	})
	require.NoError(t, repo.CreateCommission(ctx, c))

	// Duplicate insert raises ErrCommissionAlreadyExists.
	dup, _ := referral.NewCommission(referral.NewCommissionInput{
		AttributionID:    a.ID,
		MilestoneID:      c.MilestoneID,
		GrossAmountCents: 1000_00,
		RatePct:          5,
	})
	err := repo.CreateCommission(ctx, dup)
	require.ErrorIs(t, err, referral.ErrCommissionAlreadyExists)

	// MarkPaid + persist.
	require.NoError(t, c.MarkPaid("tr_test_xyz"))
	require.NoError(t, repo.UpdateCommission(ctx, c))

	got, err := repo.FindCommissionByMilestone(ctx, c.MilestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionPaid, got.Status)
	assert.Equal(t, "tr_test_xyz", got.StripeTransferID)

	// Clawback.
	require.NoError(t, got.ApplyClawback("trr_clawback"))
	require.NoError(t, repo.UpdateCommission(ctx, got))
	got2, err := repo.FindCommissionByMilestone(ctx, c.MilestoneID)
	require.NoError(t, err)
	assert.Equal(t, referral.CommissionClawedBack, got2.Status)
}

func TestReferralRepository_ListPendingKYC(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, ref))

	a, _ := referral.NewAttribution(referral.NewAttributionInput{
		ReferralID:      ref.ID,
		ProposalID:      uuid.New(),
		ProviderID:      provID,
		ClientID:        cliID,
		RatePctSnapshot: 5,
	})
	require.NoError(t, repo.CreateAttribution(ctx, a))

	c, _ := referral.NewCommission(referral.NewCommissionInput{
		AttributionID:    a.ID,
		MilestoneID:      uuid.New(),
		GrossAmountCents: 1000_00,
		RatePct:          5,
	})
	require.NoError(t, repo.CreateCommission(ctx, c))
	require.NoError(t, c.MarkPendingKYC())
	require.NoError(t, repo.UpdateCommission(ctx, c))

	pending, err := repo.ListPendingKYCByReferrer(ctx, refID)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, c.ID, pending[0].ID)
}

func TestReferralRepository_ExpiringIntros(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, ref))

	// Force last_action_at backwards by direct update — simulates 14d silence.
	_, err := db.ExecContext(ctx, `UPDATE referrals SET last_action_at = now() - interval '20 days' WHERE id = $1`, ref.ID)
	require.NoError(t, err)

	cutoff := time.Now().UTC().Add(-14 * 24 * time.Hour)
	expiring, err := repo.ListExpiringIntros(ctx, cutoff, 100)
	require.NoError(t, err)
	require.NotEmpty(t, expiring)

	found := false
	for _, e := range expiring {
		if e.ID == ref.ID {
			found = true
		}
	}
	assert.True(t, found, "stale referral must be in expiring list")
}

func TestReferralRepository_ExpiringActives(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	ref, refID, provID, cliID := newReferralFixture(t, db)
	cleanupReferral(t, db, refID, provID, cliID)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, ref))

	// Move to active artificially.
	require.NoError(t, ref.AcceptByProvider(provID))
	require.NoError(t, repo.Update(ctx, ref))
	require.NoError(t, ref.AcceptByClient(cliID))
	require.NoError(t, repo.Update(ctx, ref))

	// Force expires_at into the past.
	_, err := db.ExecContext(ctx, `UPDATE referrals SET expires_at = now() - interval '1 day' WHERE id = $1`, ref.ID)
	require.NoError(t, err)

	expiring, err := repo.ListExpiringActives(ctx, time.Now().UTC(), 100)
	require.NoError(t, err)
	require.NotEmpty(t, expiring)

	found := false
	for _, e := range expiring {
		if e.ID == ref.ID {
			found = true
		}
	}
	assert.True(t, found)
}

func TestReferralRepository_CountAndSumByReferrer(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferralRepository(db)
	referrerID := insertTestUser(t, db)
	cleanupReferral(t, db, referrerID)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		provID := insertTestUser(t, db)
		cliID := insertTestUser(t, db)
		cleanupReferral(t, db, provID, cliID)
		r, _ := referral.NewReferral(referral.NewReferralInput{
			ReferrerID:           referrerID,
			ProviderID:           provID,
			ClientID:             cliID,
			RatePct:              5,
			DurationMonths:       6,
			IntroSnapshot:        referral.IntroSnapshot{},
			IntroMessageProvider: "p",
			IntroMessageClient:   "c",
		})
		require.NoError(t, repo.Create(ctx, r))
	}

	counts, err := repo.CountByReferrer(ctx, referrerID)
	require.NoError(t, err)
	assert.Equal(t, 2, counts[referral.StatusPendingProvider])

	// SumCommissionsByReferrer with no commissions yet → empty map.
	sums, err := repo.SumCommissionsByReferrer(ctx, referrerID)
	require.NoError(t, err)
	assert.Empty(t, sums)
}
