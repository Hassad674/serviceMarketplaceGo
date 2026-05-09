package search

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// audit_spec_drift_test.go encodes the post-fix invariants for the
// five §7 + §3.2-7 + §3.2-8 ranking drifts uncovered in the
// 2026-05-09 audit. Every test in this file asserts the FIXED
// behaviour — if the production pipeline regresses on any contract
// below the suite turns red.
//
// History
//
//   - 2026-05-09: tests originally documented drift (asserted current
//     buggy behaviour and logged a TODO).
//   - 2026-05-09 (same day): tests flipped to assert the fix once
//     ranking_pipeline.go + rules/new_account_cap.go landed.

// TestAudit_SpecFix_VelocityRule_FiresWithTimestamps proves that the
// velocity rule (§7.2) now fires when the SearchDocument carries
// recent review timestamps. Prior bug: applyAntiGaming hard-coded
// RawSignals.RecentReviewTimestamps to nil.
func TestAudit_SpecFix_VelocityRule_FiresWithTimestamps(t *testing.T) {
	ap := newAuditPipeline(t)
	now := auditNowUnix
	// 8 reviews in the last 24h (above the default cap of 5).
	stamps := []int64{
		now - 30,
		now - 1200,
		now - 3600,
		now - 7200,
		now - 10800,
		now - 14400,
		now - 18000,
		now - 21600,
	}
	doc := freelanceFixtures[2] // Camille — solid baseline rating
	doc.RatingCount = 30        // > excess so the dampening factor < 1
	doc.RecentReviewTimestamps = stamps
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
	q := auditQuery("go", features.PersonaFreelance, nil)
	out := rankCandidates(t, ap.pipeline, q, hits)
	require.Equal(t, 1, len(out))

	foundVelocity := false
	for _, p := range ap.agLogger.Penalties {
		if p.Rule == antigaming.RuleReviewVelocity && p.ProfileID == doc.OrganizationID {
			foundVelocity = true
			break
		}
	}
	assert.True(t, foundVelocity,
		"velocity rule MUST fire when recent timestamps exceed the 24h cap (§7.2)")
	assert.Less(t, out[0].Candidate.Feat.RatingScoreDiverse, 1.0,
		"velocity rule must dampen rating_score_diverse below 1.0")
}

// TestAudit_SpecFix_VelocityRule_QuietWhenNoBurst proves the rule
// stays silent for organic review patterns. Negative case for the
// fix above.
func TestAudit_SpecFix_VelocityRule_QuietWhenNoBurst(t *testing.T) {
	ap := newAuditPipeline(t)
	now := auditNowUnix
	// Just 2 reviews in 24h — well under the cap.
	stamps := []int64{now - 3600, now - 7200}
	doc := freelanceFixtures[0] // Alice
	doc.RecentReviewTimestamps = stamps
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
	q := auditQuery("react", features.PersonaFreelance, nil)
	_ = rankCandidates(t, ap.pipeline, q, hits)

	for _, p := range ap.agLogger.Penalties {
		if p.ProfileID == doc.OrganizationID {
			assert.NotEqual(t, antigaming.RuleReviewVelocity, p.Rule,
				"velocity rule must stay quiet on healthy review pace")
		}
	}
}

// TestAudit_SpecFix_LinkedRule_DetectorReceivesIDs proves the linked-
// account rule (§7.3) now passes reviewer IDs to the detector. Prior
// bug: RawSignals.ReviewerIDs was hard-coded to nil so the detector
// always saw an empty slice.
func TestAudit_SpecFix_LinkedRule_DetectorReceivesIDs(t *testing.T) {
	cfg := antigaming.DefaultConfig()
	captured := [][]string{}
	det := stubLinkedDetector{
		onCallWithIDs: func(ids []string) {
			cp := make([]string, len(ids))
			copy(cp, ids)
			captured = append(captured, cp)
		},
	}
	logger := &antigaming.RecordingLogger{}
	pipe := antigaming.NewPipeline(cfg, det, logger)
	feats := &features.Features{RatingScoreDiverse: 0.6}
	raw := antigaming.RawSignals{
		ProfileID:   "test-profile",
		ReviewerIDs: []string{"a", "b", "c", "d"},
	}
	pipe.Apply(context.Background(), feats, raw)
	require.Equal(t, 1, len(captured),
		"detector must be invoked exactly once when ReviewerIDs is non-empty")
	assert.Equal(t, []string{"a", "b", "c", "d"}, captured[0],
		"detector must receive the exact slice from the document adapter")
}

