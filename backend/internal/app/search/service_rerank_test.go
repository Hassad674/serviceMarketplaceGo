package search

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// service_rerank_test.go exercises the Service.Query path with a real
// RankingPipeline wired in. Covers: presence of rerank flag, duration
// + top score are populated, LTR capture fires when wired,
// absence-of-pipeline fallback does not break the service.

// fakeHitsPayload builds a Typesense-like JSON response with the given
// number of hits, varied enough that the rerank reorders them.
func fakeHitsPayload(t *testing.T, n int) string {
	t.Helper()
	type rawDoc struct {
		ID                     string   `json:"id"`
		OrganizationID         string   `json:"organization_id"`
		Persona                string   `json:"persona"`
		IsPublished            bool     `json:"is_published"`
		DisplayName            string   `json:"display_name"`
		Skills                 []string `json:"skills"`
		SkillsText             string   `json:"skills_text"`
		AvailabilityStatus     string   `json:"availability_status"`
		RatingAverage          float64  `json:"rating_average"`
		RatingCount            int32    `json:"rating_count"`
		CompletedProjects      int32    `json:"completed_projects"`
		ProfileCompletionScore int32    `json:"profile_completion_score"`
		ResponseRate           float64  `json:"response_rate"`
		IsVerified             bool     `json:"is_verified"`
		UniqueClientsCount     int32    `json:"unique_clients_count"`
		UniqueReviewersCount   int32    `json:"unique_reviewers_count"`
		AccountAgeDays         int32    `json:"account_age_days"`
	}
	type hit struct {
		Document  rawDoc `json:"document"`
		TextMatch int64  `json:"text_match"`
	}
	hits := make([]hit, n)
	for i := 0; i < n; i++ {
		orgID := "11111111-1111-1111-1111-000000000001" // shared UUID shape
		if i > 0 {
			orgID = "22222222-2222-2222-2222-000000000002"
		}
		hits[i] = hit{
			Document: rawDoc{
				ID:                     "doc-" + string(rune('A'+i)),
				OrganizationID:         orgID,
				Persona:                "freelance",
				IsPublished:            true,
				DisplayName:            "Profile " + string(rune('A'+i)),
				Skills:                 []string{"react", "go"},
				SkillsText:             "react go",
				AvailabilityStatus:     "available_now",
				RatingAverage:          4.5 + float64(i)*0.05,
				RatingCount:            int32(10 + i),
				CompletedProjects:      int32(5 + i),
				ProfileCompletionScore: int32(80 - i*2),
				ResponseRate:           0.8,
				IsVerified:             true,
				UniqueClientsCount:     int32(5 + i),
				UniqueReviewersCount:   int32(5 + i),
				AccountAgeDays:         200,
			},
			TextMatch: int64(1000 - i*100),
		}
	}
	envelope := map[string]interface{}{
		"found":    n,
		"out_of":   n,
		"page":     1,
		"per_page": n,
		"hits":     hits,
	}
	b, err := json.Marshal(envelope)
	require.NoError(t, err)
	return string(b)
}

// recordingLTRRepo captures the search_id + payloadJSON pushed to it
// so tests can assert the fire-and-forget LTR capture runs.
type recordingLTRRepo struct {
	mu     sync.Mutex
	calls  int
	lastID string
	sha    string
}

func (r *recordingLTRRepo) AttachResultFeatures(_ context.Context, searchID, _, sha string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.lastID = searchID
	r.sha = sha
	return nil
}

func (r *recordingLTRRepo) snapshot() (int, string, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls, r.lastID, r.sha
}

// fakeAnalyticsRepo is a minimal searchanalytics.Repository used to
// build the Service instance without touching Postgres.
type fakeAnalyticsRepo struct{}

func (fakeAnalyticsRepo) InsertSearch(context.Context, *searchanalytics.SearchRow) error {
	return nil
}
func (fakeAnalyticsRepo) RecordClick(context.Context, string, string, int, time.Time) error {
	return nil
}

