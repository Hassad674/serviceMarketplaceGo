package search

import (
	"encoding/json"
	"fmt"

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
	var resp rawTypesenseResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse query result: %w", err)
	}

	docs := make([]search.SearchDocument, 0, len(resp.Hits))
	highlights := make([]map[string]string, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		doc := hit.Document
		doc.Embedding = nil // never leak the 1536-dim vector
		docs = append(docs, doc)
		highlights = append(highlights, collectHighlights(hit.Highlights))
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
	}, nil
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
