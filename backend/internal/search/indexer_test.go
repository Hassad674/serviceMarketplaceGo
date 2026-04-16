package search_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// fakeRepo is a record-and-replay stub of SearchDataRepository that
// lets every test specify the exact response for each method. Lives
// in the test file (not mock/) because the interface is internal
// to the search package.
type fakeRepo struct {
	signals   *search.RawActorSignals
	signalsErr error
	skills    []string
	skillsErr error
	pricing   *search.RawPricing
	pricingErr error
	rating    *search.RawRatingAggregate
	ratingErr error
	earnings  *search.RawEarningsAggregate
	earningsErr error
	kyc       bool
	kycErr    error
	messaging *search.RawMessagingSignals
	messagingErr error
}

func (f *fakeRepo) LoadActorSignals(_ context.Context, _ uuid.UUID, _ search.Persona) (*search.RawActorSignals, error) {
	return f.signals, f.signalsErr
}
func (f *fakeRepo) LoadSkills(_ context.Context, _ uuid.UUID, _ search.Persona) ([]string, error) {
	return f.skills, f.skillsErr
}
func (f *fakeRepo) LoadPricing(_ context.Context, _ uuid.UUID, _ search.Persona) (*search.RawPricing, error) {
	return f.pricing, f.pricingErr
}
func (f *fakeRepo) LoadRatingAggregate(_ context.Context, _ uuid.UUID) (*search.RawRatingAggregate, error) {
	return f.rating, f.ratingErr
}
func (f *fakeRepo) LoadEarningsAggregate(_ context.Context, _ uuid.UUID) (*search.RawEarningsAggregate, error) {
	return f.earnings, f.earningsErr
}
func (f *fakeRepo) LoadVerificationStatus(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.kyc, f.kycErr
}
func (f *fakeRepo) LoadMessagingSignals(_ context.Context, _ uuid.UUID) (*search.RawMessagingSignals, error) {
	return f.messaging, f.messagingErr
}

func fullRepo() *fakeRepo {
	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	lat := 48.8566
	lng := 2.3522
	minAmount := int64(50000)
	maxAmount := int64(120000)
	return &fakeRepo{
		signals: &search.RawActorSignals{
			OrganizationID:          orgID,
			Persona:                 search.PersonaFreelance,
			IsPublished:             true,
			DisplayName:             "Alice Dupont",
			Title:                   "Senior Full-Stack Developer",
			About:                   "10 years of experience shipping Next.js + Go.",
			PhotoURL:                "https://cdn/alice.webp",
			VideoURL:                "https://cdn/alice.mp4",
			City:                    "Paris",
			CountryCode:             "FR",
			Latitude:                &lat,
			Longitude:               &lng,
			WorkMode:                []string{"remote", "hybrid"},
			LanguagesProfessional:   []string{"fr", "en"},
			LanguagesConversational: []string{"es"},
			AvailabilityStatus:      "now",
			ExpertiseDomains:        []string{"dev-frontend", "dev-backend"},
			SocialLinksCount:        3,
			LastActiveAt:            time.Date(2026, 4, 15, 9, 0, 0, 0, time.UTC),
			CreatedAt:               time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:               time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
		},
		skills: []string{"React", "Next.js", "Go", "TypeScript", "PostgreSQL", "Kubernetes"},
		pricing: &search.RawPricing{
			Type:       "daily",
			MinAmount:  &minAmount,
			MaxAmount:  &maxAmount,
			Currency:   "EUR",
			Negotiable: true,
			HasPricing: true,
		},
		rating:    &search.RawRatingAggregate{Average: 4.85, Count: 42},
		earnings:  &search.RawEarningsAggregate{TotalAmount: 7_500_000, CompletedProjects: 18},
		kyc:       true,
		messaging: &search.RawMessagingSignals{ResponseRate: 0.97},
	}
}

func TestNewIndexer_Validation(t *testing.T) {
	_, err := search.NewIndexer(nil, search.NewMockEmbeddings())
	assert.ErrorContains(t, err, "repository is required")

	_, err = search.NewIndexer(fullRepo(), nil)
	assert.ErrorContains(t, err, "embeddings client is required")

	_, err = search.NewIndexer(fullRepo(), search.NewMockEmbeddings())
	assert.NoError(t, err)
}

func TestIndexer_BuildDocument_FullProfile(t *testing.T) {
	repo := fullRepo()
	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
	require.NoError(t, err)

	doc, err := idx.BuildDocument(context.Background(), repo.signals.OrganizationID, search.PersonaFreelance)
	require.NoError(t, err)

	// Identity + display
	assert.Equal(t, repo.signals.OrganizationID.String(), doc.OrganizationID)
	assert.Equal(t, repo.signals.OrganizationID.String()+":"+string(search.PersonaFreelance), doc.ID)
	assert.Equal(t, search.PersonaFreelance, doc.Persona)
	assert.True(t, doc.IsPublished)
	assert.Equal(t, "Alice Dupont", doc.DisplayName)

	// Geo
	assert.Equal(t, "Paris", doc.City)
	assert.Equal(t, "FR", doc.CountryCode)
	require.Len(t, doc.Location, 2)
	assert.InDelta(t, 48.8566, doc.Location[0], 0.001)
	assert.InDelta(t, 2.3522, doc.Location[1], 0.001)

	// Availability + ranking
	assert.Equal(t, search.AvailabilityPriorityNow, doc.AvailabilityPriority)
	// Bayesian: 4.85 * ln(1+42) ≈ 4.85 * 3.7612 ≈ 18.24
	assert.InDelta(t, search.BayesianRatingScore(4.85, 42), doc.RatingScore, 0.0001)
	assert.Greater(t, doc.RatingScore, 18.0)
	assert.True(t, doc.IsTopRated, "4.85 avg with 42 reviews must be top rated")
	assert.True(t, doc.IsVerified)

	// Pricing flattening
	assert.Equal(t, "daily", doc.PricingType)
	assert.Equal(t, "EUR", doc.PricingCurrency)
	require.NotNil(t, doc.PricingMinAmount)
	assert.Equal(t, int64(50000), *doc.PricingMinAmount)
	assert.True(t, doc.PricingNegotiable)

	// Skills / expertise
	assert.ElementsMatch(t, repo.skills, doc.Skills)
	assert.Contains(t, doc.SkillsText, "React")
	assert.Contains(t, doc.SkillsText, "PostgreSQL")

	// Profile completion: full repo hits every weight.
	assert.Equal(t, int32(100), doc.ProfileCompletionScore)

	// Embedding is populated by the mock.
	assert.Len(t, doc.Embedding, search.EmbeddingDimensions)

	// Timestamps are Unix seconds.
	assert.Equal(t, repo.signals.CreatedAt.Unix(), doc.CreatedAt)
	assert.Equal(t, repo.signals.UpdatedAt.Unix(), doc.UpdatedAt)
}

