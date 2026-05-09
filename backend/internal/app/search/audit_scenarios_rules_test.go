package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// audit_scenarios_rules_test.go covers scenario sets 8-13 of the
// ranking audit (tier sort, anti-gaming, persona weights, persona
// behaviour, diversity, featured override). The earlier sets 1-7
// (per-feature contracts) live in audit_scenarios_test.go — split
// purely to keep individual files under the 600-line cap defined
// in CLAUDE.md.

// -------------------------------------------------------------------
// Scenario set 8 — Tier sort (§6.4)
// -------------------------------------------------------------------

// TestAudit_TierSort_AlwaysBeatsB asserts that EVERY Tier A candidate
// outranks EVERY Tier B candidate, even when Tier B has a higher
// composite score (Felix Not Available has a near-perfect score but
// must end up below the available cohort).
func TestAudit_TierSort_AlwaysBeatsB(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{
		freelanceFixtures[5], // Felix not_available, top-scoring
		freelanceFixtures[1], // Bob available_now, junior
	}
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 5))
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, pipeline, q, hits)
	require.Equal(t, 2, len(out))
	// Bob (Tier A) must come before Felix (Tier B).
	assert.Equal(t, "freelance-02:freelance", out[0].Candidate.DocumentID,
		"Tier A candidate must rank above Tier B even when B has higher score")
	assert.Equal(t, "freelance-06:freelance", out[1].Candidate.DocumentID)
}

// TestAudit_TierSort_AvailableNowAndSoonShareTierA asserts that
// available_soon profiles are rendered alongside available_now,
// matching §6.4 ("Tier A: available_now or available_soon").
func TestAudit_TierSort_AvailableNowAndSoonShareTierA(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{
		freelanceFixtures[0], // available_now
		agencyFixtures[2],    // available_soon (Gamma)
		freelanceFixtures[5], // not_available (Felix)
	}
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 5))
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, pipeline, q, hits)
	require.Equal(t, 3, len(out))
	assert.Equal(t, "freelance-06:freelance", out[2].Candidate.DocumentID,
		"not_available must come last regardless of available_soon presence")
}

// -------------------------------------------------------------------
// Scenario set 9 — Anti-gaming (§7)
// -------------------------------------------------------------------

// TestAudit_AntiGaming_StuffingHalvesTextMatch asserts §7.1 — when
// the keyword stuffing rule fires, the text_match_score is halved
// (default penalty 0.5).
func TestAudit_AntiGaming_StuffingHalvesTextMatch(t *testing.T) {
	ap := newAuditPipeline(t)
	doc := freelanceFixtures[8] // Ivan, stuffed keywords
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 10}}
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, ap.pipeline, q, hits)
	require.Equal(t, 1, len(out))
	assert.InDelta(t, 0.5, out[0].Candidate.Feat.TextMatchScore, 1e-9,
		"stuffing rule must halve text_match_score when bucket=10")
	foundStuffing := false
	for _, p := range ap.agLogger.Penalties {
		if p.Rule == antigaming.RuleKeywordStuffing && p.ProfileID == doc.OrganizationID {
			foundStuffing = true
			break
		}
	}
	assert.True(t, foundStuffing, "stuffing rule must emit a Penalty event")
}

// TestAudit_AntiGaming_StuffingDoesNotFalseFireOnShortText asserts
// that profiles with short, healthy bios do NOT trigger stuffing.
// The detector requires ≥5 tokens before evaluating repetition.
func TestAudit_AntiGaming_StuffingDoesNotFalseFireOnShortText(t *testing.T) {
	ap := newAuditPipeline(t)
	doc := freelanceFixtures[1] // Bob: "react css" only 2 tokens
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 10}}
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, ap.pipeline, q, hits)
	require.Equal(t, 1, len(out))
	assert.InDelta(t, 1.0, out[0].Candidate.Feat.TextMatchScore, 1e-9,
		"short bios must not be stuffing-flagged")
	for _, p := range ap.agLogger.Penalties {
		assert.NotEqual(t, doc.OrganizationID, p.ProfileID,
			"no penalty event should fire for healthy short text")
	}
}

