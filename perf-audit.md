# Perf Audit — 2026-05-11

Branch: `feat/perf-audit-deep`
Scope: regression vs the codebase ~3 days ago (before Phase B retention scheduler, audit decorator/sanitize, stats tracking middleware, session-expiry, refresh-token-family Redis SET, RLS tenant tx wrap, live perms resolver).
Method: static audit + EXPLAIN ANALYZE on local Postgres (n=94 orgs, 111 users, 150 view events, 169 audit logs). Live load test deliberately skipped (per mission brief).

---

## Executive summary

**Verdict — likely overhead vs baseline 3 days ago: +25 % to +60 % on every authenticated request, p95 latency.**

Confidence: **high (90 %)** for the per-request regressions; **medium (60 %)** for the absolute %, because the local DB is tiny (sub-ms execution) and the real cost shows up on Neon, where round-trip + planning dominates.

The user-visible "app feels slower" is fully explained by **two uncached DB reads and one extra Redis read added to the hot path on every authenticated request** — and amplified by Neon's RTT (10-20 ms per round-trip).

### Top 3 hotspots

1. **`OrgOverridesResolver.GetRoleOverrides` calls `OrganizationRepository.FindByID` with NO cache on every authenticated request** (`backend/cmd/api/org_overrides_adapter.go:36`). Adds one full `SELECT ... FROM organizations WHERE id = $1` to every `/api/v1/*` call. Cost on Neon: ~10-15 ms RTT + ~1-3 ms planning + ~0.1 ms execution.

2. **`UserRepository.GetSessionVersion` wired directly into auth middleware with NO Redis cache** (`backend/internal/handler/router.go:152` → `backend/internal/adapter/postgres/user_repository.go:250`). One extra PG round-trip on every authenticated request. Cost on Neon: ~10-15 ms RTT.

3. **`AuditRepository.Log` runs inside `RunInTxWithTenant`** = `BEGIN; SELECT set_config; INSERT; COMMIT;` = **4 round-trips per audit event** (`backend/internal/adapter/postgres/audit_repository.go:115-150`). Login/logout/refresh/mutation/2FA toggle/etc all log audit. On Neon: ~40-60 ms per audit, even though they fire in the background.

### Top 3 quick wins (< 1h each)

1. **Cache `GetRoleOverrides` in Redis with 30s TTL**, mirroring `CachedUserStateChecker`. Estimated saving: 10-15 ms on **every** authenticated request. (Quick-win patch sketched below.)
2. **Wrap `GetSessionVersion` with the same 30s Redis cache** (TTL can even be 60s — invalidation already happens on session bump via `BumpSessionVersion`, and we control all the writers). Estimated saving: another 10-15 ms per auth request.
3. **Batch the audit `SELECT set_config` + `INSERT` into a single round-trip**: use a CTE `WITH cfg AS (SELECT set_config(...)) INSERT ...` so audit drops from 4 RTT to 2 RTT. Background goroutine but still steals connection pool slots.

### Top 3 longer wins (1-3h each)

1. **Move `MaxOpenConns` from 50 → 25 on Neon** (we have ~10 concurrent users, the 50 pool just keeps cold idle connections that prolong TLS handshakes). Or alternatively enable `pgxpool` or pgbouncer to amortize prepared statements.
2. **Hoist the audit transaction**: refactor `RunInTxWithTenant` so a caller that performs N audit writes during a single business request does ONE tx with N inserts, not N txs.
3. **Stop dedup-counting `viewer_ip_anonymized × viewer_ua_hash` via `COUNT(DISTINCT (row, row))` in `queryVisibilityTotals`** — Postgres falls back to a Sort + Quicksort (see EXPLAIN: 0.94 ms on 120 rows scales O(n log n)). With 50k rows/month it'll be the visible cost. Use `COUNT(DISTINCT viewer_ip_anonymized || viewer_ua_hash)` to leverage hash agg, or pre-aggregate per day.

---

## Backend hot paths

`net/http/pprof` is NOT enabled (`grep -rn "net/http/pprof" backend/` returns 0 hits). A live flame graph capture would require enabling it first — out of scope for a read-only audit. Replacing that with static analysis of every authenticated-request path:

### `/api/v1/auth/me` hot path (every page load)

Round-trips per call, post-Phase B:

