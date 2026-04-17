package search

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// service_phase3_test.go pins every phase 3 behaviour added to the
// service: hybrid query_by with embedding, vector_query injection,
// cursor-based pagination metadata, analytics capture, and defaults
// for empty queries.

type stubEmbedder struct {
	vec  []float32
	err  error
	call int
}

func (s *stubEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	s.call++
	if s.err != nil {
		return nil, s.err
	}
	return s.vec, nil
}

type stubAnalytics struct {
	mu  sync.Mutex
	got []AnalyticsEvent
}

func (s *stubAnalytics) CaptureSearch(_ context.Context, evt AnalyticsEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = append(s.got, evt)
}

func (s *stubAnalytics) events() []AnalyticsEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AnalyticsEvent, len(s.got))
	copy(out, s.got)
	return out
}

func newFreelanceStub(payload string) *fakeClient {
	return &fakeClient{persona: search.PersonaFreelance, respPayload: payload}
}

func TestService_Query_HybridPassesVectorQueryNotQueryBy(t *testing.T) {
	// On Typesense 28.0, the `embedding` field is a MANUAL vector
	// (not auto-embedding). Including it in `query_by` triggers a
	// 400 with the cryptic "not an auto-embedding field" error.
	// Hybrid blending is controlled exclusively by vector_query.
	stub := newFreelanceStub(`{"found":0,"hits":[]}`)
	embedder := &stubEmbedder{vec: []float32{0.1, 0.2, 0.3}}
	svc := NewService(ServiceDeps{Freelance: stub, Embedder: embedder})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react paris",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, embedder.call)
	assert.NotContains(t, stub.gotParams.QueryBy, "embedding",
		"embedding must NOT be in query_by (Typesense 28.0 manual-vector constraint)")
	assert.True(t, strings.HasPrefix(stub.gotParams.VectorQuery, "embedding:(["))
	assert.Contains(t, stub.gotParams.VectorQuery, "k:20)")
	// num_typos stays aligned with the text-only query_by fields.
	assert.Equal(t, strings.Count(stub.gotParams.QueryBy, ","), strings.Count(stub.gotParams.NumTypos, ","))
}

func TestService_Query_MatchAllSkipsEmbedding(t *testing.T) {
	stub := newFreelanceStub(`{"found":0,"hits":[]}`)
	embedder := &stubEmbedder{vec: []float32{0.1}}
	svc := NewService(ServiceDeps{Freelance: stub, Embedder: embedder})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "",
	})
	require.NoError(t, err)
	// Match-all: no embedding call, no vector_query param. query_by
	// is text-only in every path (see the hybrid test above), so we
	// only assert the vector + embedder side.
	assert.Equal(t, 0, embedder.call)
	assert.Empty(t, stub.gotParams.VectorQuery)
}

func TestService_Query_EmbedderFailureFallsBackToBM25(t *testing.T) {
	stub := newFreelanceStub(`{"found":0,"hits":[]}`)
	embedder := &stubEmbedder{err: errors.New("openai down")}
	svc := NewService(ServiceDeps{Freelance: stub, Embedder: embedder})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err, "embedder failure must not break the query")
	assert.Empty(t, stub.gotParams.VectorQuery, "no vector_query when embedding failed")
}

func TestService_Query_CursorSetsPage(t *testing.T) {
	stub := newFreelanceStub(`{"found":100,"hits":[],"page":5}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Cursor:  EncodeCursor(Cursor{Page: 5}),
	})
	require.NoError(t, err)
	assert.Equal(t, 5, stub.gotParams.Page)
}

func TestService_Query_EmitsNextCursor(t *testing.T) {
	// 100 total, 20 per page, page 2 → 40 loaded → has_more true
	stub := newFreelanceStub(`{"found":100,"hits":[],"page":2}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Page:    2,
	})
	require.NoError(t, err)
	require.True(t, res.HasMore)
	require.NotEmpty(t, res.NextCursor)

	decoded, err := DecodeCursor(res.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, 3, decoded.Page)
}

func TestService_Query_LastPageNoCursor(t *testing.T) {
	// 40 total, 20 per page, page 2 → 40 loaded == found → no more
	stub := newFreelanceStub(`{"found":40,"hits":[],"page":2}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Page:    2,
	})
	require.NoError(t, err)
	assert.False(t, res.HasMore)
	assert.Empty(t, res.NextCursor)
}

func TestService_Query_BadCursorSurfacesError(t *testing.T) {
	stub := newFreelanceStub(`{}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Cursor:  "not-valid",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCursorInvalid)
}

