package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// synonyms.go seeds the multi-way synonym list used by Typesense to
// bridge French and English terms (frontend/front-end, javascript/js,
// apporteur/business referrer, …) and short forms of common tech
// brands (nextjs/next.js, kubernetes/k8s, …).
//
// Synonyms are upserted via PUT /collections/:name/synonyms/:id which
// is idempotent on the Typesense side — calling SeedSynonyms twice in
// a row is a no-op. EnsureSchema invokes this function on every boot
// so we never end up with a stale set in the wild.
//
// Adding a new pair: append it to defaultSynonyms with a stable ID,
// then restart the API. Removing one: delete its entry AND issue a
// manual `DELETE /collections/marketplace_actors_v1/synonyms/<id>`
// because the seeder does NOT prune extras (would risk wiping
// operator-curated synonyms in production).

// Synonym is the Typesense wire format for a multi-way synonym
// definition. The Root field is OPTIONAL — when empty Typesense
// treats every term in Synonyms as bidirectionally equivalent. We
// always populate Root so the seed list reads naturally during code
// review.
type Synonym struct {
	// ID is the stable identifier used in the PUT path. Must be
	// unique per collection and stable over time so re-seeding
	// updates the existing entry instead of creating a duplicate.
	ID string `json:"-"`

	// Root is the canonical term. When set, Typesense expands
	// queries that match Root to also match any term in Synonyms,
	// AND vice versa.
	Root string `json:"root,omitempty"`

	// Synonyms is the list of equivalent terms. Order does not
	// matter — Typesense treats every pair as bidirectional.
	Synonyms []string `json:"synonyms"`
}

// defaultSynonyms is the curated FR/EN bridge for the marketplace
// search engine. ~30 pairs covering the frontend/backend stack, the
// data/AI stack, the marketing/product stack, and the marketplace's
// own role vocabulary (apporteur d'affaires).
//
// Adding to this list is welcome — the seeder is idempotent so any
// new entry takes effect on the next deploy without manual ops.
var defaultSynonyms = []Synonym{
	{ID: "s_frontend", Root: "frontend", Synonyms: []string{"front-end", "front end", "développeur front"}},
	{ID: "s_nextjs", Root: "nextjs", Synonyms: []string{"next.js", "next js", "next"}},
	{ID: "s_js", Root: "javascript", Synonyms: []string{"js", "ecmascript"}},
	{ID: "s_ts", Root: "typescript", Synonyms: []string{"ts"}},
	{ID: "s_go", Root: "golang", Synonyms: []string{"go"}},
	{ID: "s_ml", Root: "machine learning", Synonyms: []string{"ml", "apprentissage automatique"}},
	{ID: "s_ai", Root: "artificial intelligence", Synonyms: []string{"ai", "ia", "intelligence artificielle"}},
	{ID: "s_llm", Root: "large language model", Synonyms: []string{"llm", "gpt", "claude"}},
	{ID: "s_apporteur", Root: "apporteur d'affaires", Synonyms: []string{"business referrer", "apporteur", "apporteur d'affaire"}},
	{ID: "s_freelance", Root: "freelance", Synonyms: []string{"freelancer", "indépendant", "travailleur indépendant"}},
	{ID: "s_devops", Root: "devops", Synonyms: []string{"dev ops", "infrastructure", "sre"}},
	{ID: "s_k8s", Root: "kubernetes", Synonyms: []string{"k8s"}},
	{ID: "s_db", Root: "database", Synonyms: []string{"db", "base de données"}},
	{ID: "s_ux", Root: "user experience", Synonyms: []string{"ux", "expérience utilisateur"}},
	{ID: "s_ui", Root: "user interface", Synonyms: []string{"ui", "interface utilisateur"}},
	{ID: "s_saas", Root: "saas", Synonyms: []string{"software as a service"}},
	{ID: "s_b2b", Root: "b2b", Synonyms: []string{"business to business"}},
	{ID: "s_startup", Root: "startup", Synonyms: []string{"start-up", "start up"}},
	{ID: "s_pm", Root: "product manager", Synonyms: []string{"pm", "chef de produit"}},
	{ID: "s_po", Root: "product owner", Synonyms: []string{"po"}},
	{ID: "s_cto", Root: "cto", Synonyms: []string{"chief technology officer", "directeur technique"}},
	{ID: "s_ceo", Root: "ceo", Synonyms: []string{"chief executive officer", "pdg"}},
	{ID: "s_api", Root: "api", Synonyms: []string{"rest api", "rest"}},
	{ID: "s_graphql", Root: "graphql", Synonyms: []string{"graph ql"}},
	{ID: "s_rn", Root: "react native", Synonyms: []string{"react-native", "rn"}},
	{ID: "s_vue", Root: "vuejs", Synonyms: []string{"vue", "vue.js"}},
	{ID: "s_angular", Root: "angular", Synonyms: []string{"angularjs", "angular.js"}},
	{ID: "s_cloud", Root: "cloud", Synonyms: []string{"aws", "gcp", "azure"}},
	{ID: "s_data", Root: "data engineer", Synonyms: []string{"data scientist", "ingénieur data"}},
	{ID: "s_fullstack", Root: "fullstack", Synonyms: []string{"full-stack", "full stack", "développeur fullstack"}},
}

// DefaultSynonyms returns a copy of the seed list. Exposed so tests
// (and any future operator script) can iterate over the same data
// the seeder uses without reaching into package internals.
func DefaultSynonyms() []Synonym {
	out := make([]Synonym, len(defaultSynonyms))
	copy(out, defaultSynonyms)
	return out
}

// SeedSynonyms upserts every entry in defaultSynonyms onto the
// `marketplace_actors_v1` collection. Idempotent — the second call
// in a row is a no-op on the Typesense side.
//
// Errors are returned as soon as one synonym fails so the caller can
// log the diff and decide whether to abort startup. We do NOT bulk
// the upserts into a single payload because Typesense's synonym API
// does not expose a bulk endpoint, and the list is small enough
// (~30 entries) that the latency cost is negligible.
func SeedSynonyms(ctx context.Context, client *Client, logger *slog.Logger) error {
	if client == nil {
		return fmt.Errorf("synonyms seed: client is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	for _, syn := range defaultSynonyms {
		if err := client.UpsertSynonym(ctx, CollectionName, syn); err != nil {
			return fmt.Errorf("synonyms seed: upsert %q: %w", syn.ID, err)
		}
	}
	logger.Info("search: synonyms seeded",
		"collection", CollectionName, "count", len(defaultSynonyms))
	return nil
}

// UpsertSynonym puts a single synonym definition onto the collection.
// Lives here (not in client.go) so the synonyms feature can be lifted
// or removed without touching the core client surface.
func (c *Client) UpsertSynonym(ctx context.Context, collection string, syn Synonym) error {
	if syn.ID == "" {
		return fmt.Errorf("typesense upsert synonym: id is required")
	}
	if len(syn.Synonyms) == 0 {
		return fmt.Errorf("typesense upsert synonym: synonyms list is required")
	}
	body, err := json.Marshal(syn)
	if err != nil {
		return fmt.Errorf("typesense upsert synonym: marshal: %w", err)
	}
	path := fmt.Sprintf("/collections/%s/synonyms/%s",
		url.PathEscape(collection), url.PathEscape(syn.ID))
	return c.do(ctx, http.MethodPut, path, bytes.NewReader(body), nil)
}
