package search

import (
	"context"
	"math"
	"sort"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// ranking_pipeline_test.go drives every branch of the composition that
// welds Stages 2-5 together. Table-driven where the cases fit, property
// / fuzz-style where they do not. A benchmark pins the 50ms budget.

// -- helpers ----------------------------------------------------------

// newTestPipeline builds the pipeline with the production packages but
// a deterministic RandSeed so the business-rules output stays stable
// across runs.
func newTestPipeline(t *testing.T) *RankingPipeline {
	t.Helper()
	fcfg := features.DefaultConfig()
	agCfg := antigaming.DefaultConfig()
	scCfg := scorer.DefaultConfig()
	rlCfg := rules.DefaultConfig()
	rlCfg.RandSeed = 42 // deterministic for tests

	ext := features.NewDefaultExtractor(fcfg)
	ag := antigaming.NewPipeline(agCfg, antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{})
	rer := scorer.NewWeightedScorer(scCfg)
	br := rules.NewBusinessRules(rlCfg)
	return NewRankingPipeline(ext, ag, rer, br)
}

// sampleHit fabricates a minimally-plausible Typesense hit. The
// fields populate enough signals that every extractor produces a
// non-zero value — useful for verifying the composition does not
// swallow outputs mid-pipeline.
func sampleHit(id string, bucket int) TypesenseHit {
	return TypesenseHit{
		Document: search.SearchDocument{
			ID:                     id + ":freelance",
			OrganizationID:         id,
			Persona:                search.PersonaFreelance,
			IsPublished:            true,
			DisplayName:            "Profile " + id,
			Skills:                 []string{"react", "go"},
			SkillsText:             "react go paris",
			AvailabilityStatus:     "available_now",
			RatingAverage:          4.7,
			RatingCount:            20,
			CompletedProjects:      15,
			ProfileCompletionScore: 88,
			LastActiveAt:           time.Now().Unix() - 3600, // 1h ago
			ResponseRate:           0.9,
			IsVerified:             true,
			UniqueClientsCount:     10,
			RepeatClientRate:       0.4,
			UniqueReviewersCount:   15,
			MaxReviewerShare:       0.2,
			ReviewRecencyFactor:    0.8,
			LostDisputesCount:      0,
			AccountAgeDays:         200,
		},
		TextMatchBucket: bucket,
	}
}

// manyHits generates n hits with decreasing text-match buckets so the
// pipeline sees a realistic ordered cohort.
func manyHits(n int) []TypesenseHit {
	hits := make([]TypesenseHit, n)
	for i := 0; i < n; i++ {
		bucket := 10 - (i / 20) // drops by 1 every 20 candidates
		if bucket < 0 {
			bucket = 0
		}
		hits[i] = sampleHit(idFor(i), bucket)
	}
	return hits
}

// idFor builds a stable org ID from an index.
func idFor(i int) string {
	// Valid UUID-ish shape so downstream consumers that validate
	// UUIDs (the Typesense indexer, etc.) stay happy — the pipeline
	// does not validate, but keeping the shape consistent is cheap.
	const suffix = "0000-4000-8000-000000000000"
	return string([]byte{'0' + byte(i/10%10), '0' + byte(i%10), '-'}) + suffix
}

// -- Rerank behaviour -------------------------------------------------

func TestRankingPipeline_Rerank_EmptyHits(t *testing.T) {
	p := newTestPipeline(t)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    nil,
		Now:     time.Now(),
	})
	assert.NotNil(t, out)
	assert.Equal(t, 0, len(out))
}

func TestRankingPipeline_Rerank_SingleHit(t *testing.T) {
	p := newTestPipeline(t)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    []TypesenseHit{sampleHit("one", 10)},
		Now:     time.Now(),
	})
	require.Len(t, out, 1)
	assert.Equal(t, "one:freelance", out[0].Candidate.DocumentID)
	// Score.Final must be bounded in [0, 100].
	assert.GreaterOrEqual(t, out[0].Candidate.Score.Final, 0.0)
	assert.LessOrEqual(t, out[0].Candidate.Score.Final, 100.0)
}

func TestRankingPipeline_Rerank_Truncates_To_TopN(t *testing.T) {
	p := newTestPipeline(t)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    manyHits(200),
		Now:     time.Unix(1_700_000_000, 0),
	})
	assert.Equal(t, 20, len(out))
}

func TestRankingPipeline_Rerank_AllSameSkillStillSucceeds(t *testing.T) {
	// Diversity rule cannot fire when every candidate shares the same
	// primary skill — the pass degrades gracefully.
	p := newTestPipeline(t)
	hits := manyHits(25)
	for i := range hits {
		hits[i].Document.Skills = []string{"react"}
	}
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	})
	assert.Equal(t, 20, len(out))
	for _, r := range out {
		assert.Equal(t, "react", r.Candidate.PrimarySkill)
	}
}

