package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestModerationService wires a TextModerationService that talks to
// an httptest.Server instead of OpenAI. Mirrors the pattern used by
// adapter/nominatim/geocoder_test.go — we exercise the real HTTP
// pipeline (headers, body encoding, JSON decoding) without a network
// call.
func newTestModerationService(t *testing.T, handler http.Handler) (*TextModerationService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	svc := &TextModerationService{
		client: NewClient("test-api-key", srv.URL),
		model:  defaultModel,
	}
	return svc, srv
}

func TestTextModerationService_AnalyzeText_HappyPath(t *testing.T) {
	var capturedAuth, capturedCT, capturedModel, capturedInput string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedCT = r.Header.Get("Content-Type")
		assert.Equal(t, "/v1/moderations", r.URL.Path)

		var body moderationRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		capturedModel = body.Model
		capturedInput = body.Input

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": [{
				"category_scores": {
					"harassment": 0.82,
					"hate": 0.12,
					"violence": 0.04
				}
			}]
		}`))
	})

	svc, _ := newTestModerationService(t, handler)

	result, err := svc.AnalyzeText(context.Background(), "you fucking bastard")
	require.NoError(t, err)

	assert.Equal(t, "Bearer test-api-key", capturedAuth)
	assert.Equal(t, "application/json", capturedCT)
	assert.Equal(t, defaultModel, capturedModel)
	assert.Equal(t, "you fucking bastard", capturedInput)

	assert.InDelta(t, 0.82, result.MaxScore, 0.001)
	assert.False(t, result.IsSafe, "score above 0.5 should not be safe")
	assert.Len(t, result.Labels, 3)
}

func TestTextModerationService_AnalyzeText_EmptyTextShortCircuits(t *testing.T) {
	// If the handler is called, the test fails — empty input should
	// never hit the network.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP call on empty input")
	})
	svc, _ := newTestModerationService(t, handler)

	result, err := svc.AnalyzeText(context.Background(), "")
	require.NoError(t, err)
	assert.True(t, result.IsSafe)
	assert.Zero(t, result.MaxScore)
	assert.Empty(t, result.Labels)
}

func TestTextModerationService_AnalyzeText_SafeBelowThreshold(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"results": [{"category_scores": {"harassment": 0.20, "hate": 0.10}}]
		}`))
	})
	svc, _ := newTestModerationService(t, handler)

	result, err := svc.AnalyzeText(context.Background(), "bonjour ça va ?")
	require.NoError(t, err)
	assert.True(t, result.IsSafe, "score below 0.5 should be safe")
	assert.InDelta(t, 0.20, result.MaxScore, 0.001)
}

func TestTextModerationService_AnalyzeText_TruncatesLongInput(t *testing.T) {
	var capturedLen int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body moderationRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		capturedLen = len(body.Input)
		_, _ = w.Write([]byte(`{"results":[{"category_scores":{}}]}`))
	})
	svc, _ := newTestModerationService(t, handler)

	huge := strings.Repeat("a", maxInputChars+5_000)
	_, err := svc.AnalyzeText(context.Background(), huge)
	require.NoError(t, err)
	assert.Equal(t, maxInputChars, capturedLen, "input must be truncated to maxInputChars")
}

func TestTextModerationService_AnalyzeText_EmptyResultsArray(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results": []}`))
	})
	svc, _ := newTestModerationService(t, handler)

	result, err := svc.AnalyzeText(context.Background(), "hello")
	require.NoError(t, err)
	assert.True(t, result.IsSafe)
	assert.Zero(t, result.MaxScore)
}

func TestTextModerationService_AnalyzeText_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	svc, _ := newTestModerationService(t, handler)

	_, err := svc.AnalyzeText(context.Background(), "hello")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOpenAIHTTP)
	assert.Contains(t, err.Error(), "500")
}

func TestTextModerationService_AnalyzeText_RateLimitError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
	})
	svc, _ := newTestModerationService(t, handler)

	_, err := svc.AnalyzeText(context.Background(), "hello")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOpenAIHTTP)
	assert.Contains(t, err.Error(), "429")
}

func TestTextModerationService_AnalyzeText_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	})
	svc, _ := newTestModerationService(t, handler)

	_, err := svc.AnalyzeText(context.Background(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestTextModerationService_AnalyzeText_Timeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"results":[{"category_scores":{}}]}`))
	})
	svc, _ := newTestModerationService(t, handler)
	// Override the underlying http client timeout so the test runs fast.
	svc.client.http = &http.Client{Timeout: 50 * time.Millisecond}

	_, err := svc.AnalyzeText(context.Background(), "hello")
	require.Error(t, err)
}

func TestTextModerationService_MapsAllCategories(t *testing.T) {
	// The 13 omni-moderation categories must all be exposed in Labels
	// so domain/moderation.DecideStatus can match them by name against
	// its zero-tolerance matrix.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"results": [{
				"category_scores": {
					"sexual": 0.10,
					"sexual/minors": 0.01,
					"harassment": 0.30,
					"harassment/threatening": 0.85,
					"hate": 0.05,
					"hate/threatening": 0.02,
					"illicit": 0.04,
					"illicit/violent": 0.03,
					"self-harm": 0.01,
					"self-harm/intent": 0.01,
					"self-harm/instructions": 0.01,
					"violence": 0.25,
					"violence/graphic": 0.10
				}
			}]
		}`))
	})
	svc, _ := newTestModerationService(t, handler)

	result, err := svc.AnalyzeText(context.Background(), "je vais te tuer demain")
	require.NoError(t, err)

	assert.Len(t, result.Labels, 13, "all 13 categories should be mapped")
	assert.InDelta(t, 0.85, result.MaxScore, 0.001, "MaxScore must be the highest category score")

	// Spot-check that known labels are present with correct scores.
	byName := make(map[string]float64, len(result.Labels))
	for _, l := range result.Labels {
		byName[l.Name] = l.Score
	}
	assert.InDelta(t, 0.85, byName["harassment/threatening"], 0.001)
	assert.InDelta(t, 0.01, byName["sexual/minors"], 0.001)
}
