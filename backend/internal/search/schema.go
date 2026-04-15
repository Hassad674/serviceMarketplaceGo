// Package search is the backend integration for the Typesense-backed
// marketplace search engine.
//
// Scope (phase 1 — infra + sync + signals):
//   - Defines the canonical SearchDocument shape shared by the indexer
//     and the query path.
//   - Provides a thin HTTP client around the Typesense REST API (no
//     external SDK so we fully control the error handling + timeouts
//     and the dependency graph stays small).
//   - Exposes a per-persona scoped client that prepends a
//     `persona:<value> && is_published:true` filter by construction,
//     making it impossible to leak cross-persona documents through
//     the query path.
//   - Auto-creates the `marketplace_actors_v1` collection and the
//     `marketplace_actors` alias at backend startup via EnsureSchema.
//   - Pure ranking helpers: Bayesian rating score, profile completion
//     score, availability priority, default sort_by formula.
//   - Indexer that builds a SearchDocument from a set of pluggable
//     repositories + an EmbeddingsClient. Concurrent aggregate loads
//     via errgroup to keep the per-document latency budget below
//     200ms even on a slow database.
//   - EmbeddingsClient interface + OpenAI implementation + a
//     deterministic mock used by 95% of the tests.
//
// Everything related to the query side (handler, scoped API key
// generator, frontend wiring) is explicitly out of scope for phase 1
// and will land in phase 2.
//
// This package has no dependency on any other feature package. It
// receives its data through generic repository ports owned by the
// indexer, so removing the search feature entirely is a matter of
// deleting this folder and a handful of wiring lines in cmd/api/main.go.
package search

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Persona is the discriminator between the three marketplace actors.
// Stored as the `persona` field on every SearchDocument and used as
// the primary facet on the collection so a scoped API key can enforce
// `persona:<value>` without the query path having to remember it.
type Persona string

const (
	// PersonaFreelance covers independent professionals. Backed by
	// the `freelance_profiles` + `freelance_pricing` tables.
	PersonaFreelance Persona = "freelance"

	// PersonaAgency covers service-provider companies. Backed by
	// the legacy `profiles` + `profile_pricing` tables.
	PersonaAgency Persona = "agency"

	// PersonaReferrer covers business referrers (apporteurs d'affaire).
	// Backed by the `referrer_profiles` + `referrer_pricing` tables.
	PersonaReferrer Persona = "referrer"
)

// IsValid reports whether the persona is one of the recognised values.
// Used by the indexer + scoped client to reject bad input at the
// boundary.
func (p Persona) IsValid() bool {
	switch p {
	case PersonaFreelance, PersonaAgency, PersonaReferrer:
		return true
	}
	return false
}

// String implements fmt.Stringer. Used for log correlation.
func (p Persona) String() string { return string(p) }

// CollectionName is the concrete Typesense collection name. The
// frontend and the query path always go through the `marketplace_actors`
// alias — this constant is only used during schema migrations.
//
// When a schema change is needed, we build `_v2`, reindex, and swap
// the alias atomically. See `migration.go` for the alias-swap flow.
const CollectionName = "marketplace_actors_v1"

// AliasName is the stable alias every consumer queries against. Its
// target rotates between `_v1`, `_v2`, … whenever the schema evolves.
const AliasName = "marketplace_actors"

// EmbeddingDimensions is the fixed dimensionality of the vectors
// returned by OpenAI `text-embedding-3-small`. Stored here because
// the schema, the mock client, and the indexer all need to agree.
const EmbeddingDimensions = 1536

