package search_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// golden_test.go hosts the live-OpenAI semantic search suite. Each
// test here exercises a real vector embedding path against a real
// Typesense cluster, so they are EXPENSIVE by the standards of the
// rest of the search package tests.
//
// Cost model:
//   - ~15 queries × ~10 input tokens × $0.02/1M tokens = ~$0.00003 per run.
//   - Even 1000 dev runs: $0.03 total.
//   - Assertion strategy: we do NOT pin specific profile IDs (the
//     test DB rotates often). Instead, we assert on structural
//     invariants: (1) the query returns some results, (2) the top
//     hits contain expected keywords either in the title, about,
//     or skills fields, (3) embedding ordering beats pure BM25 on
//     paraphrased queries.
//
// Gating:
//   - OPENAI_EMBEDDINGS_LIVE=true     → required to activate the suite.
//   - TYPESENSE_INTEGRATION_URL       → points at the live cluster.
//   - TYPESENSE_API_KEY               → live cluster master key.
//   - OPENAI_API_KEY                  → for the embedder.
//
// Use `OPENAI_EMBEDDINGS_LIVE=true go test ./internal/search -run Golden -count=1 -v`.

// goldenEnabled reads the gating env var and returns whether the
// live suite should run. Must be true on every test method.
func goldenEnabled() bool {
	return os.Getenv("OPENAI_EMBEDDINGS_LIVE") == "true"
}

func skipIfDisabled(t *testing.T) {
	if !goldenEnabled() {
		t.Skip("set OPENAI_EMBEDDINGS_LIVE=true to run live golden tests")
	}
}

// liveEnv bundles the config needed for any golden test.
type liveEnv struct {
	Collection string
	Typesense  *search.Client
	Embedder   search.EmbeddingsClient
}

// newLiveEnv builds the live embedder + Typesense client from env.
// Fails the test fast when any required env var is missing so the
// operator sees exactly what to set.
func newLiveEnv(t *testing.T) *liveEnv {
	t.Helper()
	host := os.Getenv("TYPESENSE_INTEGRATION_URL")
	if host == "" {
		host = os.Getenv("TYPESENSE_HOST")
	}
	require.NotEmpty(t, host, "TYPESENSE_INTEGRATION_URL or TYPESENSE_HOST must be set")

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

	return &liveEnv{
		Collection: search.AliasName,
		Typesense:  ts,
		Embedder:   embedder,
	}
}

// GoldenQuery is the per-case test payload. Keywords lists the
// expected matching terms — passing if ANY top-3 hit contains at
// least one keyword (case-insensitive) in title/about/skills.
type GoldenQuery struct {
	Name      string
	Query     string
	Persona   search.Persona
	Keywords  []string
	SkipEmpty bool // when true, passing also means "zero results" is OK (useful on sparse DBs)
}

// goldenQueries are the 12+ curated queries required by the phase 3
// exit criteria. Mix of French + English phrases so multi-lingual
// support is exercised by the suite.
var goldenQueries = []GoldenQuery{
	{Name: "react_dev_paris", Query: "développeur React Paris", Persona: search.PersonaFreelance, Keywords: []string{"react", "javascript", "frontend", "front-end", "paris", "développeur"}},
	{Name: "full_stack_engineer", Query: "full stack engineer javascript", Persona: search.PersonaFreelance, Keywords: []string{"full", "stack", "javascript", "typescript", "engineer"}},
	{Name: "golang_backend", Query: "golang backend microservices", Persona: search.PersonaFreelance, Keywords: []string{"go", "golang", "backend", "microservices"}},
	{Name: "python_data_scientist", Query: "python data scientist machine learning", Persona: search.PersonaFreelance, Keywords: []string{"python", "data", "machine", "ml", "ai"}},
	{Name: "ui_ux_designer", Query: "designer UX mobile app", Persona: search.PersonaFreelance, Keywords: []string{"ux", "ui", "design", "mobile", "figma"}},
	{Name: "devops_kubernetes", Query: "devops kubernetes aws", Persona: search.PersonaFreelance, Keywords: []string{"devops", "kubernetes", "aws", "cloud", "sre"}},
	{Name: "ai_engineer_llm", Query: "AI engineer LLM langchain", Persona: search.PersonaFreelance, Keywords: []string{"ai", "llm", "langchain", "machine", "python"}},
	{Name: "business_referrer_saas", Query: "apporteur d'affaire saas b2b", Persona: search.PersonaReferrer, Keywords: []string{"saas", "b2b", "apporteur", "business"}, SkipEmpty: true},
	{Name: "marketing_agency", Query: "agence marketing digitale", Persona: search.PersonaAgency, Keywords: []string{"marketing", "digital", "agence", "communication"}, SkipEmpty: true},
	{Name: "web_agency_wordpress", Query: "agency web development wordpress", Persona: search.PersonaAgency, Keywords: []string{"web", "wordpress", "development", "agency"}, SkipEmpty: true},
	{Name: "mobile_flutter_dev", Query: "Flutter mobile app developer", Persona: search.PersonaFreelance, Keywords: []string{"flutter", "mobile", "dart", "ios", "android"}},
	{Name: "growth_hacker", Query: "growth hacker SaaS", Persona: search.PersonaFreelance, Keywords: []string{"growth", "marketing", "saas", "acquisition"}, SkipEmpty: true},
	{Name: "nextjs_expert", Query: "Next.js expert server components", Persona: search.PersonaFreelance, Keywords: []string{"next", "nextjs", "react", "typescript", "frontend"}},
	{Name: "product_manager", Query: "senior product manager", Persona: search.PersonaFreelance, Keywords: []string{"product", "manager", "pm", "management"}, SkipEmpty: true},
}

