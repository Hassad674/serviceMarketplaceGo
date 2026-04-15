package search_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

func TestMockEmbeddings_Shape(t *testing.T) {
	m := search.NewMockEmbeddings()
	vec, err := m.Embed(context.Background(), "anything")
	require.NoError(t, err)
	assert.Len(t, vec, search.EmbeddingDimensions)
}

func TestMockEmbeddings_Deterministic(t *testing.T) {
	m := search.NewMockEmbeddings()
	a, _ := m.Embed(context.Background(), "first")
	b, _ := m.Embed(context.Background(), "second")
	assert.Equal(t, a, b, "mock must produce the same vector regardless of input")
}

func TestMockEmbeddings_IndependentCopies(t *testing.T) {
	m := search.NewMockEmbeddings()
	a, _ := m.Embed(context.Background(), "x")
	a[0] = 999
	b, _ := m.Embed(context.Background(), "x")
	assert.NotEqual(t, float32(999), b[0], "caller mutation must not leak into later calls")
}

func TestMockEmbeddingsFromSeed_DifferentSeedsDifferentVectors(t *testing.T) {
	a := search.NewMockEmbeddingsFromSeed(1)
	b := search.NewMockEmbeddingsFromSeed(7)
	assert.NotEqual(t, a.Vector(), b.Vector())
}

func TestOpenAIEmbeddings_Constructor(t *testing.T) {
	_, err := search.NewOpenAIEmbeddings("", "text-embedding-3-small")
	assert.ErrorContains(t, err, "api key")

	_, err = search.NewOpenAIEmbeddings("sk-test", "")
	assert.ErrorContains(t, err, "model")

	c, err := search.NewOpenAIEmbeddings("sk-test", "text-embedding-3-small")
	assert.NoError(t, err)
	assert.NotNil(t, c)
}

// TestOpenAIEmbeddings_EmbedSuccess uses an httptest.Server to assert
// the request shape + response parsing without touching the real API.
func TestOpenAIEmbeddings_EmbedSuccess(t *testing.T) {
	// Build a 1536-dim fake embedding response.
	vec := make([]float32, search.EmbeddingDimensions)
	for i := range vec {
		vec[i] = 0.42
	}
	fakeBody := map[string]any{
		"data": []map[string]any{
			{"embedding": vec, "index": 0, "object": "embedding"},
		},
	}

	var gotAuth, gotContentType string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeBody)
	}))
	defer srv.Close()

	client, err := search.NewOpenAIEmbeddings("sk-test-123", "text-embedding-3-small",
		search.WithOpenAIEndpoint(srv.URL),
		search.WithOpenAIHTTPClient(srv.Client()),
	)
	require.NoError(t, err)

	got, err := client.Embed(context.Background(), "nextjs developer paris")
	require.NoError(t, err)
	assert.Len(t, got, search.EmbeddingDimensions)
	assert.InDelta(t, 0.42, got[0], 0.001)

	// Request shape assertions.
	assert.Equal(t, "Bearer sk-test-123", gotAuth)
	assert.Contains(t, gotContentType, "json")
	assert.Equal(t, "text-embedding-3-small", gotBody["model"])
	assert.Equal(t, "nextjs developer paris", gotBody["input"])
}

func TestOpenAIEmbeddings_EmptyInput(t *testing.T) {
	client, err := search.NewOpenAIEmbeddings("sk-test", "text-embedding-3-small",
		search.WithOpenAIEndpoint("http://should-not-hit"),
	)
	require.NoError(t, err)

	_, err = client.Embed(context.Background(), "   ")
	assert.ErrorContains(t, err, "empty input")
}

func TestOpenAIEmbeddings_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited","type":"rate_limit"}}`))
	}))
	defer srv.Close()

	client, _ := search.NewOpenAIEmbeddings("sk-test", "text-embedding-3-small",
		search.WithOpenAIEndpoint(srv.URL),
		search.WithOpenAIHTTPClient(srv.Client()),
	)

	_, err := client.Embed(context.Background(), "x")
	assert.ErrorContains(t, err, "status 429")
}

func TestOpenAIEmbeddings_WrongDimensions(t *testing.T) {
	// Return an embedding with only 3 floats instead of 1536. The
	// client must reject it so the indexer never stores a malformed
	// vector.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{
			"data": []map[string]any{
				{"embedding": []float32{0.1, 0.2, 0.3}},
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()

	client, _ := search.NewOpenAIEmbeddings("sk-test", "text-embedding-3-small",
		search.WithOpenAIEndpoint(srv.URL),
		search.WithOpenAIHTTPClient(srv.Client()),
	)

	_, err := client.Embed(context.Background(), "x")
	assert.ErrorContains(t, err, "got 3 dims")
}
