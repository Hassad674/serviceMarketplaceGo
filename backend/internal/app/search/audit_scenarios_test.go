package search

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/features"
)

// audit_scenarios_test.go enforces the per-feature contracts
// declared in docs/ranking-v1.md §3.2 + §6 + §7 by replaying
// hand-crafted fixtures through the production RankingPipeline.
//
// Every scenario follows the same pattern :
//   1. Pick the relevant fixture cohort (freelance / agency / referrer).
//   2. Decide the text-match buckets per fixture (simulating Typesense).
//   3. Rerank with the audit pipeline (deterministic seed 7).
//   4. Assert the relative ordering matches the spec contract.
//
// The audit deliberately does NOT call the live Typesense cluster —
// the live retrieval path has its own integration tests (see
// integration_test.go + golden_full_pipeline_test.go). The scope of
// this audit is the Stage 2-5 ranking pipeline.

// auditQuery builds a features.Query reusing the production token
// normaliser. Persona is a parameter so each scenario can pick the
// matching cohort without scaffold.
func auditQuery(text string, persona features.Persona, filterSkills []string) features.Query {
	return features.Query{
		Text:             text,
		NormalisedTokens: NormaliseTokens(text),
		FilterSkills:     filterSkills,
		Persona:          persona,
	}
}

// auditNow returns a deterministic time used by every scenario.
// Same value as auditNowUnix to keep last-active math stable.
func auditNow() time.Time { return time.Unix(auditNowUnix, 0).UTC() }

// rankIDs runs the pipeline + extracts the ordered document IDs
// for ergonomic assertions ("doc X is at index Y").
func rankIDs(t *testing.T, pipeline *RankingPipeline, query features.Query, hits []TypesenseHit) []string {
	t.Helper()
	out := pipeline.Rerank(context.Background(), RankInput{
		Query:   query,
		Persona: query.Persona,
		Hits:    hits,
		Now:     auditNow(),
	})
	ids := make([]string, len(out))
	for i, r := range out {
		ids[i] = r.Candidate.DocumentID
	}
	return ids
}

// indexOf returns the position of `id` in the slice, or -1 if
// missing. Used heavily for "X is ranked above Y" assertions.
func indexOf(ids []string, id string) int {
	for i, v := range ids {
		if v == id {
			return i
		}
	}
	return -1
}

// rankCandidates runs the pipeline and returns the full
// RankedCandidate slice. Used when scenarios need to inspect the
// score breakdown rather than the order alone.
func rankCandidates(t *testing.T, pipeline *RankingPipeline, query features.Query, hits []TypesenseHit) []RankedCandidate {
	t.Helper()
	return pipeline.Rerank(context.Background(), RankInput{
		Query:   query,
		Persona: query.Persona,
		Hits:    hits,
		Now:     auditNow(),
	})
}

// findCandidate returns the RankedCandidate whose DocumentID equals
// `id`. Bails the test if missing.
func findCandidate(t *testing.T, list []RankedCandidate, id string) RankedCandidate {
	t.Helper()
	for _, r := range list {
		if r.Candidate.DocumentID == id {
			return r
		}
	}
	t.Fatalf("candidate %q not in ranked output", id)
	return RankedCandidate{}
}

// -------------------------------------------------------------------
// Scenario set 1 — Text match (§3.2-1)
// -------------------------------------------------------------------

// TestAudit_TextMatch_DominantBucketWins asserts that with all other
// signals equal, the candidate with the highest text-match bucket
// ranks first. Uses two near-identical fixtures and only differs the
// bucket value.
func TestAudit_TextMatch_DominantBucketWins(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{freelanceFixtures[0], freelanceFixtures[1]}
	buckets := map[string]int{
		"freelance-01:freelance": 10,
		"freelance-02:freelance": 1,
	}
	ids := rankIDs(t, pipeline, auditQuery("react", features.PersonaFreelance, nil), hitsFromFixtures(docs, buckets))
	require.Equal(t, 2, len(ids))
	assert.Equal(t, "freelance-01:freelance", ids[0],
		"highest bucket must rank first")
}

