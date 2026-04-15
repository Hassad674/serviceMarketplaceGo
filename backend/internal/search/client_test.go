package search_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// newTestClient wires a search.Client against an httptest.Server.
// Every test that needs a fake Typesense backend goes through this
// helper so the plumbing stays consistent.
func newTestClient(t *testing.T, handler http.Handler) (*search.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client, err := search.NewClient(srv.URL, "test-key", search.WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	return client, srv
}

func validDoc() *search.SearchDocument {
	return &search.SearchDocument{
		ID:          uuid.NewString(),
		Persona:     search.PersonaFreelance,
		DisplayName: "Alice",
		IsPublished: true,
		WorkMode:    []string{"remote"},
	}
}

func TestNewClient_Validation(t *testing.T) {
	_, err := search.NewClient("", "key")
	assert.ErrorContains(t, err, "host is required")

	_, err = search.NewClient("http://localhost:8108", "")
	assert.ErrorContains(t, err, "api key is required")

	_, err = search.NewClient("not-a-url", "key")
	assert.ErrorContains(t, err, "scheme and hostname")

	_, err = search.NewClient("http://localhost:8108", "key")
	assert.NoError(t, err)
}

func TestClient_Ping(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))

		assert.NoError(t, client.Ping(context.Background()))
	})

	t.Run("unhealthy", func(t *testing.T) {
		client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))

		assert.ErrorContains(t, client.Ping(context.Background()), "unexpected status")
	})
}

func TestClient_CreateCollection(t *testing.T) {
	called := false
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/collections", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("X-TYPESENSE-API-KEY"))
		assert.Contains(t, r.Header.Get("Content-Type"), "json")

		var got search.CollectionSchema
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, search.CollectionName, got.Name)

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"name":"marketplace_actors_v1"}`))
	}))

	schema := search.CollectionSchemaDefinition()
	assert.NoError(t, client.CreateCollection(context.Background(), schema))
	assert.True(t, called)
}

func TestClient_UpsertAlias(t *testing.T) {
	var received aliasRequest
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/aliases/marketplace_actors", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"collection_name":"marketplace_actors_v1"}`))
	}))

	assert.NoError(t, client.UpsertAlias(context.Background(), search.AliasName, search.CollectionName))
	assert.Equal(t, search.CollectionName, received.CollectionName)
}

type aliasRequest struct {
	CollectionName string `json:"collection_name"`
}

func TestClient_GetAlias_NotFound(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	_, err := client.GetAlias(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, search.ErrNotFound)
}

func TestClient_UpsertDocument(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.RawQuery, "action=upsert")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))

	assert.NoError(t, client.UpsertDocument(context.Background(), search.CollectionName, validDoc()))
}

func TestClient_UpsertDocument_RejectsInvalid(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach network on invalid doc")
	}))

	doc := validDoc()
	doc.ID = "" // invalidate
	assert.ErrorContains(t, client.UpsertDocument(context.Background(), search.CollectionName, doc), "id is required")
}

func TestClient_DeleteDocument_IdempotentOnNotFound(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNotFound)
	}))

	// Not-found must be a no-op, not an error — deleting an already
	// deleted actor is the same end state as a successful delete.
	assert.NoError(t, client.DeleteDocument(context.Background(), search.CollectionName, uuid.NewString()))
}

func TestClient_DeleteDocument_Success(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))

	assert.NoError(t, client.DeleteDocument(context.Background(), search.CollectionName, uuid.NewString()))
}

func TestClient_BulkUpsert_Batches(t *testing.T) {
	batchCount := 0
	lineCount := 0
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "action=upsert")
		assert.Contains(t, r.Header.Get("Content-Type"), "ndjson")
		body, _ := io.ReadAll(r.Body)
		lineCount += strings.Count(string(body), "\n")
		batchCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))

	docs := make([]*search.SearchDocument, 0, 250)
	for i := 0; i < 250; i++ {
		docs = append(docs, validDoc())
	}
	require.NoError(t, client.BulkUpsert(context.Background(), search.CollectionName, docs))

	// 250 docs at a batch size of 100 → 3 batches (100+100+50).
	assert.Equal(t, 3, batchCount)
	assert.Equal(t, 250, lineCount)
}

func TestClient_BulkUpsert_EmptyIsNoop(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("empty bulk must not hit the network")
	}))

	assert.NoError(t, client.BulkUpsert(context.Background(), search.CollectionName, nil))
}

func TestClient_UnauthorizedMapsToSentinel(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	_, err := client.GetCollection(context.Background(), "marketplace_actors_v1")
	assert.True(t, errors.Is(err, search.ErrUnauthorized))
}

func TestClient_Query_BuildsFilterByInURL(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "go developer", q.Get("q"))
		assert.Equal(t, "title,skills_text", q.Get("query_by"))
		assert.Equal(t, "persona:freelance && is_published:true", q.Get("filter_by"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":[]}`))
	}))

	params := search.SearchParams{
		Q:        "go developer",
		QueryBy:  "title,skills_text",
		FilterBy: "persona:freelance && is_published:true",
	}
	_, err := client.Query(context.Background(), search.CollectionName, params)
	assert.NoError(t, err)
}
