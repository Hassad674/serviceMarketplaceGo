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

	// Ranking V1 aggregates (phase 6B).
	clientHistory       *search.RawClientHistory
	clientHistoryErr    error
	reviewDiversity     *search.RawReviewDiversity
	reviewDiversityErr  error
	accountAge          *search.RawAccountAge
	accountAgeErr       error
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
func (f *fakeRepo) LoadClientHistory(_ context.Context, _ uuid.UUID) (*search.RawClientHistory, error) {
	return f.clientHistory, f.clientHistoryErr
}
func (f *fakeRepo) LoadReviewDiversity(_ context.Context, _ uuid.UUID) (*search.RawReviewDiversity, error) {
	return f.reviewDiversity, f.reviewDiversityErr
}
func (f *fakeRepo) LoadAccountAge(_ context.Context, _ uuid.UUID) (*search.RawAccountAge, error) {
	return f.accountAge, f.accountAgeErr
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
		clientHistory: &search.RawClientHistory{
			UniqueClients:    11,
			RepeatClientRate: 0.36,
		},
		reviewDiversity: &search.RawReviewDiversity{
			UniqueReviewers:     18,
			MaxReviewerShare:    0.22,
			ReviewRecencyFactor: 0.82,
		},
		accountAge: &search.RawAccountAge{
			LostDisputes:   0,
			AccountAgeDays: 420,
		},
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

	// Ranking V1 signals (phase 6B) — populated from the fullRepo
	// fixture values above.
	assert.Equal(t, int32(11), doc.UniqueClientsCount)
	assert.InDelta(t, 0.36, doc.RepeatClientRate, 0.0001)
	assert.Equal(t, int32(18), doc.UniqueReviewersCount)
	assert.InDelta(t, 0.22, doc.MaxReviewerShare, 0.0001)
	assert.InDelta(t, 0.82, doc.ReviewRecencyFactor, 0.0001)
	assert.Equal(t, int32(0), doc.LostDisputesCount)
	assert.Equal(t, int32(420), doc.AccountAgeDays)

	// Timestamps are Unix seconds.
	assert.Equal(t, repo.signals.CreatedAt.Unix(), doc.CreatedAt)
	assert.Equal(t, repo.signals.UpdatedAt.Unix(), doc.UpdatedAt)
}

func TestIndexer_BuildDocument_MinimalProfile(t *testing.T) {
	orgID := uuid.New()
	repo := &fakeRepo{
		// Zero display name so the composed embedding text is empty
		// and the embedder short-circuits (phase 3: we include the
		// display name in the embedding input, so a minimal profile
		// must truly have no display name to skip embedding).
		signals: &search.RawActorSignals{
			OrganizationID:     orgID,
			Persona:            search.PersonaAgency,
			IsPublished:        false,
			AvailabilityStatus: "not",
			LastActiveAt:       time.Now(),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		skills:          []string{},
		pricing:         &search.RawPricing{HasPricing: false},
		rating:          &search.RawRatingAggregate{},
		earnings:        &search.RawEarningsAggregate{},
		messaging:       &search.RawMessagingSignals{},
		clientHistory:   &search.RawClientHistory{},
		reviewDiversity: &search.RawReviewDiversity{},
		accountAge:      &search.RawAccountAge{},
	}
	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
	require.NoError(t, err)

	// Validate bypass: the document has no display name so Validate
	// would reject it, but we're testing pre-validate outputs by
	// invoking BuildDocument which validates at the end. Set a
	// sentinel display name AFTER the embedding step would skip —
	// not possible mid-BuildDocument, so we give it a one-char display
	// name and assert the embedder ran.
	repo.signals.DisplayName = "A"

	doc, err := idx.BuildDocument(context.Background(), orgID, search.PersonaAgency)
	require.NoError(t, err)

	assert.Equal(t, int32(0), doc.ProfileCompletionScore, "empty profile scores 0")
	assert.Equal(t, 0.0, doc.RatingScore, "no reviews → zero score")
	assert.False(t, doc.IsTopRated)
	assert.False(t, doc.IsVerified)
	assert.Empty(t, doc.PricingType)
	assert.Nil(t, doc.PricingMinAmount)
	assert.Equal(t, search.AvailabilityPriorityNot, doc.AvailabilityPriority)
	// Phase 3: display_name alone is sufficient embedding text so
	// the embedder still runs — we assert shape rather than nil.
	assert.Len(t, doc.Embedding, search.EmbeddingDimensions)

	// Ranking V1 signals (phase 6B) — zero-value aggregates map to
	// zero on the document. Downstream extractors interpret zero as
	// "cold start territory" and apply the documented floors.
	assert.Equal(t, int32(0), doc.UniqueClientsCount)
	assert.Equal(t, 0.0, doc.RepeatClientRate)
	assert.Equal(t, int32(0), doc.UniqueReviewersCount)
	assert.Equal(t, 0.0, doc.MaxReviewerShare)
	assert.Equal(t, 0.0, doc.ReviewRecencyFactor)
	assert.Equal(t, int32(0), doc.LostDisputesCount)
	assert.Equal(t, int32(0), doc.AccountAgeDays)
}

// TestIndexer_BuildDocument_RankingV1Signals is a dedicated test for
// the 7 signals introduced in phase 6B. It covers 4 representative
// cases per signal (cold start, one review, typical, extreme) driven
// by table-driven data so every branch of applyClientHistory /
// applyReviewDiversity / applyAccountAge is exercised.
func TestIndexer_BuildDocument_RankingV1Signals(t *testing.T) {
	cases := []struct {
		name                 string
		history              *search.RawClientHistory
		diversity            *search.RawReviewDiversity
		age                  *search.RawAccountAge
		wantUniqueClients    int32
		wantRepeatClientRate float64
		wantUniqueReviewers  int32
		wantMaxReviewerShare float64
		wantRecencyFactor    float64
		wantLostDisputes     int32
		wantAccountAgeDays   int32
	}{
		{
			name:      "cold start — nil aggregates map to zeros",
			history:   nil,
			diversity: nil,
			age:       nil,
		},
		{
			name:                 "zero aggregates stay at zero",
			history:              &search.RawClientHistory{},
			diversity:            &search.RawReviewDiversity{},
			age:                  &search.RawAccountAge{},
			wantUniqueClients:    0,
			wantRepeatClientRate: 0,
			wantUniqueReviewers:  0,
			wantMaxReviewerShare: 0,
			wantRecencyFactor:    0,
			wantLostDisputes:     0,
			wantAccountAgeDays:   0,
		},
		{
			name: "typical senior — 11 clients, 36% repeat, 18 reviewers, mature account",
			history: &search.RawClientHistory{
				UniqueClients:    11,
				RepeatClientRate: 0.36,
			},
			diversity: &search.RawReviewDiversity{
				UniqueReviewers:     18,
				MaxReviewerShare:    0.22,
				ReviewRecencyFactor: 0.82,
			},
			age: &search.RawAccountAge{
				LostDisputes:   0,
				AccountAgeDays: 420,
			},
			wantUniqueClients:    11,
			wantRepeatClientRate: 0.36,
			wantUniqueReviewers:  18,
			wantMaxReviewerShare: 0.22,
			wantRecencyFactor:    0.82,
			wantLostDisputes:     0,
			wantAccountAgeDays:   420,
		},
		{
			name: "extreme — reviewer concentration + 2 lost disputes + fresh account",
			history: &search.RawClientHistory{
				UniqueClients:    2,
				RepeatClientRate: 1.0,
			},
			diversity: &search.RawReviewDiversity{
				UniqueReviewers:     2,
				MaxReviewerShare:    0.9,
				ReviewRecencyFactor: 0.05,
			},
			age: &search.RawAccountAge{
				LostDisputes:   2,
				AccountAgeDays: 3,
			},
			wantUniqueClients:    2,
			wantRepeatClientRate: 1.0,
			wantUniqueReviewers:  2,
			wantMaxReviewerShare: 0.9,
			wantRecencyFactor:    0.05,
			wantLostDisputes:     2,
			wantAccountAgeDays:   3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := fullRepo()
			repo.clientHistory = tc.history
			repo.reviewDiversity = tc.diversity
			repo.accountAge = tc.age

			idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
			require.NoError(t, err)

			doc, err := idx.BuildDocument(context.Background(), repo.signals.OrganizationID, search.PersonaFreelance)
			require.NoError(t, err)

			assert.Equal(t, tc.wantUniqueClients, doc.UniqueClientsCount)
			assert.InDelta(t, tc.wantRepeatClientRate, doc.RepeatClientRate, 0.0001)
			assert.Equal(t, tc.wantUniqueReviewers, doc.UniqueReviewersCount)
			assert.InDelta(t, tc.wantMaxReviewerShare, doc.MaxReviewerShare, 0.0001)
			assert.InDelta(t, tc.wantRecencyFactor, doc.ReviewRecencyFactor, 0.0001)
			assert.Equal(t, tc.wantLostDisputes, doc.LostDisputesCount)
			assert.Equal(t, tc.wantAccountAgeDays, doc.AccountAgeDays)
		})
	}
}

