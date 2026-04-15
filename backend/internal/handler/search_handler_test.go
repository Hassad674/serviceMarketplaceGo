package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsearch "marketplace-backend/internal/app/search"
	"marketplace-backend/internal/search"
)

// fakeQueryClient is a deterministic stub for the persona-scoped
// client interface the app service depends on. We reuse the same
// shape from the app layer's tests so the wiring is uniform.
type fakeQueryClient struct {
	persona search.Persona
	payload string
	err     error
	gotParams search.SearchParams
}

func (f *fakeQueryClient) Persona() search.Persona { return f.persona }

func (f *fakeQueryClient) Query(_ context.Context, params search.SearchParams) (json.RawMessage, error) {
	f.gotParams = params
	if f.err != nil {
		return nil, f.err
	}
	return json.RawMessage(f.payload), nil
}

func newTestSearchHandler(t *testing.T, freelance *fakeQueryClient) *SearchHandler {
	t.Helper()
	svc := appsearch.NewService(appsearch.ServiceDeps{Freelance: freelance})
	client, err := search.NewClient("http://localhost:8108", "test-master-key")
	require.NoError(t, err)
	return NewSearchHandler(SearchHandlerDeps{
		Service:       svc,
		Client:        client,
		TypesenseHost: "http://localhost:8108",
		APIKey:        "test-master-key",
	})
}

func TestSearchHandler_ScopedKey_HappyPath(t *testing.T) {
	stub := &fakeQueryClient{persona: search.PersonaFreelance, payload: `{}`}
	h := newTestSearchHandler(t, stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/key?persona=freelance", nil)
	rec := httptest.NewRecorder()
	h.ScopedKey(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body scopedKeyResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Key)
	assert.Equal(t, "http://localhost:8108", body.Host)
	assert.Equal(t, "freelance", body.Persona)
	assert.Greater(t, body.ExpiresAt, int64(0))

	// Sanity-check the embedded params: decode the key and look
	// for the persona filter inside.
	raw, err := base64.StdEncoding.DecodeString(body.Key)
	require.NoError(t, err)
	// Layout: 64-char digest + 4-char prefix + JSON
	require.Greater(t, len(raw), 68)
	embedded := string(raw[68:])
	assert.Contains(t, embedded, `"filter_by":"persona:freelance && is_published:true"`)
}

func TestSearchHandler_ScopedKey_UnknownPersona(t *testing.T) {
	stub := &fakeQueryClient{persona: search.PersonaFreelance, payload: `{}`}
	h := newTestSearchHandler(t, stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/key?persona=garbage", nil)
	rec := httptest.NewRecorder()
	h.ScopedKey(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_persona")
}

func TestSearchHandler_ScopedKey_PersonaUnavailable(t *testing.T) {
	// Service has no clients wired.
	svc := appsearch.NewService(appsearch.ServiceDeps{})
	client, err := search.NewClient("http://localhost:8108", "test-master-key")
	require.NoError(t, err)
	h := NewSearchHandler(SearchHandlerDeps{
		Service:       svc,
		Client:        client,
		TypesenseHost: "http://localhost:8108",
		APIKey:        "test-master-key",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/key?persona=freelance", nil)
	rec := httptest.NewRecorder()
	h.ScopedKey(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestSearchHandler_Search_HappyPath(t *testing.T) {
	payload := `{
		"found": 1,
		"out_of": 50,
		"page": 1,
		"per_page": 20,
		"search_time_ms": 4,
		"hits": [
			{
				"document": {
					"id": "11111111-1111-1111-1111-111111111111",
					"persona": "freelance",
					"is_published": true,
					"display_name": "Alice"
				},
				"highlights": []
			}
		],
		"facet_counts": []
	}`
	stub := &fakeQueryClient{persona: search.PersonaFreelance, payload: payload}
	h := newTestSearchHandler(t, stub)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/search?persona=freelance&q=alice&languages=fr,en&skills=react&pricing_min=50000&verified=true",
		nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Response body decodes into the typed result.
	var got appsearch.QueryResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, 1, got.Found)
	require.Len(t, got.Documents, 1)
	assert.Equal(t, "Alice", got.Documents[0].DisplayName)

	// The handler forwarded q and built the expected filter_by.
	assert.Equal(t, "alice", stub.gotParams.Q)
	assert.Contains(t, stub.gotParams.FilterBy, "languages_professional:[fr,en]")
	assert.Contains(t, stub.gotParams.FilterBy, "skills:[react]")
	assert.Contains(t, stub.gotParams.FilterBy, "pricing_min_amount:>=50000")
	assert.Contains(t, stub.gotParams.FilterBy, "is_verified:=true")
}

func TestSearchHandler_Search_InvalidPersona(t *testing.T) {
	stub := &fakeQueryClient{persona: search.PersonaFreelance, payload: `{}`}
	h := newTestSearchHandler(t, stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?persona=", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearchHandler_Search_PersonaNotConfigured(t *testing.T) {
	svc := appsearch.NewService(appsearch.ServiceDeps{})
	client, _ := search.NewClient("http://localhost:8108", "test-master-key")
	h := NewSearchHandler(SearchHandlerDeps{
		Service: svc, Client: client,
		TypesenseHost: "http://localhost:8108", APIKey: "test-master-key",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?persona=freelance", nil)
	rec := httptest.NewRecorder()
	h.Search(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestParseStringList(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{",,,", nil},
		{"fr", []string{"fr"}},
		{"fr,en", []string{"fr", "en"}},
		{" fr , en ", []string{"fr", "en"}},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, parseStringList(tt.in))
		})
	}
}

func TestParseBoolPointer(t *testing.T) {
	tests := []struct {
		in    string
		want  *bool
	}{
		{"", nil},
		{"true", searchBoolPtr(true)},
		{"false", searchBoolPtr(false)},
		{"1", searchBoolPtr(true)},
		{"0", searchBoolPtr(false)},
		{"banana", nil},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := parseBoolPointer(tt.in)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

func searchBoolPtr(v bool) *bool { return &v }
