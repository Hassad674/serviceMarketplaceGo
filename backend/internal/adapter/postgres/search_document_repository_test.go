package postgres_test

// Integration tests for SearchDocumentRepository.
// Gated behind MARKETPLACE_TEST_DATABASE_URL — auto-skip when unset.

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/search"
)

// searchTestDB is a local helper so we don't depend on the
// alphabetical ordering of the shared testDB helper that lives in
// the job_credit_repository_test.go file.
func searchTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping search integration test")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedFreelanceActor creates the minimum chain of rows the search
// repository expects: users + organizations + freelance_profiles +
// freelance_pricing + profile_skills + a published review + a
// released milestone.
//
// Returns the orgID so tests can exercise LoadActorSignals.
func seedFreelanceActor(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	ownerID := uuid.New()
	email := orgID.String()[:8] + "@search.test"

	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type, last_active_at)
		VALUES ($1, $2, 'hash', 'Alice', 'Dupont', 'Alice Dupont', 'provider', 'marketplace_owner', now())`,
		ownerID, email)
	require.NoError(t, err, "insert user")

	// Use a unique stripe account id per test to avoid colliding
	// on the `idx_organizations_stripe_account_id` unique index
	// when tests run sequentially against a shared DB.
	stripeAcct := "acct_test_" + orgID.String()[:8]
	_, err = db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name, photo_url, city, country_code, latitude, longitude,
		                            work_mode, languages_professional, languages_conversational, stripe_account_id)
		VALUES ($1, $2, 'provider_personal', 'Alice Freelance', 'https://cdn/alice.webp',
		        'Paris', 'FR', 48.8566, 2.3522, $3, $4, $5, $6)`,
		orgID, ownerID,
		pq.Array([]string{"remote", "hybrid"}),
		pq.Array([]string{"fr", "en"}),
		pq.Array([]string{"es"}),
		stripeAcct,
	)
	require.NoError(t, err, "insert organization")

	// Link the user to the org so the messaging/earnings queries
	// can find them.
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, ownerID)
	require.NoError(t, err, "link user to org")

	profileID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO freelance_profiles (id, organization_id, title, about, video_url, availability_status, expertise_domains)
		VALUES ($1, $2, 'Senior Full-Stack Developer', '10 years shipping Next.js + Go.', 'https://cdn/alice.mp4',
		        'available_now', $3)`,
		profileID, orgID, pq.Array([]string{"dev-frontend", "dev-backend"}))
	require.NoError(t, err, "insert freelance profile")

	_, err = db.Exec(`
		INSERT INTO freelance_pricing (profile_id, pricing_type, min_amount, max_amount, currency, negotiable)
		VALUES ($1, 'daily', 50000, 120000, 'EUR', true)`, profileID)
	require.NoError(t, err, "insert freelance pricing")

	// Seed 6 skills to hit the top completion tier. profile_skills
	// has a FK to skills_catalog, so we insert the catalog rows
	// first (idempotent). Both skill_text and display_text are
	// NOT NULL — pass the same string for both.
	for i, skill := range []string{"React", "Next.js", "Go", "TypeScript", "PostgreSQL", "Kubernetes"} {
		_, err = db.Exec(`INSERT INTO skills_catalog (skill_text, display_text) VALUES ($1, $1) ON CONFLICT DO NOTHING`, skill)
		require.NoError(t, err, "insert skills_catalog")
		_, err = db.Exec(`INSERT INTO profile_skills (organization_id, skill_text, position) VALUES ($1, $2, $3)`,
			orgID, skill, i)
		require.NoError(t, err, "insert profile_skills")
	}

	// Seed 3 social links so the completion score captures the signal.
	for _, platform := range []string{"github", "linkedin", "twitter"} {
		_, err = db.Exec(`
			INSERT INTO social_links (organization_id, platform, url, persona)
			VALUES ($1, $2, $3, 'freelance')`, orgID, platform, "https://"+platform+".com/alice")
		require.NoError(t, err, "insert social link")
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM profile_skills WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM social_links WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM freelance_pricing WHERE profile_id = $1`, profileID)
		_, _ = db.Exec(`DELETE FROM freelance_profiles WHERE id = $1`, profileID)
		_, _ = db.Exec(`DELETE FROM users WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, ownerID)
	})
	return orgID
}

func TestSearchDocumentRepository_LoadFreelanceSignals(t *testing.T) {
	db := searchTestDB(t)
	orgID := seedFreelanceActor(t, db)

	repo := postgres.NewSearchDocumentRepository(db)
	signals, err := repo.LoadActorSignals(context.Background(), orgID, search.PersonaFreelance)
	require.NoError(t, err)
	require.NotNil(t, signals)

	assert.Equal(t, orgID, signals.OrganizationID)
	assert.Equal(t, search.PersonaFreelance, signals.Persona)
	assert.Equal(t, "Alice Freelance", signals.DisplayName)
	assert.Equal(t, "Senior Full-Stack Developer", signals.Title)
	assert.Equal(t, "Paris", signals.City)
	assert.Equal(t, "FR", signals.CountryCode)
	assert.Contains(t, signals.LanguagesProfessional, "fr")
	assert.ElementsMatch(t, []string{"dev-frontend", "dev-backend"}, signals.ExpertiseDomains)
	assert.Equal(t, "available_now", signals.AvailabilityStatus)
	assert.Equal(t, 3, signals.SocialLinksCount)
	assert.NotZero(t, signals.LastActiveAt)
}

func TestSearchDocumentRepository_LoadSkills(t *testing.T) {
	db := searchTestDB(t)
	orgID := seedFreelanceActor(t, db)

	repo := postgres.NewSearchDocumentRepository(db)
	skills, err := repo.LoadSkills(context.Background(), orgID, search.PersonaFreelance)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]string{"React", "Next.js", "Go", "TypeScript", "PostgreSQL", "Kubernetes"}, skills)
}

func TestSearchDocumentRepository_LoadPricing(t *testing.T) {
	db := searchTestDB(t)
	orgID := seedFreelanceActor(t, db)

	repo := postgres.NewSearchDocumentRepository(db)
	pricing, err := repo.LoadPricing(context.Background(), orgID, search.PersonaFreelance)
	require.NoError(t, err)
	require.NotNil(t, pricing)

	assert.True(t, pricing.HasPricing)
	assert.Equal(t, "daily", pricing.Type)
	assert.Equal(t, "EUR", pricing.Currency)
	assert.True(t, pricing.Negotiable)
	require.NotNil(t, pricing.MinAmount)
	assert.Equal(t, int64(50000), *pricing.MinAmount)
}

func TestSearchDocumentRepository_LoadRatingAggregate_Empty(t *testing.T) {
	db := searchTestDB(t)
	orgID := seedFreelanceActor(t, db)

	repo := postgres.NewSearchDocumentRepository(db)
	agg, err := repo.LoadRatingAggregate(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, 0, agg.Count)
	assert.Equal(t, 0.0, agg.Average)
}

func TestSearchDocumentRepository_LoadVerificationStatus(t *testing.T) {
	db := searchTestDB(t)
	orgID := seedFreelanceActor(t, db)

	repo := postgres.NewSearchDocumentRepository(db)
	verified, err := repo.LoadVerificationStatus(context.Background(), orgID)
	require.NoError(t, err)
	assert.True(t, verified, "the seeded org has stripe_account_id so it's considered verified")
}

// TestSearchDocumentRepository_BuildDocumentEndToEnd exercises the
// full indexer loop against a real DB using the mock embeddings
// client. Confirms the adapter + indexer pipeline produces a
// document that passes Validate().
func TestSearchDocumentRepository_BuildDocumentEndToEnd(t *testing.T) {
	db := searchTestDB(t)
	orgID := seedFreelanceActor(t, db)

	repo := postgres.NewSearchDocumentRepository(db)
	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
	require.NoError(t, err)

	doc, err := idx.BuildDocument(context.Background(), orgID, search.PersonaFreelance)
	require.NoError(t, err)
	require.NoError(t, doc.Validate())

	// Phase 3 onwards, the document ID is composite `{orgID}:{persona}`
	// so a single org can carry both a freelance and a referrer doc
	// without collision. OrganizationID (string) is exposed separately
	// so delete-by-org filters can match both docs at once.
	assert.Equal(t, orgID.String()+":freelance", doc.ID)
	assert.Equal(t, orgID.String(), doc.OrganizationID)
	assert.Equal(t, "Alice Freelance", doc.DisplayName)
	assert.True(t, doc.IsVerified)
	assert.Equal(t, "Paris", doc.City)
	assert.Equal(t, search.AvailabilityPriorityNow, doc.AvailabilityPriority)
	assert.Greater(t, doc.ProfileCompletionScore, int32(90), "fully-seeded profile scores near 100")
	assert.Len(t, doc.Embedding, search.EmbeddingDimensions)
}

// TestSearchDocumentRepository_LoadActorSignals_UnknownOrg verifies
// the repository surfaces a clean "not found" error instead of
// silently returning a zero-value struct.
func TestSearchDocumentRepository_LoadActorSignals_UnknownOrg(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewSearchDocumentRepository(db)

	_, err := repo.LoadActorSignals(context.Background(), uuid.New(), search.PersonaFreelance)
	assert.ErrorContains(t, err, "not found")
}

// TestSearchDocumentRepository_LoadPricing_Missing verifies that
// an actor without a pricing row returns HasPricing=false without
// erroring out.
func TestSearchDocumentRepository_LoadPricing_Missing(t *testing.T) {
	db := searchTestDB(t)
	// Seed an org + freelance profile but no pricing row.
	orgID := uuid.New()
	ownerID := uuid.New()
	email := orgID.String()[:8] + "@nopricing.test"
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
		VALUES ($1, $2, 'h', 'A', 'B', 'A B', 'provider', 'marketplace_owner')`, ownerID, email)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, 'provider_personal', 'No Pricing')`, orgID, ownerID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, ownerID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO freelance_profiles (organization_id) VALUES ($1)`, orgID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM freelance_profiles WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM users WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, ownerID)
	})

	repo := postgres.NewSearchDocumentRepository(db)
	pricing, err := repo.LoadPricing(context.Background(), orgID, search.PersonaFreelance)
	require.NoError(t, err)
	assert.False(t, pricing.HasPricing)
}