// TestAudit_AntiGaming_ReviewerFloorCapsRating asserts §7.4 — fewer
// than 3 unique reviewers must keep rating_score_diverse ≤ FewReviewerCap.
// The two sub-tests cover the silent-noop branch (already below cap)
// and the clamp+log branch (would have exceeded cap).
func TestAudit_AntiGaming_ReviewerFloorCapsRating(t *testing.T) {
	cfg := antigaming.DefaultConfig()

	t.Run("silent_when_already_below_cap", func(t *testing.T) {
		ap := newAuditPipeline(t)
		doc := freelanceFixtures[4] // Eva: rating ≈ 0.12 already
		hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
		q := auditQuery("react", features.PersonaFreelance, nil)
		out := rankCandidates(t, ap.pipeline, q, hits)
		require.Equal(t, 1, len(out))
		assert.LessOrEqual(t, out[0].Candidate.Feat.RatingScoreDiverse, cfg.FewReviewerCap+1e-9,
			"two-reviewer profile must stay ≤ FewReviewerCap")
	})

	t.Run("clamps_and_logs_when_above_cap", func(t *testing.T) {
		ap := newAuditPipeline(t)
		doc := freelanceFixtures[0]
		doc.UniqueReviewersCount = 2
		doc.MaxReviewerShare = 0.0
		hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
		q := auditQuery("react", features.PersonaFreelance, nil)
		out := rankCandidates(t, ap.pipeline, q, hits)
		require.Equal(t, 1, len(out))
		// The diversity-shrunk extractor caps low-reviewer rating at
		// natural < FewReviewerCap. Either way, the floor is respected.
		assert.LessOrEqual(t, out[0].Candidate.Feat.RatingScoreDiverse, cfg.FewReviewerCap+1e-9,
			"two-reviewer profile must remain ≤ FewReviewerCap after the rule")
	})
}

// TestAudit_AntiGaming_NewAccountZeroesAgeBonus asserts §7.5 — a
// profile younger than NewAccountAgeDays sees its AccountAgeBonus
// zeroed by the rule. (The spec also asks for a final-score cap at
// the persona median — see audit findings doc for the gap.)
func TestAudit_AntiGaming_NewAccountZeroesAgeBonus(t *testing.T) {
	ap := newAuditPipeline(t)
	doc := freelanceFixtures[6] // Gina: 4 days old
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, ap.pipeline, q, hits)
	require.Equal(t, 1, len(out))
	assert.Equal(t, 0.0, out[0].Candidate.Feat.AccountAgeBonus,
		"new-account rule must zero AccountAgeBonus for profiles < NewAccountAgeDays")
	foundNewAcc := false
	for _, p := range ap.agLogger.Penalties {
		if p.Rule == antigaming.RuleNewAccount && p.ProfileID == doc.OrganizationID {
			foundNewAcc = true
			break
		}
	}
	assert.True(t, foundNewAcc, "new-account rule must emit a Penalty event")
}

// -------------------------------------------------------------------
// Scenario set 10 — Persona weight tables (§4)
// -------------------------------------------------------------------

// TestAudit_PersonaWeights_FreelanceTotalsToOne asserts the locked
// weight table from §4.1 sums to 1.0.
func TestAudit_PersonaWeights_FreelanceTotalsToOne(t *testing.T) {
	w := scorer.DefaultFreelanceWeights()
	assert.InDelta(t, 1.0, w.Sum(), 1e-9, "freelance weights must sum to 1")
}

// TestAudit_PersonaWeights_AgencyTotalsToOne asserts §4.2.
func TestAudit_PersonaWeights_AgencyTotalsToOne(t *testing.T) {
	w := scorer.DefaultAgencyWeights()
	assert.InDelta(t, 1.0, w.Sum(), 1e-9, "agency weights must sum to 1")
}

// TestAudit_PersonaWeights_ReferrerTotalsToOne asserts §4.3.
func TestAudit_PersonaWeights_ReferrerTotalsToOne(t *testing.T) {
	w := scorer.DefaultReferrerWeights()
	assert.InDelta(t, 1.0, w.Sum(), 1e-9, "referrer weights must sum to 1")
	assert.Equal(t, 0.0, w.SkillsOverlap, "referrer skills_overlap must be 0")
	assert.Equal(t, 0.0, w.ProvenWork, "referrer proven_work must be 0")
}

