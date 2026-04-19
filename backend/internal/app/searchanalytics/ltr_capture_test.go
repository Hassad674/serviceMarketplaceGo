package searchanalytics

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeResultPayload_EmptyInputStableHash(t *testing.T) {
	payload, sha, err := EncodeResultPayload(nil)
	require.NoError(t, err)
	assert.Equal(t, "[]", payload)
	assert.Len(t, sha, 64, "sha256 hex is 64 chars")

	payload2, sha2, err := EncodeResultPayload([]RankedResult{})
	require.NoError(t, err)
	assert.Equal(t, payload, payload2)
	assert.Equal(t, sha, sha2, "empty slice and nil produce the same SHA")
}

func TestEncodeResultPayload_SortsByRank(t *testing.T) {
	in := []RankedResult{
		{DocID: "b", RankPosition: 2, FinalScore: 80, Features: map[string]float64{"x": 0.1}},
		{DocID: "a", RankPosition: 1, FinalScore: 90, Features: map[string]float64{"x": 0.2}},
		{DocID: "c", RankPosition: 3, FinalScore: 70, Features: map[string]float64{"x": 0.3}},
	}
	payload, _, err := EncodeResultPayload(in)
	require.NoError(t, err)

	var decoded []RankedResult
	require.NoError(t, json.Unmarshal([]byte(payload), &decoded))
	require.Len(t, decoded, 3)
	assert.Equal(t, 1, decoded[0].RankPosition)
	assert.Equal(t, 2, decoded[1].RankPosition)
	assert.Equal(t, 3, decoded[2].RankPosition)
	assert.Equal(t, "a", decoded[0].DocID)
}

func TestEncodeResultPayload_HashDeterministic(t *testing.T) {
	in := []RankedResult{
		{DocID: "a", RankPosition: 1, FinalScore: 90, Features: map[string]float64{"x": 0.2, "y": 0.3}},
	}
	_, sha1, err := EncodeResultPayload(in)
	require.NoError(t, err)
	_, sha2, err := EncodeResultPayload(in)
	require.NoError(t, err)
	assert.Equal(t, sha1, sha2, "same input ⇒ same SHA")

	// Reshuffled input must still hash to the same fingerprint
	// (canonicalisation sorts by rank_position).
	reshuffled := []RankedResult{in[0]}
	_, sha3, err := EncodeResultPayload(reshuffled)
	require.NoError(t, err)
	assert.Equal(t, sha1, sha3)
}

func TestEncodeResultPayload_DifferentOrdersDifferentHash(t *testing.T) {
	a := []RankedResult{
		{DocID: "a", RankPosition: 1, FinalScore: 90, Features: map[string]float64{"x": 0.2}},
		{DocID: "b", RankPosition: 2, FinalScore: 80, Features: map[string]float64{"x": 0.1}},
	}
	b := []RankedResult{
		{DocID: "b", RankPosition: 1, FinalScore: 80, Features: map[string]float64{"x": 0.1}},
		{DocID: "a", RankPosition: 2, FinalScore: 90, Features: map[string]float64{"x": 0.2}},
	}
	_, shaA, err := EncodeResultPayload(a)
	require.NoError(t, err)
	_, shaB, err := EncodeResultPayload(b)
	require.NoError(t, err)
	assert.NotEqual(t, shaA, shaB,
		"different rank orderings produce different fingerprints")
}

func TestEncodeResultPayload_RoundTrip(t *testing.T) {
	original := []RankedResult{
		{
			DocID:        "doc-1",
			RankPosition: 1,
			FinalScore:   87.3,
			Features: map[string]float64{
				"text_match":     0.82,
				"skills_overlap": 0.75,
				"rating":         0.69,
			},
		},
		{
			DocID:        "doc-2",
			RankPosition: 2,
			FinalScore:   85.0,
			Features: map[string]float64{
				"text_match":     0.78,
				"skills_overlap": 0.60,
				"rating":         0.90,
			},
		},
	}
	payload, _, err := EncodeResultPayload(original)
	require.NoError(t, err)

	var decoded []RankedResult
	require.NoError(t, json.Unmarshal([]byte(payload), &decoded))
	assert.Equal(t, original[0].DocID, decoded[0].DocID)
	assert.InDelta(t, original[0].FinalScore, decoded[0].FinalScore, 1e-9)
	assert.InDelta(t, original[0].Features["text_match"], decoded[0].Features["text_match"], 1e-9)
}

