package postgres_test

// Integration tests for the 3 ranking V1 aggregate loaders
// (phase 6B). Gated behind MARKETPLACE_TEST_DATABASE_URL — the
// shared helper `searchTestDB` auto-skips when unset.
//
// Each aggregate has 4 scenarios per the brief's "minimum bar":
// empty, one, typical, extreme. The scenarios share the seedFreelance
// helper from search_document_repository_test.go so every fixture
// row has the full chain of user + organization + profile + ancillary
// data needed by other adapter queries.
//
// We do NOT use testcontainers here — the shared integration DB
// is bootstrapped by the caller (per the existing project pattern)
// and each test cleans up its own seeded rows via t.Cleanup.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/search"
)

// seedClientOrg creates one client organisation with its owner user
// and returns the client org ID plus the owner user ID. Used by the
// client-history tests to set up the "other side" of each proposal.
func seedClientOrg(t *testing.T, db *sql.DB, label string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	orgID := uuid.New()
	userID := uuid.New()
	email := fmt.Sprintf("%s-%s@clienthist.test", label, orgID.String()[:8])
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
		VALUES ($1, $2, 'hash', 'Client', $3, $4, 'enterprise', 'marketplace_owner')`,
		userID, email, label, "Client "+label)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, 'enterprise', $3)`,
		orgID, userID, "Client Org "+label)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, userID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM users WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
	})
	return orgID, userID
}

// seedReleasedMilestone writes one proposal + one released milestone
// between the given provider user and client user. Returns the
// proposal ID so additional milestones can be attached.
func seedReleasedMilestone(t *testing.T, db *sql.DB, clientUserID, providerUserID uuid.UUID, seq int) uuid.UUID {
	t.Helper()
	convID := uuid.New()
	_, err := db.Exec(`INSERT INTO conversations (id) VALUES ($1)`, convID)
	require.NoError(t, err)

	proposalID := uuid.New()
	// Pull client org from the users table so the denormalised
	// proposals.organization_id column is populated as production
	// code would have it.
	var clientOrgID uuid.UUID
	err = db.QueryRow(`SELECT organization_id FROM users WHERE id = $1`, clientUserID).Scan(&clientOrgID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                       title, description, amount, status, organization_id)
		VALUES ($1, $2, $3, $4, $3, $4, $5, $6, 100000, 'completed', $7)`,
		proposalID, convID, clientUserID, providerUserID,
		fmt.Sprintf("Mission %d", seq), fmt.Sprintf("Scope %d", seq), clientOrgID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO proposal_milestones (id, proposal_id, sequence, title, description, amount, status, released_at)
		VALUES ($1, $2, 1, $3, $4, 100000, 'released', NOW())`,
		uuid.New(), proposalID, fmt.Sprintf("Milestone %d", seq), "Scope")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM proposal_milestones WHERE proposal_id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})
	return proposalID
}

// fetchProviderUserID returns the first user row linked to the given
// org. The seedFreelanceActor helper sets up exactly one owner user
// per org so a single-row fetch is deterministic here.
func fetchProviderUserID(t *testing.T, db *sql.DB, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	var userID uuid.UUID
	err := db.QueryRow(`SELECT id FROM users WHERE organization_id = $1 LIMIT 1`, orgID).Scan(&userID)
	require.NoError(t, err)
	return userID
}

func TestSearchDocumentRepository_LoadClientHistory(t *testing.T) {
	db := searchTestDB(t)

	t.Run("empty — no released milestones", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadClientHistory(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 0, got.UniqueClients)
		assert.Equal(t, 0.0, got.RepeatClientRate)
	})

	t.Run("one client, one project — repeat rate 0", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)
		_, clientUserID := seedClientOrg(t, db, "one-client")
		seedReleasedMilestone(t, db, clientUserID, providerUserID, 1)

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadClientHistory(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.UniqueClients)
		assert.Equal(t, 0.0, got.RepeatClientRate, "1 project / 1 client — no repeat")
	})

	t.Run("typical — 3 clients, 1 repeats", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)

		_, c1 := seedClientOrg(t, db, "typical-c1")
		_, c2 := seedClientOrg(t, db, "typical-c2")
		_, c3 := seedClientOrg(t, db, "typical-c3")

		seedReleasedMilestone(t, db, c1, providerUserID, 1)
		seedReleasedMilestone(t, db, c1, providerUserID, 2) // c1 returns
		seedReleasedMilestone(t, db, c2, providerUserID, 3)
		seedReleasedMilestone(t, db, c3, providerUserID, 4)

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadClientHistory(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 3, got.UniqueClients)
		assert.InDelta(t, 1.0/3.0, got.RepeatClientRate, 0.0001,
			"1 of 3 clients returned → repeat rate ~0.333")
	})

	t.Run("extreme — single client, 5 repeats → repeat rate 1.0", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)
		_, c := seedClientOrg(t, db, "extreme")
		for i := 0; i < 5; i++ {
			seedReleasedMilestone(t, db, c, providerUserID, i)
		}
		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadClientHistory(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.UniqueClients)
		assert.Equal(t, 1.0, got.RepeatClientRate, "only client has ≥2 projects")
	})
}