// SearchDocument is the canonical shape of every row in the
// `marketplace_actors` Typesense collection.
//
// The struct holds exactly 32 fields organised in six sections:
// identity, display, geo, languages, availability, expertise, pricing,
// quality signals, semantic, and timestamps. JSON tags match the
// Typesense field names so the same struct is both the Go-side
// representation AND the wire format sent to the indexing endpoint.
//
// Every nullable field on the SQL side maps to an `optional: true`
// field in the collection schema so Typesense accepts documents with
// partial data during the early profile-completion phase.
type SearchDocument struct {
	// -------- Identity --------
	ID          string  `json:"id"`
	Persona     Persona `json:"persona"`
	IsPublished bool    `json:"is_published"`

	// -------- Display --------
	DisplayName string `json:"display_name"`
	Title       string `json:"title"`
	PhotoURL    string `json:"photo_url,omitempty"`

	// -------- Geo --------
	City        string    `json:"city,omitempty"`
	CountryCode string    `json:"country_code,omitempty"`
	Location    []float64 `json:"location,omitempty"` // [lat, lng]
	WorkMode    []string  `json:"work_mode"`

	// -------- Languages --------
	LanguagesProfessional  []string `json:"languages_professional"`
	LanguagesConversational []string `json:"languages_conversational"`

	// -------- Availability --------
	AvailabilityStatus   string `json:"availability_status"`
	AvailabilityPriority int32  `json:"availability_priority"`

	// -------- Expertise --------
	ExpertiseDomains []string `json:"expertise_domains"`
	Skills           []string `json:"skills"`
	SkillsText       string   `json:"skills_text"`

	// -------- Pricing --------
	// All pricing fields are optional: providers without pricing info
	// still surface in searches, just without the pricing filter match.
	PricingType       string `json:"pricing_type,omitempty"`
	PricingMinAmount  *int64 `json:"pricing_min_amount,omitempty"`
	PricingMaxAmount  *int64 `json:"pricing_max_amount,omitempty"`
	PricingCurrency   string `json:"pricing_currency,omitempty"`
	PricingNegotiable bool   `json:"pricing_negotiable"`

	// -------- Quality signals --------
	RatingAverage          float64 `json:"rating_average"`
	RatingCount            int32   `json:"rating_count"`
	RatingScore            float64 `json:"rating_score"`
	TotalEarned            int64   `json:"total_earned"`
	CompletedProjects      int32   `json:"completed_projects"`
	ProfileCompletionScore int32   `json:"profile_completion_score"`
	LastActiveAt           int64   `json:"last_active_at"`
	ResponseRate           float64 `json:"response_rate"`
	IsVerified             bool    `json:"is_verified"`
	IsTopRated             bool    `json:"is_top_rated"`
	IsFeatured             bool    `json:"is_featured"`

	// -------- Semantic --------
	Embedding []float32 `json:"embedding,omitempty"`

	// -------- Timestamps --------
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// Validate performs the cheap structural invariants that must hold
// before a document is sent to Typesense. Anything that requires
// hitting the database or the embeddings API lives in the indexer.
//
// The rules intentionally stay forgiving on the optional fields so
// early-stage profiles still make it into the index — we use the
// profile_completion_score to rank them lower instead of rejecting
// them outright.
func (d *SearchDocument) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("search document: id is required")
	}
	if _, err := uuid.Parse(d.ID); err != nil {
		return fmt.Errorf("search document: id %q is not a valid UUID: %w", d.ID, err)
	}
	if !d.Persona.IsValid() {
		return fmt.Errorf("search document: persona %q is invalid", d.Persona)
	}
	if d.DisplayName == "" {
		return fmt.Errorf("search document: display_name is required")
	}
	if d.Embedding != nil && len(d.Embedding) != EmbeddingDimensions {
		return fmt.Errorf("search document: embedding length %d, want %d",
			len(d.Embedding), EmbeddingDimensions)
	}
	return nil
}

// SetTimestamps normalises the CreatedAt/UpdatedAt fields to Unix
// epoch seconds. Typesense stores timestamps as int64 so we convert
// at the boundary rather than dragging time.Time through the wire.
func (d *SearchDocument) SetTimestamps(createdAt, updatedAt time.Time) {
	d.CreatedAt = createdAt.Unix()
	d.UpdatedAt = updatedAt.Unix()
}

