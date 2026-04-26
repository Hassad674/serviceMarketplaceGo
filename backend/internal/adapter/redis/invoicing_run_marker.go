package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// invoicingRunMarkerKey is the Redis key where the last completed
// monthly-consolidation run is stored. A single key is enough because
// only one batch runs at a time and the value is the YYYY-MM month
// that was just processed.
const invoicingRunMarkerKey = "invoicing:monthly:last_run"

// DefaultInvoicingRunMarkerTTL is 35 days — long enough that the next
// scheduled run (one calendar month later) still finds the marker
// alive when it checks. A shorter TTL would risk re-running a month
// the first time the scheduler re-ticked after a Redis flush.
const DefaultInvoicingRunMarkerTTL = 35 * 24 * time.Hour

// RunMarker tracks the last YYYY-MM successfully consolidated by the
// monthly invoicing batch. The scheduler reads it before each tick and
// short-circuits when the value matches the month it would have
// processed.
//
// On a Redis miss / blip the scheduler is biased toward re-running:
// the Service-level idempotency probe (synthetic stripe_event_id) is
// the durable backstop, so a duplicate tick is wasted work but never
// a duplicate invoice.
type RunMarker struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewRunMarker wires the marker. TTL <= 0 falls back to
// DefaultInvoicingRunMarkerTTL.
func NewRunMarker(client *goredis.Client, ttl time.Duration) *RunMarker {
	if ttl <= 0 {
		ttl = DefaultInvoicingRunMarkerTTL
	}
	return &RunMarker{client: client, ttl: ttl}
}

// GetLastMonthlyRun returns the YYYY-MM string most recently written
// by MarkMonthlyRun, or empty when nothing has been recorded (or the
// key has expired).
func (r *RunMarker) GetLastMonthlyRun(ctx context.Context) (string, error) {
	val, err := r.client.Get(ctx, invoicingRunMarkerKey).Result()
	if errors.Is(err, goredis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// MarkMonthlyRun writes the YYYY-MM marker with a 35-day TTL. Re-runs
// for the same month are idempotent — the SET overwrites and resets
// the TTL.
func (r *RunMarker) MarkMonthlyRun(ctx context.Context, monthKey string) error {
	return r.client.Set(ctx, invoicingRunMarkerKey, monthKey, r.ttl).Err()
}
