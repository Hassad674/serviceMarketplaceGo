package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// DefaultWebhookIdempotencyTTL is 7 days — Stripe retries failed
// webhooks for up to 3 days, so a 7-day window covers the full replay
// horizon with margin for clock skew.
const DefaultWebhookIdempotencyTTL = 7 * 24 * time.Hour

// WebhookIdempotencyStore dedupes Stripe webhook deliveries. Stripe
// retries events whenever we return a non-2xx, so without this guard a
// transient 500 could apply the same state change twice (e.g. activate a
// subscription and bump StartedAt a second time, poisoning the fees-saved
// stats).
//
// Pattern: the caller invokes TryClaim(eventID) BEFORE processing. On a
// first-write hit (SET NX succeeds), the caller handles the event. On a
// repeat, TryClaim returns (false, nil) and the caller ACKs the webhook
// without doing any work.
type WebhookIdempotencyStore struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewWebhookIdempotencyStore(client *goredis.Client, ttl time.Duration) *WebhookIdempotencyStore {
	if ttl <= 0 {
		ttl = DefaultWebhookIdempotencyTTL
	}
	return &WebhookIdempotencyStore{client: client, ttl: ttl}
}

// TryClaim returns true on the first call for a given eventID, false on
// every subsequent call within the TTL window. A Redis failure returns
// true conservatively — better to risk a rare double-process than to
// silently swallow a real event on a cache blip.
func (s *WebhookIdempotencyStore) TryClaim(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		// No id → can't dedupe → treat every delivery as new. Callers
		// that want strict behaviour must validate eventID themselves.
		return true, nil
	}
	key := "stripe:event:" + eventID
	ok, err := s.client.SetNX(ctx, key, "1", s.ttl).Result()
	if err != nil {
		// Cache down: claim the event rather than drop it. The downstream
		// handlers are themselves idempotent on domain state (Activate on
		// an already-active row is a no-op), so a rare duplicate apply is
		// benign.
		return true, err
	}
	return ok, nil
}
