package search_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// TestPersona_IsValid locks in the three accepted personas and
// rejects every unknown string so a future typo cannot sneak a bogus
// document into the index.
func TestPersona_IsValid(t *testing.T) {
	cases := []struct {
		in   search.Persona
		want bool
	}{
		{search.PersonaFreelance, true},
		{search.PersonaAgency, true},
		{search.PersonaReferrer, true},
		{"", false},
		{"admin", false},
		{"freelancer", false}, // typo on purpose
	}
	for _, c := range cases {
		t.Run(string(c.in), func(t *testing.T) {
			assert.Equal(t, c.want, c.in.IsValid())
		})
	}
}

func TestSearchDocument_Validate(t *testing.T) {
	base := func() *search.SearchDocument {
		orgID := uuid.NewString()
		return &search.SearchDocument{
			ID:             orgID + ":freelance",
			OrganizationID: orgID,
			Persona:        search.PersonaFreelance,
			DisplayName:    "Alice Dupont",
		}
	}

	t.Run("valid minimal document", func(t *testing.T) {
		doc := base()
		assert.NoError(t, doc.Validate())
	})

	t.Run("empty id", func(t *testing.T) {
		doc := base()
		doc.ID = ""
		assert.ErrorContains(t, doc.Validate(), "id is required")
	})

	t.Run("empty organization_id", func(t *testing.T) {
		doc := base()
		doc.OrganizationID = ""
		assert.ErrorContains(t, doc.Validate(), "organization_id is required")
	})

	t.Run("non-uuid organization_id", func(t *testing.T) {
		doc := base()
		doc.OrganizationID = "not-a-uuid"
		assert.ErrorContains(t, doc.Validate(), "not a valid UUID")
	})

	t.Run("invalid persona", func(t *testing.T) {
		doc := base()
		doc.Persona = "enterprise"
		assert.ErrorContains(t, doc.Validate(), "persona")
	})

	t.Run("missing display name", func(t *testing.T) {
		doc := base()
		doc.DisplayName = ""
		assert.ErrorContains(t, doc.Validate(), "display_name")
	})

	t.Run("embedding wrong length", func(t *testing.T) {
		doc := base()
		doc.Embedding = []float32{0.1, 0.2, 0.3}
		assert.ErrorContains(t, doc.Validate(), "embedding length")
	})

	t.Run("embedding correct length", func(t *testing.T) {
		doc := base()
		doc.Embedding = make([]float32, search.EmbeddingDimensions)
		assert.NoError(t, doc.Validate())
	})
}

func TestSearchDocument_SetTimestamps(t *testing.T) {
	doc := &search.SearchDocument{}
	created := time.Date(2026, 4, 15, 10, 30, 0, 0, time.UTC)
	updated := time.Date(2026, 4, 15, 10, 35, 0, 0, time.UTC)

	doc.SetTimestamps(created, updated)

	assert.Equal(t, created.Unix(), doc.CreatedAt)
	assert.Equal(t, updated.Unix(), doc.UpdatedAt)
}

// TestCollectionSchemaDefinition verifies the exact field count and
// the set of fields so any accidental removal (or silent rename)
// shows up immediately.
func TestCollectionSchemaDefinition(t *testing.T) {
	schema := search.CollectionSchemaDefinition()

	require.Equal(t, search.CollectionName, schema.Name)
	require.Equal(t, "rating_score", schema.DefaultSortingField)
	// 43 explicit fields (36 original + 7 ranking V1 signals added in
	// phase 6B) + the implicit `id` Typesense auto-creates = 44 total
	// on the server side.
	require.Len(t, schema.Fields, 43,
		"schema must keep its canonical field count; update both schema and test deliberately")

	byName := make(map[string]search.SchemaField, len(schema.Fields))
	for _, f := range schema.Fields {
		byName[f.Name] = f
	}

	// A handful of anchor invariants — we don't assert on every
	// field (the test would become a copy of the source) but we do
	// pin the load-bearing ones. `id` is deliberately absent: it is
	// an implicit field on every Typesense collection.
	assert.True(t, byName["persona"].Facet, "persona must be a facet")
	assert.True(t, byName["skills"].Facet)
	assert.True(t, byName["rating_score"].Sort)
	assert.True(t, byName["availability_priority"].Sort)
	assert.True(t, byName["profile_completion_score"].Sort)
	assert.True(t, byName["last_active_at"].Sort)
	assert.Equal(t, "fr", byName["display_name"].Locale)
	assert.Equal(t, "fr", byName["title"].Locale)
	assert.Equal(t, "fr", byName["skills_text"].Locale)
	assert.Equal(t, "geopoint", byName["location"].Type)
	assert.Equal(t, "float[]", byName["embedding"].Type)
	assert.Equal(t, search.EmbeddingDimensions, byName["embedding"].NumDim)

	// Ranking V1 signals (phase 6B) — sortable so future scorers can
	// reference them, optional so alias-swap on an existing collection
	// does not require backfilling legacy docs.
	for _, name := range []string{
		"unique_clients_count", "repeat_client_rate", "unique_reviewers_count",
		"max_reviewer_share", "review_recency_factor", "lost_disputes_count",
		"account_age_days",
	} {
		t.Run("ranking_v1_signal/"+name, func(t *testing.T) {
			f, ok := byName[name]
			require.True(t, ok, "%s must be declared in the schema", name)
			assert.True(t, f.Sort, "%s must be sortable", name)
			assert.True(t, f.Optional, "%s must be optional (alias-swap safe)", name)
			assert.False(t, f.Facet, "%s must not be a facet (numeric signal)", name)
		})
	}

	// Optional fields on the SQL side must be optional on the
	// Typesense side too — otherwise Typesense rejects documents
	// with missing values.
	for _, name := range []string{
		"title", "photo_url", "city", "country_code", "location",
		"pricing_type", "pricing_min_amount", "pricing_max_amount",
		"pricing_currency", "embedding",
	} {
		t.Run("optional/"+name, func(t *testing.T) {
			assert.True(t, byName[name].Optional, "%s must be optional", name)
		})
	}
}

// TestCollectionSchemaDefinition_JSONRoundTrip ensures the schema
// serialises to a payload Typesense actually accepts. We verify the
// wire-format keys rather than dumping the full JSON — the structure
// is the contract here, not the indentation.
func TestCollectionSchemaDefinition_JSONRoundTrip(t *testing.T) {
	schema := search.CollectionSchemaDefinition()
	payload, err := json.Marshal(schema)
	require.NoError(t, err)

	wire := string(payload)
	assert.Contains(t, wire, `"name":"marketplace_actors_v1"`)
	assert.Contains(t, wire, `"default_sorting_field":"rating_score"`)
	assert.Contains(t, wire, `"num_dim":1536`)
	assert.Contains(t, wire, `"type":"geopoint"`)
	// Non-indexed fields use the Index pointer — make sure we emit
	// `"index":false` for photo_url.
	assert.Contains(t, wire, `"photo_url"`)
	assert.True(t, strings.Contains(wire, `"index":false`))
}