// TestAudit_TextMatch_EmptyQueryRedistributes asserts that when the
// query is empty (q=*), the text_match weight is redistributed so a
// profile lacking the text match isn't penalised — its score still
// reflects its other features.
func TestAudit_TextMatch_EmptyQueryRedistributes(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{freelanceFixtures[0], freelanceFixtures[1]}
	// Empty-query path: every bucket = 0.
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 0))
	out := rankCandidates(t, pipeline, auditQuery("", features.PersonaFreelance, nil), hits)
	require.Equal(t, 2, len(out))
	// Both must end up with non-zero Final because rating + completion
	// + verified weights pick up the missing text-match slice.
	for _, r := range out {
		assert.Greater(t, r.Candidate.Score.Final, 0.0,
			"empty-query redistribution must produce non-zero Final for %s",
			r.Candidate.DocumentID)
	}
}

// -------------------------------------------------------------------
// Scenario set 2 — Skills overlap (§3.2-2)
// -------------------------------------------------------------------

// TestAudit_SkillsOverlap_FullMatchOverPartial asserts that a profile
// matching every query skill outranks a profile matching half — all
// other things equal.
func TestAudit_SkillsOverlap_FullMatchOverPartial(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{
		freelanceFixtures[0], // skills: react, javascript, typescript
		freelanceFixtures[1], // skills: react, css
	}
	// Hand-fix the reviewer + rating signals to be IDENTICAL so only
	// skills_overlap differs. We mutate copies, never the originals.
	a := docs[0]
	b := docs[1]
	a.RatingAverage, a.RatingCount, a.UniqueReviewersCount, a.MaxReviewerShare = 4.5, 10, 10, 0.10
	b.RatingAverage, b.RatingCount, b.UniqueReviewersCount, b.MaxReviewerShare = 4.5, 10, 10, 0.10
	a.CompletedProjects, b.CompletedProjects = 10, 10
	a.UniqueClientsCount, b.UniqueClientsCount = 10, 10
	a.ProfileCompletionScore, b.ProfileCompletionScore = 80, 80
	a.LastActiveAt, b.LastActiveAt = daysAgoUnix(2), daysAgoUnix(2)
	a.AccountAgeDays, b.AccountAgeDays = 200, 200
	a.IsVerified, b.IsVerified = true, true
	a.ResponseRate, b.ResponseRate = 0.9, 0.9
	hits := []TypesenseHit{
		{Document: a, TextMatchBucket: 5},
		{Document: b, TextMatchBucket: 5},
	}
	q := auditQuery("react javascript typescript", features.PersonaFreelance, nil)
	ids := rankIDs(t, pipeline, q, hits)
	require.Equal(t, 2, len(ids))
	assert.Equal(t, a.ID, ids[0],
		"profile matching 3/3 query skills must outrank profile matching 1/3")
}

// TestAudit_SkillsOverlap_FilterSkillsCounted asserts that filter
// chips contribute to the query skill set even if not in the typed
// query text (§3.2-2: query_skills = tokenize(query) ∪ filter.skills).
func TestAudit_SkillsOverlap_FilterSkillsCounted(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{freelanceFixtures[0], freelanceFixtures[1]}
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 5))
	// Empty text query but filter chip "react" must still pick up
	// the skill-overlap signal for both profiles.
	out := rankCandidates(t, pipeline,
		auditQuery("", features.PersonaFreelance, []string{"react"}),
		hits)
	require.Equal(t, 2, len(out))
	for _, r := range out {
		assert.Greater(t, r.Candidate.Feat.SkillsOverlapRatio, 0.0,
			"filter skill chip must contribute to skills_overlap for %s",
			r.Candidate.DocumentID)
	}
}

// TestAudit_SkillsOverlap_ReferrerAlwaysZero asserts §3.2-2 referrer
// rule: referrers don't sell skills, the feature must always be 0.
func TestAudit_SkillsOverlap_ReferrerAlwaysZero(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{referrerFixtures[0]} // skills: saas b2b
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 5))
	q := auditQuery("saas b2b", features.PersonaReferrer, []string{"saas"})
	out := rankCandidates(t, pipeline, q, hits)
	require.Equal(t, 1, len(out))
	assert.Equal(t, 0.0, out[0].Candidate.Feat.SkillsOverlapRatio,
		"skills_overlap must be 0 for referrers regardless of query content")
}

// -------------------------------------------------------------------
// Scenario set 3 — Rating diverse Bayesian (§3.2-3)
// -------------------------------------------------------------------

