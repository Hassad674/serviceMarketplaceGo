package features

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration-style test : a realistic doc flows through Extract and yields a
// sensible, in-range Features vector.
func TestDefaultExtractor_Extract_WorkedExample(t *testing.T) {
	cfg := DefaultConfig()
	ext := NewDefaultExtractor(cfg)

	// Mirrors the §12.1 worked trace — "Romain Durand", Paris, 14 projects,
	// 11 unique clients, 36% repeat, KYC verified, 420 days, 23 reviews.
	const now = 1_700_000_000
	doc := SearchDocumentLite{
		OrganizationID:         "11111111-1111-1111-1111-111111111111",
		Persona:                PersonaFreelance,
		Skills:                 []string{"react", "flutter", "typescript", "paris"},
		SkillsText:             "react flutter typescript paris",
		About:                  "Senior developer based in Paris",
		RatingAverage:          4.7,
		RatingCount:            23,
		CompletedProjects:      14,
		ProfileCompletionScore: 88,
		LastActiveAt:           now - 6*86400,
		ResponseRate:           0.91,
		IsVerified:             true,
		UniqueClientsCount:     11,
		RepeatClientRate:       0.36,
		UniqueReviewersCount:   18,
		MaxReviewerShare:       1.0 / 18.0,
		ReviewRecencyFactor:    0.82,
		LostDisputesCount:      0,
		AccountAgeDays:         420,
		NowUnix:                now,
		TextMatchBucket:        8,
	}
	q := Query{
		Text:             "développeur React Paris senior",
		NormalisedTokens: []string{"react", "paris", "senior"},
		FilterSkills:     []string{"React"},
		Persona:          PersonaFreelance,
	}

	f := ext.Extract(q, doc)

	// Every positive feature within bounds.
	for name, v := range map[string]float64{
		"TextMatchScore":      f.TextMatchScore,
		"SkillsOverlapRatio":  f.SkillsOverlapRatio,
		"RatingScoreDiverse":  f.RatingScoreDiverse,
		"ProvenWorkScore":     f.ProvenWorkScore,
		"ResponseRate":        f.ResponseRate,
		"IsVerifiedMature":    f.IsVerifiedMature,
		"ProfileCompletion":   f.ProfileCompletion,
		"LastActiveDaysScore": f.LastActiveDaysScore,
		"AccountAgeBonus":     f.AccountAgeBonus,
	} {
		assert.GreaterOrEqual(t, v, 0.0, "%s < 0", name)
		assert.LessOrEqual(t, v, 1.0, "%s > 1", name)
	}

	// Penalty in [0, cap].
	assert.GreaterOrEqual(t, f.NegativeSignals, 0.0)
	assert.LessOrEqual(t, f.NegativeSignals, cfg.DisputePenaltyCap)

	// Raw signals faithfully mirror the doc (save for clamp).
	assert.Equal(t, 8, f.RawTextMatchBucket)
	assert.Equal(t, 18, f.RawUniqueReviewers)
	assert.Equal(t, 0, f.RawLostDisputes)
	assert.Equal(t, 420, f.RawAccountAgeDays)

	// Spot-check a few expected values.
	assert.InDelta(t, 0.8, f.TextMatchScore, 1e-9)
	assert.InDelta(t, 1.0, f.IsVerifiedMature, 1e-9)
	assert.InDelta(t, 0.91, f.ResponseRate, 1e-9)
	assert.InDelta(t, 0.88, f.ProfileCompletion, 1e-9)
	assert.Greater(t, f.ProvenWorkScore, 0.3)
	assert.Greater(t, f.AccountAgeBonus, 0.99) // capped
	assert.InDelta(t, 0.0, f.NegativeSignals, 1e-9)
}

