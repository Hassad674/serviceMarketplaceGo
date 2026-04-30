package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Default cache TTL for the Redis fast-path of the webhook
// idempotency check. Five minutes is plenty: Stripe retries on a
// 5-second backoff and almost every replay lands within a minute, so
// a five-minute window catches the burst pattern while keeping Redis
// memory pressure low. The DURABLE source of truth is Postgres
// (`stripe_webhook_events`), so the cache TTL no longer has to
// cover the full Stripe retry horizon — the table does.
const DefaultWebhookIdempotencyTTL = 5 * time.Minute

// CacheError is returned by TryCacheClaim when the underlying Redis
// operation fails. Callers (the composite claimer) use it to fall
// through to the durable Postgres path without incorrectly assuming
// the event is a duplicate. The wrapped error is preserved so logs
// can still report the root cause.
type CacheError struct{ Err error }

func (e *CacheError) Error() string { return "webhook idempotency cache: " + e.Err.Error() }
func (e *CacheError) Unwrap() error { return e.Err }

// WebhookIdempotencyStore is the Redis fast-path for webhook
// dedup. It is no longer the source of truth — that role moved to
// the postgres `stripe_webhook_events` table after BUG-10 — so its
// failure modes are now strictly informational:
//
//   - hit (key already exists)  → return (false, nil), the composite
//     claimer skips Postgres entirely.
//   - miss (key was new)        → return (true, nil), the composite
//     claimer still consults Postgres before declaring "first
//     delivery" so two backends racing on the same event resolve
//     correctly.
//   - cache error               → return (false, *CacheError), the
//     composite claimer falls through to Postgres unconditionally.
type WebhookIdempotencyStore struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewWebhookIdempotencyStore wires the Redis fast-path. ttl ≤ 0
// applies DefaultWebhookIdempotencyTTL.
func NewWebhookIdempotencyStore(client *goredis.Client, ttl time.Duration) *WebhookIdempotencyStore {
	if ttl <= 0 {
		ttl = DefaultWebhookIdempotencyTTL
	}
	return &WebhookIdempotencyStore{client: client, ttl: ttl}
}

// TryCacheClaim attempts to claim `eventID` in Redis using SETNX.
//
//   - returned (true, nil)  → cache had no record, caller MUST still
//     consult the durable store before declaring "first delivery".
//   - returned (false, nil) → cache reports the event has been seen
//     in the last TTL window — caller can short-circuit without
//     hitting Postgres.
//   - returned (false, *CacheError) → Redis is unavailable. Caller
//     MUST fall through to Postgres. Crucially we DO NOT return
//     `(true, err)` here — that was the pre-fix conservative-claim
//     behaviour that allowed double-processing during Redis outages.
//
// An empty eventID is treated as a cache miss with no error so the
// composite path can validate the input downstream.
func (s *WebhookIdempotencyStore) TryCacheClaim(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return true, nil
	}
	key := "stripe:event:" + eventID
	ok, err := s.client.SetNX(ctx, key, "1", s.ttl).Result()
	if err != nil {
		return false, &CacheError{Err: err}
	}
	return ok, nil
}

// MarkSeen sets the cache entry without checking for previous
// presence. Used by the composite claimer after Postgres confirms
// the first / repeat verdict so subsequent in-window replays can
// short-circuit on Redis. Errors are returned as *CacheError so
// the caller can downgrade them to a log-and-continue.
func (s *WebhookIdempotencyStore) MarkSeen(ctx context.Context, eventID string) error {
	if eventID == "" {
		return nil
	}
	if err := s.client.Set(ctx, "stripe:event:"+eventID, "1", s.ttl).Err(); err != nil {
		return &CacheError{Err: err}
	}
	return nil
}

// TryClaim is the legacy single-store dedup probe used by features
// OTHER than the Stripe webhook handler (invoicing module's
// application-level idempotency for IssueFromSubscription /
// IssueCreditNote). Those callers tolerate cache-only semantics
// because their database paths are protected by separate UNIQUE
// constraints (invoices.stripe_event_id, credit_notes.stripe_refund_id).
//
// For webhook idempotency, callers must use the composite claimer in
// app/webhookidempotency, which combines this cache and the durable
// stripe_webhook_events table. The composite path eliminates the
// "claim conservatively on Redis error" hole that BUG-10 closed.
//
// Behaviour:
//   - claim succeeded → (true, nil)
//   - replay         → (false, nil)
//   - cache error    → (false, *CacheError) — caller decides whether
//                      to fall through to a DB-level dedup or fail
//                      the request. Application code that has its own
//                      DB UNIQUE constraint typically logs and
//                      proceeds; webhook code MUST consult Postgres.
func (s *WebhookIdempotencyStore) TryClaim(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return true, nil
	}
	return s.TryCacheClaim(ctx, eventID)
}
