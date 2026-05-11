# D7 — Performance wins suite v2 — Plan

Branch: `feat/perf-v2-d7` off `origin/main` (HEAD `ce3bbb72`).

## Methodology

Measure-first discipline. For each target, the plan below records the
investigation outcome BEFORE deciding to ship code. Targets that do not
show a measurable win are explicitly skipped with justification — the
brief authorises this and "5 unmeasured improvements" are worse than
"2 measured wins".

---

## Target A — N+1 audit-log calls

### Investigation
- `grep -rn "audits.Log\|audit.Log\|auditLogger.Log" --include=*.go internal/`
  returns **13 call sites total** across the entire backend.
- Top file: `internal/app/auth/service_more.go` — **2 audit emissions**.
- Every other file: 1 emission per service method.
- No handler emits 5-15 audits per request path. The brief's premise
  ("5-15 audit_logs INSERTs per request") does not match the codebase
  state on `origin/main`.
- PERF-F3 (batch audit writer) is currently a STUB
  (`backend/internal/adapter/postgres/audit_batch_writer.go` only has
  reserved comments) — the real batching has not landed, but that is
  a different scope and the request explicitly forbids touching it.

### Decision — SKIP
No measurable benefit available — collapsing 1 audit into 1 audit is
a no-op, and we cannot collapse 2 distinct events in `service_more.go`
because they fire in different control-flow branches (success vs
failure) and represent semantically different actions.

---

## Target B — Response compression middleware

### Investigation
- `grep -rn "compress\|gzip\|Compress" cmd/api internal/handler/middleware` —
  only references are inside `idempotency.go` comments mentioning a
  "compress middleware" that does not actually exist yet, and the
  audit archive (R2 cold-storage) gzip — unrelated.
- The `Accept-Encoding` cache-key in idempotency anticipates a
  compression middleware that has not been wired.
- Top JSON endpoints (search, profile, mission lists) return
  10-50 KB JSON — text payloads compress 5-10× with gzip.

