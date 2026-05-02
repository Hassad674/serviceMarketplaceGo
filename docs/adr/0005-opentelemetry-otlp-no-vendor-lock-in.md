# 0005. OpenTelemetry OTLP exporter, no vendor lock-in

Date: 2026-04-30

## Status

Accepted

## Context

The marketplace will eventually need distributed tracing across
HTTP, Postgres queries, Redis calls, and outbound HTTP to Stripe
/ LiveKit / OpenAI / R2 / Resend / Typesense. The earliest
diagnostic value is an end-to-end span tree per request showing
latency hotspots — which is then ingested by an APM / trace UI
(Jaeger, Honeycomb, Datadog, Grafana Tempo, Lightstep).

The choice of trace UI is operational, not architectural. A
hosted vendor (Datadog) gives a polished UI in 5 minutes; a
self-hosted backend (Tempo + Grafana) is cheaper at scale and
data-residency friendly. Either should work without changing the
application code.

OpenTelemetry is the de-facto industry standard for vendor-neutral
instrumentation. The trace data is shaped as
[OTLP](https://opentelemetry.io/docs/specs/otlp/), and every major
APM vendor accepts OTLP either directly or via the
[OpenTelemetry Collector](https://opentelemetry.io/docs/collector/).

We have three constraints:

1. We do not want to lock the codebase to one APM vendor's SDK
   (Datadog APM, NewRelic APM all require their proprietary
   library).
2. Local dev should not require running a tracing backend just
   to compile the project.
3. The instrumentation must cover inbound HTTP, outbound HTTP,
   `database/sql`, and Redis — the four hot paths where latency
   hides.

## Decision

We will use the **OpenTelemetry Go SDK with the OTLP gRPC
exporter** as the only instrumentation layer. The trace backend
(Jaeger, Honeycomb, Tempo, vendor of choice) is a deployment-time
choice, not a code change.

Concrete configuration:

1. `internal/observability/otel.go` initializes the SDK on boot.
   It reads `OTEL_EXPORTER_OTLP_ENDPOINT` from the environment.
   If unset, the SDK falls back to a **no-op tracer** so local
   dev runs without a collector. P11 #1.
2. Inbound HTTP is wrapped with `otelhttp.NewHandler` at the
   router level. Every request gets a span with HTTP method,
   path template, status, and duration. P11 #2.
3. `database/sql` is wrapped with
   `otelsql.WrapDriver(...)` so every query becomes a child span
   with SQL text (sanitized) and rows-affected. P11 #3.
4. Outbound HTTP clients (Stripe, R2, Typesense, OpenAI,
   Resend) use `otelhttp.NewTransport(...)` so external calls are
   nested in the inbound span. P11 #4.
5. Redis (`go-redis/v9`) is wrapped with the official
   `otelredis` instrumentation. P11 #4.
6. Graceful shutdown flushes the exporter so the last batch of
   spans does not get lost on container stop. P11 #5.

The exporter URL is `OTEL_EXPORTER_OTLP_ENDPOINT`. Production
deployments set it to the OTel Collector's gRPC endpoint
(`opentelemetry-collector:4317`). The Collector then fans out to
whatever backend the operator chose — the application binary is
unaware.

## Consequences

### Positive

- Switching vendors is a Helm values change. Adding Datadog APM
  is one Collector exporter line, no code change.
- Local dev runs with `OTEL_EXPORTER_OTLP_ENDPOINT` unset —
  zero overhead, zero dependency on a running collector.
- The four hot paths (HTTP in, HTTP out, SQL, Redis) have
  consistent span naming so dashboards transfer between
  backends without rewrite.
- Trace context propagation flows naturally to outbound calls:
  the Stripe SDK sees a `traceparent` header so a hosted
  Stripe-side trace tool (if Stripe ever exposes one) could
  correlate.

### Negative

- The OTel SDK adds ~3 MB to the binary size. Tolerable.
- Spans cost CPU on the hot path — measured ~80 ns per span
  on our Go 1.25 build. Negligible.
- The `otelsql` wrapper adds one driver layer; a tiny stack
  trace overhead. We keep query-text sanitization on (no PII
  leaks into trace metadata).
- The `OTEL_*` env vars are now part of the project's contract.
  We document them in `backend/.env.example`.

## Alternatives considered

- **Datadog APM SDK directly** — best UI for the price, but
  locks the codebase to a single vendor. Rejected.
- **Custom request-id + structured logging only (no traces)** —
  what we had before P11. Sufficient for "is this request
  slow?" but blind to the 5-hop fanout under load
  (`request → db → redis → outbound → db`). Rejected.
- **OpenCensus** — predecessor of OTel, deprecated since 2023.
  Rejected.
- **Sentry tracing** — viable but Sentry's primary value is
  error monitoring, not full APM. The trace UI is thin. We
  may layer Sentry for error tracking in the future regardless
  of OTel.

## References

- `backend/internal/observability/otel.go` — SDK boot and
  exporter wiring.
- `backend/.env.example` — `OTEL_EXPORTER_OTLP_ENDPOINT` and
  related env vars.
- P11 commit chain in `git log --grep="P11"`:
  `db31a052` (SDK + no-op fallback),
  `918f43ae` (inbound HTTP),
  `9d19eb55` (database/sql),
  `01c1a7ea` (Redis + outbound HTTP),
  `7c89cb70` (graceful shutdown drain).
- OpenTelemetry, *OTLP Specification*,
  <https://opentelemetry.io/docs/specs/otlp/>.
- OpenTelemetry, *Collector Architecture*,
  <https://opentelemetry.io/docs/collector/>.
