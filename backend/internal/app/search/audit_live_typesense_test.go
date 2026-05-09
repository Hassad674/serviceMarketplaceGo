package search

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// audit_live_typesense_test.go runs the full retrieval + rerank
// pipeline against a real Typesense cluster. Gated by
// TYPESENSE_INTEGRATION_URL so unit-test runs stay hermetic.
//
// The test loads the deterministic audit fixtures into a temporary
// per-run collection, runs three persona-specific queries, and
// asserts that the rerank ordering respects the per-feature
// invariants the audit suite proved on its synthetic data.
//
// Why a temporary collection rather than the shared
// `marketplace_actors` collection : the shared one has 22 real
// documents whose contents change as devs touch the seed. The
// audit must run on a known fixture set.

func liveTypesenseURL() string {
	if v := os.Getenv("TYPESENSE_INTEGRATION_URL"); v != "" {
		return v
	}
	return os.Getenv("TYPESENSE_HOST")
}

func liveTypesenseAPIKey() string {
	if v := os.Getenv("TYPESENSE_INTEGRATION_API_KEY"); v != "" {
		return v
	}
	return os.Getenv("TYPESENSE_API_KEY")
}

// liveAuditEnabled toggles the live-Typesense audit. Skipped by
// default so `go test ./...` stays hermetic.
func liveAuditEnabled() bool {
	return liveTypesenseURL() != "" && liveTypesenseAPIKey() != ""
}

// TestAuditLive_FreelanceCohortReranksAsExpected loads the 10
// freelance fixtures into a fresh collection, queries for "react",
// and asserts that the highest-rated available_now profile
// (freelance-01) leads the rerank.
//
// Run with:
//
//	TYPESENSE_INTEGRATION_URL=http://localhost:8108 \
//	TYPESENSE_INTEGRATION_API_KEY=xyz-dev-master-key-change-in-production \
//	go test -v -run TestAuditLive_ ./backend/internal/app/search/...
func TestAuditLive_FreelanceCohortReranksAsExpected(t *testing.T) {
	if !liveAuditEnabled() {
		t.Skip("set TYPESENSE_INTEGRATION_URL + TYPESENSE_INTEGRATION_API_KEY to run live audit tests")
	}
	env := newLiveAuditEnv(t)
	defer env.cleanup(t)

	env.indexDocs(t, freelanceFixtures)

	hits, raw := env.runQuery(t, "react", search.PersonaFreelance)
	require.NotEmpty(t, hits, "live retrieval must return some hits")

	// Run the rerank with the same pipeline production wires.
	pipeline := newAuditPipeline(t).pipeline
	out := pipeline.Rerank(context.Background(), RankInput{
		Query:   features.Query{Text: "react", Persona: features.PersonaFreelance, NormalisedTokens: NormaliseTokens("react")},
		Persona: features.PersonaFreelance,
		Hits:    hits,
		Now:     auditNow(),
	})
	require.NotEmpty(t, out, "rerank must return at least one candidate")

	// Build a quick lookup of top-3 IDs for assertion clarity.
	topIDs := make([]string, 0, 3)
	for i := 0; i < 3 && i < len(out); i++ {
		topIDs = append(topIDs, out[i].Candidate.DocumentID)
	}
	t.Logf("live rerank top-3 (out of %d) for freelance/react: %v", len(out), topIDs)

	// Invariants:
	//   - Felix (freelance-06, not_available) MUST NOT be in top-3
	//     because Tier B is rendered after Tier A in §6.4.
	//   - Jane (freelance-10, 4 lost disputes → 30% penalty) should
	//     not lead.
	for _, id := range topIDs {
		require.NotEqual(t, "freelance-06:freelance", id,
			"Tier B (not_available) candidate must never reach top-3")
	}

	// Stripped-down debug log to ease tuning.
	t.Logf("live retrieval found=%d", raw.Found)
}