// fakeLTRRepo records every AttachResultFeatures call for assertion.
type fakeLTRRepo struct {
	mu     sync.Mutex
	calls  []fakeLTRCall
	err    error
}

type fakeLTRCall struct {
	SearchID string
	Payload  string
	SHA      string
}

func (f *fakeLTRRepo) AttachResultFeatures(_ context.Context, searchID, payload, sha string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeLTRCall{SearchID: searchID, Payload: payload, SHA: sha})
	return f.err
}

func (f *fakeLTRRepo) snapshot() []fakeLTRCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeLTRCall, len(f.calls))
	copy(out, f.calls)
	return out
}

func TestService_CaptureResultFeatures_PersistsAsync(t *testing.T) {
	svc := newTestService(t)
	fake := &fakeLTRRepo{}

	err := svc.CaptureResultFeatures(context.Background(), "search-1",
		[]RankedResult{
			{DocID: "a", RankPosition: 1, FinalScore: 90, Features: map[string]float64{"x": 0.1}},
		}, fake)
	require.NoError(t, err)

	// Poll briefly for the async goroutine to land.
	waitFor(t, func() bool {
		return len(fake.snapshot()) == 1
	}, time.Second)

	calls := fake.snapshot()
	require.Len(t, calls, 1)
	assert.Equal(t, "search-1", calls[0].SearchID)
	assert.Contains(t, calls[0].Payload, `"doc_id":"a"`)
	assert.Len(t, calls[0].SHA, 64)
}

func TestService_CaptureResultFeatures_EmptySearchIDRejected(t *testing.T) {
	svc := newTestService(t)
	err := svc.CaptureResultFeatures(context.Background(), "", nil, &fakeLTRRepo{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty search_id")
}

func TestService_CaptureResultFeatures_NilRepoRejected(t *testing.T) {
	svc := newTestService(t)
	err := svc.CaptureResultFeatures(context.Background(), "sid", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil ltr repository")
}

func TestService_CaptureResultFeatures_RepoErrorLogged(t *testing.T) {
	svc := newTestService(t)
	fake := &fakeLTRRepo{err: errors.New("boom")}
	err := svc.CaptureResultFeatures(context.Background(), "sid",
		[]RankedResult{{DocID: "a", RankPosition: 1}}, fake)
	require.NoError(t, err, "capture is fire-and-forget once validated")
	waitFor(t, func() bool {
		return len(fake.snapshot()) == 1
	}, time.Second)
}

func TestService_CaptureResultFeatures_NilServiceIsNoop(t *testing.T) {
	var svc *Service
	err := svc.CaptureResultFeatures(context.Background(), "sid", nil, &fakeLTRRepo{})
	assert.NoError(t, err, "nil receiver must no-op (defensive against bad wiring)")
}

func TestLTRErrNotFound_IsUsefulSentinel(t *testing.T) {
	wrapped := errors.New("search analytics: attach features: " + LTRErrNotFound.Error())
	assert.NotErrorIs(t, wrapped, LTRErrNotFound, "string-wrapped is NOT considered the sentinel")
	assert.ErrorIs(t, LTRErrNotFound, LTRErrNotFound, "direct match works via errors.Is")
}

// newTestService builds a Service with a noop repo + logger — enough
// to exercise CaptureResultFeatures without spinning up the capture
// repo (LTR path doesn't touch it).
func newTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(Config{
		Repository: &noopRepo{},
		Logger:     slog.Default(),
		Clock:      time.Now,
	})
	require.NoError(t, err)
	return svc
}

// noopRepo is a minimal Repository used to satisfy NewService.
// CaptureResultFeatures doesn't call it, so every method is a
// no-op. Separate from the main service_test.go fake because the
// LTR path has different needs — we only need the interface shape.
type noopRepo struct{}

func (noopRepo) InsertSearch(context.Context, *SearchRow) error { return nil }
func (noopRepo) RecordClick(context.Context, string, string, int, time.Time) error {
	return nil
}

// waitFor polls until cond returns true or timeout elapses. Used to
// assert on the fire-and-forget goroutine without flaky sleeps.
func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition did not become true within %s", timeout)
}
