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

func TestEnsureSchema_DriftDetectedWithoutAutoMigrate(t *testing.T) {
	fake := newTypesenseFake()
	// Pre-populate an alias pointing at a "legacy" collection that
	// has a different field count than the canonical schema.
	legacyName := "marketplace_actors_legacy"
	fake.aliases[search.AliasName] = legacyName
	fake.collections[legacyName] = search.CollectionSchema{
		Name:   legacyName,
		Fields: []search.SchemaField{{Name: "id", Type: "string"}}, // only 1 field
	}

	client, _ := newTestClient(t, fake)

	// Migration must NOT error and must NOT try to auto-migrate.
	err := search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
		Client: client,
		Logger: silentLogger(),
	})
	assert.NoError(t, err)

	fake.mu.Lock()
	defer fake.mu.Unlock()
	// Alias still points at the legacy collection.
	assert.Equal(t, legacyName, fake.aliases[search.AliasName])
	// No new canonical collection was created.
	_, created := fake.collections[search.CollectionName]
	assert.False(t, created, "EnsureSchema must not auto-migrate on drift")
}
