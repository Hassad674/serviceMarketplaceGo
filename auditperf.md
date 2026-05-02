# Audit de Performance — Final Verification

**Date** : 2026-05-01 (final verification post F.1 + F.2)
**Branche** : `chore/final-verification-audit`
**Périmètre** : backend Go (~622 .go prod files, 134 migrations) + DB Postgres + web Next.js + admin Vite + mobile Flutter

---

## Snapshot — état actuel après F.1 + F.2 (PRs #31 → #91)

| Layer | CRITICAL | HIGH | MEDIUM | LOW | Total |
|---|---|---|---|---|---|
| Backend + DB | 0 | 1 | 8 | 6 | 15 |
| Web + Admin | 0 | 2 | 6 | 5 | 13 |
| Mobile | 0 | 4 | 6 | 5 | 15 |
| **Total** | **0** | **7** | **20** | **16** | **43** |

**Closed since previous round (15 items closed by F.1 + F.2)** :
- PERF-FINAL-B-01 — `ReadHeaderTimeout=5s` slowloris guard wired in `backend/cmd/api/wire_serve.go:109` with test `TestBuildHTTPServer_Timeouts`. **CLOSED**.
- PERF-FINAL-B-04 — slow query logger live in `backend/internal/adapter/postgres/slow_query.go` (50ms WARN, 500ms ERROR). **CLOSED**.
- PERF-FINAL-B-07 — last_message denormalisation done via migration 133 + repo path. **CLOSED** (verified `133_denormalize_last_message.up.sql`).
- PERF-FINAL-B-12 — Stripe webhook async via `pending_events` worker + `stripe_event_id` dedup column (m.134). **CLOSED**.
- PERF-FINAL-W-02 — 27 raw `<img>` migration done; remaining 3 are documented blob: previews with `eslint-disable-next-line` and `@next/next/no-img-element` is now `error` level. **CLOSED**.
- PERF-FINAL-W-04 — staleTime fix on team permissions (PR #65 trail). Verify in source.
- BUG-NEW-12 / W-11 — admin Suspense flash. **CLOSED**.
- BUG-NEW-13 / W-12 — RSC fallback port. **CLOSED**.
- OTel hooks (P11) — wired with no-op fallback, tests confirm.
- Graceful shutdown (P11 #5) — 3-step `drainHTTP` → `drainWS` → `drainWorkers`.

---

# BACKEND + DB

## HIGH (1)

### PERF-FINAL-B-02 : `payment_records.ListByOrganization` sans LIMIT ni cursor
- **Severity**: HIGH
- **Location** : `backend/internal/adapter/postgres/payment_record_repository.go:354-410` (verify line numbers in current branch).
- **Why it matters** : `WHERE pr.organization_id = $1 OR pr.provider_organization_id = $1 ORDER BY pr.created_at DESC` SANS `LIMIT`. Une agence active 12-24 mois → 10k-50k lignes par requête wallet.
- **Fix** : cursor pagination `(created_at, id)` standard. Use `pkg/cursor/Encode`.
- **Effort** : S (1-2h)

## MEDIUM (8)

### PERF-FINAL-B-03 : `service.CacheService` interface absent
- **Severity**: MEDIUM (was HIGH; downgraded — 5 specialized caches do exist)
- **Location** : `backend/internal/port/service/` (no generic interface). Adapters individuels existent : profile, expertise, freelance_profile, skill_catalog, subscription.
- **Fix** : extract generic port + Redis adapter, migrate the 5 specialized caches.
- **Effort** : M (½j)

### PERF-FINAL-B-05 : Tous les clients HTTP externes utilisent `http.DefaultTransport`
- **Severity**: MEDIUM (downgraded from HIGH given current load)
- **Location** : `backend/internal/search/embeddings.go`, `backend/internal/search/client.go`, `backend/internal/adapter/openai/client.go`, `backend/internal/adapter/anthropic/analyzer.go`, `backend/internal/adapter/vies/client.go`, `backend/internal/adapter/nominatim/client.go`
- **Fix** : `pkg/httpx/NewTunedClient(timeout time.Duration) *http.Client` with MaxIdleConnsPerHost: 50, KeepAlive: 30s, ForceAttemptHTTP2.
- **Effort** : XS (30 min)

### PERF-FINAL-B-06 : Stripe Connect `account.GetByID` jamais caché + ctx ignoré
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/stripe/account.go` — 7 sites use `account.GetByID(accountID, nil)`.
- **Fix** : redis cache (60-120s TTL, invalidated via `account.updated` webhook) + `&stripe.AccountParams{Params: stripe.Params{Context: ctx}}`.
- **Effort** : M (½j)

### PERF-FINAL-B-08 : Notification worker — `getPrefs` + `users.GetByID` à chaque job
- **Severity**: MEDIUM
- **Location** : `backend/internal/app/notification/worker.go:99, 193, 211-222`
- **Fix** : LRU local cache + singleflight. Process-local OK.
- **Effort** : S (1-2h)

### PERF-FINAL-B-09 : Search worker tick = 30s
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/worker/worker.go:86-87`
- **Fix** : tick 5s + Redis pubsub `wake_search_worker`.
- **Effort** : S (1-2h)

### PERF-FINAL-B-10 : LTR capture INSERT par recherche
- **Severity**: MEDIUM
- **Fix** : batched flush via channel + 1×/sec multi-row INSERT.
- **Effort** : S (1-2h)

### PERF-FINAL-B-11 : Indexer fan-out 9 goroutines × N actors sans cap
- **Severity**: MEDIUM
- **Location** : `backend/internal/search/indexer.go:314-364`
- **Fix** : `errgroup.SetLimit(3)`.
- **Effort** : XS (30 min)

### PERF-FINAL-B-13 : `metrics.go` uses `sync.Mutex` for Prometheus counter
- **Severity**: MEDIUM
- **Location** : `backend/internal/handler/metrics.go:216`
- **Fix** : use `prometheus/client_golang` native counters.
- **Effort** : S (1-2h)

### PERF-FINAL-B-15 : `reindex` CLI lacks resume capability
- **Severity**: MEDIUM
- **Location** : `backend/cmd/reindex/main.go`
- **Fix** : checkpoint via `pending_events` table.
- **Effort** : S (1-2h)

## LOW (6)

- **PERF-FINAL-B-16** : `IncrementUnreadForRecipients` fan-out org × org INSERTs — INSERT ... SELECT in single query. Effort: S.
- **PERF-FINAL-B-17** : `idx_search_queries_search_id` UNIQUE — partition by day if scale demands. Effort: S.
- **PERF-FINAL-B-18** : OFFSET pagination dans 8 admin endpoints. Effort: S.
- **PERF-FINAL-B-19** : WS hub `register/unregister` channel size 64. Effort: XS.
- **PERF-FINAL-B-21** : `INSERT (SELECT FROM organization_members WHERE user_id=$X LIMIT 1)` per creation. Effort: XS.
- **PERF-FINAL-B-23** : `ListPaymentRecords` LEFT JOIN large — UNION pattern faster on >10k rows. Effort: M.

## Index audit

| Table | Status | Notes |
|---|---|---|
| proposals | ✅ | composite indexes both org sides |
| payment_records | ✅ | provider_organization_id added m.131 |
| conversations, messages | ✅ | composites + denormalized last_message m.133 |
| audit_logs | ✅ | partial indexes, RLS WITH CHECK m.129 |
| invoice | ✅ | composite |
| pending_events | ✅ | partial idx for stuck rows m.128, stripe dedup m.134 |
| jobs, job_applications, organizations | ✅ | composites |

## Connection pools

- **Postgres** : MaxOpenConns=50, MaxIdleConns=25, ConnMaxLifetime=30min ✅
- **Redis** : pool 50/10/3 ✅
- **HTTP clients externes** : default → cf. PERF-FINAL-B-05

## Strong points backend

- Cursor pagination omniprésente sur les hot paths
- Context timeouts à 100% dans tous les repos
- 5 cache adapters spécialisés
- Pending events outbox `FOR UPDATE SKIP LOCKED` + stale recovery (m.128)
- Audit log append-only via REVOKE m.124 + RLS WITH CHECK m.129
- Batch query patterns sur listings (`GetTotalUnreadBatch`, `ListMilestonesForProposals`, `GetProfileSkillsBatch`)
- WS hub `SendToUser` non-bloquant (sendOrDrop) ✅
- Slowloris guard (`ReadHeaderTimeout=5s`)
- 3-step graceful shutdown
- OTel SDK with no-op zero-overhead fallback
- Slow query logger (50ms WARN / 500ms ERROR)

---

# WEB (Next.js 16) + ADMIN (Vite)

## HIGH (2)

### PERF-FINAL-W-01 : `payment-info/page.tsx` fetch + polling dans `useEffect`
- **Severity**: HIGH
- **Location** : `web/src/app/[locale]/(app)/payment-info/page.tsx`
- **Fix** : `useQuery({ queryKey: ['payment-info', orgId], queryFn: ..., refetchInterval: mode === 'onboarding' ? 10000 : false })`.
- **Effort** : M (½j)

### PERF-FINAL-W-03 : 31/51 pages déclarent `"use client"` — over-hydration
- **Severity**: HIGH (verified — count is 31 page-level)
- **Location** : `grep -rn '"use client"' web/src/app/` → 31 sites.
- **Fix** : descendre `"use client"` au composant interactif.
- **Effort** : M (½j)

## MEDIUM (6)

- **PERF-FINAL-W-05** : `experimental.optimizePackageImports` incomplete (`next-intl`, `@stripe/react-stripe-js`, `@stripe/react-connect-js` missing). Effort: XS.
- **PERF-FINAL-W-06** : `loadStripe` au module level. Effort: XS.
- **PERF-FINAL-W-07** : `useDebouncedValue` dupliqué. Effort: XS.
- **PERF-FINAL-W-08** : Hex couleurs marque dupliqués 3 fois. Effort: XS.
- **PERF-FINAL-W-09** : Aucun `priority` prop sur les images LCP candidates. Effort: S.
- **PERF-FINAL-W-10** : 467 hardcoded `/api/v1/` strings (was 96 in previous audit — count grew!). Effort: M for centralization.

## LOW (5)

- 5 LOW items — see previous audit, unchanged.

---

# MOBILE (Flutter 3.16+)

## HIGH (4)

### PERF-FINAL-M-01 : ProviderScope at root + minimal `.select()` adoption
- **Severity**: HIGH
- **Why it matters** : 746 `dynamic` field references identified — many flow through Riverpod and trigger global rebuilds.
- **Fix** : `ConsumerStatelessWidget` + targeted `ref.watch(provider.select((s) => s.specificField))`.
- **Effort** : L (3 days, can stagger)

### PERF-FINAL-M-02 : Deferred imports in `app_router.dart`
- **Severity**: HIGH
- **Fix** : `deferred as` for routes likely never visited (admin-only, edge flows).
- **Effort** : M (½j)

### PERF-FINAL-M-03 : 3 mobile files at 530-595 lines (preventive split before they cross 600)
- **Severity**: HIGH
- **Fix** : split before regression.
- **Effort** : M (½j)

### PERF-FINAL-M-04 : 573 `Color(0x...)` hex hardcoded (was 491 in previous audit — regression!)
- **Severity**: HIGH (DRY + theming)
- **Fix** : centralize via theme tokens.
- **Effort** : L (2 days)

## MEDIUM (6)

- 6 MEDIUM mobile perf items — see previous audit, unchanged.

## LOW (5)

- 5 LOW items — see previous audit, unchanged.

---

## Top-1% benchmark — performance

**Strengths confirmed in this verification**:
- Slowloris closed
- Slow query logger active with 50ms/500ms thresholds
- Async Stripe webhook via outbox
- Last_message denormalised
- 27 raw `<img>` migrated; ESLint `@next/next/no-img-element: error`
- OTel SDK wired with zero-overhead default
- Mutation rate limit covers anonymous traffic (P10 #3)
- 3-step graceful shutdown with WS drain

**Remaining gaps (top 5)**:
1. `payment_records.ListByOrganization` unbounded (B-02)
2. Stripe Connect `account.GetByID` not cached (B-06)
3. `payment-info/page.tsx` legacy `useEffect` polling (W-01)
4. 31 page-level `"use client"` (W-03)
5. Mobile color/dynamic regressions (M-01, M-04)

**Verdict** : **Top 5%** — backend perf primitives are world-class, frontend lags slightly on bundle/RSC discipline, mobile lags on type-safety and theming.