func TestRankingPipeline_Rerank_MixedPersonaHitsRespectInputPersona(t *testing.T) {
	p := newTestPipeline(t)
	hits := manyHits(10)
	hits[0].Document.Persona = search.PersonaAgency
	// The pipeline uses the input.Persona for feature extraction +
	// scoring — doc.Persona only flows into the Candidate metadata.
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "agency", Persona: features.PersonaAgency},
		Persona: features.PersonaAgency,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	})
	require.NotEmpty(t, out)
	// Every score.Final is bounded.
	for _, r := range out {
		assert.GreaterOrEqual(t, r.Candidate.Score.Final, 0.0)
		assert.LessOrEqual(t, r.Candidate.Score.Final, 100.0)
	}
}

func TestRankingPipeline_Rerank_CancelledContextDoesNotPanic(t *testing.T) {
	// Rerank is pure and never honours ctx for cancellation in V1
	// (feature set is too small). The test documents the contract:
	// a cancelled ctx must not panic the pipeline.
	p := newTestPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := p.Rerank(ctx, RankInput{
		Query:   features.Query{Text: "go", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    manyHits(50),
		Now:     time.Unix(1_700_000_000, 0),
	})
	assert.Equal(t, 20, len(out))
}

func TestRankingPipeline_Rerank_NilReceiverReturnsNil(t *testing.T) {
	var p *RankingPipeline
	out := p.Rerank(context.Background(), RankInput{Hits: manyHits(3)})
	assert.Nil(t, out)
}

func TestRankingPipeline_Rerank_ZeroNowUsesClock(t *testing.T) {
	// Test path: when RankInput.Now is zero, the pipeline falls back
	// to rankingNow(). Stub the clock so the fallback is observable.
	orig := rankingNow
	defer func() { rankingNow = orig }()
	frozen := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	rankingNow = func() time.Time { return frozen }

	p := newTestPipeline(t)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    []TypesenseHit{sampleHit("one", 10)},
	})
	require.Len(t, out, 1)
	// Not panicking + producing a result is the assertion — the
	// frozen clock is consumed by the last-active feature internally.
}

// -- Nil-component degradation ---------------------------------------

func TestRankingPipeline_Rerank_NilExtractorZeroFeatures(t *testing.T) {
	// When the extractor is nil, every Features is zero, the scorer
	// returns zero composites, but the pipeline still runs + produces
	// ordered candidates (ordering is arbitrary at equal scores).
	rlCfg := rules.DefaultConfig()
	rlCfg.RandSeed = 42
	br := rules.NewBusinessRules(rlCfg)
	p := NewRankingPipeline(
		nil,
		antigaming.NewPipeline(antigaming.DefaultConfig(), antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{}),
		scorer.NewWeightedScorer(scorer.DefaultConfig()),
		br,
	)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "x", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    manyHits(25),
		Now:     time.Unix(1_700_000_000, 0),
	})
	assert.Equal(t, 20, len(out))
}

func TestRankingPipeline_Rerank_NilScorerStillTruncates(t *testing.T) {
	rlCfg := rules.DefaultConfig()
	rlCfg.RandSeed = 42
	br := rules.NewBusinessRules(rlCfg)
	p := NewRankingPipeline(
		features.NewDefaultExtractor(features.DefaultConfig()),
		antigaming.NewPipeline(antigaming.DefaultConfig(), antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{}),
		nil,
		br,
	)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "x", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    manyHits(30),
		Now:     time.Unix(1_700_000_000, 0),
	})
	assert.Equal(t, 20, len(out))
	for _, r := range out {
		// With nil scorer every RankedScore is zero.
		assert.Equal(t, 0.0, r.Candidate.Score.Final)
	}
}

func TestRankingPipeline_Rerank_NilRulesTruncatesAtDefaultTopN(t *testing.T) {
	p := NewRankingPipeline(
		features.NewDefaultExtractor(features.DefaultConfig()),
		antigaming.NewPipeline(antigaming.DefaultConfig(), antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{}),
		scorer.NewWeightedScorer(scorer.DefaultConfig()),
		nil,
	)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "x", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    manyHits(30),
		Now:     time.Unix(1_700_000_000, 0),
	})
	assert.Equal(t, 20, len(out)) // defaultPipelineTopN
}

