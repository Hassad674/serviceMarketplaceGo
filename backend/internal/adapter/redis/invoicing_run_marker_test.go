package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/redis"
)

func newRunMarkerTest(t *testing.T, ttl time.Duration) (*adapter.RunMarker, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return adapter.NewRunMarker(client, ttl), mr
}

func TestRunMarker_EmptyInitialState(t *testing.T) {
	marker, _ := newRunMarkerTest(t, time.Hour)

	got, err := marker.GetLastMonthlyRun(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "", got, "an unset marker reads back as empty, not an error")
}

func TestRunMarker_MarkAndReadRoundTrip(t *testing.T) {
	marker, _ := newRunMarkerTest(t, time.Hour)

	require.NoError(t, marker.MarkMonthlyRun(context.Background(), "2026-04"))

	got, err := marker.GetLastMonthlyRun(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "2026-04", got)

	// Overwrite with a newer month — value must update verbatim.
	require.NoError(t, marker.MarkMonthlyRun(context.Background(), "2026-05"))
	got2, err := marker.GetLastMonthlyRun(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "2026-05", got2)
}

func TestRunMarker_TTLIsSet(t *testing.T) {
	marker, mr := newRunMarkerTest(t, 0) // 0 → DefaultInvoicingRunMarkerTTL

	require.NoError(t, marker.MarkMonthlyRun(context.Background(), "2026-04"))

	ttl := mr.TTL("invoicing:monthly:last_run")
	assert.Greater(t, ttl, 30*24*time.Hour, "default TTL must outlive a calendar month so the marker is alive at next tick")
}
