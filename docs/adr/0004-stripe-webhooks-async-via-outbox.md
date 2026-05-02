# 0004. Async Stripe webhooks via the pending\_events outbox

Date: 2026-04-30

## Status

Accepted

## Context

Stripe webhooks signal the backend of payment events
(`payment_intent.succeeded`, `charge.refunded`,
`account.updated`, etc.). Stripe enforces a strict
acknowledgement contract:

- The endpoint must respond **2xx within 10 seconds**, otherwise
  Stripe retries with exponential backoff up to 72 hours.
- A 5xx response is a retry signal. Repeated 5xxs eventually
  disable the webhook on Stripe's side.

Some events trigger heavy downstream work:

- `payment_intent.succeeded` for a proposal payment activates
  the proposal, sends notifications to both parties, allocates
  the platform fee, and updates the agency's wallet.
- `account.updated` recomputes the agency's KYC + payout
  eligibility, re-indexes their search document, and emits
  notifications.

A naive synchronous handler exceeds the 10 s budget under load
(measured: p95 of 14 s when the search index is hot). Past that,
we lose events.

In addition, our infrastructure occasionally has transient outages
(Redis blip, Typesense reindex). A synchronous handler that fails
mid-flight gives Stripe a 5xx — which is retried, but the partial
work in our database (e.g. proposal half-activated) needs explicit
compensation logic.

## Decision

We will **acknowledge Stripe webhooks immediately** and process
them **asynchronously** via a transactional outbox.

Concrete flow:

1. Webhook handler verifies the Stripe signature and parses the
   event.
2. **In one local transaction**, the handler:
   - Inserts a row in `pending_events` with the event id,
     payload, and `status='pending'`.
   - Returns `200 OK` to Stripe with no further side effects.
3. A background worker (`internal/app/pendingevent/worker.go`)
   polls `pending_events` for unprocessed rows and dispatches
   each to its registered handler.
4. The dispatcher uses the event id as an idempotency key —
   repeated insertions of the same event id are no-ops (unique
   constraint on `stripe_event_id`).
5. A handler can mark an event `processed`, `failed_retryable`
   (worker re-attempts later), or `failed_permanent` (alert and
   move on).
6. A stale-event recovery cron (`migration 128`) sweeps events
   stuck in `processing` for over 5 minutes and resets them.

The pattern is the **transactional outbox** —
[Microservices.io: Transactional Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html).
We keep the simple shape: one outbox table, a single worker, no
broker.

## Consequences

### Positive

- Stripe always sees `200 OK` within milliseconds; no lost
  events, no retry storms.
- Handlers can be slow (search reindex, notification fanout)
  without affecting the webhook ack.
- Idempotency is built into the architecture: re-running the
  worker on a partially-processed event is safe because every
  step looks up the event id in the database first.
- Crash recovery: the stale-event recovery cron unblocks events
  whose worker died mid-handler.
- Handlers are unit-testable in isolation (no Stripe SDK in
  the test path; the test enqueues a `pending_events` row and
  asserts the handler processed it).

### Negative

- Latency between "Stripe says paid" and "user sees activated
  proposal" is now bounded by the worker's poll interval (1 s
  in production). This is acceptable; the UX shows a "processing"
  state during that window.
- One additional table to operate. We mitigate with metrics:
  `pending_events_lag_seconds` is logged by the worker on every
  poll cycle.
- The worker's at-least-once semantics impose idempotency
  discipline on every handler. Concretely: every handler that
  writes to a domain table must check for an existing row by
  the event's natural key before inserting. Most handlers do
  this anyway (e.g. proposal activation checks
  `proposal.status != 'active'` first), but it's a discipline
  to maintain.

## Alternatives considered

- **Synchronous handlers** — the original implementation. Lost
  events under load and partial-state bugs on transient
  failures. Rejected.
- **Push the event to a message broker (Redis Streams,
  RabbitMQ, Kafka)** — proper separation of concerns, but adds
  operational surface (broker monitoring, consumer groups, dead
  letter queues). For our throughput (peak ~50 webhooks/min)
  the database table is simpler and sufficient. We will
  reconsider if event volume grows by 100x.
- **Stripe's native event delivery via Workers / Edge** — vendor
  lock-in to Stripe. Rejected; we want the webhook architecture
  to apply uniformly to any provider (PayPal, Mollie) we add
  later.

## References

- `backend/migrations/087_create_pending_events.up.sql` — initial
  outbox table.
- `backend/migrations/128_pending_events_stale_recovery.up.sql`
  — stale-event recovery cron support.
- `backend/migrations/134_pending_events_stripe_event_id.up.sql`
  — dedupe column for Stripe-sourced events.
- `backend/internal/app/pendingevent/worker.go` — the
  async dispatcher.
- `backend/internal/handler/stripe_handler.go` — the
  webhook endpoint that enqueues into `pending_events`.
- Microservices.io, *Transactional Outbox Pattern*,
  <https://microservices.io/patterns/data/transactional-outbox.html>.