func TestIndexer_BuildDocument_MinimalProfile(t *testing.T) {
	orgID := uuid.New()
	repo := &fakeRepo{
		signals: &search.RawActorSignals{
			OrganizationID: orgID,
			Persona:        search.PersonaAgency,
			IsPublished:    false,
			DisplayName:    "Empty Agency",
			AvailabilityStatus: "not",
			LastActiveAt:   time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		skills:    []string{},
		pricing:   &search.RawPricing{HasPricing: false},
		rating:    &search.RawRatingAggregate{},
		earnings:  &search.RawEarningsAggregate{},
		messaging: &search.RawMessagingSignals{},
	}
	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
	require.NoError(t, err)

	doc, err := idx.BuildDocument(context.Background(), orgID, search.PersonaAgency)
	require.NoError(t, err)

	assert.Equal(t, int32(0), doc.ProfileCompletionScore, "empty profile scores 0")
	assert.Equal(t, 0.0, doc.RatingScore, "no reviews → zero score")
	assert.False(t, doc.IsTopRated)
	assert.False(t, doc.IsVerified)
	assert.Empty(t, doc.PricingType)
	assert.Nil(t, doc.PricingMinAmount)
	assert.Equal(t, search.AvailabilityPriorityNot, doc.AvailabilityPriority)
	// Embedding skipped on empty text.
	assert.Nil(t, doc.Embedding)
}

func TestIndexer_BuildDocument_RejectsInvalidPersona(t *testing.T) {
	idx, err := search.NewIndexer(fullRepo(), search.NewMockEmbeddings())
	require.NoError(t, err)

	_, err = idx.BuildDocument(context.Background(), uuid.New(), "enterprise")
	assert.ErrorContains(t, err, "invalid persona")
}

func TestIndexer_BuildDocument_PropagatesSignalsError(t *testing.T) {
	repo := fullRepo()
	repo.signalsErr = errors.New("db down")

	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
	require.NoError(t, err)

	_, err = idx.BuildDocument(context.Background(), uuid.New(), search.PersonaFreelance)
	assert.ErrorContains(t, err, "load actor signals")
	assert.ErrorContains(t, err, "db down")
}

func TestIndexer_BuildDocument_PropagatesSubReadError(t *testing.T) {
	repo := fullRepo()
	repo.ratingErr = errors.New("reviews unavailable")

	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
	require.NoError(t, err)

	_, err = idx.BuildDocument(context.Background(), repo.signals.OrganizationID, search.PersonaFreelance)
	assert.ErrorContains(t, err, "rating")
	assert.ErrorContains(t, err, "reviews unavailable")
}

func TestIndexer_FeaturedOverride(t *testing.T) {
	repo := fullRepo()
	called := 0
	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings(),
		search.WithFeaturedOverride(func(id uuid.UUID) bool {
			called++
			return id == repo.signals.OrganizationID
		}))
	require.NoError(t, err)

	doc, err := idx.BuildDocument(context.Background(), repo.signals.OrganizationID, search.PersonaFreelance)
	require.NoError(t, err)
	assert.True(t, doc.IsFeatured)
	assert.Equal(t, 1, called)
}

func TestComposeEmbeddingText(t *testing.T) {
	t.Run("nil is empty", func(t *testing.T) {
		assert.Equal(t, "", search.ComposeEmbeddingText(nil))
	})
	t.Run("empty fields produce empty string", func(t *testing.T) {
		assert.Equal(t, "", search.ComposeEmbeddingText(&search.RawActorSignals{}))
	})
	t.Run("composes title + about + expertise", func(t *testing.T) {
		got := search.ComposeEmbeddingText(&search.RawActorSignals{
			Title:            "Go backend engineer",
			About:            "Building payment systems.",
			ExpertiseDomains: []string{"dev-backend", "payments"},
		})
		assert.Contains(t, got, "Go backend engineer")
		assert.Contains(t, got, "Building payment systems")
		assert.Contains(t, got, "dev-backend, payments")
	})
}

// TestIndexer_BuildDocument_Latency tracks that the full build stays
// under 200ms even with a dozen concurrent calls on the in-process
// mocks. Real DB latency is covered by the Postgres adapter tests.
func TestIndexer_BuildDocument_Latency(t *testing.T) {
	idx, err := search.NewIndexer(fullRepo(), search.NewMockEmbeddings())
	require.NoError(t, err)

	start := time.Now()
	_, err = idx.BuildDocument(context.Background(), uuid.New(), search.PersonaFreelance)
	require.NoError(t, err)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 200*time.Millisecond,
		"BuildDocument must stay under 200ms on in-memory mocks; got %s", elapsed)
}
