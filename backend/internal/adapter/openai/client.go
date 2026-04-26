// Package openai wraps calls to the OpenAI REST API. Two endpoints are
// used by this backend today: /v1/moderations (free content safety
// classifier — see text_moderation.go) and /v1/embeddings (used from
// cmd/reindex for Typesense). The shared Client below centralises the
// HTTP plumbing (base URL, auth header, JSON encoding, timeout) so each
// feature adapter stays focused on request/response mapping.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// defaultBaseURL is the production OpenAI API root. Tests override it
// via NewClient's optional baseURL argument (see httptest wiring in the
// test files).
const defaultBaseURL = "https://api.openai.com"

// defaultTimeout bounds a single external call. Consistent with the
// "external HTTP ≤ 10s" rule from backend/CLAUDE.md.
const defaultTimeout = 10 * time.Second

// Client is a thin wrapper around *http.Client scoped to one OpenAI
// account (one API key). Safe for concurrent use.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient builds a Client. Pass an empty baseURL to hit the real
// OpenAI API; tests pass an httptest.Server URL.
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// ErrOpenAIHTTP is the sentinel wrapped by every non-2xx response.
// Adapters match against it with errors.Is so they can surface clean
// domain errors without caring about HTTP details.
var ErrOpenAIHTTP = errors.New("openai http error")

// postJSON sends body as JSON to path and decodes the JSON response
// into out. Callers pass a context whose deadline should not exceed
// defaultTimeout — the client's own timeout acts as a hard ceiling.
func (c *Client) postJSON(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("openai: encode body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("openai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("openai: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read a capped body so huge error responses don't explode memory.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%w: status=%d body=%q", ErrOpenAIHTTP, resp.StatusCode, string(snippet))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("openai: decode response: %w", err)
	}
	return nil
}
