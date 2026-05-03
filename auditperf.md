# Audit de Performance — Final Deep Audit V2

**Date** : 2026-05-03 (final-deep-audit-v2 post F.1 + F.2 + F.3.1 + F.3.3)
**Branch** : `chore/final-deep-audit-v2`
**Périmètre** : backend Go (~674 fichiers prod, 132 migrations) + DB Postgres + web Next.js + admin Vite + mobile Flutter

---

## Snapshot — état actuel

| Layer | CRITICAL | HIGH | MEDIUM | LOW | Total |
|---|---|---|---|---|---|
| Backend + DB | 0 | 1 | 8 | 6 | 15 |
| Web + Admin | 0 | 2 | 5 | 5 | 12 |
| Mobile | 0 | 3 | 6 | 5 | 14 |
| **Total** | **0** | **6** | **19** | **16** | **41** |

**Δ vs 2026-05-01** : -2 (PERF-FINAL-W-02 raw `<img>` reduced to 3 prod sites; PERF-FINAL-M-04 Color hex regression closed by F.3.3 — count went 491 → 573 → 124).

---

# BACKEND + DB

## HIGH (1)

### PERF-FINAL-B-02 : `payment_records.ListByOrganization` sans LIMIT ni cursor
- **Severity**: HIGH
- **Location**: `backend/internal/adapter/postgres/payment_record_repository.go` — `WHERE pr.organization_id = $1 OR pr.provider_organization_id = $1 ORDER BY pr.created_at DESC` SANS LIMIT.
- **Why**: 12-24 mois d'activité agence → 10k-50k lignes par requête wallet.
- **Fix**: cursor pagination `(created_at, id)` via `pkg/cursor/Encode`.
- **Effort**: S (1-2h)

## MEDIUM (8)

### PERF-FINAL-B-03 : Generic `service.CacheService` interface absent
- **Severity**: MEDIUM
- **Location**: `backend/internal/port/service/` — no generic interface. 5 specialised adapters exist (profile, expertise, freelance_profile, skill_catalog, subscription).
- **Fix**: extract generic port + Redis adapter, migrate the 5 specialised caches.
- **Effort**: M (½j)

### PERF-FINAL-B-05 : External HTTP clients use `http.DefaultTransport`
- **Severity**: MEDIUM
- **Location**: `backend/internal/search/embeddings.go`, `client.go`, `adapter/openai/client.go`, `anthropic/analyzer.go`, `vies/client.go`, `nominatim/client.go`
- **Fix**: `pkg/httpx/NewTunedClient(timeout time.Duration) *http.Client` with MaxIdleConnsPerHost: 50, KeepAlive: 30s, ForceAttemptHTTP2.
- **Effort**: XS (30 min)

### PERF-FINAL-B-06 : Stripe Connect `account.GetByID` jamais caché + ctx ignoré
- **Severity**: MEDIUM
- **Location**: `backend/internal/adapter/stripe/account.go` — 7 sites use `account.GetByID(accountID, nil)`.
- **Fix**: redis cache 60-120s TTL + `&stripe.AccountParams{Params: stripe.Params{Context: ctx}}`.
- **Effort**: M (½j)

### PERF-FINAL-B-08 : Notification worker — `getPrefs` + `users.GetByID` à chaque job
- **Severity**: MEDIUM
- **Location**: `backend/internal/app/notification/worker.go:99, 193, 211-222`
- **Fix**: LRU local cache + singleflight.
- **Effort**: S (1-2h)

### PERF-FINAL-B-09 : Search worker tick = 30s
- **Severity**: MEDIUM
- **Location**: `backend/internal/adapter/worker/worker.go:86-87`
- **Fix**: tick 5s + Redis pubsub `wake_search_worker`.
- **Effort**: S (1-2h)

### PERF-FINAL-B-10 : LTR capture INSERT par recherche
- **Severity**: MEDIUM
- **Fix**: batched flush via channel + 1×/sec multi-row INSERT.
- **Effort**: S (1-2h)

### PERF-FINAL-B-11 : Indexer fan-out 9 goroutines × N actors sans cap
- **Severity**: MEDIUM
- **Location**: `backend/internal/search/indexer.go:314-364`
- **Fix**: `errgroup.SetLimit(3)`.
- **Effort**: XS (30 min)

### PERF-FINAL-B-13 : `metrics.go` uses `sync.Mutex` for Prometheus counter
- **Severity**: MEDIUM
- **Location**: `backend/internal/handler/metrics.go:216`
- **Fix**: use `prometheus/client_golang` native counters.
- **Effort**: S (1-2h)

### PERF-FINAL-B-15 : `reindex` CLI lacks resume capability
- **Severity**: MEDIUM
- **Location**: `backend/cmd/reindex/main.go`
- **Fix**: checkpoint via `pending_events` table.
- **Effort**: S (1-2h)

## LOW (6)

- PERF-FINAL-B-16 : `IncrementUnreadForRecipients` fan-out org × org INSERTs — INSERT ... SELECT in single query. Effort: S.
- PERF-FINAL-B-17 : `idx_search_queries_search_id` UNIQUE — partition by day if scale demands. Effort: S.
- PERF-FINAL-B-18 : OFFSET pagination dans 8 admin endpoints. Effort: S.
- PERF-FINAL-B-19 : WS hub `register/unregister` channel size 64. Effort: XS.
- PERF-FINAL-B-21 : `INSERT (SELECT FROM organization_members WHERE user_id=$X LIMIT 1)` per creation. Effort: XS.
- PERF-FINAL-B-23 : `ListPaymentRecords` LEFT JOIN large — UNION pattern faster on >10k rows. Effort: M.

## Connection pools