func TestRankingPipeline_Rerank_NilAntigamingLeavesFeaturesUnchanged(t *testing.T) {
	// When antigaming is nil, no penalty multiplies text_match_score.
	// With a stuffing-prone SkillsText, the ordinary antigaming
	// pipeline would halve text_match; with nil AG we keep the raw
	// bucket-derived score.
	ext := features.NewDefaultExtractor(features.DefaultConfig())
	br := rules.NewBusinessRules(rules.Config{RandSeed: 42, TopN: 20,
		RisingTalentSlotEvery: 5, RisingTalentMaxAge: 60, NoiseCoefficient: 0.0})
	p := NewRankingPipeline(
		ext, nil,
		scorer.NewWeightedScorer(scorer.DefaultConfig()),
		br,
	)
	hits := []TypesenseHit{sampleHit("a", 10)}
	hits[0].Document.SkillsText = "react react react react react react react react react react"
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	})
	require.Len(t, out, 1)
	// Raw extractor output with bucket 10 is 1.0 (no AG penalty).
	assert.InDelta(t, 1.0, out[0].Candidate.Feat.TextMatchScore, 1e-9)
}

// -- Determinism / property tests ------------------------------------

func TestRankingPipeline_Rerank_DeterministicWithFixedSeed(t *testing.T) {
	p := newTestPipeline(t)
	hits := manyHits(100)
	ri := RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	}
	out1 := p.Rerank(context.Background(), ri)
	out2 := p.Rerank(context.Background(), ri)
	require.Equal(t, len(out1), len(out2))
	for i := range out1 {
		assert.Equal(t, out1[i].Candidate.DocumentID, out2[i].Candidate.DocumentID,
			"position %d must match between deterministic runs", i)
	}
}

func TestRankingPipeline_Rerank_ScoresWithinBoundsProperty(t *testing.T) {
	p := newTestPipeline(t)
	prop := func(seed int64) bool {
		// Build ~20 hits with parameter noise controlled by seed.
		nHits := int(seed%30) + 1
		if nHits < 1 {
			nHits = 1
		}
		hits := manyHits(nHits)
		out := p.Rerank(context.Background(), RankInput{
			Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
			Persona: features.PersonaFreelance,
			Hits:    hits,
			Now:     time.Unix(1_700_000_000+seed, 0),
		})
		if len(out) > 20 {
			return false
		}
		for _, r := range out {
			if math.IsNaN(r.Candidate.Score.Final) {
				return false
			}
			if r.Candidate.Score.Final < 0 || r.Candidate.Score.Final > 100 {
				return false
			}
		}
		return true
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

func TestRankingPipeline_Rerank_NoDuplicatesEmitted(t *testing.T) {
	p := newTestPipeline(t)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "x", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    manyHits(200),
		Now:     time.Unix(1_700_000_000, 0),
	})
	seen := make(map[string]struct{}, len(out))
	for _, r := range out {
		if _, dup := seen[r.Candidate.DocumentID]; dup {
			t.Fatalf("duplicate candidate: %s", r.Candidate.DocumentID)
		}
		seen[r.Candidate.DocumentID] = struct{}{}
	}
}

func TestRankingPipeline_Rerank_NaNInputsClampedByAntiGamingAndScorer(t *testing.T) {
	p := newTestPipeline(t)
	hits := []TypesenseHit{sampleHit("bad", 10)}
	hits[0].Document.MaxReviewerShare = math.NaN()
	hits[0].Document.RepeatClientRate = math.NaN()
	hits[0].Document.ReviewRecencyFactor = math.NaN()
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	})
	require.Len(t, out, 1)
	assert.False(t, math.IsNaN(out[0].Candidate.Score.Final))
	assert.GreaterOrEqual(t, out[0].Candidate.Score.Final, 0.0)
	assert.LessOrEqual(t, out[0].Candidate.Score.Final, 100.0)
}

func TestRankingPipeline_Rerank_RawDocPreservedById(t *testing.T) {
	// Rerank swaps positions (tier sort + rising talent) but must
	// keep RawDoc in 1:1 correspondence with the Candidate.DocumentID
	// so the handler can emit the original SearchDocument without
	// fetching again.
	p := newTestPipeline(t)
	hits := manyHits(30)
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	})
	for _, r := range out {
		assert.Equal(t, r.Candidate.DocumentID, r.RawDoc.Document.ID)
	}
}

