package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsearch "marketplace-backend/internal/app/search"
	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/search"
)

type fakeTracker struct {
	err      error
	searchID string
	docID    string
	position int
	calls    int
}

func (f *fakeTracker) RecordClick(_ context.Context, searchID, docID string, position int) error {
	f.calls++
	f.searchID = searchID
	f.docID = docID
	f.position = position
	return f.err
}

func newTestTrackHandler(t *testing.T, tracker ClickTracker) *SearchHandler {
	t.Helper()
	stub := &fakeQueryClient{persona: search.PersonaFreelance, payload: `{}`}
	svc := appsearch.NewService(appsearch.ServiceDeps{Freelance: stub})
	client, err := search.NewClient("http://localhost:8108", "test-master-key")
	require.NoError(t, err)
	return NewSearchHandler(SearchHandlerDeps{
		Service:       svc,
		Client:        client,
		TypesenseHost: "http://localhost:8108",
		APIKey:        "test-master-key",
		ClickTracker:  tracker,
	})
}

func TestTrack_HappyPath(t *testing.T) {
	tracker := &fakeTracker{}
	h := newTestTrackHandler(t, tracker)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/track?search_id=s1&doc_id=d1&position=2", nil)
	rec := httptest.NewRecorder()
	h.Track(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 1, tracker.calls)
	assert.Equal(t, "s1", tracker.searchID)
	assert.Equal(t, "d1", tracker.docID)
	assert.Equal(t, 2, tracker.position)
}

func TestTrack_MissingTracker(t *testing.T) {
	h := newTestTrackHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/track?search_id=s1&doc_id=d1&position=2", nil)
	rec := httptest.NewRecorder()
	h.Track(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestTrack_InvalidParams(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"missing search_id", "/api/v1/search/track?doc_id=d1&position=0"},
		{"missing doc_id", "/api/v1/search/track?search_id=s1&position=0"},
		{"missing position", "/api/v1/search/track?search_id=s1&doc_id=d1"},
		{"negative position", "/api/v1/search/track?search_id=s1&doc_id=d1&position=-5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tracker := &fakeTracker{}
			h := newTestTrackHandler(t, tracker)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			h.Track(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, 0, tracker.calls)
		})
	}
}

func TestTrack_SearchNotFound(t *testing.T) {
	tracker := &fakeTracker{err: searchanalytics.ErrNotFound}
	h := newTestTrackHandler(t, tracker)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/track?search_id=gone&doc_id=d1&position=0", nil)
	rec := httptest.NewRecorder()
	h.Track(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
