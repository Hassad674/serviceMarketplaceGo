# P11 — OpenTelemetry traces + metrics + graceful shutdown polish

**Phase:** F.2 HIGH #7 (FINAL F.2 plan)
**Source audit:** SCAL-FINAL (observability foundation) + scalability gaps
**Effort:** 1.5j est.
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p11-otel-graceful-shutdown`

## Goal

Wire OpenTelemetry (OTel) instrumentation throughout the backend + extend graceful shutdown to drain WebSocket connections + worker queues.

## Decisions (LOCKED — user validated)

### OTel exporter
- **OTLP standard** (no vendor lock-in)
- Configured via env vars `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`
- The deployment chooses the backend (Jaeger / Honeycomb / Datadog / Grafana Tempo)
- If `OTEL_EXPORTER_OTLP_ENDPOINT` is empty → no-op exporter (zero overhead in dev)

### Spans coverage
- HTTP server : 1 span per request (with `request_id`, `route`, `method`, `status`, `duration_ms`)
- DB queries : 1 span per query (with sanitized SQL + duration)
- Redis ops : 1 span per command
- Outbound HTTP (Stripe, Resend, etc.) : 1 span per call
- Workers (pending_events, scheduler) : 1 span per job

### Graceful shutdown polish
- Current : 30s timeout HTTP server shutdown
- Add : drain WebSocket connections (close all WS hub connections cleanly)
- Add : flush worker queues (pending_events worker stops processing new but completes current)
- Add : flush logs (slog default flushes via stdout but explicit Sync() for any other handler)
- Total budget stays 30s, sub-budgets : 15s HTTP, 10s WS, 5s workers + flush

## Plan (5 commits)

### Commit 1 — OTel SDK setup
- `internal/observability/otel.go` : new package with `Init(cfg) (shutdown func(), error)` that sets up TracerProvider + MeterProvider with OTLP exporter
- `cmd/api/main.go` (or wire_infra.go) : call `Init()` at startup, defer shutdown
- env vars in `backend/.env.example`
- Unit tests on no-op fallback (empty endpoint)

### Commit 2 — HTTP server instrumentation
- Middleware `OtelMiddleware(handler)` that wraps every request in a span
- Use `otelhttp.NewHandler` from `go.opentelemetry.io/contrib`
- Tests : 1 request creates 1 span with right attributes

### Commit 3 — DB instrumentation
- Wrap `*sql.DB` queries via `otelsql` (`go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql`)
- Update `wire_infra.go::wireInfrastructure` to use the otel-wrapped DB
- Tests : 1 query creates 1 span

### Commit 4 — Redis + outbound instrumentation
- Redis : use `redisotel` from `github.com/redis/go-redis/extra/redisotel`
- Outbound HTTP : wrap http.Client with `otelhttp.NewTransport`
- Tests

### Commit 5 — Graceful shutdown polish
- `cmd/api/wire_serve.go::shutdown()` (or wherever) :
  - Step 1 (15s) : `srv.Shutdown(ctx)` — drain HTTP requests
  - Step 2 (10s) : `wsHub.GracefulShutdown(ctx)` — close WS connections cleanly with close-frame
  - Step 3 (5s) : `pendingEventsWorker.Stop(ctx)` + `notifWorkerCancel()` — drain workers
  - Total : 30s budget
- Add `wsHub.GracefulShutdown(ctx)` method if not exists (close all conns with `1001 Going Away`)
- Tests : simulate SIGTERM, asserter all 3 steps run within budget

## Hard constraints

- **Validation pipeline before EVERY commit**: `go build && go vet && go test ./... -count=1 -short -race`
- **Zero overhead when OTel disabled** : empty `OTEL_EXPORTER_OTLP_ENDPOINT` → no spans created (no-op exporter, no allocations)
- **No PII in spans** : same redaction policy as logs (no auth tokens, no email values, no full credit cards)

## OFF-LIMITS

- LiveKit / call code, workflow files, other plans
- Vendor-specific exporters (Jaeger / Datadog libs) — OTLP only
- Metrics dashboards / Grafana setup — out of scope

## Branch ownership

`fix/p11-otel-graceful-shutdown` only. Created from main.

## Final report (under 800 words)

PR URL first. Then:
1. OTel SDK wired (yes/no)
2. Spans coverage (HTTP yes, DB yes, Redis yes, outbound yes)
3. Graceful shutdown 3-step (yes)
4. Validation pipeline output
5. Env vars added (list)
6. "Branch ownership confirmed"