// TestAudit_Rating_HighCountBeatsHighAvgLowCount asserts the Bayesian
// shrinkage prevents a 1-review 5-star profile from beating a high-
// volume 4.6-star profile. Pin everything else equal so rating drives.
func TestAudit_Rating_HighCountBeatsHighAvgLowCount(t *testing.T) {
	// Build two synthetic docs: same skills/text-match/etc but very
	// different rating profiles.
	mk := func(id string, avg float64, count int32, unique int32, share float64) search.SearchDocument {
		return search.SearchDocument{
			ID:                     id + ":freelance",
			OrganizationID:         "11111111-1111-1111-1111-" + id + "00000000",
			Persona:                search.PersonaFreelance,
			IsPublished:            true,
			DisplayName:            id,
			Skills:                 []string{"react"},
			SkillsText:             "react",
			AvailabilityStatus:     "available_now",
			AvailabilityPriority:   3,
			RatingAverage:          avg,
			RatingCount:            count,
			CompletedProjects:      10,
			ProfileCompletionScore: 80,
			LastActiveAt:           daysAgoUnix(2),
			ResponseRate:           0.9,
			IsVerified:             true,
			UniqueClientsCount:     8,
			RepeatClientRate:       0.30,
			UniqueReviewersCount:   unique,
			MaxReviewerShare:       share,
			ReviewRecencyFactor:    0.85,
			LostDisputesCount:      0,
			AccountAgeDays:         300,
		}
	}
	highCount := mk("100000", 4.6, 50, 45, 0.05)
	highAvg := mk("200000", 5.0, 1, 1, 1.0)
	pipeline := newAuditPipeline(t).pipeline
	hits := []TypesenseHit{
		{Document: highCount, TextMatchBucket: 5},
		{Document: highAvg, TextMatchBucket: 5},
	}
	q := auditQuery("react", features.PersonaFreelance, nil)
	ids := rankIDs(t, pipeline, q, hits)
	require.Equal(t, 2, len(ids))
	assert.Equal(t, highCount.ID, ids[0],
		"Bayesian rating must rank 50× 4.6 above 1× 5.0")
}

// TestAudit_Rating_ColdStartFloor asserts that a profile with zero
// reviews receives the cold-start floor (default 0.15) rather than 0.
func TestAudit_Rating_ColdStartFloor(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	docs := []search.SearchDocument{freelanceFixtures[6]} // Gina: 0 reviews
	hits := hitsFromFixtures(docs, uniformBuckets(docs, 5))
	q := auditQuery("react html", features.PersonaFreelance, nil)
	out := rankCandidates(t, pipeline, q, hits)
	require.Equal(t, 1, len(out))
	cfg := features.DefaultConfig()
	assert.InDelta(t, cfg.ColdStartFloor, out[0].Candidate.Feat.RatingScoreDiverse, 1e-9,
		"zero-review profile must surface the cold-start floor")
}

// TestAudit_Rating_DiversityFactorPenalisesConcentration asserts the
// "3 friends leave 10 reviews" attack: a profile whose reviews come
// from a few high-share reviewers must score lower than a comparable
// profile with diverse reviewers.
func TestAudit_Rating_DiversityFactorPenalisesConcentration(t *testing.T) {
	mkDoc := func(id string, unique int32, share float64) search.SearchDocument {
		return search.SearchDocument{
			ID:                     id + ":freelance",
			OrganizationID:         "33333333-3333-4444-5555-" + id + "00000000",
			Persona:                search.PersonaFreelance,
			IsPublished:            true,
			DisplayName:            id,
			Skills:                 []string{"react"},
			SkillsText:             "react",
			AvailabilityStatus:     "available_now",
			AvailabilityPriority:   3,
			RatingAverage:          4.8,
			RatingCount:            10,
			CompletedProjects:      10,
			ProfileCompletionScore: 80,
			LastActiveAt:           daysAgoUnix(2),
			ResponseRate:           0.9,
			IsVerified:             true,
			UniqueClientsCount:     8,
			UniqueReviewersCount:   unique,
			MaxReviewerShare:       share,
			ReviewRecencyFactor:    0.85,
			AccountAgeDays:         300,
		}
	}
	diverse := mkDoc("100000", 9, 0.15)
	concentrated := mkDoc("200000", 9, 0.80) // one reviewer = 80% of all reviews
	pipeline := newAuditPipeline(t).pipeline
	hits := []TypesenseHit{
		{Document: diverse, TextMatchBucket: 5},
		{Document: concentrated, TextMatchBucket: 5},
	}
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, pipeline, q, hits)
	require.Equal(t, 2, len(out))
	dCand := findCandidate(t, out, diverse.ID)
	cCand := findCandidate(t, out, concentrated.ID)
	assert.Greater(t, dCand.Candidate.Feat.RatingScoreDiverse,
		cCand.Candidate.Feat.RatingScoreDiverse,
		"diverse reviewers must score higher than concentrated reviewers")
}

