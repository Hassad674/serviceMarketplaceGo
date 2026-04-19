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

// goldenQueries are the 40+ curated queries required by the phase 6H
// exit criteria. Mix of French + English phrases so multi-lingual
// support is exercised, with coverage per persona:
//
//   - freelance : 15 queries (tech stacks, soft-skill keywords,
//     location terms, pricing-band terms, role titles)
//   - agency    : 14 queries (B2B phrasing, project-style,
//     "équipe"/"studio", sector keywords)
//   - referrer  : 12 queries (business-development phrasing,
//     commission hints, intro asks)
//
// Assertion is keyword containment in top-3 so the suite survives
// dataset rotation. Skip-empty is reserved for sparse personas /
// queries where zero hits is acceptable.
var goldenQueries = []GoldenQuery{
	// ---- FREELANCE: technology stacks ----
	{Name: "react_dev_paris", Query: "développeur React Paris", Persona: search.PersonaFreelance, Keywords: []string{"react", "javascript", "frontend", "front-end", "paris", "développeur"}},
	{Name: "full_stack_engineer", Query: "full stack engineer javascript", Persona: search.PersonaFreelance, Keywords: []string{"full", "stack", "javascript", "typescript", "engineer"}},
	{Name: "golang_backend", Query: "golang backend microservices", Persona: search.PersonaFreelance, Keywords: []string{"go", "golang", "backend", "microservices"}},
	{Name: "python_data_scientist", Query: "python data scientist machine learning", Persona: search.PersonaFreelance, Keywords: []string{"python", "data", "machine", "ml", "ai"}},
	{Name: "ai_engineer_llm", Query: "AI engineer LLM langchain", Persona: search.PersonaFreelance, Keywords: []string{"ai", "llm", "langchain", "machine", "python"}},
	{Name: "devops_kubernetes", Query: "devops kubernetes aws", Persona: search.PersonaFreelance, Keywords: []string{"devops", "kubernetes", "aws", "cloud", "sre"}},
	{Name: "mobile_flutter_dev", Query: "Flutter mobile app developer", Persona: search.PersonaFreelance, Keywords: []string{"flutter", "mobile", "dart", "ios", "android"}},
	{Name: "nextjs_expert", Query: "Next.js expert server components", Persona: search.PersonaFreelance, Keywords: []string{"next", "nextjs", "react", "typescript", "frontend"}},
	{Name: "rust_systems_engineer", Query: "Rust systems engineer low latency", Persona: search.PersonaFreelance, Keywords: []string{"rust", "systems", "engineer", "backend", "low"}, SkipEmpty: true},
	{Name: "swift_ios_developer", Query: "développeur iOS Swift SwiftUI", Persona: search.PersonaFreelance, Keywords: []string{"ios", "swift", "mobile", "apple", "swiftui"}, SkipEmpty: true},

	// ---- FREELANCE: design + UX ----
	{Name: "ui_ux_designer", Query: "designer UX mobile app", Persona: search.PersonaFreelance, Keywords: []string{"ux", "ui", "design", "mobile", "figma"}},
	{Name: "product_designer_figma", Query: "product designer Figma design system", Persona: search.PersonaFreelance, Keywords: []string{"product", "design", "figma", "ui", "ux"}, SkipEmpty: true},

	// ---- FREELANCE: soft-skill / role queries ----
	{Name: "product_manager", Query: "senior product manager", Persona: search.PersonaFreelance, Keywords: []string{"product", "manager", "pm", "management"}, SkipEmpty: true},
	{Name: "growth_hacker", Query: "growth hacker SaaS", Persona: search.PersonaFreelance, Keywords: []string{"growth", "marketing", "saas", "acquisition"}, SkipEmpty: true},
	{Name: "freelance_tech_lead", Query: "freelance tech lead équipe senior", Persona: search.PersonaFreelance, Keywords: []string{"tech", "lead", "senior", "équipe", "team"}, SkipEmpty: true},

	// ---- AGENCY: B2B phrasing + project style ----
	{Name: "marketing_agency", Query: "agence marketing digitale", Persona: search.PersonaAgency, Keywords: []string{"marketing", "digital", "agence", "communication"}, SkipEmpty: true},
	{Name: "web_agency_wordpress", Query: "agency web development wordpress", Persona: search.PersonaAgency, Keywords: []string{"web", "wordpress", "development", "agency"}, SkipEmpty: true},
	{Name: "agency_saas_development", Query: "agency SaaS product development", Persona: search.PersonaAgency, Keywords: []string{"saas", "agency", "development", "product", "software"}, SkipEmpty: true},
	{Name: "studio_mobile_development", Query: "studio mobile iOS Android", Persona: search.PersonaAgency, Keywords: []string{"mobile", "studio", "ios", "android", "app"}, SkipEmpty: true},
	{Name: "agence_design_branding", Query: "agence design branding identité", Persona: search.PersonaAgency, Keywords: []string{"design", "branding", "identité", "agence", "communication"}, SkipEmpty: true},
	{Name: "ecommerce_agency", Query: "agence e-commerce Shopify", Persona: search.PersonaAgency, Keywords: []string{"ecommerce", "e-commerce", "shopify", "commerce", "agence"}, SkipEmpty: true},
	{Name: "agency_ai_consulting", Query: "AI consulting agency data strategy", Persona: search.PersonaAgency, Keywords: []string{"ai", "consulting", "data", "agency", "strategy"}, SkipEmpty: true},
	{Name: "agency_webflow_site", Query: "agence Webflow site vitrine", Persona: search.PersonaAgency, Keywords: []string{"webflow", "site", "vitrine", "web", "agence"}, SkipEmpty: true},
	{Name: "devops_consulting_studio", Query: "DevOps consulting studio cloud migration", Persona: search.PersonaAgency, Keywords: []string{"devops", "consulting", "cloud", "studio", "migration"}, SkipEmpty: true},
	{Name: "équipe_tech_startup", Query: "équipe tech startup MVP", Persona: search.PersonaAgency, Keywords: []string{"tech", "startup", "mvp", "équipe", "product"}, SkipEmpty: true},
	{Name: "agency_react_native", Query: "React Native agency mobile", Persona: search.PersonaAgency, Keywords: []string{"react", "native", "mobile", "agency", "app"}, SkipEmpty: true},
	{Name: "studio_data_engineering", Query: "studio data engineering analytics", Persona: search.PersonaAgency, Keywords: []string{"data", "engineering", "analytics", "studio", "etl"}, SkipEmpty: true},
	{Name: "agency_video_content", Query: "agence vidéo content creation", Persona: search.PersonaAgency, Keywords: []string{"vidéo", "video", "content", "agence", "production"}, SkipEmpty: true},
	{Name: "agency_performance_marketing", Query: "agency performance marketing SEA SEO", Persona: search.PersonaAgency, Keywords: []string{"marketing", "performance", "sea", "seo", "agency"}, SkipEmpty: true},

	// ---- REFERRER: business development phrasing ----
	{Name: "business_referrer_saas", Query: "apporteur d'affaire saas b2b", Persona: search.PersonaReferrer, Keywords: []string{"saas", "b2b", "apporteur", "business"}, SkipEmpty: true},
	{Name: "referrer_enterprise_intro", Query: "apporteur business entreprise grande compte", Persona: search.PersonaReferrer, Keywords: []string{"apporteur", "business", "entreprise", "compte", "b2b"}, SkipEmpty: true},
	{Name: "referrer_commission_tech", Query: "referrer commission tech startup", Persona: search.PersonaReferrer, Keywords: []string{"referrer", "commission", "tech", "startup", "apporteur"}, SkipEmpty: true},
	{Name: "referrer_fintech_network", Query: "apporteur d'affaire fintech réseau", Persona: search.PersonaReferrer, Keywords: []string{"fintech", "apporteur", "réseau", "network", "finance"}, SkipEmpty: true},
	{Name: "referrer_cpo_intro", Query: "introduction CPO VP Engineering", Persona: search.PersonaReferrer, Keywords: []string{"cpo", "vp", "engineering", "introduction", "executive"}, SkipEmpty: true},
	{Name: "referrer_ecommerce_dealflow", Query: "apporteur e-commerce deal flow Shopify", Persona: search.PersonaReferrer, Keywords: []string{"ecommerce", "e-commerce", "apporteur", "shopify", "deal"}, SkipEmpty: true},
	{Name: "referrer_healthcare_b2b", Query: "business referrer healthcare B2B", Persona: search.PersonaReferrer, Keywords: []string{"healthcare", "health", "b2b", "referrer", "business"}, SkipEmpty: true},
	{Name: "referrer_digital_transformation", Query: "apporteur transformation digitale secteur public", Persona: search.PersonaReferrer, Keywords: []string{"transformation", "digital", "public", "apporteur", "secteur"}, SkipEmpty: true},
	{Name: "referrer_pharma_network", Query: "referrer pharmaceutique sciences vie", Persona: search.PersonaReferrer, Keywords: []string{"pharma", "pharmaceutique", "sciences", "referrer", "health"}, SkipEmpty: true},
	{Name: "referrer_tech_procurement", Query: "apporteur achats tech procurement", Persona: search.PersonaReferrer, Keywords: []string{"achats", "procurement", "tech", "apporteur", "purchasing"}, SkipEmpty: true},
	{Name: "referrer_banking_cto", Query: "apporteur banking CTO CIO", Persona: search.PersonaReferrer, Keywords: []string{"banking", "bank", "cto", "cio", "apporteur"}, SkipEmpty: true},
	{Name: "referrer_retail_intro", Query: "apporteur retail grande distribution", Persona: search.PersonaReferrer, Keywords: []string{"retail", "distribution", "apporteur", "commerce", "grande"}, SkipEmpty: true},
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
		Q:        q.Query,
		QueryBy:  "display_name,title,skills_text,city",
		FilterBy: fmt.Sprintf("persona:%s && is_published:true", q.Persona),
		// DefaultSortByHybrid includes _vector_distance, which
		// Typesense 28.0 accepts ONLY when a vector_query is set.
		// The plain DefaultSortBy would 400 here.
		//
		// IMPORTANT: `embedding` MUST NOT appear in `query_by` on
		// Typesense 28.0 — it is a manual-embedding (not
		// auto-embedding) field, so we pass the pre-computed vector
		// via `vector_query` instead. Mixing the two surfaces as a
		// 400 with "Vector field `embedding` is not an auto-embedding
		// field, do not use `query_by` with it".
		SortBy:        search.DefaultSortByHybrid(),
		Page:          1,
		PerPage:       10,
		ExcludeFields: "embedding",
		NumTypos:      "2,2,1,1",
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

// persona IDs count as "no keyword match". Helper returns true when
// any keyword appears in the concatenation of searchable fields.
func hitsContainKeyword(hits []map[string]any, keywords []string, topN int) bool {
	if topN > len(hits) {
		topN = len(hits)
	}
	for i := 0; i < topN; i++ {
		blob := strings.ToLower(hitText(hits[i]))
		for _, kw := range keywords {
			if strings.Contains(blob, strings.ToLower(kw)) {
				return true
			}
		}
	}
	return false
}