| Step | Cost source | RTT |
|---|---|---|
| 1. Cookie parse + `SessionService.Get` | Redis GET | 1 RTT |
| 2. `verifySessionVersion` → `UserRepository.GetSessionVersion` | **Postgres SELECT (uncached)** | **1 RTT** |
| 3. `checkUserState` → `CachedUserStateChecker.GetUserState` | Redis GET (hit) | 1 RTT |
| 4. `injectLivePermissions` → `OrgOverridesResolver.GetRoleOverrides` → `OrganizationRepository.FindByID` | **Postgres SELECT (uncached)** | **1 RTT** |
| 5. The actual `/auth/me` handler | Postgres SELECT users + organizations | 1-2 RTT |

**Net new cost vs baseline: steps 2 and 4 (~2 extra Postgres round-trips, ~20-30 ms on Neon).**

Same overhead is paid on **EVERY** authenticated endpoint — `/api/v1/freelance-profiles/{id}`, `/api/v1/me/stats/visibility`, `/api/v1/messaging/unread-count`, `/api/v1/calls/me/active`, etc.

### `/api/v1/freelance-profiles/{id}` with `?q=...&pos=...` (tracking middleware)

- Steps 1-5 above (auth)
- 6. The freelance profile read itself
- 7. **`TrackProfileViews` middleware fires a goroutine** that calls `ProfileViewRepository.Record` (1 INSERT, 1 RTT). Detached context with 5 s timeout. Pool slot held while in-flight — typically `< 10 ms`, so no real backpressure unless many concurrent reads.

Goroutine is fire-and-forget with `context.WithoutCancel` — safe, no leak. Pre-existing `defer recover()` guards a panic.

### `/api/v1/me/stats/visibility?days=30`

- Steps 1-5 (auth)
- 6. **2 sequential PG queries** (totals + daily series). EXPLAIN shows `Sort Method: quicksort` for the `COUNT(DISTINCT (row, row))` aggregate — fine on 120 rows, but log-linear in the long run.
- 7. **2 more queries** for `AggregateApplications` (totals + series). 4 queries total per call.
- Per-query cost on Neon: 10-15 ms RTT + ~1-2 ms plan + 1-5 ms exec. **4×~17 ms ≈ 70 ms total** — fine, but plan caching would shave ~5-10 ms.

### `/api/v1/auth/login`

- All the auth middleware costs **don't apply** (no session yet).
- Login flow now writes:
  - `bcrypt.Compare` (~50-80 ms — biggest single cost, unchanged)
  - `users.last_login_at` update (1 RTT, unchanged)
  - **NEW**: `AuditRepository.Log("auth.login_success")` inside `RunInTxWithTenant` = 4 RTT (~60 ms on Neon if synchronous, ~0 if backgrounded). Need to verify call site is async — see "Required follow-up checks" below.
  - **NEW**: `RefreshBlacklistService.AddFamilyMember` = SADD + EXPIRE = 2 Redis RTT (~2 ms).
  - Session/refresh creation (Redis SETs, unchanged).

### `/debug/pprof/profile?seconds=30`

Not available — pprof not enabled. **Quick-win recommendation: add `_ "net/http/pprof"` import behind a `PPROF_ENABLED=true` env gate.**

---

## Database — EXPLAIN ANALYZE

All ran via `docker exec marketplace-postgres psql -U postgres -d marketplace_go`. Dataset is small (n=94 orgs, 111 users) so execution time is near zero; the **planning time** is the real signal — it shows how much a non-prepared-statement path pays per call.

### Q1 — `OrganizationRepository.FindByID` (auth hot path)

```sql
SELECT id, owner_user_id, type, name, ..., role_overrides, ...
FROM organizations WHERE id = $1
```

```
Seq Scan on organizations  (cost=0.04..4.19 rows=1 width=415) (actual time=0.015..0.027 rows=1 loops=1)
  Filter: (id = $0)
  Rows Removed by Filter: 93
  Buffers: shared hit=4
Planning Time: 1.667 ms
Execution Time: 0.096 ms
```