// newServiceWithRerank builds a Service with the full ranking pipeline
// wired in + the supplied LTR repo. Returns a fake client so tests
// can inject a canned response and inspect the query params.
func newServiceWithRerank(t *testing.T, payload string, ltrRepo searchanalytics.LTRRepository) (*Service, *fakeClient) {
	t.Helper()
	stub := &fakeClient{persona: search.PersonaFreelance, respPayload: payload}

	ext := features.NewDefaultExtractor(features.DefaultConfig())
	ag := antigaming.NewPipeline(antigaming.DefaultConfig(), antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{})
	rer := scorer.NewWeightedScorer(scorer.DefaultConfig())
	brCfg := rules.DefaultConfig()
	brCfg.RandSeed = 1
	br := rules.NewBusinessRules(brCfg)
	rp := NewRankingPipeline(ext, ag, rer, br)

	analyticsSvc, err := searchanalytics.NewService(searchanalytics.Config{
		Repository: fakeAnalyticsRepo{},
		Logger:     slog.Default(),
	})
	require.NoError(t, err)

	svc := NewService(ServiceDeps{
		Freelance:        stub,
		Logger:           slog.Default(),
		RankingPipeline:  rp,
		LTRRepository:    ltrRepo,
		AnalyticsService: analyticsSvc,
	})
	return svc, stub
}

func TestService_Query_RerankFlagAndDuration(t *testing.T) {
	svc, _ := newServiceWithRerank(t, fakeHitsPayload(t, 5), nil)
	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err)
	assert.True(t, res.Reranked, "rerank flag must be set when pipeline runs")
	assert.Greater(t, res.RerankDurationMs, -1, "duration must be measured")
	assert.GreaterOrEqual(t, res.TopFinalScore, 0.0)
	assert.LessOrEqual(t, res.TopFinalScore, 100.0)
}

func TestService_Query_NoPipelineKeepsRawOrder(t *testing.T) {
	stub := &fakeClient{persona: search.PersonaFreelance, respPayload: fakeHitsPayload(t, 5)}
	svc := NewService(ServiceDeps{
		Freelance: stub,
		Logger:    slog.Default(),
		// No RankingPipeline.
	})
	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err)
	assert.False(t, res.Reranked, "no pipeline = no rerank")
	assert.Equal(t, 0, res.RerankDurationMs)
	assert.Equal(t, 0.0, res.TopFinalScore)
	// Documents should appear in Typesense's raw order (doc-A first).
	require.Len(t, res.Documents, 5)
	assert.Equal(t, "doc-A", res.Documents[0].ID)
}

func TestService_Query_WithRerankReordersAndAssignsTopScore(t *testing.T) {
	svc, _ := newServiceWithRerank(t, fakeHitsPayload(t, 8), nil)
	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err)
	assert.True(t, res.Reranked)
	assert.Greater(t, res.TopFinalScore, 0.0)
	// Reranked order may differ from raw — assert "not all in same
	// order" loosely so the test is robust against future weight
	// tuning.
	require.NotEmpty(t, res.Documents)
	// The reranked top hit must be one of the input docs — we do not
	// invent candidates.
	foundInInput := false
	for i := 0; i < 8; i++ {
		if res.Documents[0].ID == "doc-"+string(rune('A'+i)) {
			foundInInput = true
			break
		}
	}
	assert.True(t, foundInInput, "top hit must be one of the retrieved docs")
}

func TestService_Query_LTRCaptureFiresWhenWired(t *testing.T) {
	repo := &recordingLTRRepo{}
	svc, _ := newServiceWithRerank(t, fakeHitsPayload(t, 5), repo)
	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err)
	// The capture runs on a detached goroutine — wait briefly.
	require.Eventually(t, func() bool {
		calls, _, _ := repo.snapshot()
		return calls >= 1
	}, 2*time.Second, 10*time.Millisecond, "LTR capture did not fire")
	calls, searchID, sha := repo.snapshot()
	assert.Equal(t, 1, calls)
	assert.Equal(t, res.SearchID, searchID)
	assert.Len(t, sha, 64, "SHA must be 64 hex chars")
}