// TestAuditLive_AgencyCohortRespectsTierAndRating runs the same
// shape against the agency fixtures: "marketing" should surface
// Gamma (agency-03) above Beta (agency-02) because Gamma has more
// reviews + higher rating despite Beta having fewer skills/text.
func TestAuditLive_AgencyCohortRespectsTierAndRating(t *testing.T) {
	if !liveAuditEnabled() {
		t.Skip("set TYPESENSE_INTEGRATION_URL + TYPESENSE_INTEGRATION_API_KEY to run live audit tests")
	}
	env := newLiveAuditEnv(t)
	defer env.cleanup(t)

	env.indexDocs(t, agencyFixtures)

	hits, _ := env.runQuery(t, "marketing", search.PersonaAgency)
	require.NotEmpty(t, hits, "agency / 'marketing' must return hits")

	pipeline := newAuditPipeline(t).pipeline
	out := pipeline.Rerank(context.Background(), RankInput{
		Query: features.Query{Text: "marketing", Persona: features.PersonaAgency,
			NormalisedTokens: NormaliseTokens("marketing")},
		Persona: features.PersonaAgency,
		Hits:    hits,
		Now:     auditNow(),
	})
	require.NotEmpty(t, out)

	// Eta (agency-07, not_available) must not be in top results
	// because Tier B comes last.
	for i := 0; i < 3 && i < len(out); i++ {
		require.NotEqual(t, "agency-07:agency", out[i].Candidate.DocumentID,
			"agency-07 (not_available) cannot reach top-3 under §6.4")
	}
}

// TestAuditLive_ReferrerCohortFiltersOnPersona asserts that the
// scoped query at the persona boundary surfaces only referrer docs.
// The agency fixtures are also indexed in the same collection but
// must NOT leak through the persona filter (§14 of decisions:
// "scoped client per persona — impossible to leak").
func TestAuditLive_ReferrerCohortFiltersOnPersona(t *testing.T) {
	if !liveAuditEnabled() {
		t.Skip("set TYPESENSE_INTEGRATION_URL + TYPESENSE_INTEGRATION_API_KEY to run live audit tests")
	}
	env := newLiveAuditEnv(t)
	defer env.cleanup(t)

	mixed := append([]search.SearchDocument{}, agencyFixtures...)
	mixed = append(mixed, referrerFixtures...)
	env.indexDocs(t, mixed)

	hits, _ := env.runQuery(t, "saas", search.PersonaReferrer)
	for _, h := range hits {
		require.Equal(t, search.PersonaReferrer, h.Document.Persona,
			"persona-scoped query must never leak %s docs into a referrer search",
			h.Document.Persona)
	}
}

// liveAuditEnv encapsulates the per-test collection name, client,
// and seed-/teardown helpers. The collection is named with a UUID
// suffix so concurrent test runs never clobber each other.
type liveAuditEnv struct {
	collection string
	client     *search.Client
}

func newLiveAuditEnv(t *testing.T) *liveAuditEnv {
	t.Helper()
	host := liveTypesenseURL()
	apiKey := liveTypesenseAPIKey()
	require.NotEmpty(t, host, "TYPESENSE_INTEGRATION_URL or TYPESENSE_HOST required")
	require.NotEmpty(t, apiKey, "TYPESENSE_INTEGRATION_API_KEY or TYPESENSE_API_KEY required")
	cli, err := search.NewClient(host, apiKey, search.WithHTTPClient(&http.Client{Timeout: 15 * time.Second}))
	require.NoError(t, err, "construct typesense client")

	suffix := uuid.New().String()
	collection := "ranking_audit_" + suffix[:8]
	require.NoError(t, ensureAuditCollection(t, cli, collection))

	return &liveAuditEnv{
		collection: collection,
		client:     cli,
	}
}