- **Postgres** : MaxOpenConns=50, MaxIdleConns=25, ConnMaxLifetime=30min ✅ (`adapter/postgres/db.go:31`)
- **Redis** : pool 50/10/3 ✅
- **HTTP clients externes** : default → cf. PERF-FINAL-B-05

## Strong points backend

- Cursor pagination omniprésente sur les hot paths
- Context timeouts à 100% dans tous les repos
- 5 cache adapters spécialisés
- Pending events outbox `FOR UPDATE SKIP LOCKED` + stale recovery (m.128)
- Audit log append-only via REVOKE m.124 + RLS WITH CHECK m.129
- Batch query patterns (`GetTotalUnreadBatch`, `ListMilestonesForProposals`, `GetProfileSkillsBatch`)
- WS hub `SendToUser` non-bloquant (sendOrDrop)
- Slowloris guard (`ReadHeaderTimeout=5s` `wire_serve.go:109`)
- 3-step graceful shutdown
- OTel SDK with no-op zero-overhead fallback
- Slow query logger (50ms WARN / 500ms ERROR `slow_query.go`)
- Stripe webhook async via outbox + dedup (m.134)
- Last_message denormalised on conversation (m.133)
- Mutation rate limit anonymous-fallback (P10)

---

# WEB (Next.js 16) + ADMIN (Vite)

## HIGH (2)

### PERF-FINAL-W-01 : `payment-info/page.tsx` fetch + polling dans `useEffect`
- **Severity**: HIGH
- **Location**: `web/src/app/[locale]/(app)/payment-info/page.tsx`
- **Fix**: TanStack Query with `refetchInterval: mode === 'onboarding' ? 10000 : false`.
- **Effort**: M (½j)

### PERF-FINAL-W-03 : 31/51 pages declare `"use client"` — over-hydration
- **Severity**: HIGH
- **Location**: `grep -rln '"use client"' web/src/app/` → 31 sites.
- **Fix**: descendre `"use client"` au composant interactif.
- **Effort**: M (½j)

## MEDIUM (5)

- PERF-FINAL-W-05 : `experimental.optimizePackageImports` incomplete. Effort: XS.
- PERF-FINAL-W-06 : `loadStripe` au module level. Effort: XS.
- PERF-FINAL-W-07 : `useDebouncedValue` dupliqué. Effort: XS.
- PERF-FINAL-W-09 : Aucun `priority` prop sur les images LCP. Effort: S.
- PERF-FINAL-W-10 : 467 hardcoded `/api/v1/` strings. **Will be reduced when F.3.2 lands** (typed apiClient migration on `feat/f3-2-openapi-and-typed-paths` covers 174/178 sites). Effort: M for full centralization.

## LOW (5)

- 5 LOW items unchanged.

---

# MOBILE (Flutter 3.16+)

## HIGH (3)

### PERF-FINAL-M-01 : ProviderScope at root + minimal `.select()` adoption
- **Severity**: HIGH
- **Why**: 508 `dynamic` field references (down from 746 thanks to F.3.3). Many flow through Riverpod and trigger global rebuilds.
- **Fix**: `ConsumerStatelessWidget` + targeted `ref.watch(provider.select((s) => s.specificField))`.
- **Effort**: L (3 days, can stagger)

### PERF-FINAL-M-02 : Deferred imports in `app_router.dart`
- **Severity**: HIGH
- **Fix**: `deferred as` for routes likely never visited.
- **Effort**: M (½j)

### PERF-FINAL-M-03 : Mobile files at 530-600 lines (3 files near cap)
- **Severity**: HIGH (preventive)
- **Fix**: split before regression.
- **Effort**: M (½j)

## CLOSED in F.3.3

- PERF-FINAL-M-04 (Color hex regression) — **CLOSED**: 491 → 573 → 124 hex strings in `mobile/lib/`. Most migrated to AppPalette tokens. Remaining 124 are documented exceptions.

## MEDIUM (6)

- 6 MEDIUM mobile perf items — see previous audit, unchanged.

## LOW (5)

- 5 LOW items unchanged.

---

## Index audit (132 migrations)

| Table | Status | Notes |
|---|---|---|
| proposals | ✅ | composite indexes both org sides |
| payment_records | ✅ | provider_organization_id added m.131 (CONCURRENTLY) |
| conversations, messages | ✅ | composites + denormalised last_message m.133 |
| audit_logs | ✅ | partial indexes, RLS WITH CHECK m.129 |
| invoice | ✅ | composite |
| pending_events | ✅ | partial idx for stuck rows m.128, stripe dedup m.134 |
| jobs, job_applications, organizations | ✅ | composites |

`CREATE INDEX CONCURRENTLY` count: 1 / 200+ index migrations — m.131 documents the manual workflow.

---

## Top-1% benchmark — performance

**Strengths confirmed**:
- Slowloris closed
- Slow query logger 50ms/500ms thresholds
- Async Stripe webhook via outbox
- Last_message denormalised
- Raw `<img>` migration (3 production sites left, vs 27 originally)
- OTel SDK with zero-overhead default
- Mutation rate limit covers anonymous traffic
- 3-step graceful shutdown with WS drain
- Mobile Color hex regression closed (124 vs 573)

**Remaining gaps**:
1. `payment_records.ListByOrganization` unbounded (B-02)
2. Stripe Connect `account.GetByID` not cached (B-06)
3. `payment-info/page.tsx` legacy `useEffect` polling (W-01)
4. 31 page-level `"use client"` (W-03)
5. Mobile `dynamic` regression to investigate (M-01)

**Verdict**: backend perf primitives **TOP 1%** (slowloris, slow query log, outbox, RLS, graceful shutdown, OTel). Frontend at **TOP 5%** (over-hydration + image priority gap). Mobile at **TOP 5%** (Riverpod over-rebuild + dynamic types).
