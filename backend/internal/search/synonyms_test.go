package search

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// synonyms_test.go covers the synonyms seed list + upsert wire
// format using a stub HTTP server. The integration test in
// integration_test.go exercises the same path against a real
// Typesense container so we know the contract is honoured by the
// upstream.

func TestDefaultSynonyms_HasExpectedShape(t *testing.T) {
	got := DefaultSynonyms()
	assert.GreaterOrEqual(t, len(got), 30,
		"default synonyms list must contain at least 30 FR/EN pairs (phase 2 spec)")

	// Every entry must have a stable ID and a non-empty synonyms
	// slice. Root is allowed to be empty (multi-way synonyms) but
	// every seed in the spec uses a Root, so we assert it here.
	seen := make(map[string]struct{}, len(got))
	for _, syn := range got {
		assert.NotEmpty(t, syn.ID, "every synonym must have an ID")
		_, dup := seen[syn.ID]
		assert.False(t, dup, "synonym IDs must be unique: %q", syn.ID)
		seen[syn.ID] = struct{}{}

		assert.NotEmpty(t, syn.Synonyms, "synonym %q must have at least one synonym", syn.ID)
		assert.NotEmpty(t, syn.Root, "synonym %q must have a non-empty Root", syn.ID)
	}
}

func TestUpsertSynonym_BuildsCorrectRequest(t *testing.T) {
	var calls int32
	var capturedPath string
	var capturedMethod string
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "test-key")
	require.NoError(t, err)

	syn := Synonym{
		ID:       "s_test",
		Root:     "frontend",
		Synonyms: []string{"front-end", "front end"},
	}
	require.NoError(t, client.UpsertSynonym(context.Background(), "marketplace_actors_v1", syn))

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
	assert.Equal(t, http.MethodPut, capturedMethod)
	assert.Equal(t, "/collections/marketplace_actors_v1/synonyms/s_test", capturedPath)

	var payload Synonym
	require.NoError(t, json.Unmarshal(capturedBody, &payload))
	assert.Equal(t, "frontend", payload.Root)
	assert.Equal(t, []string{"front-end", "front end"}, payload.Synonyms)
}

func TestUpsertSynonym_RejectsInvalidInputs(t *testing.T) {
	client, err := NewClient("http://localhost:8108", "test-key")
	require.NoError(t, err)

	tests := []struct {
		name    string
		syn     Synonym
		wantErr string
	}{
		{
			name:    "missing id",
			syn:     Synonym{Synonyms: []string{"a"}},
			wantErr: "id is required",
		},
		{
			name:    "empty synonyms slice",
			syn:     Synonym{ID: "s_x"},
			wantErr: "synonyms list is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.UpsertSynonym(context.Background(), "marketplace_actors_v1", tt.syn)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSeedSynonyms_UpsertsEverySynonymOnce(t *testing.T) {
	var calls int32
	upsertedIDs := make(map[string]bool)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		// Path: /collections/marketplace_actors_v1/synonyms/s_xxx
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(parts) >= 4 {
			upsertedIDs[parts[3]] = true
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "test-key")
	require.NoError(t, err)

	require.NoError(t, SeedSynonyms(context.Background(), client, slog.Default()))

	expected := DefaultSynonyms()
	assert.Equal(t, int32(len(expected)), atomic.LoadInt32(&calls),
		"every synonym must be upserted exactly once")
	for _, syn := range expected {
		assert.True(t, upsertedIDs[syn.ID], "synonym %q must be upserted", syn.ID)
	}
}

func TestSeedSynonyms_ReturnsErrorOnFirstFailure(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "boom"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "test-key")
	require.NoError(t, err)

	err = SeedSynonyms(context.Background(), client, slog.Default())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "synonyms seed")
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "should fail fast on first error")
}