// TestAudit_SpecFix_LinkedRule_FullPipelineRoute proves the IDs flow
// from SearchDocument → RawSignals through the production pipeline
// (not just the antigaming sub-package).
func TestAudit_SpecFix_LinkedRule_FullPipelineRoute(t *testing.T) {
	captured := [][]string{}
	det := stubLinkedDetector{
		onCallWithIDs: func(ids []string) {
			cp := make([]string, len(ids))
			copy(cp, ids)
			captured = append(captured, cp)
		},
	}
	cfg := antigaming.DefaultConfig()
	logger := &antigaming.RecordingLogger{}
	ag := antigaming.NewPipeline(cfg, det, logger)
	pipeline := newAuditPipelineWithAntigaming(t, ag)

	doc := freelanceFixtures[0]
	doc.ReviewerIDs = []string{"r1", "r2", "r3"}
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
	q := auditQuery("react", features.PersonaFreelance, nil)
	_ = pipeline.Rerank(context.Background(), RankInput{
		Query:   q,
		Persona: q.Persona,
		Hits:    hits,
		Now:     auditNow(),
	})
	require.Equal(t, 1, len(captured),
		"full pipeline must propagate ReviewerIDs into the linked-account detector")
	assert.Equal(t, []string{"r1", "r2", "r3"}, captured[0])
}

// stubLinkedDetector replaces the no-op detector for spec-fix tests.
// It records each invocation through the ID hook so we can verify
// reachability AND the exact slice passed in.
type stubLinkedDetector struct {
	onCallWithIDs func([]string)
}

func (s stubLinkedDetector) LinkedCount(_ context.Context, ids []string) (int, error) {
	if s.onCallWithIDs != nil {
		s.onCallWithIDs(ids)
	}
	return 0, nil // never trigger the dampening — we only care reachability
}

// TestAudit_SpecFix_NewAccountCap_FinalScoreEnforced proves §7.5 —
// the new-account profile's Final score is now capped at the cohort
// median. Prior bug: PipelineResult.NewAccountCapped was discarded
// by applyAntiGaming, so a freshly gamed account could legitimately
// reach the top.
func TestAudit_SpecFix_NewAccountCap_FinalScoreEnforced(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline

	// Build a strong-signal new account: 4 days old (< 7 day cap)
	// but with rating + verified + completion all maxed out as if
	// the account were fully gamed before any check fires.
	newAcc := search.SearchDocument{
		ID:                     "freelance-fresh-attacker:freelance",
		OrganizationID:         "ffffffff-1111-1111-1111-000000000901",
		Persona:                search.PersonaFreelance,
		IsPublished:            true,
		DisplayName:            "Fresh Attacker",
		Skills:                 []string{"react"},
		SkillsText:             "react",
		AvailabilityStatus:     "available_now",
		AvailabilityPriority:   3,
		RatingAverage:          5.0,
		RatingCount:            30,
		CompletedProjects:      30,
		ProfileCompletionScore: 100,
		LastActiveAt:           daysAgoUnix(0),
		ResponseRate:           1.0,
		IsVerified:             true,
		UniqueClientsCount:     20,
		UniqueReviewersCount:   28,
		MaxReviewerShare:       0.05,
		ReviewRecencyFactor:    1.0,
		AccountAgeDays:         4, // <-- triggers the rule
	}

	// And a legitimate established cohort to measure the median.
	mature1 := freelanceFixtures[0] // Alice — mature, top-rated
	mature2 := freelanceFixtures[2] // Camille — mature, proven
	mature3 := freelanceFixtures[1] // Bob — mature, mid-tier

	hits := []TypesenseHit{
		{Document: newAcc, TextMatchBucket: 5},
		{Document: mature1, TextMatchBucket: 5},
		{Document: mature2, TextMatchBucket: 5},
		{Document: mature3, TextMatchBucket: 5},
	}

	out := pipeline.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     auditNow(),
	})
	require.Equal(t, 4, len(out))

	// Locate the new account + the mature cohort scores.
	freshFinal := -1.0
	matureFinals := make([]float64, 0, 3)
	for _, r := range out {
		if r.Candidate.DocumentID == newAcc.ID {
			freshFinal = r.Candidate.Score.Final
			continue
		}
		matureFinals = append(matureFinals, r.Candidate.Score.Final)
	}
	require.GreaterOrEqual(t, freshFinal, 0.0, "new-account doc must be present")
	require.Equal(t, 3, len(matureFinals))

	median := medianOf(matureFinals)
	assert.LessOrEqual(t, freshFinal, median+1e-6,
		"§7.5: new-account profile Final must be ≤ persona median (got %.2f, median=%.2f)",
		freshFinal, median)
	t.Logf("FIX confirmed — new-account Final=%.2f ≤ median=%.2f",
		freshFinal, median)
}

