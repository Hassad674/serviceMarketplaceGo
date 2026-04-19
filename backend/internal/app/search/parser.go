package search

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"marketplace-backend/internal/search"
)

// parser.go decodes the raw Typesense /documents/search response
// into the typed QueryResult struct exposed by the service. The
// wire format is stable enough that we can parse it directly with
// `encoding/json` — no unmarshaller hooks, no reflection tricks.
//
// Reference:
// https://typesense.org/docs/latest/api/search.html#search-results

// rawTypesenseResponse mirrors the subset of fields we care about.
// Anything we do not decode here (e.g. request_params,
// search_cutoff) is silently dropped by the JSON decoder.
type rawTypesenseResponse struct {
	Found        int                  `json:"found"`
	OutOf        int                  `json:"out_of"`
	Page         int                  `json:"page"`
	PerPage      int                  `json:"per_page"`
	SearchTimeMs int                  `json:"search_time_ms"`
	Hits         []rawTypesenseHit    `json:"hits"`
	FacetCounts  []rawTypesenseFacet  `json:"facet_counts"`

	// RequestParams contains the actual q Typesense ran. When
	// the spell-checker fires, Typesense exposes the corrected
	// term via this field's `q` plus the legacy `corrected_query`
	// field at the top level. We read both so we are robust
	// against future server versions that drop one or the other.
	RequestParams  rawRequestParams `json:"request_params"`
	CorrectedQuery string           `json:"corrected_query"`
}

type rawRequestParams struct {
	FirstQ string `json:"first_q"`
	Q      string `json:"q"`
}

type rawTypesenseHit struct {
	Document   search.SearchDocument `json:"document"`
	Highlights []rawTypesenseHilite  `json:"highlights"`

	// TextMatch is Typesense's raw BM25 score for the hit (integer,
	// unbucketed). Present on every hit of a /documents/search
	// response — see https://typesense.org/docs/latest/api/search.html
	// §Response format.
	TextMatch int64 `json:"text_match"`

	// TextMatchInfo is the structured match info Typesense returns
	// when buckets:N is set in sort_by. It carries the raw score as a
	// string so callers can distinguish the bucketed sort order from
	// the pure BM25 value. We only use `Score` (the raw value) for
	// normalisation; the `Fields` nested object is dropped to avoid
	// allocating arrays we never read.
	TextMatchInfo rawTypesenseTextMatchInfo `json:"text_match_info"`
}

// rawTypesenseTextMatchInfo mirrors the `text_match_info` sub-object
// Typesense emits when bucketed sorting is active.
type rawTypesenseTextMatchInfo struct {
	Score string `json:"score"`
}

type rawTypesenseHilite struct {
	Field   string `json:"field"`
	Snippet string `json:"snippet"`
}

type rawTypesenseFacet struct {
	FieldName string                  `json:"field_name"`
	Counts    []rawTypesenseFacetItem `json:"counts"`
}

type rawTypesenseFacetItem struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// parseQueryResult is the entry point used by Service.Query. It
// unwraps the raw JSON, normalises the hits + facet counts, and
// strips the embedding vectors from each document so the API
// response stays small.
func parseQueryResult(raw json.RawMessage) (*QueryResult, error) {
	result, _, err := parseQueryResultWithHits(raw)
	return result, err
}

// parseQueryResultWithHits extends parseQueryResult with the per-hit
// TypesenseHit slice the ranking pipeline consumes. The QueryResult
// still carries the stripped-down SearchDocument slice used by the
// JSON response; the parallel []TypesenseHit carries the same docs
// plus the bucketed _text_match score needed for feature extraction.
//
// Returning two slices rather than extending QueryResult keeps the
// API shape stable — adding a non-json field to QueryResult worked,
// but returning a second value makes the contract explicit and lets
// tests pin the two decoders independently.
func parseQueryResultWithHits(raw json.RawMessage) (*QueryResult, []TypesenseHit, error) {
	var resp rawTypesenseResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, nil, fmt.Errorf("parse query result: %w", err)
	}

	docs := make([]search.SearchDocument, 0, len(resp.Hits))
	highlights := make([]map[string]string, 0, len(resp.Hits))
	hits := make([]TypesenseHit, 0, len(resp.Hits))
	rawScores := make([]float64, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		doc := hit.Document
		doc.Embedding = nil // never leak the 1536-dim vector
		docs = append(docs, doc)
		highlights = append(highlights, collectHighlights(hit.Highlights))
		hits = append(hits, TypesenseHit{Document: doc})
		rawScores = append(rawScores, resolveTextMatchRaw(hit))
	}

	// Bucket the raw scores into [0, 10]. Typesense returns BM25
	// values whose magnitude varies per query — normalising against
	// the top-hit score makes the bucket meaningful across queries.
	buckets := computeTextMatchBuckets(rawScores)
	for i := range hits {
		hits[i].TextMatchBucket = buckets[i]
	}

	return &QueryResult{
		Documents:      docs,
		Found:          resp.Found,
		OutOf:          resp.OutOf,
		Page:           resp.Page,
		PerPage:        resp.PerPage,
		SearchTimeMs:   resp.SearchTimeMs,
		FacetCounts:    transformFacetCounts(resp.FacetCounts),
		CorrectedQuery: pickCorrectedQuery(resp),
		Highlights:     highlights,
	}, hits, nil
}