// Referrer-persona docs always have SkillsOverlapRatio and ProvenWorkScore
// equal to 0 regardless of skill overlap or project count.
func TestDefaultExtractor_Extract_ReferrerForcesZero(t *testing.T) {
	cfg := DefaultConfig()
	ext := NewDefaultExtractor(cfg)

	doc := SearchDocumentLite{
		Persona:            PersonaReferrer,
		Skills:             []string{"react", "flutter"},
		CompletedProjects:  50,
		UniqueClientsCount: 25,
		RepeatClientRate:   0.5,
	}
	q := Query{
		Persona:          PersonaReferrer,
		NormalisedTokens: []string{"react"},
	}
	f := ext.Extract(q, doc)
	assert.Equal(t, 0.0, f.SkillsOverlapRatio)
	assert.Equal(t, 0.0, f.ProvenWorkScore)
}

// Cold-start profile : no reviews, no projects, fresh account. The extractor
// must still produce a valid vector and NOT panic.
func TestDefaultExtractor_Extract_ColdStart(t *testing.T) {
	cfg := DefaultConfig()
	ext := NewDefaultExtractor(cfg)

	doc := SearchDocumentLite{
		Persona:        PersonaFreelance,
		AccountAgeDays: 1,
		NowUnix:        1_700_000_000,
		LastActiveAt:   1_700_000_000,
	}
	q := Query{Persona: PersonaFreelance}

	f := ext.Extract(q, doc)
	assert.InDelta(t, cfg.ColdStartFloor, f.RatingScoreDiverse, 1e-9)
	assert.Equal(t, 0.0, f.ProvenWorkScore)
	assert.Equal(t, 0.0, f.SkillsOverlapRatio)
	assert.Equal(t, 0.0, f.IsVerifiedMature)
	assert.InDelta(t, 1.0, f.LastActiveDaysScore, 1e-9)
}

// Property test — every feature stays in [0, 1] across 500 random inputs,
// and NegativeSignals stays in [0, cap].
func TestDefaultExtractor_Extract_PropertyBounds(t *testing.T) {
	cfg := DefaultConfig()
	ext := NewDefaultExtractor(cfg)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 500; i++ {
		doc := randomDoc(rng)
		q := randomQuery(rng)
		f := ext.Extract(q, doc)
		assertFeaturesInBounds(t, f, cfg)
	}
}

// Property test — determinism : same inputs produce identical outputs 100
// iterations in a row.
func TestDefaultExtractor_Extract_Deterministic(t *testing.T) {
	cfg := DefaultConfig()
	ext := NewDefaultExtractor(cfg)
	rng := rand.New(rand.NewSource(99))

	doc := randomDoc(rng)
	q := randomQuery(rng)
	baseline := ext.Extract(q, doc)
	for i := 0; i < 100; i++ {
		got := ext.Extract(q, doc)
		require.Equal(t, baseline, got,
			"Extract is non-deterministic at iteration %d", i)
	}
}

// ExtractorFunc adapts a bare function — used by the scorer for stubs.
func TestExtractorFunc_SatisfiesInterface(t *testing.T) {
	var called bool
	var f Extractor = ExtractorFunc(func(q Query, d SearchDocumentLite) Features {
		called = true
		return Features{TextMatchScore: 0.42}
	})
	out := f.Extract(Query{}, SearchDocumentLite{})
	assert.True(t, called)
	assert.Equal(t, 0.42, out.TextMatchScore)
}

// Config accessor returns a copy of the immutable config — mutating it must
// not affect subsequent Extract calls.
func TestDefaultExtractor_Config_ReturnsCopy(t *testing.T) {
	ext := NewDefaultExtractor(DefaultConfig())
	cfg := ext.Config()
	cfg.ColdStartFloor = 999
	// The extractor still uses the original cfg.
	doc := SearchDocumentLite{Persona: PersonaFreelance}
	f := ext.Extract(Query{Persona: PersonaFreelance}, doc)
	assert.InDelta(t, DefaultConfig().ColdStartFloor, f.RatingScoreDiverse, 1e-9)
}

