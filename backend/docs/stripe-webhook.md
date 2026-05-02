# Stripe webhook async dispatch (P8)

This document describes the async-dispatch architecture installed by
P8 for the Stripe webhook endpoint. The HTTP handler now responds 200
OK in <50ms; the heavy work (PDF generation, multi-row DB writes,
fan-out emails) runs in a background worker against the
`pending_events` queue.

## Why async

Pre-P8 the Stripe webhook dispatcher was synchronous:

```
HTTP request arrives
  → verify signature
  → idempotency claim (Redis + Postgres)
  → dispatch (PDF generation 2-5s, DB writes, email sends)
  → reply 200 (or 5xx + Release on error)
```

On `invoice.paid`, the chrome-headless PDF render alone could take 2-5
seconds. Combined with email delivery, multi-row invoice persistence,
and credit-note triggers, a single delivery routinely consumed 6-8
seconds — uncomfortably close to Stripe's 10s timeout. Under load
(e.g. monthly billing run) the tail latency tipped over and Stripe
retried, multiplying load and wasting compute.

P8 splits the pipeline: the HTTP handler does only the cheap work
(signature verification + queue write), and the worker drains the
queue in the background.

## The async path

```
HTTP request arrives
  → verify Stripe signature      (sync, fast — pure crypto)
  → marshal projected event JSON (sync)
  → INSERT pending_events
       ON CONFLICT (stripe_event_id) DO NOTHING
  → reply 200 OK                 (≤50ms target)
```

The dispatch chain runs in `adapter/worker/handlers/stripe_handlers.go`,
which decodes the persisted event and calls
`StripeHandler.Dispatch(ctx, event)` exactly as the synchronous path
used to.

## Idempotency model

Three layers, evaluated in order:

1. **`pending_events.stripe_event_id` partial unique index**
   (migration 134). The webhook handler's INSERT uses
   `ON CONFLICT (stripe_event_id) DO NOTHING`, so a Stripe re-delivery
   of the same `evt_*` is a silent no-op. **This is the primary
   idempotency line.**
2. **Worker stale-row recovery (BUG-NEW-03).** A worker that crashes
   between claiming a row and calling `MarkDone`/`MarkFailed` leaves
   the row in `processing` status. After 5 minutes, another worker
   re-claims and re-dispatches. This is why every per-event handler
   downstream of `Dispatch` must be idempotent.
3. **Per-handler idempotency.** Each Stripe-event handler (subscription
   register/snapshot, invoice issuance, credit notes, payment
   confirmation) already guards on a domain-specific marker:
   - `payment_intent.succeeded` → `loadProposalForActor` short-circuits
     when the proposal is already active.
   - `customer.subscription.created/updated/deleted` →
     `RegisterFromCheckout` and `HandleSubscriptionSnapshot` are
     UPSERT-shaped on the Stripe subscription id.
   - `invoice.paid` → `IssueFromSubscription` guards on
     `(stripe_event_id, stripe_invoice_id)` — duplicate inserts hit
     a unique constraint, the handler logs and returns nil so the
     worker marks the row done without retrying.
   - `charge.refunded` → `IssueCreditNote` guards on `stripe_event_id`.

The legacy `IdempotencyClaimer` (Redis + `stripe_webhook_events`
table) is preserved on the inline-dispatch fallback path used by unit
tests that don't wire a queue. In production the queue is always
wired, and the claimer is never consulted.

## Retry behaviour

Stripe retries on any non-2xx response with exponential backoff (3
days max). The webhook handler can return 5xx in three places:

- Body read failure / missing `Stripe-Signature` header → 400, no retry.
- Signature verification failure → 400, no retry.
- `ScheduleStripe` returns an error (Postgres down, schema drift,
  etc.) → 503 with `Retry-After` from the standard `res.Error`
  builder. Stripe retries; the next attempt re-runs the same flow.

Once the row is enqueued, the worker is the only entity responsible
for retries. If `Dispatch` returns an error, the worker bumps the
attempts counter and reschedules the row via the domain backoff
schedule (1m → 5m → 15m → 1h → 6h, capped at `MaxAttempts = 5`). After
5 attempts the row stays in `failed` status and surfaces in the admin
pending-events view for manual triage.

## Event types covered

Every Stripe event the synchronous dispatcher handled is now async:

| Stripe event type | Handler | Side effects |
|---|---|---|
| `payment_intent.succeeded` | `handlePaymentSucceeded` | Confirm payment + activate proposal |
| `payment_intent.payment_failed` | (log only) | None |
| `account.updated` and friends | `dispatchEmbeddedNotif` | Diff-based notifications via embedded notifier |
| `customer.subscription.created` | `handleSubscriptionCreated` | Register subscription from checkout, enforce auto-renew flag |
| `customer.subscription.updated/deleted` | `handleSubscriptionSnapshot` | Reflect Stripe state into our row |
| `invoice.payment_failed` | `handleInvoicePaymentFailed` | Audit log only (snapshot path covers state transition) |
| `invoice.paid` | `handleInvoicePaid` | Issue customer-facing FAC invoice, render PDF, email |
| `charge.refunded` | `handleChargeRefunded` | Issue credit note (AV) for refunded invoice |

Unknown event types log at `DEBUG` and the worker marks the row
done (no retry).

## Observability

Structured logs emitted on the enqueue path:

| Log line | Fields |
|---|---|
| `stripe webhook: enqueued for async dispatch` | `event_id`, `event_type`, `enqueue_ms` |
| `stripe webhook: duplicate delivery deduplicated by ON CONFLICT` | `event_id`, `event_type`, `enqueue_ms` |
| `stripe webhook: enqueue failed — Stripe will retry` | `event_id`, `event_type`, `error` |

Worker-side logs (existing in `adapter/worker/worker.go`):

| Log line | Fields |
|---|---|
| `worker: dispatching event` (DEBUG) | `event_id`, `event_type`, `attempts` |
| `worker: event handler failed` (WARN) | `event_id`, `event_type`, `attempts`, `next_fires_at`, `error` |

The worker handler itself logs the per-event helper traces from the
synchronous code path verbatim — moving to async did not change any
business-logic logging.

## Operational notes

- The worker's `TickInterval` is 30s by default. Cold-start latency
  is bounded above by 30s, but the worker runs an immediate tick on
  startup so events backed up while the process was down get
  processed without waiting.
- The worker is safe to run on multiple instances concurrently —
  `PopDue` uses `FOR UPDATE SKIP LOCKED` so workers never claim the
  same row.
- The pre-5xx response path (signature verify → enqueue) executes
  three small SQL/no-op steps and is well within Stripe's 10s
  budget on a healthy database. The pre-async slow tail is fully
  eliminated.
- Stripe's webhook signature verification logic is verbatim from the
  pre-P8 code; P8 only changes what runs *after* the signature has
  been verified.

## Migration to async

Migration 133 added `stripe_event_id TEXT` to `pending_events` plus a
partial unique index. The migration is fast metadata-only (Postgres
11+) and backfill-free — existing rows get `NULL` and the index
ignores them.

Rollback is reversible: `134_pending_events_stripe_event_id.down.sql`
drops the index and the column. The application code falls back to
inline dispatch when `WithPendingEventsQueue` is not called, so
disabling the async path requires only un-wiring the setter in
`cmd/api/wire_late_handlers.go`.
