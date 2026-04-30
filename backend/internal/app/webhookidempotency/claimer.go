// Package webhookidempotency owns the composite cache+durable
// claimer used by the Stripe webhook handler to dedupe replays.
//
// Why a dedicated package: the claimer composes two independent
// adapters (Redis fast-path + Postgres source-of-truth) and the
// composition logic itself has correctness consequences (BUG-10).
// Putting it next to the handler would couple the test surface to
// the HTTP layer; putting it inside one of the adapters would force
// that adapter to import the other. The app/ layer is the natural
// home for the orchestration.
//
// Source-of-truth contract (BUG-10):
//
//   - Redis is a 5-minute fast-path cache.
//   - Postgres `stripe_webhook_events` is the durable, infinite
//     source of truth — Stripe retries failed webhooks for up to
//     3 days, but the upstream window of concern (a payment fund
//     being claimed twice) requires a permanent record.
//   - Redis HIT  → skip Postgres, return "already processed".
//   - Redis MISS → consult Postgres → record verdict in Redis.
//   - Redis DOWN → fall through to Postgres unconditionally.
//   - Postgres DOWN AND Redis DOWN → return error so the handler
//     can reply 503. The pre-fix code returned "claimed" on any
//     cache failure, which let Stripe re-deliver and apply the
//     same state change twice. The new behaviour explicitly fails
//     loud so the operator notices.
package webhookidempotency

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"marketplace-backend/internal/adapter/redis"
)

// CacheStore is the narrow port the composite claimer expects from
// the Redis adapter. Defined locally so this package does not pull
// in the goredis client.
type CacheStore interface {
	// TryCacheClaim returns:
	//   (true, nil)         → cache miss, caller must verify in DB
	//   (false, nil)        → cache hit, event was seen recently
	//   (false, *CacheError) → Redis is unavailable
	TryCacheClaim(ctx context.Context, eventID string) (bool, error)

	// MarkSeen seeds the cache after Postgres has decided.
	MarkSeen(ctx context.Context, eventID string) error
}

// DurableStore is the Postgres-backed source of truth.
type DurableStore interface {
	// TryClaim returns:
	//   (true, nil)  → INSERT succeeded; this is the first delivery
	//   (false, nil) → ON CONFLICT path; this is a replay, skip it
	//   (_, err)     → SQL exec failed; caller must NOT process
	TryClaim(ctx context.Context, eventID, eventType string) (bool, error)
}

// Claimer composes a cache fast-path and a durable source of truth.
type Claimer struct {
	cache   CacheStore
	durable DurableStore
}

// NewClaimer wires the composite claimer. The durable store is
// REQUIRED — passing nil produces an error at construction time so
// no caller can accidentally degrade to "Redis only" semantics
// (which is exactly what BUG-10 was). The cache store is optional;
// when nil the claimer always consults the durable store.
func NewClaimer(durable DurableStore, cache CacheStore) (*Claimer, error) {
	if durable == nil {
		return nil, errors.New("webhook idempotency: durable store is required")
	}
	return &Claimer{cache: cache, durable: durable}, nil
}

// TryClaim returns true on the first delivery for `eventID`, false
// when the event has already been processed, and a non-nil error
// only when BOTH layers are unavailable. The signature stays
// 3-argument with eventType so the durable INSERT can populate the
// event_type column without a separate fetch.
//
// The flow:
//   1. Empty eventID → reject ("can't dedupe").
//   2. Try Redis fast-path:
//        - hit → return (false, nil)  [skip durable, fastest path]
//        - miss → fall through to Postgres
//        - cache error → fall through, log warn, do NOT short-circuit
//   3. Postgres INSERT ON CONFLICT DO NOTHING:
//        - rows == 1 → first delivery, populate cache, return true
//        - rows == 0 → replay, populate cache, return false
//        - SQL error → return (false, err) so the handler can 503
func (c *Claimer) TryClaim(ctx context.Context, eventID, eventType string) (bool, error) {
	if eventID == "" {
		return false, fmt.Errorf("webhook idempotency: empty event_id")
	}

	if c.cache != nil {
		claimed, cacheErr := c.cache.TryCacheClaim(ctx, eventID)
		var ce *redis.CacheError
		switch {
		case cacheErr == nil && !claimed:
			// Fast-path hit: this event was processed in the last
			// TTL window. Return immediately without touching
			// Postgres — this is the hot path for Stripe retries
			// during normal operation.
			return false, nil
		case cacheErr == nil && claimed:
			// Cache miss but claim succeeded → caller is the first
			// to see this id IN-CACHE. We MUST still consult
			// Postgres in case another backend instance saw the
			// same event between our SETNX and now.
		case errors.As(cacheErr, &ce):
			// Cache layer down. Fall through to durable. Log so the
			// operator sees the degradation; don't return.
			slog.Warn("webhook idempotency: cache unavailable, falling through to durable",
				"event_id", eventID, "error", cacheErr)
		default:
			// Unknown error class — should not happen with the
			// current adapter, but defensively fall through to
			// durable rather than fail the request outright.
			slog.Warn("webhook idempotency: unexpected cache error, falling through to durable",
				"event_id", eventID, "error", cacheErr)
		}
	}

	first, err := c.durable.TryClaim(ctx, eventID, eventType)
	if err != nil {
		return false, fmt.Errorf("webhook idempotency: durable claim failed: %w", err)
	}

	c.populateCacheBestEffort(ctx, eventID)
	return first, nil
}

// populateCacheBestEffort writes the verdict to Redis so subsequent
// in-window replays can short-circuit. Failures are logged but never
// surfaced — the durable verdict has already been returned.
func (c *Claimer) populateCacheBestEffort(ctx context.Context, eventID string) {
	if c.cache == nil {
		return
	}
	if err := c.cache.MarkSeen(ctx, eventID); err != nil {
		slog.Debug("webhook idempotency: cache populate failed (non-fatal)",
			"event_id", eventID, "error", err)
	}
}