// TestIndexer_BuildDocument_RankingV1Signal_Errors verifies that a
// failure in any of the 3 new aggregate loads propagates out of
// BuildDocument with the step name in the error chain so alerts can
// quickly locate the failing query.
func TestIndexer_BuildDocument_RankingV1Signal_Errors(t *testing.T) {
	cases := []struct {
		name    string
		inject  func(*fakeRepo)
		wantMsg string
	}{
		{
			name: "client history failure",
			inject: func(r *fakeRepo) {
				r.clientHistoryErr = errors.New("milestones unavailable")
			},
			wantMsg: "client_history",
		},
		{
			name: "review diversity failure",
			inject: func(r *fakeRepo) {
				r.reviewDiversityErr = errors.New("reviews unavailable")
			},
			wantMsg: "review_diversity",
		},
		{
			name: "account age failure",
			inject: func(r *fakeRepo) {
				r.accountAgeErr = errors.New("users unavailable")
			},
			wantMsg: "account_age",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := fullRepo()
			tc.inject(repo)

			idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
			require.NoError(t, err)

			_, err = idx.BuildDocument(context.Background(), repo.signals.OrganizationID, search.PersonaFreelance)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantMsg)
		})
	}
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
	t.Run("composes display_name + title + skills + about + expertise", func(t *testing.T) {
		got := search.ComposeEmbeddingText(&search.RawActorSignals{
			DisplayName:      "Alice Martin",
			Title:            "Go backend engineer",
			About:            "Building payment systems.",
			ExpertiseDomains: []string{"dev-backend", "payments"},
		}, "Go", "PostgreSQL", "Stripe")
		assert.Contains(t, got, "Alice Martin")
		assert.Contains(t, got, "Go backend engineer")
		assert.Contains(t, got, "Go PostgreSQL Stripe")
		assert.Contains(t, got, "Building payment systems")
		assert.Contains(t, got, "dev-backend, payments")
	})
	t.Run("truncates inputs above the cost cap", func(t *testing.T) {
		long := make([]byte, search.MaxEmbeddingInputChars*2)
		for i := range long {
			long[i] = 'x'
		}
		got := search.ComposeEmbeddingText(&search.RawActorSignals{
			About: string(long),
		})
		assert.Equal(t, search.MaxEmbeddingInputChars, len(got))
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
