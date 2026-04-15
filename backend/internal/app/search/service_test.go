package search

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// fakeClient is a deterministic stub used by every service test.
// It captures the SearchParams the service produces and returns a
// canned JSON response so we can assert on the parsed result.
type fakeClient struct {
	persona      search.Persona
	gotParams    search.SearchParams
	respPayload  string
	respErr      error
}

func (f *fakeClient) Persona() search.Persona { return f.persona }

func (f *fakeClient) Query(_ context.Context, params search.SearchParams) (json.RawMessage, error) {
	f.gotParams = params
	if f.respErr != nil {
		return nil, f.respErr
	}
	return json.RawMessage(f.respPayload), nil
}

func newFreelanceClient(payload string) *fakeClient {
	return &fakeClient{persona: search.PersonaFreelance, respPayload: payload}
}

func TestNewService_DropsNilPersonas(t *testing.T) {
	svc := NewService(ServiceDeps{
		Freelance: newFreelanceClient(`{}`),
	})
	assert.True(t, svc.HasPersona(search.PersonaFreelance))
	assert.False(t, svc.HasPersona(search.PersonaAgency))
	assert.False(t, svc.HasPersona(search.PersonaReferrer))
}

func TestService_Query_PersonaNotConfigured(t *testing.T) {
	svc := NewService(ServiceDeps{})
	_, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPersonaNotConfigured))
}

func TestService_Query_DefaultsApplied(t *testing.T) {
	stub := newFreelanceClient(`{"found":0,"hits":[]}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.NoError(t, err)

	// Empty query becomes "*"
	assert.Equal(t, "*", stub.gotParams.Q)
	// Default sort_by from the search package
	assert.Equal(t, search.DefaultSortBy(), stub.gotParams.SortBy)
	// Default page + per_page
	assert.Equal(t, DefaultPage, stub.gotParams.Page)
	assert.Equal(t, DefaultPerPage, stub.gotParams.PerPage)
	// Embedding excluded so the wire payload stays small
	assert.Contains(t, stub.gotParams.ExcludeFields, "embedding")
	// Highlights enabled
	assert.NotEmpty(t, stub.gotParams.HighlightFields)
}

func TestService_Query_PerPageCappedAtMax(t *testing.T) {
	stub := newFreelanceClient(`{"found":0,"hits":[]}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		PerPage: 10000,
	})
	require.NoError(t, err)
	assert.Equal(t, MaxPerPage, stub.gotParams.PerPage)
}

func TestService_Query_FilterByForwarded(t *testing.T) {
	stub := newFreelanceClient(`{"found":0,"hits":[]}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Filters: FilterInput{
			Skills:    []string{"react"},
			Languages: []string{"fr"},
		},
	})
	require.NoError(t, err)
	// Order is fixed by the filter builder (languages before skills).
	assert.Equal(t, "languages_professional:[fr] && skills:[react]", stub.gotParams.FilterBy)
}

func TestService_Query_ParsesHits(t *testing.T) {
	payload := `{
		"found": 2,
		"out_of": 100,
		"page": 1,
		"per_page": 20,
		"search_time_ms": 5,
		"hits": [
			{
				"document": {
					"id": "11111111-1111-1111-1111-111111111111",
					"persona": "freelance",
					"is_published": true,
					"display_name": "Alice",
					"title": "Go Developer",
					"languages_professional": ["fr","en"],
					"skills": ["go","react"],
					"work_mode": ["remote"]
				},
				"highlights": [
					{"field":"display_name","snippet":"<mark>Alice</mark>"}
				]
			},
			{
				"document": {
					"id": "22222222-2222-2222-2222-222222222222",
					"persona": "freelance",
					"is_published": true,
					"display_name": "Bob",
					"title": "Designer",
					"languages_professional": ["en"],
					"skills": ["figma"],
					"work_mode": ["hybrid"]
				},
				"highlights": []
			}
		],
		"facet_counts": [
			{
				"field_name": "skills",
				"counts": [
					{"value":"go","count":12},
					{"value":"react","count":8}
				]
			}
		]
	}`
	stub := newFreelanceClient(payload)
	svc := NewService(ServiceDeps{Freelance: stub})

	got, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.NoError(t, err)
	assert.Equal(t, 2, got.Found)
	assert.Equal(t, 100, got.OutOf)
	assert.Equal(t, 5, got.SearchTimeMs)
	require.Len(t, got.Documents, 2)
	assert.Equal(t, "Alice", got.Documents[0].DisplayName)
	assert.Equal(t, "Bob", got.Documents[1].DisplayName)
	require.Len(t, got.Highlights, 2)
	assert.Equal(t, "<mark>Alice</mark>", got.Highlights[0]["display_name"])
	assert.Empty(t, got.Highlights[1])
	assert.Equal(t, 12, got.FacetCounts["skills"]["go"])
	assert.Equal(t, 8, got.FacetCounts["skills"]["react"])
}

func TestService_Query_StripsEmbedding(t *testing.T) {
	payload := `{
		"found":1,"hits":[
			{"document":{"id":"33333333-3333-3333-3333-333333333333","persona":"freelance","is_published":true,"display_name":"Alice","embedding":[0.1,0.2,0.3]},"highlights":[]}
		]
	}`
	stub := newFreelanceClient(payload)
	svc := NewService(ServiceDeps{Freelance: stub})

	got, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.NoError(t, err)
	require.Len(t, got.Documents, 1)
	assert.Nil(t, got.Documents[0].Embedding, "embedding must be stripped from the typed response")
}

func TestService_Query_DidYouMean(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "top-level corrected_query",
			payload: `{"found":0,"hits":[],"corrected_query":"react"}`,
			want:    "react",
		},
		{
			name:    "request_params delta",
			payload: `{"found":0,"hits":[],"request_params":{"first_q":"reakt","q":"react"}}`,
			want:    "react",
		},
		{
			name:    "no correction",
			payload: `{"found":0,"hits":[],"request_params":{"first_q":"react","q":"react"}}`,
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newFreelanceClient(tt.payload)
			svc := NewService(ServiceDeps{Freelance: stub})
			got, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.CorrectedQuery)
		})
	}
}

func TestService_Query_PropagatesClientError(t *testing.T) {
	stub := &fakeClient{persona: search.PersonaFreelance, respErr: errors.New("typesense unavailable")}
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "typesense unavailable")
}

func TestService_Query_QueryByIncludesExpectedFields(t *testing.T) {
	stub := newFreelanceClient(`{"found":0,"hits":[]}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.NoError(t, err)

	for _, field := range []string{"display_name", "title", "skills_text"} {
		assert.True(t, strings.Contains(stub.gotParams.QueryBy, field),
			"query_by must include %q", field)
	}
}

func TestService_Query_FacetByIncludesSidebarFields(t *testing.T) {
	stub := newFreelanceClient(`{"found":0,"hits":[]}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.NoError(t, err)

	wanted := []string{"skills", "languages_professional", "expertise_domains", "work_mode", "city"}
	for _, field := range wanted {
		assert.True(t, strings.Contains(stub.gotParams.FacetBy, field),
			"facet_by must include %q", field)
	}
}