// SchemaField is the JSON representation of one field in a Typesense
// collection definition. We redeclare it locally (instead of using the
// typesense-go SDK) to keep this package's import surface minimal.
//
// See https://typesense.org/docs/28.0/api/collections.html for the
// full reference of accepted field types and options.
type SchemaField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Facet    bool   `json:"facet,omitempty"`
	Sort     bool   `json:"sort,omitempty"`
	Index    *bool  `json:"index,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Locale   string `json:"locale,omitempty"`
	NumDim   int    `json:"num_dim,omitempty"`
}

// CollectionSchema is the JSON payload posted to POST /collections when
// the backend creates the `marketplace_actors_v1` collection on first
// boot. The schema is intentionally flat — one document per actor —
// because nested facets hurt query performance in Typesense.
type CollectionSchema struct {
	Name                string        `json:"name"`
	Fields              []SchemaField `json:"fields"`
	DefaultSortingField string        `json:"default_sorting_field,omitempty"`
}

// boolPtr returns a pointer to the given bool. Used to fill the
// optional Index field of SchemaField without dragging a helper
// package in.
func boolPtr(b bool) *bool { return &b }

// CollectionSchemaDefinition returns the full 32-field Typesense
// collection schema for the marketplace search index.
//
// Implementation notes:
//   - Facet-enabled fields are those exposed in the filter sidebar
//     (persona, skills, expertise, languages, city, work_mode, etc.).
//   - Sort-enabled fields are those the default ranking formula
//     references (rating_score, profile_completion_score, last_active_at, …).
//   - Text fields use `locale: fr` so Typesense's French tokenizer
//     picks up plurals, accents, and stopwords correctly.
//   - The `embedding` field is the OpenAI 1536-dim vector used for
//     semantic search. `num_dim` is mandatory on float[] fields.
//   - default_sorting_field fires when a query omits `sort_by`.
func CollectionSchemaDefinition() CollectionSchema {
	// Note: Typesense implicitly creates the `id` field on every
	// collection and does NOT accept it in the fields list. We
	// skip declaring it here so the wire payload matches what the
	// server actually persists — otherwise every EnsureSchema run
	// on an existing collection logs a spurious drift warning.
	fields := []SchemaField{
		// Identity
		{Name: "persona", Type: "string", Facet: true},
		{Name: "is_published", Type: "bool", Facet: true},

		// Display
		{Name: "display_name", Type: "string", Locale: "fr"},
		{Name: "title", Type: "string", Locale: "fr", Optional: true},
		{Name: "photo_url", Type: "string", Optional: true, Index: boolPtr(false)},

		// Geo
		{Name: "city", Type: "string", Facet: true, Optional: true},
		{Name: "country_code", Type: "string", Facet: true, Optional: true},
		{Name: "location", Type: "geopoint", Optional: true},
		{Name: "work_mode", Type: "string[]", Facet: true},

		// Languages
		{Name: "languages_professional", Type: "string[]", Facet: true},
		{Name: "languages_conversational", Type: "string[]", Facet: true},

		// Availability
		{Name: "availability_status", Type: "string", Facet: true},
		{Name: "availability_priority", Type: "int32", Sort: true},

		// Expertise
		{Name: "expertise_domains", Type: "string[]", Facet: true},
		{Name: "skills", Type: "string[]", Facet: true},
		{Name: "skills_text", Type: "string", Locale: "fr"},

		// Pricing
		{Name: "pricing_type", Type: "string", Facet: true, Optional: true},
		{Name: "pricing_min_amount", Type: "int64", Sort: true, Optional: true},
		{Name: "pricing_max_amount", Type: "int64", Sort: true, Optional: true},
		{Name: "pricing_currency", Type: "string", Facet: true, Optional: true},
		{Name: "pricing_negotiable", Type: "bool", Facet: true},

		// Quality signals
		{Name: "rating_average", Type: "float", Sort: true},
		{Name: "rating_count", Type: "int32", Sort: true},
		{Name: "rating_score", Type: "float", Sort: true},
		{Name: "total_earned", Type: "int64", Sort: true},
		{Name: "completed_projects", Type: "int32", Sort: true},
		{Name: "profile_completion_score", Type: "int32", Sort: true},
		{Name: "last_active_at", Type: "int64", Sort: true},
		{Name: "response_rate", Type: "float", Sort: true},
		{Name: "is_verified", Type: "bool", Facet: true},
		{Name: "is_top_rated", Type: "bool", Facet: true},
		{Name: "is_featured", Type: "bool", Facet: true},

		// Semantic
		{Name: "embedding", Type: "float[]", NumDim: EmbeddingDimensions, Optional: true},

		// Timestamps
		{Name: "created_at", Type: "int64", Sort: true},
		{Name: "updated_at", Type: "int64", Sort: true},
	}
	return CollectionSchema{
		Name:                CollectionName,
		Fields:              fields,
		DefaultSortingField: "rating_score",
	}
}