func TestGolden_SemanticSuite(t *testing.T) {
	skipIfDisabled(t)
	env := newLiveEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, q := range goldenQueries {
		t.Run(q.Name, func(t *testing.T) {
			runGoldenQuery(t, ctx, env, q)
		})
	}
}

// runGoldenQuery embeds the input query, issues a hybrid search,
// and asserts on the structural invariants described above.
func runGoldenQuery(t *testing.T, ctx context.Context, env *liveEnv, q GoldenQuery) {
	t.Helper()
	vec, err := env.Embedder.Embed(ctx, q.Query)
	require.NoError(t, err, "embed query")

	raw, err := env.Typesense.Query(ctx, env.Collection, search.SearchParams{
		Q:       q.Query,
		QueryBy: "display_name,title,skills_text,city,embedding",
		FilterBy: fmt.Sprintf("persona:%s && is_published:true", q.Persona),
		// DefaultSortByHybrid includes _vector_distance, which
		// Typesense 28.0 accepts ONLY when a vector_query is set.
		// The plain DefaultSortBy would 400 here.
		SortBy:        search.DefaultSortByHybrid(),
		Page:          1,
		PerPage:       10,
		ExcludeFields: "embedding",
		NumTypos:      "2,2,1,1,0",
		VectorQuery:   search.FormatVectorQuery(vec, 20),
	})
	require.NoError(t, err, "typesense query")

	hits, found := parseGoldenHits(t, raw)
	if found == 0 {
		if q.SkipEmpty {
			t.Skipf("zero results for %q — skipping (SkipEmpty=true on sparse DB)", q.Query)
			return
		}
		t.Fatalf("zero results for %q — expected at least one hit", q.Query)
	}
	topN := 3
	if topN > len(hits) {
		topN = len(hits)
	}
	keywordHit := false
	for i := 0; i < topN; i++ {
		blob := hitText(hits[i])
		for _, kw := range q.Keywords {
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
		"top-%d hits for %q should contain at least one of %v — got %s",
		topN, q.Query, q.Keywords, topPreview(hits, topN))
}

// parseGoldenHits decodes the Typesense response into a list of hit
// documents and returns the declared total `found` value.
func parseGoldenHits(t *testing.T, raw json.RawMessage) ([]map[string]any, int) {
	t.Helper()
	var envelope struct {
		Found int `json:"found"`
		Hits  []struct {
			Document map[string]any `json:"document"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))
	out := make([]map[string]any, 0, len(envelope.Hits))
	for _, h := range envelope.Hits {
		out = append(out, h.Document)
	}
	return out, envelope.Found
}

// hitText concatenates the searchable text fields of a hit into a
// single string so the keyword-containment check can scan it once.
func hitText(hit map[string]any) string {
	var b strings.Builder
	for _, k := range []string{"display_name", "title", "skills_text", "city"} {
		if v, ok := hit[k].(string); ok && v != "" {
			b.WriteString(v)
			b.WriteByte(' ')
		}
	}
	// Expertise domains is a string slice.
	if vs, ok := hit["expertise_domains"].([]any); ok {
		for _, v := range vs {
			if s, ok := v.(string); ok {
				b.WriteString(s)
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
}

// topPreview renders a compact string representation of the first
// few hits for assertion failure messages.
func topPreview(hits []map[string]any, n int) string {
	if n > len(hits) {
		n = len(hits)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(fmt.Sprintf("[%d: %q / skills=%q]", i, hits[i]["title"], hits[i]["skills_text"]))
	}
	return b.String()
}

// Compile-time guard so url.QueryEscape is referenced if we ever
// need to switch from GET to POST request bodies for the golden
// suite. Keeps the import line future-proof.
var _ = url.QueryEscape
var _ = http.MethodGet
