package search

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parser_ranking_test.go pins the text-match bucketing logic + the
// new parseQueryResultWithHits entry point consumed by the ranking
// pipeline wiring.

func TestParseQueryResultWithHits_SurfacesRawTextMatch(t *testing.T) {
	raw := json.RawMessage(`{
		"found": 3,
		"out_of": 3,
		"page": 1,
		"per_page": 3,
		"hits": [
			{"document": {"id":"a","organization_id":"o","persona":"freelance","is_published":true,"display_name":"A"},
			 "text_match": 1000, "text_match_info": {"score": "1000"}},
			{"document": {"id":"b","organization_id":"o","persona":"freelance","is_published":true,"display_name":"B"},
			 "text_match": 500},
			{"document": {"id":"c","organization_id":"o","persona":"freelance","is_published":true,"display_name":"C"},
			 "text_match": 100}
		]
	}`)
	_, hits, err := parseQueryResultWithHits(raw)
	require.NoError(t, err)
	require.Len(t, hits, 3)
	// Normalisation rule: top hit = bucket 10.
	assert.Equal(t, 10, hits[0].TextMatchBucket)
	// Half score → bucket 5.
	assert.Equal(t, 5, hits[1].TextMatchBucket)
	// 10% score → bucket 1.
	assert.Equal(t, 1, hits[2].TextMatchBucket)
}

func TestParseQueryResultWithHits_EmptyQueryReturnsZeroBuckets(t *testing.T) {
	// On q=* the backend skips embedding AND Typesense returns 0 /
	// omits the text_match field — every hit lands in bucket 0.
	raw := json.RawMessage(`{
		"found": 2,
		"hits": [
			{"document": {"id":"a","organization_id":"o","persona":"freelance","is_published":true,"display_name":"A"}},
			{"document": {"id":"b","organization_id":"o","persona":"freelance","is_published":true,"display_name":"B"}}
		]
	}`)
	_, hits, err := parseQueryResultWithHits(raw)
	require.NoError(t, err)
	require.Len(t, hits, 2)
	assert.Equal(t, 0, hits[0].TextMatchBucket)
	assert.Equal(t, 0, hits[1].TextMatchBucket)
}

func TestComputeTextMatchBuckets_Linear(t *testing.T) {
	cases := []struct {
		name string
		raw  []float64
		want []int
	}{
		{"empty", []float64{}, []int{}},
		{"all_zero", []float64{0, 0, 0}, []int{0, 0, 0}},
		{"single_top", []float64{5}, []int{10}},
		{"decreasing", []float64{100, 50, 10}, []int{10, 5, 1}},
		{"ties_same_bucket", []float64{10, 10, 10}, []int{10, 10, 10}},
		{"negative_clamped", []float64{-1, 10}, []int{0, 10}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeTextMatchBuckets(c.raw)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestResolveTextMatchRaw_PrefersTextMatchInfoScore(t *testing.T) {
	// When both fields are present, text_match_info.score wins
	// because it retains full precision — the int field may truncate
	// on some Typesense builds.
	got := resolveTextMatchRaw(rawTypesenseHit{
		TextMatch:     42,
		TextMatchInfo: rawTypesenseTextMatchInfo{Score: "123.5"},
	})
	assert.InDelta(t, 123.5, got, 1e-9)
}

func TestResolveTextMatchRaw_FallsBackToInt(t *testing.T) {
	got := resolveTextMatchRaw(rawTypesenseHit{TextMatch: 42})
	assert.InDelta(t, 42.0, got, 1e-9)
}

func TestResolveTextMatchRaw_InvalidScoreFallsBackToInt(t *testing.T) {
	got := resolveTextMatchRaw(rawTypesenseHit{
		TextMatch:     77,
		TextMatchInfo: rawTypesenseTextMatchInfo{Score: "not-a-number"},
	})
	assert.InDelta(t, 77.0, got, 1e-9)
}

func TestParseQueryResultWithHits_InvalidJSONReturnsError(t *testing.T) {
	_, _, err := parseQueryResultWithHits(json.RawMessage(`not a json`))
	assert.Error(t, err)
}

func TestParseQueryResult_BackwardCompatible(t *testing.T) {
	// parseQueryResult still works — the two-value wrapper above must
	// not break the legacy single-value entry point.
	raw := json.RawMessage(`{"found":1,"hits":[{"document":{"id":"a","organization_id":"o","persona":"freelance","is_published":true,"display_name":"A"}}]}`)
	r, err := parseQueryResult(raw)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, 1, r.Found)
}

func TestParseQueryResultWithHits_KeepsDocumentEmbeddingStripped(t *testing.T) {
	// Regression guard — the ranking path must not re-introduce
	// the embedding vector into either the QueryResult.Documents
	// slice or the []TypesenseHit.
	raw := json.RawMessage(`{
		"found": 1,
		"hits": [
			{"document": {"id":"a","organization_id":"o","persona":"freelance","is_published":true,"display_name":"A","embedding":[0.1,0.2,0.3]},
			 "text_match": 100}
		]
	}`)
	res, hits, err := parseQueryResultWithHits(raw)
	require.NoError(t, err)
	require.Len(t, res.Documents, 1)
	require.Len(t, hits, 1)
	assert.Nil(t, res.Documents[0].Embedding)
	assert.Nil(t, hits[0].Document.Embedding)
}