func TestService_Query_AnalyticsCaptured(t *testing.T) {
	stub := newFreelanceStub(`{"found":12,"hits":[],"page":1}`)
	rec := &stubAnalytics{}
	svc := NewService(ServiceDeps{Freelance: stub, Analytics: rec})

	res, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
		UserID:  "user-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.SearchID)

	// Capture is async; wait a short beat.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && len(rec.events()) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	events := rec.events()
	require.Len(t, events, 1)
	assert.Equal(t, res.SearchID, events[0].SearchID)
	assert.Equal(t, "user-1", events[0].UserID)
	assert.Equal(t, 12, events[0].ResultsCount)
	assert.Equal(t, "freelance", events[0].Persona)
}

func TestService_Query_WithoutAnalyticsDoesNotPanic(t *testing.T) {
	stub := newFreelanceStub(`{"found":0,"hits":[],"page":1}`)
	svc := NewService(ServiceDeps{Freelance: stub, Analytics: nil})

	_, err := svc.Query(context.Background(), QueryInput{Persona: search.PersonaFreelance})
	require.NoError(t, err)
}

func TestService_Query_SearchIDStableAcrossPages(t *testing.T) {
	// Two calls with the same input params (different pages) should
	// produce the same search_id because we bucket by minute.
	stub := newFreelanceStub(`{"found":50,"hits":[],"page":1}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	res1, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
		UserID:  "u1",
		Page:    1,
	})
	require.NoError(t, err)

	stub.respPayload = `{"found":50,"hits":[],"page":2}`
	res2, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
		UserID:  "u1",
		Page:    2,
	})
	require.NoError(t, err)
	assert.Equal(t, res1.SearchID, res2.SearchID)
}

func TestParseQueryResult_UsesResponsePagination(t *testing.T) {
	raw := []byte(`{"found":30,"out_of":100,"page":2,"hits":[],"search_time_ms":5}`)
	res, err := parseQueryResult(raw)
	require.NoError(t, err)
	assert.Equal(t, 2, res.Page)
	assert.Equal(t, 30, res.Found)
	assert.Equal(t, 5, res.SearchTimeMs)
}

func TestService_Query_SkipsEmbeddingWithoutClient(t *testing.T) {
	stub := newFreelanceStub(`{"found":0,"hits":[]}`)
	svc := NewService(ServiceDeps{Freelance: stub})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Query:   "react",
	})
	require.NoError(t, err)
	assert.NotContains(t, stub.gotParams.QueryBy, "embedding")
	assert.Empty(t, stub.gotParams.VectorQuery)
}

// Ensures the service wires the analytics event's FilterBy from the
// built search params, not from the (untyped) input filters. Past
// bugs have sent the raw filter struct to analytics instead.
func TestService_Query_AnalyticsFilterByFromParams(t *testing.T) {
	stub := newFreelanceStub(`{"found":0,"hits":[],"page":1}`)
	rec := &stubAnalytics{}
	svc := NewService(ServiceDeps{Freelance: stub, Analytics: rec})

	_, err := svc.Query(context.Background(), QueryInput{
		Persona: search.PersonaFreelance,
		Filters: FilterInput{RatingMin: floatPtr(4.0)},
	})
	require.NoError(t, err)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && len(rec.events()) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	events := rec.events()
	require.Len(t, events, 1)
	// The persona filter is baked into the scoped client, not the
	// service-level filter_by — we only see rating here.
	assert.Contains(t, events[0].FilterBy, "rating_average:>=4")
}

func floatPtr(f float64) *float64 { return &f }

// Ensure JSON round-trip of the QueryResult keeps the phase 3 fields.
func TestQueryResult_JSONIncludesPhase3Fields(t *testing.T) {
	res := QueryResult{
		SearchID:   "abc",
		NextCursor: "n1",
		HasMore:    true,
	}
	raw, err := json.Marshal(res)
	require.NoError(t, err)
	payload := string(raw)
	assert.Contains(t, payload, `"search_id":"abc"`)
	assert.Contains(t, payload, `"next_cursor":"n1"`)
	assert.Contains(t, payload, `"has_more":true`)
}
