package search_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// typesenseFake is a tiny in-memory stand-in for the Typesense REST
// API that EnsureSchema uses: just enough to record alias lookups,
// collection creations, and alias upserts so we can assert the
// migration flow without starting a real container.
type typesenseFake struct {
	mu          sync.Mutex
	aliases     map[string]string
	collections map[string]search.CollectionSchema
}

func newTypesenseFake() *typesenseFake {
	return &typesenseFake{
		aliases:     make(map[string]string),
		collections: make(map[string]search.CollectionSchema),
	}
}

func (f *typesenseFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/aliases/"):
		name := strings.TrimPrefix(r.URL.Path, "/aliases/")
		target, ok := f.aliases[name]
		if !ok {
			http.Error(w, `{"message":"alias not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"collection_name": target})

	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/aliases/"):
		name := strings.TrimPrefix(r.URL.Path, "/aliases/")
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		f.aliases[name] = payload["collection_name"]
		_ = json.NewEncoder(w).Encode(payload)

	case r.Method == http.MethodPost && r.URL.Path == "/collections":
		var schema search.CollectionSchema
		if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		if _, exists := f.collections[schema.Name]; exists {
			http.Error(w, `{"message":"collection already exists"}`, http.StatusConflict)
			return
		}
		f.collections[schema.Name] = schema
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(schema)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/collections/"):
		name := strings.TrimPrefix(r.URL.Path, "/collections/")
		schema, ok := f.collections[name]
		if !ok {
			http.Error(w, `{"message":"collection not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(schema)

	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/collections/"):
		// Additive schema PATCH: append the incoming fields to the
		// existing collection. Sufficient to exercise the happy
		// path of inspectExistingAlias's auto-migration logic.
		name := strings.TrimPrefix(r.URL.Path, "/collections/")
		schema, ok := f.collections[name]
		if !ok {
			http.Error(w, `{"message":"collection not found"}`, http.StatusNotFound)
			return
		}
		var payload struct {
			Fields []search.SchemaField `json:"fields"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, `{"message":"bad patch"}`, http.StatusBadRequest)
			return
		}
		schema.Fields = append(schema.Fields, payload.Fields...)
		f.collections[name] = schema
		_ = json.NewEncoder(w).Encode(map[string]any{"fields": payload.Fields})

	// Synonym endpoints (not the focus of migration tests).
	case strings.HasPrefix(r.URL.Path, "/collections/") && strings.Contains(r.URL.Path, "/synonyms"):
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))

	default:
		// Not implemented for the migration tests — helps catch
		// accidental reliance on unmocked endpoints.
		http.Error(w, `{"message":"fake: not implemented: `+r.Method+" "+r.URL.Path+`"}`, http.StatusNotImplemented)
	}
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEnsureSchema_FreshBootstrap(t *testing.T) {
	fake := newTypesenseFake()
	client, _ := newTestClient(t, fake)

	err := search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
		Client: client,
		Logger: silentLogger(),
	})
	require.NoError(t, err)

	fake.mu.Lock()
	defer fake.mu.Unlock()

	// Collection was created.
	_, hasCollection := fake.collections[search.CollectionName]
	assert.True(t, hasCollection, "collection %s must exist", search.CollectionName)
	// Alias points to it.
	assert.Equal(t, search.CollectionName, fake.aliases[search.AliasName])
}

func TestEnsureSchema_IdempotentOnSecondRun(t *testing.T) {
	fake := newTypesenseFake()
	client, _ := newTestClient(t, fake)

	// First call: creates everything.
	require.NoError(t, search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
		Client: client,
		Logger: silentLogger(),
	}))

	// Second call: must be a no-op, no fresh create.
	require.NoError(t, search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
		Client: client,
		Logger: silentLogger(),
	}))

	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.Len(t, fake.collections, 1, "second run must not create a new collection")
}

func TestEnsureSchema_NilClient(t *testing.T) {
	err := search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{Client: nil})
	assert.ErrorContains(t, err, "client is required")
}

// TestEnsureSchema_AdditiveDrift_AutoPatched locks the auto-apply
// behaviour introduced in phase 6B: when the only difference between
// live and expected is additive (new fields with matching types),
// EnsureSchema PATCHes the live collection in place so operators
// don't have to run a manual `_vN` alias swap.
func TestEnsureSchema_AdditiveDrift_AutoPatched(t *testing.T) {
	fake := newTypesenseFake()
	legacyName := "marketplace_actors_legacy"
	fake.aliases[search.AliasName] = legacyName
	// The expected schema drops the implicit `id` but keeps N other
	// fields — we simulate additive drift by starting from the full
	// expected schema MINUS the ranking V1 signals. Any 7-field
	// mismatch would do, but using the actual missing fields makes
	// the intent self-documenting.
	expected := search.CollectionSchemaDefinition()
	base := make([]search.SchemaField, 0, len(expected.Fields)-7)
	skipNames := map[string]bool{
		"unique_clients_count": true, "repeat_client_rate": true,
		"unique_reviewers_count": true, "max_reviewer_share": true,
		"review_recency_factor": true, "lost_disputes_count": true,
		"account_age_days": true,
	}
	for _, f := range expected.Fields {
		if skipNames[f.Name] {
			continue
		}
		base = append(base, f)
	}
	fake.collections[legacyName] = search.CollectionSchema{
		Name:   legacyName,
		Fields: base,
	}

	client, _ := newTestClient(t, fake)

	err := search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
		Client: client,
		Logger: silentLogger(),
	})
	require.NoError(t, err)

	fake.mu.Lock()
	defer fake.mu.Unlock()
	// Alias still points at the legacy collection (we did NOT
	// swap — we patched in place).
	assert.Equal(t, legacyName, fake.aliases[search.AliasName])
	// The live collection now has the full expected field count.
	assert.Equal(t, len(expected.Fields), len(fake.collections[legacyName].Fields),
		"additive drift must be patched to reach parity with expected schema")
}

// TestEnsureSchema_NonAdditiveDrift_NoAutoMigrate verifies the
// safety net: when the live collection has a field with a
// different TYPE than the expected schema, EnsureSchema logs a
// warning and leaves the cluster alone. PATCH cannot express a
// type change without data loss, so automation is deliberately
// off for this case.
func TestEnsureSchema_NonAdditiveDrift_NoAutoMigrate(t *testing.T) {
	fake := newTypesenseFake()
	legacyName := "marketplace_actors_legacy"
	fake.aliases[search.AliasName] = legacyName
	fake.collections[legacyName] = search.CollectionSchema{
		Name: legacyName,
		// Conflicts: persona declared with the wrong type.
		Fields: []search.SchemaField{
			{Name: "persona", Type: "int32"},
			{Name: "display_name", Type: "string"},
		},
	}

	client, _ := newTestClient(t, fake)
	err := search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
		Client: client,
		Logger: silentLogger(),
	})
	assert.NoError(t, err, "non-additive drift should warn, not fail")

	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.Equal(t, 2, len(fake.collections[legacyName].Fields),
		"non-additive drift must not be patched")
}