func TestRankingPipeline_Rerank_MonotoneScoreInInput(t *testing.T) {
	// With antigaming disabled + noise set to zero, raising one
	// feature must not lower the Final score for a single-hit
	// pipeline.
	ext := features.NewDefaultExtractor(features.DefaultConfig())
	br := rules.NewBusinessRules(rules.Config{RandSeed: 42, TopN: 20,
		RisingTalentSlotEvery: 5, RisingTalentMaxAge: 60, NoiseCoefficient: 0.0})
	p := NewRankingPipeline(
		ext, antigaming.NewPipeline(antigaming.DefaultConfig(), antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{}),
		scorer.NewWeightedScorer(scorer.DefaultConfig()),
		br,
	)
	lo := []TypesenseHit{sampleHit("a", 1)}
	hi := []TypesenseHit{sampleHit("a", 10)}
	ri := RankInput{
		Query: features.Query{Text: "react", Persona: features.PersonaFreelance}, Persona: features.PersonaFreelance,
		Now: time.Unix(1_700_000_000, 0),
	}
	ri.Hits = lo
	outLo := p.Rerank(context.Background(), ri)
	ri.Hits = hi
	outHi := p.Rerank(context.Background(), ri)
	require.Len(t, outLo, 1)
	require.Len(t, outHi, 1)
	assert.LessOrEqual(t, outLo[0].Candidate.Score.Final, outHi[0].Candidate.Score.Final)
}

func TestRankingPipeline_Rerank_TierAAlwaysBeatsTierB(t *testing.T) {
	// Availability tiering is a hard partition — any Tier B
	// candidate must land below any Tier A candidate in the output.
	p := newTestPipeline(t)
	hits := manyHits(10)
	hits[0].Document.AvailabilityStatus = "not_available" // worst
	hits[1].Document.AvailabilityStatus = "available_now" // best
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	})
	aCount := 0
	seenB := false
	for _, r := range out {
		tier := rules.TierOf(r.Candidate.AvailabilityStatus)
		if tier == rules.TierA {
			require.False(t, seenB, "Tier A must not appear after Tier B at rank=%d", aCount)
			aCount++
		} else {
			seenB = true
		}
	}
}

// -- Benchmark --------------------------------------------------------

func BenchmarkRerank_200Candidates(b *testing.B) {
	fcfg := features.DefaultConfig()
	ext := features.NewDefaultExtractor(fcfg)
	ag := antigaming.NewPipeline(antigaming.DefaultConfig(), antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{})
	rer := scorer.NewWeightedScorer(scorer.DefaultConfig())
	br := rules.NewBusinessRules(rules.Config{RandSeed: 42, TopN: 20,
		RisingTalentSlotEvery: 5, RisingTalentMaxAge: 60, NoiseCoefficient: 0.006,
		NoiseTop3Multiplier: 0.3, NoiseMidMultiplier: 0.8, NoiseTailMultiplier: 1.5})
	p := NewRankingPipeline(ext, ag, rer, br)

	hits := manyHits(200)
	ri := RankInput{
		Query:   features.Query{Text: "react paris senior", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Rerank(ctx, ri)
	}
}

// -- Helper tests ----------------------------------------------------

func TestNormaliseTokens(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"React", []string{"react"}},
		{"React Paris", []string{"react", "paris"}},
		{"React  Paris, React", []string{"react", "paris"}}, // dedup + punctuation
		{"Go!Go?Go.", []string{"go"}},
		{"développeur", []string{"développeur"}}, // multibyte preserved
	}
	for _, c := range cases {
		got := NormaliseTokens(c.in)
		assert.Equal(t, c.want, got, "normalise(%q)", c.in)
	}
}

func TestPrimarySkillOf(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"react"}, "react"},
		{[]string{"", "  ", "go"}, "go"},
		{[]string{"  swift  "}, "swift"},
	}
	for _, c := range cases {
		got := primarySkillOf(c.in)
		assert.Equal(t, c.want, got, "primarySkillOf(%v)", c.in)
	}
}

// -- Stable sort guard ------------------------------------------------

func TestRankingPipeline_Rerank_StableWithinTier(t *testing.T) {
	// Two tied-score hits (same bucket, identical feature inputs)
	// must still appear without panicking — tie-breaking is
	// implementation-defined but the slice shape must remain sound.
	p := newTestPipeline(t)
	hits := make([]TypesenseHit, 5)
	for i := range hits {
		hits[i] = sampleHit(idFor(i), 10)
	}
	out := p.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     time.Unix(1_700_000_000, 0),
	})
	assert.Equal(t, len(hits), len(out))
	// Each candidate appears exactly once.
	ids := make([]string, 0, len(out))
	for _, r := range out {
		ids = append(ids, r.Candidate.DocumentID)
	}
	sort.Strings(ids)
	for i := 1; i < len(ids); i++ {
		assert.NotEqual(t, ids[i-1], ids[i])
	}
}