// TestAudit_SpecFix_NewAccountFlag_ReachesScorerWhenWired retains the
// contract test from the original drift suite — the antigaming
// pipeline correctly computes PipelineResult.NewAccountCapped.
func TestAudit_SpecFix_NewAccountFlag_ReachesScorerWhenWired(t *testing.T) {
	cfg := antigaming.DefaultConfig()
	logger := &antigaming.RecordingLogger{}
	pipe := antigaming.NewPipeline(cfg, antigaming.NoopLinkedReviewersDetector{}, logger)
	feats := &features.Features{}
	raw := antigaming.RawSignals{
		ProfileID:      "fresh",
		AccountAgeDays: 3, // < NewAccountAgeDays (7)
	}
	res := pipe.Apply(context.Background(), feats, raw)
	assert.True(t, res.NewAccountCapped,
		"PipelineResult.NewAccountCapped must be set for accounts younger than the cap")
}

// TestAudit_SpecFix_NewAccountCap_PropagatesToCandidate proves the
// fix handles the integration boundary: scoreCandidates must transfer
// the antigaming flag onto rules.Candidate.NewAccountCapped so the
// rules layer can find it.
func TestAudit_SpecFix_NewAccountCap_PropagatesToCandidate(t *testing.T) {
	ap := newAuditPipeline(t)
	doc := freelanceFixtures[6] // Gina — 4 days old
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, ap.pipeline, q, hits)
	require.Equal(t, 1, len(out))
	assert.True(t, out[0].Candidate.NewAccountCapped,
		"Candidate.NewAccountCapped must be true for accounts < NewAccountAgeDays")
}

// TestAudit_SpecFix_About_FieldFlowsToFeatures proves §3.2-7 — the
// SearchDocument now carries `about` and the document adapter now
// passes it through to features.SearchDocumentLite.About. Prior bug:
// document_adapter.go hard-coded About to "".
func TestAudit_SpecFix_About_FieldFlowsToFeatures(t *testing.T) {
	doc := search.SearchDocument{
		ID:             "junk-bio:freelance",
		OrganizationID: "00000000-0000-4000-8000-000000000001",
		Persona:        search.PersonaFreelance,
		IsPublished:    true,
		DisplayName:    "Junk Bio Profile",
		About:          "lorem ipsum dolor sit amet — recap of expertise",
	}
	hit := TypesenseHit{Document: doc, TextMatchBucket: 5}
	lite := hit.ToSearchDocumentLite(time.Now().Unix())
	assert.Equal(t, doc.About, lite.About,
		"document_adapter.go must propagate SearchDocument.About into the feature lite copy")
}

// TestAudit_SpecFix_LastActiveAt_DormantBaseline proves §3.2-8 — when
// LastActiveAt is unknown the score now collapses to the dormant
// baseline (≈ 1 / (1 + 180/30) at default decay) instead of 0. Prior
// bug: ExtractLastActiveDays returned 0 on missing LastActiveAt.
func TestAudit_SpecFix_LastActiveAt_DormantBaseline(t *testing.T) {
	cfg := features.DefaultConfig()
	doc := features.SearchDocumentLite{
		// LastActiveAt explicitly missing.
		NowUnix: 1700000000,
	}
	got := features.ExtractLastActiveDays(doc, cfg)
	// At default decay (30 days), the 6-month baseline gives 1/7 ≈ 0.1429.
	assert.InDelta(t, 1.0/7.0, got, 1e-9,
		"§3.2-8: missing LastActiveAt must yield the 6-month dormant baseline (got %.4f)", got)
}

// medianOf returns the median of the provided floats. Mirrors the
// rules.medianFinalNonCapped helper but operates on a slice of
// already-extracted scores.
func medianOf(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := make([]float64, len(values))
	copy(cp, values)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

// newAuditPipelineWithAntigaming builds an audit pipeline that uses
// the supplied antigaming.Pipeline instead of the default. Used by
// the linked-account fix test to plug in a stub detector. Re-creates
// the production-default extractor + scorer + rules locally so the
// test stays self-contained without sharing state with other audits.
func newAuditPipelineWithAntigaming(t *testing.T, ag *antigaming.Pipeline) *RankingPipeline {
	t.Helper()
	ext := features.NewDefaultExtractor(features.DefaultConfig())
	scCfg := scorer.DefaultConfig()
	require.NoError(t, scCfg.Validate())
	rlCfg := rules.DefaultConfig()
	rlCfg.RandSeed = 7
	return NewRankingPipeline(
		ext,
		ag,
		scorer.NewWeightedScorer(scCfg),
		rules.NewBusinessRules(rlCfg),
	)
}