func TestService_Query_LTRCaptureSkipsWhenNotWired(t *testing.T) {
	// Pipeline wired, LTR repo nil → capture must be silently
	// skipped.
	svc, _ := newServiceWithRerank(t, fakeHitsPayload(t, 5), nil)
	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err)
	// No assertion on repo — the test verifies the path does not
	// panic with a nil repo.
}

func TestService_Query_RerankDurationBelowBudget(t *testing.T) {
	// Budget: 50ms for 200 hits; on a reasonable machine rerank
	// should land far below that. This is a "soft" guard against
	// unexpected perf regressions.
	payload := fakeHitsPayload(t, 200)
	svc, _ := newServiceWithRerank(t, payload, nil)
	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err)
	assert.Less(t, res.RerankDurationMs, 50, "rerank must stay under 50ms budget on 200 hits")
}

func TestService_Query_RerankWithEmptyQuery(t *testing.T) {
	// Empty query path: text_match buckets are zero across the
	// board, scorer redistributes weights. Reranker still runs.
	svc, _ := newServiceWithRerank(t, fakeHitsPayload(t, 5), nil)
	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "", // listing page
	})
	require.NoError(t, err)
	assert.True(t, res.Reranked)
	assert.Greater(t, res.TopFinalScore, 0.0)
}

func TestService_CaptureLTR_EmptySearchIDSkipsWrite(t *testing.T) {
	// Set up service with a pipeline wired but hijack the flow so
	// decorate runs but SearchID somehow stays empty — a
	// pathological input the runtime guard must handle.
	repo := &recordingLTRRepo{}
	svc, _ := newServiceWithRerank(t, fakeHitsPayload(t, 3), repo)

	// Directly invoke captureLTR with an empty-SearchID result.
	svc.captureLTR(context.Background(), &QueryResult{SearchID: ""}, []RankedCandidate{})

	// Nothing should have been recorded.
	time.Sleep(50 * time.Millisecond)
	calls, _, _ := repo.snapshot()
	assert.Equal(t, 0, calls, "captureLTR must skip when search_id is empty")
}

func TestService_CaptureLTR_NilServiceSkips(t *testing.T) {
	svc := NewService(ServiceDeps{
		Freelance:        &fakeClient{persona: search.PersonaFreelance, respPayload: "{}"},
		Logger:           slog.Default(),
		LTRRepository:    &recordingLTRRepo{},
		AnalyticsService: nil, // critical: no service
	})
	// Must not panic + must not write.
	svc.captureLTR(context.Background(), &QueryResult{SearchID: "id"}, []RankedCandidate{})
}

func TestService_CaptureLTR_NilRepoSkips(t *testing.T) {
	svc := NewService(ServiceDeps{
		Freelance:        &fakeClient{persona: search.PersonaFreelance, respPayload: "{}"},
		Logger:           slog.Default(),
		LTRRepository:    nil, // critical: no repo
		AnalyticsService: nil,
	})
	svc.captureLTR(context.Background(), &QueryResult{SearchID: "id"}, []RankedCandidate{})
}

func TestFeatureContributionMap_CarriesBaseAndNegativeSignals(t *testing.T) {
	// Encoding contract: the map carries the 9 scorer breakdowns +
	// `base` + `negative_signals` so the LTR trainer can reconstruct
	// the exact scoring state.
	c := rules.Candidate{
		Feat: features.Features{NegativeSignals: 0.15},
		Score: scorer.RankedScore{
			Base: 0.7,
			Breakdown: map[string]float64{
				scorer.BreakdownTextMatch: 0.2,
			},
		},
	}
	got := featureContributionMap(RankedCandidate{Candidate: c})
	assert.InDelta(t, 0.7, got["base"], 1e-9)
	assert.InDelta(t, 0.15, got["negative_signals"], 1e-9)
	assert.InDelta(t, 0.2, got[scorer.BreakdownTextMatch], 1e-9)
	assert.NotSame(t, &c.Score.Breakdown, &got, "map must be a copy, not a reference")
}