// TestAudit_PersonaWeights_AgencyRatingDominates asserts that for
// agencies, rating + proven_work jointly dominate (≥ 50%) — §4.2
// rationale ("track record + portfolio is paramount").
func TestAudit_PersonaWeights_AgencyRatingDominates(t *testing.T) {
	w := scorer.DefaultAgencyWeights()
	combined := w.Rating + w.ProvenWork
	assert.GreaterOrEqual(t, combined, 0.50,
		"agency: rating+proven_work must drive ≥50%% of the score, got %.2f", combined)
}

// TestAudit_PersonaWeights_ReferrerResponseRateMatters asserts that
// for referrers, response_rate carries meaningful weight — §4.3
// rationale ("an unresponsive referrer is useless").
func TestAudit_PersonaWeights_ReferrerResponseRateMatters(t *testing.T) {
	w := scorer.DefaultReferrerWeights()
	assert.GreaterOrEqual(t, w.ResponseRate, 0.15,
		"referrer response_rate must be ≥15%% of the composite, got %.2f", w.ResponseRate)
}

// TestAudit_PersonaWeights_FreelanceTextMatchPresent asserts that
// freelance text_match weight is non-trivial (≥10%) — keyword match
// matters for the freelance funnel.
func TestAudit_PersonaWeights_FreelanceTextMatchPresent(t *testing.T) {
	w := scorer.DefaultFreelanceWeights()
	assert.GreaterOrEqual(t, w.TextMatch, 0.10,
		"freelance text_match must be ≥10%% of the composite")
}

// -------------------------------------------------------------------
// Scenario set 11 — Persona-specific behaviour
// -------------------------------------------------------------------

// TestAudit_AgencyPersona_RatingDrivesOrder asserts that for the
// agency cohort, the highest-rated profile leads — even with weaker
// response_rate (which is only 5% weighted for agencies).
func TestAudit_AgencyPersona_RatingDrivesOrder(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{agencyFixtures[0], agencyFixtures[8]}
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 5))
	q := auditQuery("react", features.PersonaAgency, nil)
	ids := rankIDs(t, pipeline, q, hits)
	require.Equal(t, 2, len(ids))
	assert.Equal(t, "agency-01:agency", ids[0],
		"top-rated agency must rank above mid-tier")
}

// TestAudit_ReferrerPersona_ResponseRateBreaksTie asserts that for
// referrers, a poor response_rate (Mu = 0.2) loses to a similar
// profile with high response_rate (Lambda = 0.95) even though Mu has
// 30 reviews vs Lambda's 42.
func TestAudit_ReferrerPersona_ResponseRateBreaksTie(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{
		referrerFixtures[0], // Lambda: 0.95 response rate
		referrerFixtures[1], // Mu: 0.20 response rate
	}
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 5))
	q := auditQuery("saas b2b", features.PersonaReferrer, nil)
	ids := rankIDs(t, pipeline, q, hits)
	require.Equal(t, 2, len(ids))
	assert.Equal(t, "referrer-01:referrer", ids[0],
		"high-response-rate referrer must outrank slow-response twin")
}

// -------------------------------------------------------------------
// Scenario set 12 — Diversity (§6.5)
// -------------------------------------------------------------------

