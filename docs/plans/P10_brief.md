# P10 — Slow queries observability + slowloris guard + mutation rate limit

**Phase:** F.2 HIGH #6
**Source audit:** PERF-FINAL-B-04 (slow query log) + PERF-FINAL-B-01 (slowloris) + SEC-FINAL (rate limit on mutations)
**Effort:** 1j est.
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p10-observability-slowloris-mutation-ratelimit`

## Goal

3 quick infra hardenings:
1. **Slow query log** — every DB query exceeding 50ms WARN, 500ms ERROR with structured fields
2. **Slowloris guard** — set `http.Server.ReadHeaderTimeout=5s` (currently unset → infinite, vulnerable)
3. **Mutation rate limit** — 30 req/min/user on POST/PUT/PATCH/DELETE via existing rate limiter middleware

## Decisions (LOCKED — user validated)

### Slow query log
- Threshold WARN @ 50ms, ERROR @ 500ms (CLAUDE.md spec)
- Wrap every `*sql.DB` query method via a small instrumentation layer
- Structured fields: `query` (sanitized — first 200 chars, no values), `duration_ms`, `caller` (file:line), `request_id`

### Slowloris guard
- `http.Server.ReadHeaderTimeout = 5 * time.Second` (in `wire_serve.go` or main.go where the server is built)
- Confirm `ReadTimeout = 15s` is already there (per current main.go)
- `IdleTimeout = 60s` already there

### Mutation rate limit
- Middleware that matches POST|PUT|PATCH|DELETE
- 30 req/min/user (per JWT user_id, fallback to IP if unauthenticated)
- Use existing `middleware/ratelimit.go` infrastructure
- Returns 429 with `Retry-After` header

## Plan (4 commits)

### Commit 1 — Slow query log
- New file `internal/adapter/postgres/slow_query.go` :
  - Wrap `*sql.DB` queries via a custom driver decorator OR via context-tracked timing in the existing helpers (less invasive)
  - Decision: use **context-tracked timing** — wrap `db.QueryContext`, `db.ExecContext`, `db.QueryRowContext` via small helpers in `internal/adapter/postgres/instrumented_db.go`
  - Update repos to use the instrumented wrapper (or wrap at the `WithTxRunner` layer for write paths)
- Tests asserting log emission on slow path

### Commit 2 — Slowloris guard
- `cmd/api/wire_serve.go::buildHTTPServer` (or wherever) :
  ```go
  srv := &http.Server{
    Addr:              ":" + cfg.Port,
    Handler:           r,
    ReadHeaderTimeout: 5 * time.Second,  // NEW — slowloris guard
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      0,
    IdleTimeout:       60 * time.Second,
  }
  ```
- Test: HTTP server config struct field assertion

### Commit 3 — Mutation rate limit
- New middleware `MutationRateLimit(rateLimiter, threshold=30 req/min)`
- Wire in router for all `POST|PUT|PATCH|DELETE` route groups
- Per-user (JWT) key, fallback IP
- Returns 429 + `Retry-After` header
- Tests: 31 mutations in <1min asserter 31st returns 429

### Commit 4 — Docs
- `backend/docs/observability.md` (or update existing) : threshold values, log format, how to query in production
- `backend/docs/security.md` : slowloris guard rationale + mutation rate limit policy

## Hard constraints

- **Validation pipeline before EVERY commit**: `go build && go vet && go test ./... -count=1 -short -race`
- **Zero behaviour change** on the hot path : the slow query wrapper must add < 1µs overhead per query (just a Now()+sub)
- **Slowloris guard MUST not break legitimate slow uploads** : ReadHeaderTimeout only covers HEADERS, not body. Verify with a test uploading a large file — should not 408.

## OFF-LIMITS

- LiveKit / call code, workflow files, other plans
- Major restructuring of the DB driver — only wrap, no replace

## Branch ownership

`fix/p10-observability-slowloris-mutation-ratelimit` only. Created from main.

## Final report (under 600 words)

PR URL first. Then:
1. Slow query log: number of repos instrumented
2. Slowloris guard set : ReadHeaderTimeout=5s confirmed
3. Mutation rate limit active : route count covered
4. Validation pipeline output
5. Branch ownership confirmed