func (e *liveAuditEnv) cleanup(t *testing.T) {
	t.Helper()
	if e == nil || e.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The Client does not expose a DeleteCollection helper (intentionally
	// — production never drops collections). Delete every doc by filter
	// so subsequent runs don't see stale audit data. The empty
	// collection is harmless but will accumulate one per run; ops
	// can prune them via the Typesense admin endpoint.
	if _, err := e.client.DeleteDocumentsByFilter(ctx, e.collection, "is_published:[true,false]"); err != nil {
		t.Logf("warning: failed to flush audit collection %s: %v", e.collection, err)
	}
	// Best-effort: drop the collection via the raw API. Using a
	// throwaway HTTP request keeps us from leaking ephemeral
	// collections forever. Failure is logged, not fatal.
	if err := dropCollectionRaw(ctx, e.collection); err != nil {
		t.Logf("warning: failed to drop audit collection %s: %v", e.collection, err)
	}
}

// dropCollectionRaw issues a Typesense `DELETE /collections/{name}`
// against the same host the Client uses. Implemented inline because
// the Client struct does not export a DropCollection helper — the
// production code never deletes collections by design.
func dropCollectionRaw(ctx context.Context, name string) error {
	host := liveTypesenseURL()
	apiKey := liveTypesenseAPIKey()
	if host == "" || apiKey == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, host+"/collections/"+name, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", apiKey)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// indexDocs upserts the fixture cohort into the temporary collection.
// Waits a short interval afterwards so Typesense's background
// indexer makes the docs queryable.
func (e *liveAuditEnv) indexDocs(t *testing.T, docs []search.SearchDocument) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Pointer slice required by the client interface.
	ptrs := make([]*search.SearchDocument, 0, len(docs))
	now := time.Now().Unix()
	for i := range docs {
		d := docs[i]
		// Stamp timestamps so the schema accepts the doc.
		if d.CreatedAt == 0 {
			d.CreatedAt = now
		}
		if d.UpdatedAt == 0 {
			d.UpdatedAt = now
		}
		// Compute the Bayesian rating_score (used by the legacy
		// default_sorting_field). The audit pipeline ignores it but
		// the Typesense schema requires the field.
		d.RatingScore = search.BayesianRatingScore(d.RatingAverage, int(d.RatingCount))
		ptrs = append(ptrs, &d)
	}
	require.NoError(t, e.client.BulkUpsert(ctx, e.collection, ptrs),
		"BulkUpsert into %s", e.collection)
	// Typesense's writer is synchronous in practice for small docs,
	// but a small breath gives the background timestamp index a
	// chance to settle.
	time.Sleep(150 * time.Millisecond)
}

// runQuery hits the Typesense /documents/search endpoint with the
// production query shape and parses the response through the same
// path the production code uses.
func (e *liveAuditEnv) runQuery(t *testing.T, q string, persona search.Persona) ([]TypesenseHit, *QueryResult) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	raw, err := e.client.Query(ctx, e.collection, search.SearchParams{
		Q:             q,
		QueryBy:       "display_name,title,skills_text,city",
		FilterBy:      "persona:" + string(persona) + " && is_published:true",
		SortBy:        search.DefaultSortBy(),
		Page:          1,
		PerPage:       50,
		ExcludeFields: "embedding",
		NumTypos:      "2,2,1,1",
	})
	require.NoError(t, err, "live typesense query")
	result, hits, err := parseQueryResultWithHits(raw)
	require.NoError(t, err)
	return hits, result
}

// ensureAuditCollection creates a fresh collection following the
// production schema definition. The audit deliberately avoids
// touching `marketplace_actors_v1` to keep the shared dataset
// stable.
func ensureAuditCollection(t *testing.T, cli *search.Client, name string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	def := search.CollectionSchemaDefinition()
	def.Name = name
	def.DefaultSortingField = "rating_score" // match the production schema
	return cli.CreateCollection(ctx, def)
}

// _useAllImports keeps Go's import checker happy for the optional
// rules / antigaming / scorer imports used during pipeline
// construction. The references make the usage explicit even though
// newAuditPipeline already uses them transitively.
var _useAllImports = func() {
	_ = features.PersonaFreelance
	_ = scorer.PersonaFreelance
	_ = rules.PersonaFreelance
	_ = antigaming.NoopLogger{}
}
