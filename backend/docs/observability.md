# Observability

This document captures the observability surface of the marketplace
backend: structured logging, slow-query detection, request tracing,
and the health / metrics scrape endpoints. Every entry here is
implementation-anchored — change the code and update the doc in the
same commit.

## Slow query log (PERF-FINAL-B-04 / P10)

Every database call routed through the
`marketplace-backend/internal/adapter/postgres` helpers
(`Query`, `QueryRow`, `Exec` in `slow_query.go`) is timed and a
structured slog line is emitted when the elapsed wall-clock
duration crosses one of two thresholds:

| Threshold variable          | Value | slog level |
| --------------------------- | ----- | ---------- |
| `SlowQueryWarnThreshold`    | 50ms  | `WARN`     |
| `SlowQueryErrorThreshold`   | 500ms | `ERROR`    |

These match the CLAUDE.md performance spec for backend p95 latency
(< 100ms CRUD) — a query that takes longer than half the budget is
worth a log line. The `ERROR` floor is the "this WILL show up in
the request's p95" line that should trigger an alert.

### Log fields

```json
{
  "time": "2026-05-01T10:30:00Z",
  "level": "WARN",
  "msg": "slow query",
  "op": "Query",
  "duration_ms": 67,
  "query": "SELECT id, email FROM users WHERE id = $1",
  "caller": "user_repository.go:91",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field         | Description |
| ------------- | ----------- |
| `op`          | One of `Query`, `QueryRow`, `Exec` |
| `duration_ms` | Wall-clock milliseconds (rounded to nearest ms) |
| `query`       | Sanitised SQL — whitespace collapsed, truncated to 200 bytes. **Parameter values are NEVER logged** |
| `caller`      | File:line of the repo method that initiated the call |
| `request_id`  | UUID stamped by the `RequestID` middleware. Empty for non-HTTP callers (background jobs, CLI tools) |
| `err`         | Present only when the underlying call failed |

### Hot-path overhead

Below the WARN floor the helper is a single `time.Since` comparison
with no allocations — verified by `TestLogSlowQuery_FastPath_NoAllocs`
and `BenchmarkQuery_FastPath`. The brief budget is < 1µs per query.

### Adoption

The helpers are opt-in per repo to keep the migration incremental.
Currently instrumented (P10):

- `internal/adapter/postgres/user_repository.go` (5 hot paths)
- `internal/adapter/postgres/social_link_repository.go`
- `internal/adapter/postgres/webhook_idempotency.go`

Adopting in a new repo is a one-line per call site swap:

```go
// Before
rows, err := r.db.QueryContext(ctx, query, args...)

// After
rows, err := postgres.Query(ctx, r.db, query, args...)
```

### Production query

Look for slow queries by request:

```
{"msg":"slow query","request_id":"<uuid>"}
```

Or by caller:

```
{"msg":"slow query","caller":"user_repository.go:91"}
```

## HTTP request log (existing)

Every request gets a `request_id` (UUID v4) stamped by the
`RequestID` middleware and propagated in the request context. The
`Logger` middleware emits one `INFO` line per completed request:

```json
{
  "time": "2026-05-01T10:30:00Z",
  "level": "INFO",
  "msg": "request",
  "method": "POST",
  "path": "/api/v1/auth/register",
  "status": 201,
  "duration_ms": 42,
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "remote_addr": "203.0.113.5:12345"
}
```

Cross-correlation: every slow query log line that fires inside an
HTTP request shares the same `request_id`. Grep on the id to see
the request and all DB calls it triggered.

## Health endpoints

| Endpoint    | Purpose          | Response |
| ----------- | ---------------- | -------- |
| `GET /health` | Liveness probe | `200 OK` with `{"status":"ok"}` |
| `GET /ready`  | Readiness probe | `200 OK` if DB + Redis connected, `503` otherwise |

## Metrics

`GET /metrics` exposes a Prometheus-format scrape endpoint when
the metrics handler is wired. Public by design (no credentials) so
a Grafana Agent or Prometheus scraper can target it without
secrets — bind the backend port to an internal-only network in
production OR front the path with a reverse-proxy ACL.
