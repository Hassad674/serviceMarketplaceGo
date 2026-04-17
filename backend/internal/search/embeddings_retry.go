package search

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// embeddings_retry.go implements a transparent retry wrapper around any
// EmbeddingsClient. Phase 3 makes live embedding generation mandatory
// when OPENAI_API_KEY is set, so transient failures (network blips,
// rate limits, 5xx) must not bubble up as fatal indexing errors — we
// retry with exponential backoff to give the API time to recover.
//
// The wrapper is intentionally a separate type rather than code baked
// into OpenAIEmbeddingsClient so the retry behaviour can wrap ANY
// EmbeddingsClient implementation (including test fakes) and stay
// unit-testable in isolation.

// RetryAttempts is the total number of Embed calls made on a single
// input (1 initial attempt + 3 retries, per the phase 3 scope).
const RetryAttempts = 4

// RetryBackoffs are the sleep durations applied between attempts, in
// order. Length must be RetryAttempts-1 because the first attempt is
// immediate. Delays chosen from the phase 3 spec: 500ms, 1s, 2s.
var RetryBackoffs = []time.Duration{
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
}

// RetryingEmbeddingsClient wraps another EmbeddingsClient and replays
// calls with exponential backoff on transient failure. Non-transient
// errors (empty input, bad API key) surface immediately so the caller
// can stop wasting API calls.
type RetryingEmbeddingsClient struct {
	inner     EmbeddingsClient
	sleep     func(time.Duration) // injectable for tests (time.Sleep in prod)
	backoffs  []time.Duration
}

// NewRetryingEmbeddings wraps the inner client with exponential
// backoff. Passing nil sleeps the real time.Sleep — tests inject a
// no-op to keep the suite fast.
func NewRetryingEmbeddings(inner EmbeddingsClient) *RetryingEmbeddingsClient {
	return &RetryingEmbeddingsClient{
		inner:    inner,
		sleep:    time.Sleep,
		backoffs: append([]time.Duration(nil), RetryBackoffs...),
	}
}

// WithRetryClock replaces the sleep function. Tests use this to drop
// the wall-clock delay while still asserting on the number of calls.
func (c *RetryingEmbeddingsClient) WithRetryClock(sleep func(time.Duration)) *RetryingEmbeddingsClient {
	c.sleep = sleep
	return c
}

// Embed runs the underlying Embed call with retry. Returns the first
// successful result, or the last error encountered after exhausting
// every attempt.
func (c *RetryingEmbeddingsClient) Embed(ctx context.Context, text string) ([]float32, error) {
	var lastErr error
	for attempt := 0; attempt < RetryAttempts; attempt++ {
		if attempt > 0 {
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("embeddings retry: context cancelled: %w", err)
			}
			c.sleep(c.backoffs[attempt-1])
		}
		vec, err := c.inner.Embed(ctx, text)
		if err == nil {
			return vec, nil
		}
		if !isTransientEmbeddingError(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("embeddings retry: exhausted %d attempts: %w", RetryAttempts, lastErr)
}

// isTransientEmbeddingError reports whether the error looks like a
// network / rate-limit / 5xx situation worth retrying. Conservative
// by default — anything that smells like a client-side bug (empty
// input, 400, 401) is treated as non-transient so we stop wasting
// API calls.
func isTransientEmbeddingError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Non-retryable: deterministic client errors.
	for _, s := range []string{
		"empty input",
		"status 400",
		"status 401",
		"status 403",
		"api key",
	} {
		if strings.Contains(msg, s) {
			return false
		}
	}
	// Retryable: anything that signals network / server / rate-limit
	// transient failure.
	for _, s := range []string{
		"status 429",
		"status 500",
		"status 502",
		"status 503",
		"status 504",
		"deadline exceeded",
		"timeout",
		"connection",
		"eof",
		"reset by peer",
		"temporarily",
		"http:",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	// Default: treat unknown errors as transient so callers get the
	// benefit of the backoff on unexpected failures.
	return true
}