// -------------------------------------------------------------------
// Scenario set 4 — Proven work (§3.2-4)
// -------------------------------------------------------------------

// TestAudit_ProvenWork_VolumeBeatsSparse asserts the 0.4·log(projects)
// + 0.35·log(clients) + 0.25·sqrt(repeat) composite ordering with
// other signals pinned equal.
func TestAudit_ProvenWork_VolumeBeatsSparse(t *testing.T) {
	mk := func(id string, projects, clients int32, repeat float64) search.SearchDocument {
		return search.SearchDocument{
			ID:                     id + ":freelance",
			OrganizationID:         "44444444-aaaa-aaaa-aaaa-" + id + "00000000",
			Persona:                search.PersonaFreelance,
			IsPublished:            true,
			DisplayName:            id,
			Skills:                 []string{"react"},
			SkillsText:             "react",
			AvailabilityStatus:     "available_now",
			AvailabilityPriority:   3,
			RatingAverage:          4.5,
			RatingCount:            10,
			CompletedProjects:      projects,
			ProfileCompletionScore: 80,
			LastActiveAt:           daysAgoUnix(2),
			ResponseRate:           0.9,
			IsVerified:             true,
			UniqueClientsCount:     clients,
			RepeatClientRate:       repeat,
			UniqueReviewersCount:   10,
			MaxReviewerShare:       0.10,
			ReviewRecencyFactor:    0.85,
			AccountAgeDays:         300,
		}
	}
	rich := mk("100000", 50, 30, 0.40)
	sparse := mk("200000", 5, 5, 0.10)
	pipeline := newAuditPipeline(t).pipeline
	hits := []TypesenseHit{
		{Document: rich, TextMatchBucket: 5},
		{Document: sparse, TextMatchBucket: 5},
	}
	q := auditQuery("react", features.PersonaFreelance, nil)
	out := rankCandidates(t, pipeline, q, hits)
	require.Equal(t, 2, len(out))
	rCand := findCandidate(t, out, rich.ID)
	sCand := findCandidate(t, out, sparse.ID)
	assert.Greater(t, rCand.Candidate.Feat.ProvenWorkScore, sCand.Candidate.Feat.ProvenWorkScore,
		"50 projects / 30 clients / 40%% repeat must yield higher proven_work_score than 5/5/10%%")
}

// TestAudit_ProvenWork_ReferrerAlwaysZero asserts referrers never
// score on proven_work (§3.2-4 referrer rule).
func TestAudit_ProvenWork_ReferrerAlwaysZero(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	doc := referrerFixtures[0]
	doc.CompletedProjects = 999 // would otherwise saturate
	doc.UniqueClientsCount = 999
	doc.RepeatClientRate = 1.0
	hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
	q := auditQuery("saas", features.PersonaReferrer, nil)
	out := rankCandidates(t, pipeline, q, hits)
	require.Equal(t, 1, len(out))
	assert.Equal(t, 0.0, out[0].Candidate.Feat.ProvenWorkScore,
		"proven_work must be 0 for referrers regardless of project counts")
}

// -------------------------------------------------------------------
// Scenario set 5 — Verified-mature (§3.2-6)
// -------------------------------------------------------------------

// TestAudit_VerifiedMature_BinaryGate asserts the gate fires only
// when both is_verified AND account_age_days ≥ 30.
func TestAudit_VerifiedMature_BinaryGate(t *testing.T) {
	cases := []struct {
		name     string
		verified bool
		ageDays  int32
		want     float64
	}{
		{"unverified mature", false, 90, 0},
		{"verified young", true, 5, 0},
		{"verified borderline", true, 30, 1},
		{"verified mature", true, 200, 1},
	}
	pipeline := newAuditPipeline(t).pipeline
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := freelanceFixtures[0]
			doc.IsVerified = tc.verified
			doc.AccountAgeDays = tc.ageDays
			hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
			out := rankCandidates(t, pipeline,
				auditQuery("react", features.PersonaFreelance, nil), hits)
			require.Equal(t, 1, len(out))
			assert.Equal(t, tc.want, out[0].Candidate.Feat.IsVerifiedMature,
				"is_verified_mature gate")
		})
	}
}

// -------------------------------------------------------------------
// Scenario set 6 — Last active + account age (§3.2-8 / §3.2-9)
// -------------------------------------------------------------------

