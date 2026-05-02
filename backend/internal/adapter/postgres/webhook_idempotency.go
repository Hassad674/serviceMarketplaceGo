package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// WebhookIdempotencyStore is the durable webhook-dedup adapter
// backed by the `stripe_webhook_events` table (migration 089).
//
// Why durable: the Redis fast-path on its own fails open — if Redis
// is unavailable, the original code "claimed conservatively" (treat
// every event as new) and could process the same webhook twice in
// the rare case Stripe also retried during the Redis outage. Doubling
// up subscription rows or commission invoices is the costliest class
// of bugs we can ship, so the pre-merge fix (BUG-10) makes Postgres
// the source of truth: even when Redis is down, the unique-constraint
// INSERT settles whether we have seen this event_id before.
//
// The pattern is INSERT ... ON CONFLICT DO NOTHING — a single
// round-trip that takes a row-level lock only on the new key, so
// concurrent webhook deliveries with different event_ids don't
// contend.
type WebhookIdempotencyStore struct {
	db *sql.DB
}

// NewWebhookIdempotencyStore wires the durable store against the
// shared *sql.DB pool.
func NewWebhookIdempotencyStore(db *sql.DB) *WebhookIdempotencyStore {
	return &WebhookIdempotencyStore{db: db}
}

// TryClaim attempts to record `eventID` as the first delivery for
// the given event_type. Returns (true, nil) when the row was inserted
// (= we are the first to see this event), (false, nil) when the row
// was already present (= duplicate, skip), and (false, err) when the
// underlying SQL exec failed for an infrastructure reason.
//
// Callers MUST treat (false, err) as fatal — they cannot blindly
// process the webhook because they don't know whether the event was
// new or replayed. The composite cache+postgres claimer reflects
// that: it only fails the request when BOTH layers are down.
func (s *WebhookIdempotencyStore) TryClaim(ctx context.Context, eventID, eventType string) (bool, error) {
	if eventID == "" {
		return false, fmt.Errorf("webhook idempotency: empty event_id")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := Exec(ctx, s.db, `
		INSERT INTO stripe_webhook_events (stripe_event_id, event_type)
		VALUES ($1, $2)
		ON CONFLICT (stripe_event_id) DO NOTHING`,
		eventID, eventType,
	)
	if err != nil {
		return false, fmt.Errorf("insert webhook event: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	// rows == 0 means the conflict path executed → the event was
	// already recorded → this is a replay, the caller must skip.
	return rows == 1, nil
}

// Release deletes the row keyed by eventID, reversing a prior TryClaim.
// BUG-NEW-06 — used by the webhook dispatcher when a downstream handler
// returns an error AFTER the claim succeeded; without this DELETE the
// durable claim is permanent and Stripe's next retry would be silently
// deduped, dropping the state change forever.
//
// Idempotent: a missing row returns nil (no error) so a concurrent
// release on the same id does not surface as a failure.
func (s *WebhookIdempotencyStore) Release(ctx context.Context, eventID string) error {
	if eventID == "" {
		return fmt.Errorf("webhook idempotency: empty event_id on release")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := Exec(ctx, s.db,
		`DELETE FROM stripe_webhook_events WHERE stripe_event_id = $1`,
		eventID,
	)
	if err != nil {
		return fmt.Errorf("delete webhook event: %w", err)
	}
	return nil
}
