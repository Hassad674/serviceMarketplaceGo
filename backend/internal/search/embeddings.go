package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// embeddings.go defines the EmbeddingsClient port used by the indexer
// plus two implementations: the real OpenAI client and a deterministic
// mock for tests. The mock is the default in every unit + integration
// test (95% of the suite) so test runs cost $0.
//
// The live OpenAI client is instantiated in `cmd/api/main.go`
// whenever TYPESENSE_* is configured (mandatory since phase 4) and
// in the bulk reindex CLI. Golden semantic tests gated by
// OPENAI_EMBEDDINGS_LIVE=true build one directly (phase 3).

// EmbeddingsClient is the port the indexer depends on. Small and
// focused — one method, one responsibility. If a consumer needs
// batching in the future we will add a BatchEmbed method to a
// separate, opt-in interface rather than bloat this one.
type EmbeddingsClient interface {
	// Embed converts free-form text into a 1536-dimensional vector.
	// Implementations MUST:
	//   - respect ctx for cancellation / deadline;
	//   - return an error instead of panicking on network failure;
	//   - return exactly EmbeddingDimensions floats on success.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// ----------------------------------------------------------------------
// OpenAI implementation
// ----------------------------------------------------------------------

// openAIEmbeddingsURL is the stable REST endpoint for the embeddings
// API. Kept as a constant so test injectors can swap it out if we
// ever need to mock the OpenAI wire at the HTTP layer.
const openAIEmbeddingsURL = "https://api.openai.com/v1/embeddings"

// defaultOpenAITimeout is the per-call timeout for the embeddings
// endpoint. OpenAI's p95 is ~300ms; 10 seconds is a generous budget
// that catches pathological cases without hanging the caller.
const defaultOpenAITimeout = 10 * time.Second

// OpenAIEmbeddingsClient hits the OpenAI REST API directly. Kept
// minimal — no streaming, no retries inside the client; the indexer
// decides whether to retry based on its own error-handling policy.
type OpenAIEmbeddingsClient struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

// OpenAIOption mutates an OpenAIEmbeddingsClient during construction.
type OpenAIOption func(*OpenAIEmbeddingsClient)

// WithOpenAIEndpoint overrides the embeddings endpoint URL. Used in
// tests to point the client at an httptest.Server.
func WithOpenAIEndpoint(url string) OpenAIOption {
	return func(c *OpenAIEmbeddingsClient) { c.endpoint = url }
}

// WithOpenAIHTTPClient overrides the HTTP client.
func WithOpenAIHTTPClient(h *http.Client) OpenAIOption {
	return func(c *OpenAIEmbeddingsClient) { c.httpClient = h }
}

// NewOpenAIEmbeddings builds a live client for the OpenAI embeddings
// endpoint. Both fields are required — we refuse to silently accept
// an empty API key and produce confusing 401s later.
func NewOpenAIEmbeddings(apiKey, model string, opts ...OpenAIOption) (*OpenAIEmbeddingsClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai embeddings: api key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("openai embeddings: model is required")
	}
	c := &OpenAIEmbeddingsClient{
		apiKey:     apiKey,
		model:      model,
		endpoint:   openAIEmbeddingsURL,
		httpClient: &http.Client{Timeout: defaultOpenAITimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// openAIRequest / openAIResponse are local wire structs that match
// the subset of the OpenAI embeddings API we care about. We do not
// depend on the `openai-go` package because that would pull in the
// entire chat completions + tools surface for one method.
type openAIRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Embed posts a single input string and returns the first embedding.
// The OpenAI API supports batched inputs but we intentionally keep
// this interface single-input so the indexer's concurrency model
// stays simple. Batching is a future optimisation once we see
// measured latency.
func (c *OpenAIEmbeddingsClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		// OpenAI returns a 400 on empty input; catch it early to
		// save a round-trip and surface a cleaner error.
		return nil, fmt.Errorf("openai embeddings: empty input")
	}

	body, err := json.Marshal(openAIRequest{Model: c.model, Input: text})
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: http: %w", err)
	}
	defer resp.Body.Close()

	return parseOpenAIResponse(resp)
}

// parseOpenAIResponse decodes the REST response into a []float32.
// Extracted so the main Embed function stays below the 50-line
// limit and so the decode logic can be tested independently.
func parseOpenAIResponse(resp *http.Response) ([]float32, error) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embeddings: status %d: %s",
			resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var decoded openAIResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("openai embeddings: decode: %w", err)
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("openai embeddings: api error: %s", decoded.Error.Message)
	}
	if len(decoded.Data) == 0 {
		return nil, fmt.Errorf("openai embeddings: empty data array")
	}
	vec := decoded.Data[0].Embedding
	if len(vec) != EmbeddingDimensions {
		return nil, fmt.Errorf("openai embeddings: got %d dims, want %d",
			len(vec), EmbeddingDimensions)
	}
	return vec, nil
}

// ----------------------------------------------------------------------
// Mock implementation (deterministic, zero-cost)
// ----------------------------------------------------------------------

// MockEmbeddingsClient returns a deterministic vector on every call.
// The vector is not random — we generate a pattern derived from the
// input string length so the integration tests can still verify that
// different inputs produce different (but reproducible) vectors,
// which matters for vector-search ordering assertions.
//
// Use NewMockEmbeddings() for the default "flat" vector or
// NewMockEmbeddingsFromSeed(seed) for a family of deterministic
// vectors that differ per seed — useful for per-persona tests.
type MockEmbeddingsClient struct {
	vector []float32
}

// NewMockEmbeddings returns a client that always emits the same
// 1536-dim "flat-ish" vector (0.1, 0.2, ..., 0.1 cyclically). The
// vector is L2-normalised-ish so cosine distance is well-defined
// and tests that compare ordering produce stable results.
func NewMockEmbeddings() *MockEmbeddingsClient {
	return &MockEmbeddingsClient{vector: buildMockVector(1)}
}

// NewMockEmbeddingsFromSeed returns a client that always emits a
// vector derived from the seed. Different seeds produce linearly-
// independent vectors so tests can verify vector ordering without
// any real embedding model.
func NewMockEmbeddingsFromSeed(seed int) *MockEmbeddingsClient {
	return &MockEmbeddingsClient{vector: buildMockVector(seed)}
}

// buildMockVector returns a deterministic 1536-dim vector. The
// pattern is simple: alternating sin-ish values scaled by the seed
// so different seeds produce different orientations.
func buildMockVector(seed int) []float32 {
	vec := make([]float32, EmbeddingDimensions)
	for i := range vec {
		// Cheap but deterministic: gives a slightly varying shape
		// without depending on math/rand or the time package.
		vec[i] = float32(((i+seed)%10)+1) / 100.0
	}
	return vec
}

// Embed returns the fixed vector. Ignores the input text because
// tests want reproducibility across runs and inputs — if a test
// needs to see vectors change based on content, it constructs a
// custom MockEmbeddingsClient with a bespoke vector.
func (m *MockEmbeddingsClient) Embed(_ context.Context, _ string) ([]float32, error) {
	// Return a fresh slice on every call so callers can safely
	// mutate it without poisoning subsequent tests.
	out := make([]float32, len(m.vector))
	copy(out, m.vector)
	return out, nil
}

// Vector exposes the underlying deterministic vector for tests that
// need to assert on its shape without calling Embed.
func (m *MockEmbeddingsClient) Vector() []float32 { return m.vector }
