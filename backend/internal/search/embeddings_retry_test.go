package search

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingEmbedder is a test helper that returns a preset sequence of
// errors (one per call) and finally a fixed vector. Lets each test
// reason about exactly how many retries happened.
type countingEmbedder struct {
	calls    int
	errs     []error
	vector   []float32
}

func (c *countingEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	idx := c.calls
	c.calls++
	if idx < len(c.errs) && c.errs[idx] != nil {
		return nil, c.errs[idx]
	}
	return c.vector, nil
}

func TestRetryingEmbeddings_SuccessFirstAttempt(t *testing.T) {
	vec := []float32{0.1, 0.2}
	inner := &countingEmbedder{vector: vec}
	r := NewRetryingEmbeddings(inner).WithRetryClock(func(time.Duration) {})

	got, err := r.Embed(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, vec, got)
	assert.Equal(t, 1, inner.calls, "should succeed on first try")
}

func TestRetryingEmbeddings_RetriesTransient(t *testing.T) {
	vec := []float32{0.1}
	inner := &countingEmbedder{
		errs:   []error{errors.New("openai embeddings: status 503"), errors.New("openai embeddings: status 503")},
		vector: vec,
	}
	var sleeps []time.Duration
	r := NewRetryingEmbeddings(inner).WithRetryClock(func(d time.Duration) {
		sleeps = append(sleeps, d)
	})

	got, err := r.Embed(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, vec, got)
	assert.Equal(t, 3, inner.calls)
	assert.Equal(t, []time.Duration{500 * time.Millisecond, 1 * time.Second}, sleeps)
}

func TestRetryingEmbeddings_GivesUpAfterMaxAttempts(t *testing.T) {
	inner := &countingEmbedder{
		errs: []error{
			errors.New("status 500"),
			errors.New("status 500"),
			errors.New("status 500"),
			errors.New("status 500"),
		},
	}
	r := NewRetryingEmbeddings(inner).WithRetryClock(func(time.Duration) {})

	_, err := r.Embed(context.Background(), "hello")
	require.Error(t, err)
	assert.Equal(t, RetryAttempts, inner.calls)
	assert.Contains(t, err.Error(), "exhausted")
}

func TestRetryingEmbeddings_NonTransientSurfacesImmediately(t *testing.T) {
	inner := &countingEmbedder{
		errs: []error{errors.New("openai embeddings: empty input")},
	}
	r := NewRetryingEmbeddings(inner).WithRetryClock(func(time.Duration) {})

	_, err := r.Embed(context.Background(), "x")
	require.Error(t, err)
	assert.Equal(t, 1, inner.calls, "client errors must not retry")
}

func TestRetryingEmbeddings_CancelledContextAborts(t *testing.T) {
	inner := &countingEmbedder{
		errs: []error{errors.New("status 503"), errors.New("status 503"), errors.New("status 503"), errors.New("status 503")},
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := NewRetryingEmbeddings(inner).WithRetryClock(func(time.Duration) { cancel() })

	_, err := r.Embed(ctx, "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestIsTransientEmbeddingError(t *testing.T) {
	cases := []struct {
		err  string
		want bool
	}{
		{"openai embeddings: status 503", true},
		{"openai embeddings: status 429", true},
		{"openai embeddings: status 500", true},
		{"connection reset by peer", true},
		{"context deadline exceeded", true},
		{"openai embeddings: empty input", false},
		{"openai embeddings: status 400", false},
		{"openai embeddings: status 401: bad api key", false},
		{"totally unexpected error", true}, // default retry
	}
	for _, tc := range cases {
		t.Run(tc.err, func(t *testing.T) {
			got := isTransientEmbeddingError(errors.New(tc.err))
			assert.Equal(t, tc.want, got)
		})
	}
}