// resolveTextMatchRaw returns the raw match score for a hit, preferring
// the structured text_match_info.score (string) when present because
// it retains full precision — the integer text_match field loses the
// lower bits above 2^31 on 32-bit JSON parsers.
//
// Falls back to 0 for empty-query hits (q=*) where Typesense does not
// emit a match score at all. The ranking pipeline's stage-2 extractor
// maps 0 to a bucket of 0, which the scorer then redistributes via the
// empty-query branch (§5.2).
func resolveTextMatchRaw(hit rawTypesenseHit) float64 {
	if hit.TextMatchInfo.Score != "" {
		v, err := strconv.ParseFloat(hit.TextMatchInfo.Score, 64)
		if err == nil {
			return v
		}
	}
	return float64(hit.TextMatch)
}

// computeTextMatchBuckets maps the per-query raw BM25 scores into the
// [0, 10] bucket range docs/ranking-v1.md §3.2-1 consumes. The
// top-scoring hit is normalised to 10; ties land in the same bucket;
// hits with a zero raw score stay in bucket 0 (useful on q=*).
//
// The bucket is deliberately linear in the raw score (bucket_i =
// round(10 * score_i / max_score)) rather than log-based because the
// scorer already applies the downstream mapping; a log here would
// stack the non-linearity twice.
func computeTextMatchBuckets(raw []float64) []int {
	out := make([]int, len(raw))
	if len(raw) == 0 {
		return out
	}
	var maxScore float64
	for _, v := range raw {
		if v > maxScore {
			maxScore = v
		}
	}
	if maxScore <= 0 {
		return out
	}
	for i, v := range raw {
		if v <= 0 {
			out[i] = 0
			continue
		}
		bucket := int(math.Round(10 * v / maxScore))
		if bucket < 0 {
			bucket = 0
		}
		if bucket > 10 {
			bucket = 10
		}
		out[i] = bucket
	}
	return out
}

// collectHighlights converts Typesense's `highlights` array into a
// flat map keyed by field name. We pick the first snippet per field
// (Typesense already returns them ordered by relevance) so the
// frontend can render `<mark>` tags without iterating a list.
func collectHighlights(in []rawTypesenseHilite) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for _, h := range in {
		if _, ok := out[h.Field]; ok {
			continue
		}
		out[h.Field] = h.Snippet
	}
	return out
}

// transformFacetCounts turns the slice-of-facets shape Typesense
// returns into a nested map for ergonomic frontend access:
//
//	{
//	  "skills":              {"react": 12, "go": 8, ...},
//	  "languages_professional": {"fr": 30, "en": 22, ...},
//	}
func transformFacetCounts(in []rawTypesenseFacet) map[string]map[string]int {
	if len(in) == 0 {
		return map[string]map[string]int{}
	}
	out := make(map[string]map[string]int, len(in))
	for _, facet := range in {
		bucket := make(map[string]int, len(facet.Counts))
		for _, c := range facet.Counts {
			bucket[c.Value] = c.Count
		}
		out[facet.FieldName] = bucket
	}
	return out
}

// pickCorrectedQuery returns the spell-corrected query when Typesense
// ran one, or an empty string when it did not. We prefer the
// top-level `corrected_query` field because newer Typesense
// versions populate it; older clusters expose the corrected term
// indirectly via `request_params.q != first_q`.
func pickCorrectedQuery(resp rawTypesenseResponse) string {
	if resp.CorrectedQuery != "" {
		return resp.CorrectedQuery
	}
	if resp.RequestParams.FirstQ != "" && resp.RequestParams.Q != "" &&
		resp.RequestParams.FirstQ != resp.RequestParams.Q {
		return resp.RequestParams.Q
	}
	return ""
}
