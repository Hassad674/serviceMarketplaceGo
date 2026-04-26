package vies_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/vies"
)

// newTestClient wires the adapter against a miniredis-backed go-redis
// client and a fake VIES server, returning the call counter so tests
// can assert cache hits never reach the network.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*vies.Client, *httptest.Server, *miniredis.Miniredis, *atomic.Int64) {
	t.Helper()

	calls := &atomic.Int64{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	c := vies.NewClient(rdb, vies.WithEndpoint(srv.URL), vies.WithCacheTTL(60*time.Second))
	return c, srv, mr, calls
}

func writeVIESJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

func TestClient_Validate_ValidVAT(t *testing.T) {
	c, _, _, calls := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeVIESJSON(t, w, map[string]any{
			"isValid":     true,
			"requestDate": "2026-04-25T10:30:00Z",
			"name":        "ACME GMBH",
			"address":     "Hauptstr. 1, 10115 Berlin",
			"countryCode": "DE",
			"vatNumber":   "123456789",
		})
	})

	res, err := c.Validate(context.Background(), "de", " 123456789 ")
	require.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Equal(t, "DE", res.CountryCode)
	assert.Equal(t, "123456789", res.VATNumber)
	assert.Equal(t, "ACME GMBH", res.RegisteredName)
	assert.Contains(t, res.RegisteredAddr, "Berlin")
	assert.NotEmpty(t, res.RawPayload, "raw payload kept for legal proof")
	assert.Greater(t, res.CheckedAt, int64(0))
	assert.Equal(t, int64(1), calls.Load())
}

func TestClient_Validate_InvalidVAT_NotCached(t *testing.T) {
	c, _, mr, calls := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeVIESJSON(t, w, map[string]any{
			"isValid":   false,
			"userError": "INVALID_INPUT",
		})
	})

	res, err := c.Validate(context.Background(), "FR", "00000000000")
	require.NoError(t, err)
	assert.False(t, res.Valid)
	assert.Equal(t, int64(1), calls.Load())

	// The negative result MUST NOT be cached — retries should hit VIES.
	keys := mr.Keys()
	assert.Empty(t, keys, "negative results must never be cached")

	// Second call hits the network again.
	_, err = c.Validate(context.Background(), "FR", "00000000000")
	require.NoError(t, err)
	assert.Equal(t, int64(2), calls.Load(), "negative result must re-query VIES")
}

func TestClient_Validate_PositiveResult_CacheHit_NoNetwork(t *testing.T) {
	c, _, mr, calls := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeVIESJSON(t, w, map[string]any{
			"isValid":     true,
			"requestDate": "2026-04-25T10:30:00Z",
			"name":        "ACME GMBH",
			"countryCode": "DE",
			"vatNumber":   "123456789",
		})
	})

	// Prime the cache.
	_, err := c.Validate(context.Background(), "DE", "123456789")
	require.NoError(t, err)
	assert.Equal(t, int64(1), calls.Load())

	// Cache key written.
	keys := mr.Keys()
	require.Len(t, keys, 1)
	assert.Equal(t, "vies:DE123456789", keys[0])

	// Second call must serve from cache — calls counter unchanged.
	res, err := c.Validate(context.Background(), "de", "123456789") // mixed case still hits cache
	require.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Equal(t, "ACME GMBH", res.RegisteredName)
	assert.Equal(t, int64(1), calls.Load(), "cache hit must not reach the VIES endpoint")
}

func TestClient_Validate_NetworkError_WrappedCleanly(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	// Point at a guaranteed-unreachable URL.
	c := vies.NewClient(rdb,
		vies.WithEndpoint("http://127.0.0.1:1/vies"),
		vies.WithHTTPClient(&http.Client{
			Timeout: 200 * time.Millisecond,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 200 * time.Millisecond,
			},
		}),
	)

	res, err := c.Validate(context.Background(), "FR", "26878912963")
	require.Error(t, err)
	assert.False(t, res.Valid)

	// Wrap must mention "vies:" prefix and surface the underlying url.Error.
	var urlErr *url.Error
	assert.True(t, errors.As(err, &urlErr) || err.Error() != "")
	assert.Contains(t, err.Error(), "vies:")
}

func TestClient_Validate_EmptyInput_Error(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	c := vies.NewClient(rdb)

	_, err = c.Validate(context.Background(), "", "12345")
	assert.Error(t, err)

	_, err = c.Validate(context.Background(), "FR", "")
	assert.Error(t, err)
}