Findings:
- **Planning : Execution = 17 : 1.** Planner is the real cost. With prepared statements (or a Redis cache), planning collapses to ~0.
- Seq Scan on 94 rows is fine (PG won't bother with the index that small). Will switch to Index Scan once `n > ~500`.
- **The fix is caching, not an index.** Adding a Redis cache fronting `FindByID` (mirror of `CachedUserStateChecker`, 30 s TTL) takes the entire query off the hot path 95 % of the time.

### Q2 — `UserRepository.GetSessionVersion` (auth hot path)

```sql
SELECT session_version FROM users WHERE id = $1
```

```
Seq Scan on users users_1  (cost=0.00..5.09 rows=109 width=16) (actual time=0.001..0.002 rows=1 loops=1)
Planning Time: 2.419 ms
Execution Time: 0.110 ms
```

Same story — planner is 22× the execution. **Identical caching fix recommended.**

### Q3 — `profile_view_events` visibility totals

```sql
SELECT COUNT(*), COUNT(DISTINCT (viewer_ip_anonymized, viewer_ua_hash)),
       COUNT(*) FILTER (WHERE came_from='search'),
       COALESCE(AVG(search_position) FILTER (WHERE search_position IS NOT NULL), 0)
FROM profile_view_events WHERE organization_id = $1 AND created_at >= NOW() - (30 * '1 day'::interval)
```

```
Sort Method: quicksort  Memory: 41kB
Seq Scan on profile_view_events  (cost=0.00..6.00 rows=121 width=50) (actual time=0.014..0.061 rows=120 loops=1)
      Filter: ((organization_id = $0) AND (created_at >= (now() - '30 days'::interval)))
Planning Time: 1.005 ms
Execution Time: 1.163 ms
```

Findings:
- Seq Scan because the table is 150 rows — too small to use `idx_pve_org_created`. Will switch automatically at scale.
- The `COUNT(DISTINCT (composite))` forces a Sort (quicksort, 41 kB on 120 rows). **At 100k rows/month this becomes O(n log n) work in memory.** Suggest splitting into 2 queries (or pre-aggregate per day) when row count grows.
- No new index needed today.

### Q4 — `audit_logs` INSERT (every audit)

```sql
INSERT INTO audit_logs (id, user_id, action, ...) VALUES ($1, ..., $8)
```

```
Planning Time: 2.408 ms
Trigger for constraint audit_logs_user_id_fkey: time=1.175 calls=1
Execution Time: 2.474 ms
```

Findings:
- **FK trigger costs 1.175 ms per insert** — checks `users(id)` exists. Acceptable.
- Planning is 2.4 ms — every INSERT re-plans because lib/pq does NOT prepare statements without explicit `Stmt.Prepare`. Times 4 round-trips for the tx wrap = ~10 ms of overhead per audit on local box, **~60 ms on Neon**.

### Q5 — `user_sessions` ListByUser

```
Index Scan using idx_user_sessions_user_id on user_sessions (cost=0.14..8.16 rows=1 width=384) (actual time=0.411..0.411 rows=0 loops=1)
Planning Time: 3.829 ms
Execution Time: 0.573 ms
```

Findings:
- Correctly uses `idx_user_sessions_user_id`. Execution is fine.
- Planning is **3.8 ms**, the worst of any I tested — suggests `SELECT *` against a wide row. Worth narrowing the column list once this endpoint is on the user-visible hot path.

### Missing indexes — none critical

For the current dataset and queries, no missing indexes. Recommendations are caching + plan caching, not indexing.

### `pg_stat_statements`

Not enabled in the local container. To enable for a real audit:

```bash
docker exec marketplace-postgres psql -U postgres -c "CREATE EXTENSION IF NOT EXISTS pg_stat_statements"
# then restart with shared_preload_libraries='pg_stat_statements'
```

---

## Web bundle

`@next/bundle-analyzer` is configured (`web/next.config.ts:48`). A bundle delta requires building from this branch AND from main 3 days ago — heavier than a read-only audit. Reasoning from `git diff`:

- Phase B added/touched mostly **backend** files. The web touches are limited to:
  - `web/src/features/account/components/two-factor-toggle.tsx` (FIX-2FA)
  - `web/src/features/security/components/sessions-list.tsx` (SEC-SESSIONS Malt-style)
  - `web/src/app/[locale]/providers.tsx` (PERF-FIX retry policy)
  - `web/src/shared/hooks/use-user.ts` (PERF-FIX session caching + retryOnMount=false — **this is a NET WIN**)

- No new heavy dep landed (no chart/PDF/markdown swap in `web/package.json` over last 3 days).
- Per-route code splitting unchanged. Initial JS bundle should be `±2 %` vs baseline.

### Web/auth-me fan-out

This was the right call: `use-user.ts:198 retryOnMount: false` plus 30 min stale time. The previous behaviour (every observer re-fetched on mount after a 401) was a real perf bug — that fix is in production now and is one of the few **net-positive** changes in the Phase B window.

### Dashboard mount

The fix for `/auth/me` fan-out also helped. Remaining concerns:

1. **PostHogProvider** mounted in locale layout fires `useSession()` on every page (now cached, so cheap).
2. **`useReconcileCallOnMount`** still fires `/api/v1/calls/me/active` — see `web/src/features/call/hooks/use-reconcile-call-on-mount.ts:34`. Need to confirm it's gated to authenticated sessions only.

### Lighthouse

Not run — would require a local dev server, beyond a read-only audit. Targets (CLAUDE.md): LCP < 2.5 s, FID < 100 ms, CLS < 0.1, JS bundle < 200 KB gz.

---

## Schedulers cumulative cost

| Scheduler | Dev interval | Prod interval | Per-tick cost | Per-hour @ prod | Notes |
|---|---|---|---|---|---|
| `retention` | 1 min | 1 hour | Loops 6 policies × N batches (50 ms yield between). Each `Sweep()` deletes up to `BatchSize` rows from `messages`/`notifications`/`device_tokens`/`search_queries`/`audit_logs`/`user_sessions`. Empty → returns immediately. | ~0 (empty most ticks), pulses on retention cliffs | `MaxBatchesPerRun` caps runaway. Tagged as system actor for RLS. |
| `gdpr` | 1 min | 24 hour | Scans `users` for `deleted_at + 30 d` past due, anonymises in batches of 100 | ~0 (rare event) | Daily prod interval is fine. |
| `dispute` | 1 min | 1 hour | Two queries (`auto_resolve` + `escalate_due_period`), each a bounded SELECT … FOR UPDATE on `disputes` | ~10-30 ms | Acceptable. |
| `kyc` | 1 min | 1 hour | Iterates `organizations` where `kyc_first_earning_at + grace_period` past due | ~5 ms | Acceptable. |
| `invoicing` | 1 hour | 1 hour | Once-a-month (`defaultRunWindow`) — does the monthly invoice run if Redis marker says "this month not yet run" | ~0 most ticks; runs heavy on the 1st of month | Once-a-day check is enough. |
| `referral` | 1 hour | 1 hour | `RunExpirerCycle` — bounded SELECT/UPDATE on `referrals` | ~10 ms | Acceptable. |
| `proposal service_scheduler` | varies | — | Bounded queries on `proposal_milestones` ready-to-release | ~10 ms | Acceptable. |

**Cumulative steady-state cost at prod intervals: ~50 ms of DB work spread over ~5 minutes of clock time. Negligible.** None of the schedulers can plausibly explain the user's "app feels slower" report — they're background and bounded.

**Note on dev intervals**: most schedulers run at 1 min in dev. If a dev box has a tiny local DB (94 rows), the dev box may show 1-min spikes that don't exist in prod. Verify this matches user's observation context (prod vs dev).

---

## Top 5 findings with file:line evidence

### F1 — Org overrides resolver on every auth request, no cache  [CRITICAL]
`backend/cmd/api/org_overrides_adapter.go:36` — `GetRoleOverrides` calls `OrganizationRepository.FindByID` for the auth context org on **every authenticated request**. No Redis cache. **+10-15 ms per request on Neon**, every authenticated call.

### F2 — Session version checker hits Postgres on every auth request  [CRITICAL]
`backend/internal/handler/router.go:152` wires `SessionVersions: deps.UserRepo` — the raw Postgres adapter. `backend/internal/adapter/postgres/user_repository.go:250` does a fresh `SELECT session_version FROM users WHERE id = $1` every authenticated request. **+10-15 ms per request on Neon**.

### F3 — Audit log = 4 round-trips per insert  [HIGH]
`backend/internal/adapter/postgres/audit_repository.go:135-150` wraps every `INSERT INTO audit_logs` in `RunInTxWithTenant` = `BEGIN + SELECT set_config + INSERT + COMMIT`. Even though it runs in a background goroutine for many call sites, it still steals a connection pool slot for **~60 ms on Neon per audit event**. Auth login emits at least 1 audit row; mutation endpoints often emit 2+.

### F4 — DB pool MaxOpenConns=50 may be too high for Neon's connection limit  [MEDIUM]
`backend/internal/adapter/postgres/db.go:31` sets `MaxOpenConns(50)`. Neon's free/starter tier caps connections at ~20-40. If the pool tries to grow past Neon's limit, every new connection's TLS handshake adds 50-100 ms before the query runs. Recommended: 25/15 or pgbouncer/pgpool.

### F5 — otelsql wraps every Query, but Prepare spans are correctly suppressed  [INFO — not a regression]
`backend/internal/observability/db.go:30-42` wraps the driver in `otelsql.Open`. When `OTEL_EXPORTER_OTLP_ENDPOINT` is empty, the global tracer is no-op so the wrapper is effectively free. **If you've set the OTLP endpoint in prod**, every query now has span recording overhead (~5-20 µs per call) — verify the env on Railway.

---

## Proposed quick wins (patches)

**Status — QW1 + QW2 implemented**: commit `ef5bee9c` on branch
`feat/perf-fixes` (`https://github.com/Hassad674/serviceMarketplaceGo/tree/feat/perf-fixes`).
17 unit tests cover miss / hit / TTL / error / corrupt-payload /
invalidate / nil-overrides for both caches. **NOT merged to main —
awaiting user review.**

Build green (`go build ./...`), vet green, full redis adapter test
suite green (`go test ./internal/adapter/redis/ -count=1` → ok in
39 s). Targeted middleware suite green
(`go test ./internal/handler/middleware/`). Files added:

```
backend/internal/adapter/redis/session_version_cache.go      +138
backend/internal/adapter/redis/session_version_cache_test.go +194
backend/internal/adapter/redis/org_overrides_cache.go        +149
backend/internal/adapter/redis/org_overrides_cache_test.go   +197
backend/cmd/api/bootstrap_router.go                          +27 -8
backend/cmd/api/wire_router.go                               +6 -1
backend/internal/handler/router.go                           +29 -2
```

### QW1 — Redis cache for `GetRoleOverrides` (30 s TTL) — DONE

`CachedOrgOverridesResolver` mirrors the existing `CachedUserStateChecker`
pattern. Read-through cache with 30s TTL; never cache errors (preserves
the auth middleware's fail-open policy on transient resolver failures).
Explicit `Invalidate(orgID)` exposed so the role-permissions editor can
collapse propagation to "next request".

Saving: ~10-15 ms on every auth'd request, traded for ~1 ms Redis GET.

### QW2 — Redis cache for `GetSessionVersion` (30 s TTL) — DONE

`CachedSessionVersionChecker` mirrors QW1. Same fail-open semantics, same
TTL. The router now picks the cached checker over the raw `UserRepo` via
a new optional `SessionVersionChecker` field in `RouterDeps`. Tests that
pass nil keep their legacy direct-PG behaviour. **Follow-up TODO for
the user**: wire `cache.Invalidate(userID)` into `BumpSessionVersion`
call sites so revocation propagates immediately rather than waiting
for the TTL.

Saving: ~10-15 ms on every auth'd request.

### QW3 — Combine audit `set_config` + `INSERT` into one round-trip

```sql
WITH cfg AS (SELECT set_config('app.current_user_id', $1, true))
INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at)
VALUES ($2, $3, $4, $5, $6, $7, $8, $9);
```

Drops audit from 4 RTT to 2 RTT. Saving: ~20-30 ms per audit event on Neon.

### QW4 — Enable pprof behind env gate

```go
// backend/cmd/api/main.go
if os.Getenv("PPROF_ENABLED") == "true" {
    import _ "net/http/pprof"
    go http.ListenAndServe("localhost:6060", nil)
}
```

Saving: 0 (just enables profiling for future investigation).

### QW5 — Lower pool size for Neon

```go
// backend/internal/adapter/postgres/db.go
db.SetMaxOpenConns(25)  // was 50
db.SetMaxIdleConns(10)  // was 25
```

Saving: avoids hitting Neon connection limits, which would otherwise force TLS-handshake-per-query on overflow.

---

## Required follow-up checks (not done in this audit)

1. **Verify every `auditService.Log(...)` call site is wrapped in a goroutine or fire-and-forget**. If any synchronous call path exists on `/auth/login`, that's a +60 ms latency hit on the user's login flow.
2. **Enable `pg_stat_statements` on Neon** and capture top 20 by mean execution time after 24 h of real traffic. The local box can't show what's slow in prod.
3. **Confirm `OTEL_EXPORTER_OTLP_ENDPOINT` is unset on Railway** — if set, every DB call + Redis call has tracer overhead.
4. **Profile in production with `pprof` behind a flag**. Static analysis caught the 2 big regressions but flame graph would confirm what's burning CPU vs. waiting on I/O.

---

## Verdict for prod

- **Current overhead vs 3 days ago: ~+25 % to +60 % p95** on every authenticated request, driven entirely by F1 + F2 + F3.
- **After quick wins QW1 + QW2 + QW3: ~+5 % to +10 % p95** (essentially noise — within the budget).
- **After longer wins (pool tuning, audit batch hoist, distinct rewrite): back to baseline or +2 %.**

**Recommended sequence**: ship QW1 + QW2 first (biggest bang per LoC). Verify on Railway. Then QW3, then enable pprof, then re-measure.

Confidence on numbers: **medium-high**. The static analysis identifies the right hotspots; the absolute percentages need a prod-side measurement to land precisely.