// TestAudit_LastActive_HyperbolicDecay asserts the reference table
// from §3.2-8 (1 / (1 + days / 30)).
func TestAudit_LastActive_HyperbolicDecay(t *testing.T) {
	cases := []struct {
		days int
		want float64
	}{
		{0, 1.00},
		{15, 0.667},
		{30, 0.500},
		{90, 0.250},
		{180, 0.143},
		{365, 0.076},
	}
	cfg := features.DefaultConfig()
	for _, tc := range cases {
		t.Run("days_"+itoa(tc.days), func(t *testing.T) {
			doc := freelanceFixtures[0]
			doc.LastActiveAt = daysAgoUnix(tc.days)
			lite := features.SearchDocumentLite{
				LastActiveAt: doc.LastActiveAt,
				NowUnix:      auditNowUnix,
			}
			got := features.ExtractLastActiveDays(lite, cfg)
			assert.InDelta(t, tc.want, got, 0.005,
				"last_active_days_score(%d days)", tc.days)
		})
	}
}

// TestAudit_AccountAge_LogScaleSaturates asserts the reference table
// from §3.2-9: log-normalised, capped at 1 year.
func TestAudit_AccountAge_LogScaleSaturates(t *testing.T) {
	cases := []struct {
		days int32
		want float64
	}{
		{0, 0},
		{7, 0.351},
		{30, 0.578},
		{90, 0.766},
		{365, 1.0},
		{1000, 1.0}, // capped
	}
	cfg := features.DefaultConfig()
	for _, tc := range cases {
		t.Run("days_"+itoa(int(tc.days)), func(t *testing.T) {
			lite := features.SearchDocumentLite{AccountAgeDays: tc.days}
			got := features.ExtractAccountAgeBonus(lite, cfg)
			assert.InDelta(t, tc.want, got, 0.005,
				"account_age_bonus(%d days)", tc.days)
		})
	}
}

// -------------------------------------------------------------------
// Scenario set 7 — Negative penalty (§5.3)
// -------------------------------------------------------------------

// TestAudit_LostDisputes_PenaltyCappedAt30Pct asserts §5.3 — three
// lost disputes saturate the penalty at 30% and additional disputes
// don't push past it.
func TestAudit_LostDisputes_PenaltyCappedAt30Pct(t *testing.T) {
	cases := []struct {
		count int32
		want  float64
	}{
		{0, 0.0},
		{1, 0.10},
		{3, 0.30},
		{4, 0.30}, // capped
		{10, 0.30},
	}
	pipeline := newAuditPipeline(t).pipeline
	for _, tc := range cases {
		t.Run("disputes_"+itoa(int(tc.count)), func(t *testing.T) {
			doc := freelanceFixtures[0]
			doc.LostDisputesCount = tc.count
			hits := []TypesenseHit{{Document: doc, TextMatchBucket: 5}}
			out := rankCandidates(t, pipeline,
				auditQuery("react", features.PersonaFreelance, nil), hits)
			require.Equal(t, 1, len(out))
			assert.InDelta(t, tc.want, out[0].Candidate.Feat.NegativeSignals, 1e-9,
				"NegativeSignals at %d lost disputes", tc.count)
		})
	}
}

// TestAudit_LostDisputes_LowerFinalScore asserts the penalty actually
// reduces the displayed Final score: same fixture, with vs without
// disputes.
func TestAudit_LostDisputes_LowerFinalScore(t *testing.T) {
	pipeline := newAuditPipeline(t).pipeline
	clean := freelanceFixtures[0]
	clean.LostDisputesCount = 0
	disputed := clean
	disputed.ID = "freelance-01-bad:freelance"
	disputed.OrganizationID = "fffffff1-0000-0000-0000-000000000999"
	disputed.LostDisputesCount = 4
	hits := []TypesenseHit{
		{Document: clean, TextMatchBucket: 5},
		{Document: disputed, TextMatchBucket: 5},
	}
	out := rankCandidates(t, pipeline,
		auditQuery("react", features.PersonaFreelance, nil), hits)
	require.Equal(t, 2, len(out))
	cleanCand := findCandidate(t, out, clean.ID)
	dispCand := findCandidate(t, out, disputed.ID)
	assert.Greater(t, cleanCand.Candidate.Score.Final, dispCand.Candidate.Score.Final,
		"clean profile must outrank disputed twin")
}


// itoa is a tiny helper for naming sub-tests with integer suffixes
// without dragging strconv into every assertion. Avoids the call-site
// clutter of `strconv.Itoa(tc.days)`.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