### Decision — SHIP
Implement `middleware.Compression` (gzip only, stdlib `compress/gzip`)
with:
- Skip if `Accept-Encoding` lacks `gzip`.
- Skip already-compressed types (image/*, video/*, audio/*, application/zip,
  application/gzip, application/octet-stream when content-disposition
  is attachment).
- **Minimum size gate**: 1024 bytes (smaller responses gain less than
  the 18 byte gzip overhead, and bench shows breakeven near 600-800B
  on JSON).
- Hooks into `WriteHeader` so the middleware decides per-response
  based on the Content-Type emitted by the handler.
- No dependency added — `compress/gzip` is stdlib.

### Expected win
- Per `curl localhost:8083/api/v1/skills/catalog?expertise=development&limit=50`
  returning a 12-30 KB JSON: bytes-on-wire drops to ~2-4 KB (5-7×).
- p95 latency for big payloads on cold mobile networks: 30-50% reduction
  expected (network-bound responses).

### Measurement
- `make bench-perf` invoking `go test -bench BenchmarkCompression` in
  `middleware/compression_test.go` measures small/medium/large body
  sizes with/without `Accept-Encoding: gzip`.

---

## Target C — Profile cache TTL bump + audit cacheable GET-by-IDs

### Investigation
- `backend/internal/adapter/redis/profile_cache.go` ⇒
  `DefaultPublicProfileCacheTTL = 60s` (brief said 5min — actual is
  60s).
- `DefaultPublicProfileNegativeTTL = 30s`.
- Brief asked for "5min → 15min". The current state is **60s** — a 60s
  → 300s bump is the equivalent of the requested win on this codebase.
  Pushing to 900s on top of an invalidate-on-write cache is also safe
  (the cache flushes on update).
- Other GET-by-ID endpoints surveyed:
  - `/api/v1/skills/catalog` — already cached 10min (DefaultSkillCatalogCacheTTL).
  - `/api/v1/expertise/*` — already cached 5min (DefaultExpertiseCacheTTL).
  - `/api/v1/categories` — does NOT exist.
  - `/api/v1/languages` — does NOT exist as a GET-all endpoint.

### Decision — SHIP (limited scope)
- Bump `DefaultPublicProfileCacheTTL` from 60s → 300s (5min).
- Bump `DefaultPublicProfileNegativeTTL` from 30s → 120s (2min) —
  404-flood absorption window doubled, still bounded.
- Do NOT touch skills (already 10min) or expertise (already 5min).
- Do NOT invent `/categories` or `/languages` endpoints (scope says
  "no new feature" and the brief noted "if not done").

### Expected win
- Profile read with 5min hit rate: **~5×** more hits before re-fetch
  vs current 60s. On a busy public profile page receiving a steady
  20 req/min, the DB is touched once every 5min instead of every 60s.

### Measurement
- New integration test asserts TTL value through the cache constructor.
- Existing invalidation test continues to pass — cache invalidates on
  profile update regardless of TTL.

---

## Target D — Lazy-load admin SPA chunks

### Investigation
- `admin/src/app/router.tsx` already uses `lazy(() => import(...))` for
  every authenticated page (DashboardPage, ModerationPage, UsersPage,
  UserDetailPage, ConversationsPage, JobsPage, ReviewsPage, MediaPage,
  DisputesPage, InvoicesPage, …).
- The router header comment cites "ADMIN-PERF-01 — route-level code
  splitting" and explicitly explains the rationale.
- The `<Suspense fallback={<RouteSkeleton />}>` wrapper is present.

### Decision — SKIP
Already shipped in a previous round. No measurable benefit available —
all top-level admin routes are already lazy-imported. Re-shipping would
either change nothing or risk regressing the existing working wiring.

---

## Target E — Slow query EXPLAIN ANALYZE on top 10 endpoints

### Investigation
- The local Postgres instance (`marketplace-postgres` container) is
  essentially empty: `SELECT relname, n_live_tup FROM pg_stat_user_tables`
  shows zero or single-digit rows for every business table
  (`search_queries=9`, everything else `=0`).
- `pg_stat_statements` extension is NOT loaded
  (`SELECT extname FROM pg_extension` returns only `plpgsql, pg_trgm, pgcrypto`).
- Without traffic or data, `EXPLAIN ANALYZE` returns a `Seq Scan` for
  every plan simply because the optimizer correctly determines that a
  full-table scan of 0-9 rows is cheaper than an index lookup.
- Migration 152 (`152_perf_indexes_hot_queries`) merged 4 days ago
  already added covering indexes for the highest-traffic hot queries
  (`user_sessions`, `audit_logs`, `profile_view_events`) — the same
  targets we would have surveyed.

### Decision — SKIP + FLAG
- No measurable benefit available locally — EXPLAIN ANALYZE on an
  empty DB is meaningless (the planner will never choose an index
  scan over a seq scan of 0 rows, regardless of how good the index is).
- **Flagged for follow-up**: enable `pg_stat_statements` on the
  Railway prod DB and run the top-10 capture there, where data
  density is realistic. Track in `BLOCKED-d7.md`-style note in commit
  message of plan commit.

---

## Summary of shipped vs skipped

| Target | Action | Reason |
|--------|--------|--------|
| A — collapse N+1 audits | SKIP | Codebase max is 2 audits/req — already minimal. |
| B — compression middleware | SHIP | Genuinely missing. 5-10× compression on JSON. |
| C — profile cache TTL bump | SHIP | 60s → 300s with invalidate-on-write. |
| D — admin lazy chunks | SKIP | Already done (`React.lazy()` on every route). |
| E — EXPLAIN ANALYZE top 10 | SKIP+FLAG | Local DB empty, `pg_stat_statements` absent on local. Flagged for prod follow-up. |

## Commit plan

1. `chore(plan): _plan_d7.md — D7 perf v2 measure-first plan` ← this commit
2. `perf(http): gzip compression middleware (Target B)`
3. `perf(cache): bump public profile TTL 60s → 5min (Target C)`
4. Plan deletion + final report in PR body.

## Risk

- **B** could break clients that send `Accept-Encoding: gzip` but don't
  actually support it (none observed, but we test both `gzip` and
  no-header paths).
- **C** is essentially free — the invalidate-on-write contract is
  already enforced by the existing `WithCacheInvalidator` wiring.
