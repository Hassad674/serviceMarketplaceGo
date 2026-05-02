# P8 — Stripe webhook async via `pending_events` worker

**Phase:** F.2 HIGH #4
**Source audit:** PERF-FINAL-B-12 + scheduler RLS gap
**Effort:** 2j est.
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p8-stripe-webhook-async`

## Problem

`handler/stripe_handler.go::dispatch` runs Stripe webhook handlers **synchronously** with up to 5+ DB writes + PDF generation (chromedp 2-5s) + email sends. Stripe webhook timeout = 10s. PR #65 fixed BUG-NEW-06 (release claim on error) but the timeout risk under load remains.

Plus: scheduler paths (auto-approve milestone, auto-close, dispute auto-resolve, AI summary) currently run on the privileged role bypassing RLS. Need to migrate them to the system-actor pattern from P1.

## Decision (LOCKED — user validated)

**Enqueue immediately + worker async**:
- Webhook handler verifies signature (sync, fast)
- Inserts row in `pending_events` table (already exists, used by Phase 6 worker)
- Returns 200 OK immediately
- The `pending_events_worker` polls + dispatches to the appropriate handler
- **Idempotency**: claim PERSISTS (don't release on success) so a Stripe re-delivery of the same event_id creates 0 second pending_event row (uniqueness constraint on event_id)

## Plan (5 commits)

### Commit 1 — pending_events schema extension
- Migration 134 : ADD COLUMN `event_type TEXT` if not exists already, ensure unique index on `event_id` for Stripe events
- Tests

### Commit 2 — Stripe webhook → enqueue
- `handler/stripe_handler.go` :
  - Strip the inline `dispatch()` chain
  - Add `pending_events.Insert(...)` with `event_type='stripe.<event.type>'`, `payload=event.JSON`, `priority=1`
  - Return 200 OK immediately
- Tests : webhook handler test asserting <50ms response time, asserting pending_event row exists

### Commit 3 — Worker handler registration
- `app/searchindex/publisher.go` (or wherever worker handlers register) :
  - Register `stripe.checkout.session.completed` → handler
  - `stripe.customer.subscription.updated` → handler
  - `stripe.invoice.paid` → handler
  - etc. (mirror current dispatch() handlers)
- Each worker handler runs the existing logic (PDF gen, etc.) but in worker context, not request context
- Tests

### Commit 4 — Scheduler RLS migration to system-actor
- `app/proposal/service_scheduler.go` AutoApproveMilestone, AutoCloseProposal :
  - Wrap in `system.WithSystemActor(ctx)` at entry point
  - Repo calls already work via legacy GetByID + warnIfNotSystemActor (P1 setup)
- `app/dispute/scheduler.go` auto-resolve : same pattern
- Wire entry points in `cmd/api/wire_*.go`
- Integration test : run scheduler under non-superuser role + system-actor context, asserter all paths green

### Commit 5 — Docs + observability
- `backend/docs/stripe-webhook.md` : explain the async architecture, retry behavior, idempotency guarantees
- Add structured logging: `event_id`, `event_type`, `enqueue_ms`, `process_ms`
- Tests

## Hard constraints

- **Zero data loss**: Stripe re-delivers if no 2xx response within 10s. Our 200 must come BEFORE any DB write fails. Use a single insert into pending_events with `ON CONFLICT (event_id) DO NOTHING` so re-deliveries are safe.
- **Validation pipeline**: build + vet + test -race before every commit
- **Worker idempotency**: each worker handler must be idempotent (can be re-run on same event without side-effects). Use existing webhook idempotency table (Redis + Postgres) per-handler if needed.

## OFF-LIMITS

- LiveKit / call code, workflow files, other plans
- Stripe webhook signature verification logic — keep as is

## Branch ownership

`fix/p8-stripe-webhook-async` only. Created from main.

## Final report (under 700 words)

PR URL first. Then:
1. Webhook response time (before sync N seconds → after <50ms)
2. Worker handlers registered (count + list)
3. Migration 134 applied yes/no
4. Scheduler paths migrated to system-actor (count)
5. Validation pipeline output
6. Branch ownership confirmed