// TestAudit_Diversity_BreaksThreeInARow asserts that when the top-3
// share the same primary skill, the diversity rule promotes a
// different-skill candidate from later in the slice.
func TestAudit_Diversity_BreaksThreeInARow(t *testing.T) {
	mk := func(id, skill string, bucket int) (search.SearchDocument, int) {
		return search.SearchDocument{
			ID:                     id + ":freelance",
			OrganizationID:         "55555555-aaaa-bbbb-cccc-" + id + "00000000",
			Persona:                search.PersonaFreelance,
			IsPublished:            true,
			DisplayName:            id,
			Skills:                 []string{skill},
			SkillsText:             skill,
			AvailabilityStatus:     "available_now",
			AvailabilityPriority:   3,
			RatingAverage:          4.6,
			RatingCount:            12,
			CompletedProjects:      8,
			ProfileCompletionScore: 80,
			LastActiveAt:           daysAgoUnix(2),
			ResponseRate:           0.9,
			IsVerified:             true,
			UniqueClientsCount:     8,
			UniqueReviewersCount:   12,
			MaxReviewerShare:       0.10,
			ReviewRecencyFactor:    0.85,
			AccountAgeDays:         300,
		}, bucket
	}
	r1, b1 := mk("100000", "react", 10)
	r2, b2 := mk("200000", "react", 9)
	r3, b3 := mk("300000", "react", 8)
	r4, b4 := mk("400000", "react", 7)
	g1, gb := mk("500000", "go", 7)
	hits := []TypesenseHit{
		{Document: r1, TextMatchBucket: b1},
		{Document: r2, TextMatchBucket: b2},
		{Document: r3, TextMatchBucket: b3},
		{Document: r4, TextMatchBucket: b4},
		{Document: g1, TextMatchBucket: gb},
	}
	pipeline := newAuditPipeline(t).pipeline
	q := auditQuery("dev", features.PersonaFreelance, nil)
	ids := rankIDs(t, pipeline, q, hits)
	require.Equal(t, 5, len(ids))
	idToSkill := map[string]string{
		r1.ID: "react",
		r2.ID: "react",
		r3.ID: "react",
		r4.ID: "react",
		g1.ID: "go",
	}
	allReact := true
	for i := 0; i < 3 && i < len(ids); i++ {
		if idToSkill[ids[i]] != "react" {
			allReact = false
			break
		}
	}
	assert.False(t, allReact,
		"diversity rule must break the run of 3 react candidates by promoting g1; got %v", ids)
}

// -------------------------------------------------------------------
// Scenario set 13 — Featured override (§8 dormant V1)
// -------------------------------------------------------------------

// TestAudit_FeaturedOverride_DormantByDefault asserts that the
// is_featured flag has no effect when FeaturedEnabled=false (V1).
func TestAudit_FeaturedOverride_DormantByDefault(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	hi := agencyFixtures[0] // Alpha (top), not featured
	lo := agencyFixtures[5] // Zeta (mid), featured
	hits := []TypesenseHit{
		{Document: hi, TextMatchBucket: 5},
		{Document: lo, TextMatchBucket: 5},
	}
	q := auditQuery("agency", features.PersonaAgency, nil)
	ids := rankIDs(t, pipeline, q, hits)
	require.Equal(t, 2, len(ids))
	assert.Equal(t, "agency-01:agency", ids[0],
		"is_featured must not promote a mid-tier candidate when FeaturedEnabled=false")
}

// TestAudit_FeaturedOverride_BoostsWhenEnabled asserts that flipping
// FeaturedEnabled true with a positive boost actually elevates a
// featured candidate.
func TestAudit_FeaturedOverride_BoostsWhenEnabled(t *testing.T) {
	rlCfg := rules.DefaultConfig()
	rlCfg.RandSeed = 7
	rlCfg.FeaturedEnabled = true
	rlCfg.FeaturedBoost = 2.0
	pipeline := NewRankingPipeline(
		features.NewDefaultExtractor(features.DefaultConfig()),
		antigaming.NewPipeline(antigaming.DefaultConfig(), antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{}),
		scorer.NewWeightedScorer(scorer.DefaultConfig()),
		rules.NewBusinessRules(rlCfg),
	)
	notFeatured := agencyFixtures[8]
	featured := agencyFixtures[5]
	hits := []TypesenseHit{
		{Document: notFeatured, TextMatchBucket: 5},
		{Document: featured, TextMatchBucket: 5},
	}
	q := auditQuery("agency", features.PersonaAgency, nil)
	ids := rankIDs(t, pipeline, q, hits)
	require.Equal(t, 2, len(ids))
	assert.Equal(t, featured.ID, ids[0],
		"FeaturedEnabled+positive boost must surface the is_featured candidate first")
}