// assertFeaturesInBounds validates the invariants every Features value must
// satisfy. Shared helper for property tests + table tests.
func assertFeaturesInBounds(t *testing.T, f Features, cfg Config) {
	t.Helper()
	checks := map[string]float64{
		"TextMatchScore":      f.TextMatchScore,
		"SkillsOverlapRatio":  f.SkillsOverlapRatio,
		"RatingScoreDiverse":  f.RatingScoreDiverse,
		"ProvenWorkScore":     f.ProvenWorkScore,
		"ResponseRate":        f.ResponseRate,
		"IsVerifiedMature":    f.IsVerifiedMature,
		"ProfileCompletion":   f.ProfileCompletion,
		"LastActiveDaysScore": f.LastActiveDaysScore,
		"AccountAgeBonus":     f.AccountAgeBonus,
	}
	for name, v := range checks {
		if math.IsNaN(v) {
			t.Fatalf("%s is NaN", name)
		}
		if v < 0 || v > 1 {
			t.Fatalf("%s = %f outside [0, 1]", name, v)
		}
	}
	if f.NegativeSignals < 0 || f.NegativeSignals > cfg.DisputePenaltyCap {
		t.Fatalf("NegativeSignals = %f outside [0, %f]", f.NegativeSignals, cfg.DisputePenaltyCap)
	}
}

// randomDoc returns a SearchDocumentLite with varied, occasionally extreme
// values — covers the interesting corners of the input space.
func randomDoc(rng *rand.Rand) SearchDocumentLite {
	personas := []Persona{PersonaFreelance, PersonaAgency, PersonaReferrer}
	skills := [][]string{
		nil,
		{"react"},
		{"react", "typescript", "node"},
		{"go", "kubernetes", "terraform", "aws", "postgres"},
	}
	return SearchDocumentLite{
		Persona:                personas[rng.Intn(3)],
		Skills:                 skills[rng.Intn(len(skills))],
		RatingAverage:          rng.Float64() * 5,
		RatingCount:            int32(rng.Intn(100)),
		CompletedProjects:      int32(rng.Intn(150)),
		ProfileCompletionScore: int32(rng.Intn(120) - 10), // occasionally out-of-bounds
		LastActiveAt:           int64(1_600_000_000 + rng.Intn(1_000_000_000)),
		ResponseRate:           rng.Float64() * 1.2, // occasionally > 1
		IsVerified:             rng.Intn(2) == 0,
		UniqueClientsCount:     int32(rng.Intn(80)),
		RepeatClientRate:       rng.Float64() * 1.3, // occasionally > 1
		UniqueReviewersCount:   int32(rng.Intn(60)),
		MaxReviewerShare:       rng.Float64(),
		ReviewRecencyFactor:    rng.Float64(),
		LostDisputesCount:      int32(rng.Intn(8)),
		AccountAgeDays:         int32(rng.Intn(1500)),
		NowUnix:                1_700_000_000,
		TextMatchBucket:        rng.Intn(15) - 2, // occasionally negative / > 10
	}
}

// randomQuery returns a varied Query.
func randomQuery(rng *rand.Rand) Query {
	tokens := [][]string{
		nil,
		{"react"},
		{"react", "senior", "paris"},
		{"go", "devops"},
	}
	personas := []Persona{PersonaFreelance, PersonaAgency, PersonaReferrer}
	return Query{
		Text:             "random",
		NormalisedTokens: tokens[rng.Intn(len(tokens))],
		FilterSkills:     tokens[rng.Intn(len(tokens))],
		Persona:          personas[rng.Intn(3)],
	}
}

// Benchmark — Extract must stay well below 2 µs on a single document. The
// budget is 50 ms for re-ranking 200 docs × extract+score.
func BenchmarkDefaultExtractor_Extract(b *testing.B) {
	cfg := DefaultConfig()
	ext := NewDefaultExtractor(cfg)
	rng := rand.New(rand.NewSource(1))
	doc := randomDoc(rng)
	q := randomQuery(rng)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ext.Extract(q, doc)
	}
}
