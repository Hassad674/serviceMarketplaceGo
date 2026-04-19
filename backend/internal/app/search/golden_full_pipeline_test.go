package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// golden_full_pipeline_test.go is the live end-to-end counterpart of
// internal/search/golden_test.go: it runs the full retrieval + rerank
// path against a real Typesense cluster + real OpenAI embedder and
// asserts that, after the Stage 2-5 pipeline runs, the top-ranked
// profile still matches the expected keywords AND has
// NegativeSignals == 0 (we do not surface disputed profiles at the
// top without reason).
//
// Gated by OPENAI_EMBEDDINGS_LIVE=true + TYPESENSE_INTEGRATION_URL +
// TYPESENSE_API_KEY + OPENAI_API_KEY.
//
// The rest of the gate is the same as the BM25-only golden suite in
// internal/search/golden_test.go — we skip when the env vars are
// absent so the test never runs on a fresh dev machine.

func fullPipelineEnabled() bool {
	return os.Getenv("OPENAI_EMBEDDINGS_LIVE") == "true"
}

type fullPipelineEnv struct {
	collection string
	client     *search.Client
	embedder   search.EmbeddingsClient
	pipeline   *RankingPipeline
}

func newFullPipelineEnv(t *testing.T) *fullPipelineEnv {
	t.Helper()
	host := os.Getenv("TYPESENSE_INTEGRATION_URL")
	if host == "" {
		host = os.Getenv("TYPESENSE_HOST")
	}
	require.NotEmpty(t, host, "TYPESENSE_INTEGRATION_URL must be set")
	apiKey := os.Getenv("TYPESENSE_API_KEY")
	require.NotEmpty(t, apiKey, "TYPESENSE_API_KEY must be set")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	require.NotEmpty(t, openaiKey, "OPENAI_API_KEY must be set")

	model := os.Getenv("OPENAI_EMBEDDINGS_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	ts, err := search.NewClient(host, apiKey)
	require.NoError(t, err)
	raw, err := search.NewOpenAIEmbeddings(openaiKey, model)
	require.NoError(t, err)
	embedder := search.NewRetryingEmbeddings(raw)

	// Build the pipeline with the production defaults + a deterministic
	// seed so a rerun produces the same ordering on the same data.
	fcfg := features.DefaultConfig()
	agCfg := antigaming.DefaultConfig()
	scCfg := scorer.DefaultConfig()
	rlCfg := rules.DefaultConfig()
	rlCfg.RandSeed = 1
	pipeline := NewRankingPipeline(
		features.NewDefaultExtractor(fcfg),
		antigaming.NewPipeline(agCfg, antigaming.NoopLinkedReviewersDetector{}, antigaming.NoopLogger{}),
		scorer.NewWeightedScorer(scCfg),
		rules.NewBusinessRules(rlCfg),
	)
	return &fullPipelineEnv{
		collection: search.AliasName,
		client:     ts,
		embedder:   embedder,
		pipeline:   pipeline,
	}
}

// fullPipelineCase is a trimmed-down version of the BM25-only
// GoldenQuery struct. We do not import that struct directly to keep
// the two suites decoupled — their assertions differ in scope.
type fullPipelineCase struct {
	name      string
	query     string
	persona   search.Persona
	keywords  []string
	skipEmpty bool
}

// fullPipelineCases is a subset of goldenQueries focused on the three
// most common persona queries. Running the full pipeline against 40
// queries would multiply OpenAI costs + test duration by 3 — the
// reduced subset still covers each persona + French/English mixing.
var fullPipelineCases = []fullPipelineCase{
	{name: "freelance_react_paris", query: "développeur React Paris", persona: search.PersonaFreelance, keywords: []string{"react", "javascript", "paris", "frontend", "développeur"}},
	{name: "freelance_golang_backend", query: "golang backend microservices", persona: search.PersonaFreelance, keywords: []string{"go", "golang", "backend", "microservices"}},
	{name: "freelance_ai_engineer", query: "AI engineer LLM langchain", persona: search.PersonaFreelance, keywords: []string{"ai", "llm", "langchain", "python", "machine"}},
	{name: "agency_marketing_digital", query: "agence marketing digitale", persona: search.PersonaAgency, keywords: []string{"marketing", "digital", "agence", "communication"}, skipEmpty: true},
	{name: "agency_web_dev", query: "agency web development", persona: search.PersonaAgency, keywords: []string{"web", "development", "agency"}, skipEmpty: true},
	{name: "referrer_saas_b2b", query: "apporteur saas b2b", persona: search.PersonaReferrer, keywords: []string{"saas", "b2b", "apporteur", "business"}, skipEmpty: true},
}

// TestGolden_FullPipeline exercises retrieval + rerank on the real
// stack. Each case asserts:
//  1. retrieval returned at least one hit (unless skipEmpty),
//  2. the reranked top-3 contains at least one expected keyword,
//  3. the top-1 profile has NegativeSignals == 0 (no disputed
//     profile silently slipped to the top).
//
// The suite is advisory-strict: a failure indicates the rerank
// ordering drifted away from the retrieval quality, or the test
// fixture no longer carries the expected keywords. Either way the
// dev investigates rather than muting the test.
func TestGolden_FullPipeline(t *testing.T) {
	if !fullPipelineEnabled() {
		t.Skip("set OPENAI_EMBEDDINGS_LIVE=true to run full-pipeline golden tests")
	}
	env := newFullPipelineEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, tc := range fullPipelineCases {
		t.Run(tc.name, func(t *testing.T) {
			runFullPipelineCase(t, ctx, env, tc)
		})
	}
}

func runFullPipelineCase(t *testing.T, ctx context.Context, env *fullPipelineEnv, tc fullPipelineCase) {
	t.Helper()
	vec, err := env.embedder.Embed(ctx, tc.query)
	require.NoError(t, err, "embed")

	raw, err := env.client.Query(ctx, env.collection, search.SearchParams{
		Q:             tc.query,
		QueryBy:       "display_name,title,skills_text,city",
		FilterBy:      fmt.Sprintf("persona:%s && is_published:true", tc.persona),
		SortBy:        search.DefaultSortByHybrid(),
		Page:          1,
		PerPage:       50,
		ExcludeFields: "embedding",
		NumTypos:      "2,2,1,1",
		VectorQuery:   search.FormatVectorQuery(vec, 20),
	})
	require.NoError(t, err, "typesense query")

	// Parse with the ranking-aware path so we get the []TypesenseHit
	// with bucketed text_match scores.
	result, hits, err := parseQueryResultWithHits(raw)
	require.NoError(t, err)
	if len(hits) == 0 {
		if tc.skipEmpty {
			t.Skipf("zero results for %q on sparse persona %s", tc.query, tc.persona)
			return
		}
		t.Fatalf("zero results for %q", tc.query)
	}

	// Rerank with the production pipeline. The embedding has already
	// reordered the raw hits; the rerank re-orders them against the
	// full 10-feature vector.
	persona := features.Persona(tc.persona)
	reranked := env.pipeline.Rerank(ctx, RankInput{
		Query: features.Query{
			Text:             tc.query,
			NormalisedTokens: NormaliseTokens(tc.query),
			Persona:          persona,
		},
		Persona: persona,
		Hits:    hits,
		Now:     time.Now(),
	})
	require.NotEmpty(t, reranked, "rerank produced empty output for %q", tc.query)

	// Assertion 1 — keyword containment in reranked top-3.
	topN := 3
	if topN > len(reranked) {
		topN = len(reranked)
	}
	keywordHit := false
	for i := 0; i < topN; i++ {
		blob := docBlob(reranked[i].RawDoc.Document)
		for _, kw := range tc.keywords {
			if strings.Contains(strings.ToLower(blob), strings.ToLower(kw)) {
				keywordHit = true
				break
			}
		}
		if keywordHit {
			break
		}
	}
	require.True(t, keywordHit,
		"reranked top-%d for %q should contain one of %v — got %s",
		topN, tc.query, tc.keywords, renderTop(reranked, topN))

	// Assertion 2 — top-1 must not have NegativeSignals > 0. The
	// rerank already penalises disputed profiles via the
	// (1 - NegativeSignals) multiplication, so a disputed profile
	// reaching rank 1 would contradict the spec.
	top := reranked[0]
	require.Equal(t, 0.0, top.Candidate.Feat.NegativeSignals,
		"top-ranked doc %s carries NegativeSignals=%.3f — check extractor",
		top.Candidate.DocumentID, top.Candidate.Feat.NegativeSignals)

	// Lightweight debug log at INFO to help tune weights when the
	// assertion fails post-merge.
	slog.Info("golden.full_pipeline",
		"query", tc.query,
		"persona", tc.persona,
		"top_doc", top.Candidate.DocumentID,
		"top_final", top.Candidate.Score.Final,
		"retrieval_count", result.Found,
		"rerank_count", len(reranked))
	_ = result
}

// docBlob concatenates the searchable fields of a document so a
// keyword scan is a single strings.Contains call.
func docBlob(doc search.SearchDocument) string {
	var b strings.Builder
	b.WriteString(doc.DisplayName)
	b.WriteByte(' ')
	b.WriteString(doc.Title)
	b.WriteByte(' ')
	b.WriteString(doc.SkillsText)
	b.WriteByte(' ')
	b.WriteString(doc.City)
	b.WriteByte(' ')
	for _, s := range doc.Skills {
		b.WriteString(s)
		b.WriteByte(' ')
	}
	for _, d := range doc.ExpertiseDomains {
		b.WriteString(d)
		b.WriteByte(' ')
	}
	return b.String()
}

// renderTop is a compact previewer used in assertion failure messages.
func renderTop(ranked []RankedCandidate, n int) string {
	if n > len(ranked) {
		n = len(ranked)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		r := ranked[i]
		fmt.Fprintf(&b, "[%d: %s final=%.1f skills=%q]",
			i+1, r.Candidate.DocumentID, r.Candidate.Score.Final, r.RawDoc.Document.SkillsText)
	}
	return b.String()
}

// Ensure encoding/json is referenced — some builds drop unused
// imports; parseQueryResultWithHits pulls it transitively but keep
// the compile-time guard to make test-file refactors obvious.
var _ = json.Marshal