// seedPublishedReview inserts one published client→provider review
// from the given reviewer against the provider org. Returns the
// review ID for cleanup tracking.
func seedPublishedReview(t *testing.T, db *sql.DB, reviewerUserID, reviewerOrgID, reviewedUserID, reviewedOrgID uuid.UUID, createdAt time.Time, rating int) uuid.UUID {
	t.Helper()
	proposalID := uuid.New()
	convID := uuid.New()
	_, err := db.Exec(`INSERT INTO conversations (id) VALUES ($1)`, convID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                       title, description, amount, status)
		VALUES ($1, $2, $3, $4, $3, $4, 'review-seed', 'scope', 1, 'completed')`,
		proposalID, convID, reviewerUserID, reviewedUserID)
	require.NoError(t, err)
	reviewID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO reviews (id, proposal_id, reviewer_id, reviewed_id,
		                     global_rating, timeliness, communication, quality, comment,
		                     reviewer_organization_id, reviewed_organization_id, side,
		                     moderation_status, moderation_score, title_visible,
		                     created_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $5, $5, $5, 'Solid work',
		        $6, $7, 'client_to_provider',
		        'clean', 0, true,
		        $8, $8)`,
		reviewID, proposalID, reviewerUserID, reviewedUserID, rating,
		reviewerOrgID, reviewedOrgID, createdAt)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM reviews WHERE id = $1`, reviewID)
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})
	return reviewID
}

func TestSearchDocumentRepository_LoadReviewDiversity(t *testing.T) {
	db := searchTestDB(t)

	t.Run("empty — no reviews", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadReviewDiversity(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 0, got.UniqueReviewers)
		assert.Equal(t, 0.0, got.MaxReviewerShare)
		assert.Equal(t, 0.0, got.ReviewRecencyFactor)
	})

	t.Run("one reviewer — max share 1.0", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)
		reviewerOrg, reviewerUser := seedClientOrg(t, db, "div-one")
		seedPublishedReview(t, db, reviewerUser, reviewerOrg, providerUserID, orgID, time.Now(), 5)

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadReviewDiversity(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.UniqueReviewers)
		assert.Equal(t, 1.0, got.MaxReviewerShare)
		assert.Greater(t, got.ReviewRecencyFactor, 0.99,
			"brand-new review → recency factor near 1.0")
	})

	t.Run("typical — 3 reviewers, balanced counts, mixed ages", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)

		org1, user1 := seedClientOrg(t, db, "div-typ1")
		org2, user2 := seedClientOrg(t, db, "div-typ2")
		org3, user3 := seedClientOrg(t, db, "div-typ3")

		now := time.Now()
		seedPublishedReview(t, db, user1, org1, providerUserID, orgID, now, 5)
		seedPublishedReview(t, db, user2, org2, providerUserID, orgID, now.AddDate(0, 0, -180), 4)
		seedPublishedReview(t, db, user3, org3, providerUserID, orgID, now.AddDate(-1, 0, 0), 5)

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadReviewDiversity(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 3, got.UniqueReviewers)
		assert.InDelta(t, 1.0/3.0, got.MaxReviewerShare, 0.0001,
			"balanced 1/1/1 → max share 0.333")
		assert.Greater(t, got.ReviewRecencyFactor, 0.0)
		assert.Less(t, got.ReviewRecencyFactor, 1.0,
			"mixed ages → recency factor between 0 and 1")
	})

	t.Run("extreme — 1 reviewer with 8 reviews, 1 with 2 → max share 0.8", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)

		hogOrg, hogUser := seedClientOrg(t, db, "div-hog")
		otherOrg, otherUser := seedClientOrg(t, db, "div-oth")

		now := time.Now()
		for i := 0; i < 8; i++ {
			seedPublishedReview(t, db, hogUser, hogOrg, providerUserID, orgID,
				now.AddDate(0, 0, -i*10), 5)
		}
		for i := 0; i < 2; i++ {
			seedPublishedReview(t, db, otherUser, otherOrg, providerUserID, orgID,
				now.AddDate(0, 0, -(i+1)*20), 5)
		}

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadReviewDiversity(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 2, got.UniqueReviewers)
		assert.InDelta(t, 0.8, got.MaxReviewerShare, 0.0001,
			"hog reviewer owns 8/10 reviews")
	})
}

func TestSearchDocumentRepository_LoadAccountAge(t *testing.T) {
	db := searchTestDB(t)

	t.Run("fresh account — age 0, no disputes", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		// Force the owner's created_at to NOW so age_days rounds to 0
		_, err := db.Exec(`UPDATE users SET created_at = NOW() WHERE organization_id = $1`, orgID)
		require.NoError(t, err)

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadAccountAge(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 0, got.LostDisputes)
		assert.Equal(t, 0, got.AccountAgeDays,
			"freshly created user → age 0 days")
	})

	t.Run("mature account — age 400 days, no disputes", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		_, err := db.Exec(`UPDATE users SET created_at = NOW() - INTERVAL '400 days' WHERE organization_id = $1`, orgID)
		require.NoError(t, err)

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadAccountAge(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 0, got.LostDisputes)
		assert.InDelta(t, 400, got.AccountAgeDays, 1,
			"created_at 400 days ago → age 400 ± 1 day")
	})

	t.Run("one lost dispute (full_refund) counts", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)
		clientOrgID, clientUserID := seedClientOrg(t, db, "disp-full")
		seedResolvedDispute(t, db, clientOrgID, clientUserID, orgID, providerUserID, "full_refund")

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadAccountAge(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.LostDisputes)
	})

	t.Run("extreme — 3 disputes: 1 full_refund + 1 partial_refund + 1 full_release (not counted)", func(t *testing.T) {
		orgID := seedFreelanceActor(t, db)
		providerUserID := fetchProviderUserID(t, db, orgID)

		clientOrg1, clientUser1 := seedClientOrg(t, db, "disp-ex-1")
		clientOrg2, clientUser2 := seedClientOrg(t, db, "disp-ex-2")
		clientOrg3, clientUser3 := seedClientOrg(t, db, "disp-ex-3")

		seedResolvedDispute(t, db, clientOrg1, clientUser1, orgID, providerUserID, "full_refund")
		seedResolvedDispute(t, db, clientOrg2, clientUser2, orgID, providerUserID, "partial_refund")
		seedResolvedDispute(t, db, clientOrg3, clientUser3, orgID, providerUserID, "full_release")

		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadAccountAge(context.Background(), orgID)
		require.NoError(t, err)
		assert.Equal(t, 2, got.LostDisputes,
			"full_release is a win for the provider — not counted")
	})

	t.Run("unknown org — clean zero-value result, no error", func(t *testing.T) {
		repo := postgres.NewSearchDocumentRepository(db)
		got, err := repo.LoadAccountAge(context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, 0, got.LostDisputes)
		assert.Equal(t, 0, got.AccountAgeDays)
	})
}

// seedResolvedDispute inserts a dispute between the given client
// (initiator) and provider (respondent) with the supplied
// resolution type. The dispute is attached to a freshly seeded
// proposal + milestone so every FK chain stays valid.
func seedResolvedDispute(t *testing.T, db *sql.DB, clientOrgID, clientUserID, providerOrgID, providerUserID uuid.UUID, resolutionType string) uuid.UUID {
	t.Helper()

	// Minimal proposal to anchor the dispute.
	convID := uuid.New()
	proposalID := uuid.New()
	milestoneID := uuid.New()
	_, err := db.Exec(`INSERT INTO conversations (id) VALUES ($1)`, convID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                       title, description, amount, status, organization_id)
		VALUES ($1, $2, $3, $4, $3, $4, 'dispute-seed', 'scope', 100000, 'disputed', $5)`,
		proposalID, convID, clientUserID, providerUserID, clientOrgID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO proposal_milestones (id, proposal_id, sequence, title, description, amount, status)
		VALUES ($1, $2, 1, 'step', 'scope', 100000, 'disputed')`,
		milestoneID, proposalID)
	require.NoError(t, err)

	disputeID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO disputes (id, proposal_id, conversation_id,
		                      initiator_id, respondent_id,
		                      client_id, provider_id,
		                      client_organization_id, provider_organization_id,
		                      milestone_id,
		                      reason, description, requested_amount, proposal_amount,
		                      status, resolution_type, resolved_at)
		VALUES ($1, $2, $3,
		        $4, $5,
		        $4, $5,
		        $6, $7,
		        $8,
		        'quality_issue', 'Work was not up to agreed spec.', 100000, 100000,
		        'resolved', $9, NOW())`,
		disputeID, proposalID, convID,
		clientUserID, providerUserID,
		clientOrgID, providerOrgID,
		milestoneID,
		resolutionType)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM disputes WHERE id = $1`, disputeID)
		_, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, milestoneID)
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})
	return disputeID
}

// Compile-time check: the SearchDocumentRepository still satisfies
// the port — important because phase 6B added 3 methods to the
// SearchDataRepository interface and we must not drift.
var _ search.SearchDataRepository = (*postgres.SearchDocumentRepository)(nil)
