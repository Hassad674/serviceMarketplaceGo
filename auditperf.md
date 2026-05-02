# Audit de Performance — Final Deep

**Date** : 2026-05-01 (final audit before public showcase)
**Branche** : `chore/final-audit-deep`
**Périmètre** : backend Go (~622 .go files prod, 131 migrations) + DB Postgres + web Next.js + admin Vite + mobile Flutter
**Méthodologie** : Audit statique exhaustif. Lecture du code (pas seulement file names). Cross-référence avec PRs #31-#66 fusionnés. Tout finding cite un fichier:ligne précis et propose un fix concret.

---

## Snapshot — état actuel après PRs #31-#66

| Layer | CRITICAL | HIGH | MEDIUM | LOW | Total |
|---|---|---|---|---|---|
| Backend + DB | 0 | 6 | 11 | 6 | 23 |
| Web + Admin | 0 | 5 | 8 | 5 | 18 |
| Mobile | 0 | 4 | 8 | 5 | 17 |
| **Total** | **0** | **15** | **27** | **16** | **58** |

**Closed since previous round (28 items)** : streaming uploads (PR #34), notification worker pool (PR #36), WS sendOrDrop (PR #40), LiveKit lazy (PR #41), RSC public listings (PR #41), loading/error/sitemap (PR #41), TestDB removed (Phase 0), cross-feature imports partial (PR #37), unused web deps (Phase 0), admin lazy routes (PR #41), mobile dead deps (Phase 0), `GetParticipantNames` N+1 (PERF-B-02 closed), `GetTotalUnread` partial index (PERF-B-11 closed), milestone `CreateBatch` bulk (PERF-B-04 closed), `provider_organization_id` indexes (PERF-B-08 closed via m.131), Firebase deferred (PR #41), FCM post-frame (PR #41 cat F), memCacheWidth (PR #41 cat D), const constructors (PR #41 cat A), unread badge scope (PR #41 cat B), stable keys + cacheExtent (PR #41 cat C), RepaintBoundary (PR #41 cat E), reuse hub (PR #41), search worker tick + Redis pubsub (still open per audit), profile cache infra (Phase 4 M closed), expertise/skill catalog cache (Phase 4 M).

---

# BACKEND + DB

## HIGH (6)

### PERF-FINAL-B-01 (was PERF-B-07) : `WriteTimeout: 0` + pas de `ReadHeaderTimeout` — Slowloris
- **Severity**: HIGH
- **Location** : `backend/cmd/api/main.go:866-874`
- **Why it matters** : aucun timeout côté HTTP server hors `ReadTimeout: 15s`. Avec `MaxOpenConns=50`, un attaquant peut ouvrir 50 connexions et envoyer des headers byte-par-byte → saturation triviale du pool DB. `WriteTimeout: 0` est commenté "pour les WS long-lived" mais c'est précisément ce qui doit être surchargé localement, pas globalement.
- **How to fix** : `ReadHeaderTimeout = 5*time.Second`, `WriteTimeout = 60*time.Second`, garder `IdleTimeout = 60s`. Pour SSE/WS, utiliser `http.NewResponseController(w).SetWriteDeadline(time.Time{})` dans le handler concerné (ws.ServeWS et SSE handlers) — c'est l'API Go 1.20+ exactement faite pour ça.
- **Test required** : `slowloris_test.go` qui démarre le serveur, ouvre 50 connexions, envoie 1 byte/sec, vérifie que le serveur ferme après 5s (`assert.LessOrEqual(connDuration, 6*time.Second)`).
- **Effort** : XS (30 min)

### PERF-FINAL-B-02 (was PERF-B-03) : `payment_records.ListByOrganization` sans LIMIT ni cursor
- **Severity**: HIGH
- **Location** : `backend/internal/adapter/postgres/payment_record_repository.go:354-410`
- **Why it matters** : `WHERE pr.organization_id = $1 OR pr.provider_organization_id = $1 ORDER BY pr.created_at DESC` SANS `LIMIT`. Une agence active 12-24 mois → 10k-50k lignes par requête wallet. ~10 MB RAM Go par appel. Le commentaire dit "nouvelle plan = Index Scan" mais le `ORDER BY ... DESC` non borné force toujours un Sort matériel ou un Index Scan complet. Un user peut DoS un nœud en spammant `/wallet/list`.
- **How to fix** : cursor pagination `(created_at, id)` standard. Signature `ListByOrganization(ctx, orgID, cursor, limit)`. Utiliser exactement le pattern `pkg/cursor/Encode` qui existe déjà. Pour les rapports historiques, créer un endpoint dédié `/admin/payment-records/export` qui stream en CSV.
- **Test required** : table-driven `payment_record_repository_test.go::TestListByOrganization_Pagination` avec 250 records, asserte que `len(page1) == 50` ET que `page2 = ListByOrganization(ctx, orgID, page1Cursor, 50)` continue sans dupliquer.
- **Effort** : S (1-2h)

### PERF-FINAL-B-03 (was PERF-B-05) : `service.CacheService` interface absent — pas de cache-aside générique
- **Severity**: HIGH
- **Location** : `backend/internal/port/service/` (interface manquante). Adapters individuels existent : `profile_cache.go`, `expertise_cache.go`, `freelance_profile_cache.go`, `skill_catalog_cache.go`, `subscription_cache.go` — mais chacun a sa propre interface ad-hoc.
- **Why it matters** : pas d'API unifiée signifie : (a) chaque feature qui veut du cache redéfinit ses propres signatures (DRY violation), (b) impossible de tracer les hits/misses globalement, (c) impossible de switcher le backend Redis vers un L1+L2 (in-memory + Redis) sans toucher 5 fichiers. Aucun cache pour `jobapp.ListPublic`, `reviewapp.GetAggregateForOrg`, `searchapp.GetCuratedExpertises` (les rares hot-paths SEO non-cachés).
- **How to fix** : 
  1. Port `service.CacheService { Get(ctx, key) ([]byte, error); Set(ctx, key, val, ttl); Delete(ctx, key); InvalidatePrefix(ctx, prefix) }`. 
  2. Adapter `redis/generic_cache.go` qui l'implémente.
  3. Migrer les 5 caches existants vers ce port (préserver les wrappers typés en option).
  4. Métriques `cache_hits_total{key_prefix}` / `cache_misses_total{key_prefix}` pour observabilité.
- **Test required** : `cache_test.go` table-driven : Get-after-Set retourne la valeur ; Get après TTL retourne miss ; InvalidatePrefix dégage tous les keys d'un prefix ; race test 100 goroutines Get/Set/Delete simultanés.
- **Effort** : M (½j)

### PERF-FINAL-B-04 (was PERF-B-06) : Slow-query logger inexistant
- **Severity**: HIGH
- **Location** : pas de `pkg/dbx/` ni `query_logger.go`. Seules traces : `slog.Warn` ad-hoc dans certains repos.
- **Why it matters** : régressions de plan invisibles en prod. Un nouveau filtre qui casse l'index, un VACUUM ANALYZE qui change le planner, une croissance de table — aucun signal. CLAUDE.md ligne 1166-1181 prescrit son existence.
- **How to fix** : 
  1. `pkg/dbx/Wrapper { db *sql.DB; threshold time.Duration }` qui wrappe `QueryContext`/`ExecContext`/`QueryRowContext`.
  2. Mesure `time.Since(start)` ; si > 50ms (configurable via env `DB_SLOW_THRESHOLD_MS`), `slog.Warn("slow query", "duration_ms", dur, "query", q[:200], "args_count", len(args))`.
  3. Wirer dans `cmd/api/main.go` : remplacer `db := postgres.Open(cfg)` par `db := dbx.Wrap(postgres.Open(cfg), 50*time.Millisecond)`.
  4. Métrique `db_query_duration_seconds_bucket{operation}` Prometheus en bonus.
- **Test required** : `dbx_test.go` : exécute une query lente (sleep 100ms via Postgres `pg_sleep`), capture le slog output, asserte la présence de `"slow query"`. Mock `slog.Default()` avec une `slog.NewTextHandler(buf, ...)`.
- **Effort** : S (1-2h)

### PERF-FINAL-B-05 (was PERF-B-09) : Tous les clients HTTP externes utilisent `http.DefaultTransport` (`MaxIdleConnsPerHost=2`)
- **Severity**: HIGH
- **Location** : 
  - `backend/internal/search/embeddings.go:89`
  - `backend/internal/search/client.go:96`
  - `backend/internal/adapter/openai/client.go:46`
  - `backend/internal/adapter/anthropic/analyzer.go:53`
  - `backend/internal/adapter/vies/client.go:91`
  - `backend/internal/adapter/nominatim/client.go:39`
- **Why it matters** : `http.DefaultTransport` plafonne à 2 conns idle par host. Sous burst (100 hits Typesense/s pour un dashboard searchy), 50-100 ms gaspillés par cold connection TLS. Typesense particulièrement touché — c'est le hot-path utilisateur.
- **How to fix** : créer `pkg/httpx/NewTunedClient(timeout time.Duration) *http.Client` qui wrappe `&http.Transport{ MaxIdleConns: 200, MaxIdleConnsPerHost: 50, MaxConnsPerHost: 100, IdleConnTimeout: 90*time.Second, ForceAttemptHTTP2: true, DialContext: (&net.Dialer{Timeout: 5s, KeepAlive: 30s}).DialContext, TLSHandshakeTimeout: 5s }`. Migrer les 6 sites.
- **Test required** : `httpx_test.go` benchmark avec 100 requêtes parallèles vers un test server : avant fix → ~10s, après fix → ~2s. Plus un test que le timeout dial est respecté.
- **Effort** : XS (30 min)

### PERF-FINAL-B-06 (was PERF-B-14, PERF-B-15) : Stripe Connect `account.GetByID` jamais caché + ctx ignoré
- **Severity**: HIGH
- **Location** : `backend/internal/adapter/stripe/account.go:156, 306, 370, 381, 390, 420, 450` (7 sites)
- **Why it matters** : 100-300 ms par appel Stripe externe sur chaque endpoint vérifiant ChargesEnabled/PayoutsEnabled. Au checkout, 2-3 appels consécutifs = 600-900 ms gaspillés sur un cold path. De plus, ces appels passent `nil` (line 306 etc.) ou `&stripe.AccountParams{}` sans `Context: ctx` — annulation impossible si le client se déconnecte.
- **How to fix** : 
  1. Cache : `redisadapter.AccountStatusCache` clé `stripe:acct:{accountID}`, valeur snapshot des flags (`charges_enabled, payouts_enabled, details_submitted, requirements_count`), TTL 60-120s. Invalider sur webhook `account.updated` (déjà reçu, juste appeler `cache.Delete`).
  2. Ctx : remplacer `account.GetByID(accountID, nil)` par `account.GetByID(accountID, &stripe.AccountParams{Params: stripe.Params{Context: ctx}})`. Le SDK stripe-go v82 accepte `Params.Context`.
- **Test required** : `account_cache_test.go` : 1er appel hit Stripe, 2e appel hit cache (vérifier via mock du Stripe SDK call count == 1) ; ctx-cancel test : créer un `context.WithTimeout(ctx, 1ms)`, asserter `account.GetByID` retourne `context.DeadlineExceeded` (avec un Stripe SDK qui sleep).
- **Effort** : M (½j)

## MEDIUM (11)

### PERF-FINAL-B-07 (was PERF-B-10) : `ConversationRepository.ListConversations` — 2 LATERAL JOIN par row
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/postgres/conversation_queries.go:89-141`
- **Why it matters** : double LATERAL pour `conversation_participants → users` + `last_message`. À 10k+ conversations, 30-50 ms par page p50. C'est l'endpoint le plus chargé de la home dashboard.
- **How to fix** : dénormaliser `last_message_seq`, `last_message_content_preview` (≤100 chars), `last_message_at`, `last_message_sender_id` directement sur la table `conversations`. Maintenu par trigger `AFTER INSERT ON messages` (atomique) OU par l'app au moment de l'INSERT messages dans la même tx (préféré, contrôle explicite). Migration `133_denormalize_last_message.up.sql` + backfill.
- **Test required** : `conversation_queries_test.go::TestListConversations_LastMessageDenormalised` — insert 5 conversations + 20 messages, asserter chaque conversation a son last_message correctement, et asserter qu'aucune LATERAL join apparaît dans `EXPLAIN`.
- **Effort** : M (½j)

### PERF-FINAL-B-08 (was PERF-B-13) : Notification worker — `getPrefs` + `users.GetByID` à chaque job
- **Severity**: MEDIUM
- **Location** : `backend/internal/app/notification/worker.go:99, 193, 211-222`
- **Why it matters** : 2 queries DB par notif × 1000 notifs/min = 2000 queries/min purement administratives. Quand le pool DB est sous pression, ces queries amplifient le contention.
- **How to fix** : cache LRU local `golang.org/x/sync/singleflight` (déjà transitive dep) + `hashicorp/golang-lru/v2` (à ajouter, ~1KB). TTL 2 min, taille 1000. Clé `prefs:{userID}` et `user:{userID}`. Le cache est process-local, pas de cohérence cross-instance — acceptable car les changements de prefs sont rares (UI manual update).
- **Test required** : `worker_cache_test.go` : 100 notifs pour le même user, asserter `users.GetByID` mock count == 1 (pas 100). Test TTL : avancer le clock de 3 min, asserter cache invalidée.
- **Effort** : S (1-2h)

### PERF-FINAL-B-09 (was PERF-B-17) : Search worker tick = 30s — embeddings différés trop longuement
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/worker/worker.go:86-87`
- **Why it matters** : profil mis à jour, index Typesense obsolète ~30s. Le user voit son profil updated dans l'UI mais ne le retrouve pas en search pendant 30s — confusion + signaling tickets.
- **How to fix** : tick 5s + signal Redis pubsub `wake_search_worker`. Le publisher (PublishReindexTx) `PUBLISH wake_search_worker 1` après commit. Le worker écoute en parallèle de son ticker via `redis.PSubscribe`. Sur réception, drain immédiat. Le ticker reste en backstop pour les events insérés avant que le subscriber soit ready.
- **Test required** : intégration test avec 1 instance backend + 1 instance Redis : INSERT pendingevent, mesurer le time-to-process (cible < 1s avec pubsub vs 15s en moyenne avec tick 30s).
- **Effort** : S (1-2h)

### PERF-FINAL-B-10 (was PERF-B-18) : LTR capture INSERT par recherche
- **Severity**: MEDIUM
- **Location** : `backend/internal/app/searchanalytics/ltr_capture.go:102-111`
- **Why it matters** : 100 recherches/s = 100 UPDATE/s additionnels. À l'échelle, c'est de la pression DB pure pour de l'analytics asynchrone.
- **How to fix** : batcher via `chan SearchEvent` buffered (cap 1024), goroutine dédiée flush 1×/seconde via `INSERT ... VALUES (...), (...), (...)` multi-row. Drop l'event si le channel est full (slog.Warn) — analytics n'est pas critique.
- **Test required** : load test 1000 events/s, asserter < 5 INSERTs effectifs par seconde + zéro perte d'events sous le throughput nominal.
- **Effort** : S (1-2h)

### PERF-FINAL-B-11 (was PERF-B-19) : Indexer fan-out 9 goroutines × N actors sans cap
- **Severity**: MEDIUM
- **Location** : `backend/internal/search/indexer.go:314-364`
- **Why it matters** : 9 reads concurrents par actor reindexed. Reindex full = 90k goroutines en rafale → DB pool saturation, OOM Go potentiel.
- **How to fix** : `golang.org/x/sync/errgroup`+`SetLimit(3)` au niveau actor (3 actors parallèles × 9 reads internes = 27 connections max, raisonnable face au pool de 50). Le pattern `SetLimit` retourne `nil` au dépassement et `Go()` bloque jusqu'à libération d'un slot.
- **Test required** : `indexer_test.go::TestReindexAllRespectsConcurrencyCap` — avec un mock repo qui sleep 100ms par read, lancer un reindex de 100 actors, asserter `max(active_goroutines) <= 30`.
- **Effort** : XS (30 min)

### PERF-FINAL-B-12 (was PERF-B-20) : Stripe webhook handlers synchrones inline
- **Severity**: MEDIUM
- **Location** : `backend/internal/handler/stripe_handler.go:225-272` (`dispatch`)
- **Why it matters** : `handleInvoicePaid` peut faire 5+ DB writes + PDF generation (chromedp 2-5s) + email synchrones. Stripe timeout webhook = 10s. Le BUG-NEW-06 fix renvoie 503 + release claim, mais le timeout natural reste un risque sous load.
- **How to fix** : enqueuer dans `pending_events` (table existe déjà) avec `event_type='stripe.webhook.dispatch'` + `payload=event.JSON`, retourner 200 immédiatement. Worker `pending_events_worker` poll et exécute `dispatch` async. L'idempotency key reste claim-on-receipt mais release seulement après processing async.
- **Test required** : webhook handler test avec mock service qui sleep 5s : asserter response < 200ms et que l'event est en `pending_events` à status `pending`.
- **Effort** : M (½j)

### PERF-FINAL-B-13 (NEW) : `metrics.go` uses `sync.Mutex` for Prometheus counter operations
- **Severity**: MEDIUM
- **Location** : `backend/internal/handler/metrics.go:216` (`_, _ = w.Write(...)`)
- **Why it matters** : metrics scrape endpoint serializes via mutex. Under high scrape frequency (e.g. Prometheus 5s interval × multiple replicas via service discovery), this becomes a contention point. The mutex also protects multiple counters — a long write blocks observability calls.
- **How to fix** : use `prometheus/client_golang` instead of hand-rolled string-builder. Native counters use atomic ops, no mutex. Drop ~60 lines of code.
- **Test required** : `metrics_concurrent_test.go` — 100 goroutines incrementing counters + 1 scraping endpoint, asserter no race detected via `-race` and total count is correct.
- **Effort** : S (1-2h)

### PERF-FINAL-B-14 (NEW) : 35 legacy `.GetByID()` callers in app layer bypass tenant context
- **Severity**: MEDIUM (technical debt that becomes HIGH at prod role rotation)
- **Location** : `backend/internal/app/{proposal,dispute,review,referral}/*.go` — 35 sites identified by `grep -rn ".GetByID(" backend/internal/app/ | grep -v "GetByIDForOrg|GetByIDWithVersion"`. Examples: `service_actions.go:20, 72, 101, 165, 260, 296` (proposal); `service_actions.go:119, 269, 353, 437, 478, 528, 554, 600, 686, 767, 792, 838, 849` (dispute); `service.go:86, 280` (review).
- **Why it matters** : these callers use the legacy `GetByID(ctx, id)` signature which doesn't install tenant context via RunInTxWithTenant. Today they work because the migration owner role bypasses RLS. Once the production rotation to `marketplace_app NOSUPERUSER NOBYPASSRLS` is performed (per `backend/docs/rls.md`), every one of these calls returns `ErrProposalNotFound` / `ErrDisputeNotFound` immediately. The 8-path PR series only migrated REPO methods; the APP CALLERS still pass through unguarded GetByID.
- **How to fix** : two options:
  - (a) Add `GetByIDForOrg(ctx, id, callerOrgID)` to every repo, migrate every caller. ~3 days of mechanical work.
  - (b) Keep `GetByID` but document it as "system-actor only" and add a runtime check `if !systemActorContext(ctx) { return ErrSystemActorOnly }`. Faster but uglier.
  
  Recommended: (a). Each migration site adds an explicit `orgID := mustGetOrgID(ctx)` at the top of the action method, then threads `orgID` to every call site.
- **Test required** : `rls_caller_audit_test.go` (integration) — create a `marketplace_test_app` role with NOBYPASSRLS, run every public service method through the role, asserter all return correctly. Currently the test only covers the migrated repos.
- **Effort** : L (3 jours)

### PERF-FINAL-B-15 (NEW) : `reindex` CLI lacks resume capability
- **Severity**: MEDIUM
- **Location** : `backend/cmd/reindex/main.go:155` (`reindexPersona` — 7-param function)
- **Why it matters** : `reindex` is a 1.5h+ operation on full prod data. Crash mid-run = restart from zero. No checkpoint, no `--resume-from` flag.
- **How to fix** : write checkpoint to `pending_events` table with `event_type='reindex.checkpoint'` after every 100 actors. On startup, `LoadCheckpoint(ctx, persona)` returns the last completed offset; reindex resumes there. Cleanup checkpoint on completion.
- **Test required** : `reindex_resume_test.go` — start a reindex, kill mid-batch, restart, asserter total processed == total expected (no duplicates from re-processing, no gaps).
- **Effort** : S (1-2h)

### PERF-FINAL-B-16 (was PERF-B-22) : `IncrementUnreadForRecipients` fan-out org × org INSERTs
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/postgres/conversation_queries.go` (queryIncrementUnreadForRecipients)
- **Why it matters** : pour une org de 50 membres, 50 INSERTs séquentiels par message envoyé. À 100 msgs/min sur 5 grosses orgs = 25k inserts/min purement comptables.
- **How to fix** : INSERT ... SELECT en une seule query : `INSERT INTO conversation_read_state (conversation_id, user_id, unread_count) SELECT $1, om.user_id, 1 FROM organization_members om WHERE om.organization_id = $2 AND om.user_id != $3 ON CONFLICT (conversation_id, user_id) DO UPDATE SET unread_count = unread_count + 1`.
- **Test required** : table-driven test : 50 members, 1 message, asserter exactly 1 INSERT statement executed (mock the *sql.DB).
- **Effort** : S (1-2h)

### PERF-FINAL-B-17 (was PERF-B-29) : `idx_search_queries_search_id` UNIQUE
- **Severity**: MEDIUM (negligible aujourd'hui, mais à mesurer si search analytics scale)
- **Location** : migration création `idx_search_queries_search_id`
- **Why it matters** : peut bloquer inserts concurrents sur hot search_id. À 100+ recherches/s sur le même query, contention de lock.
- **How to fix** : si LTR capture devient hot, migrer search_queries vers un partitioned table par jour OU utiliser un id généré côté client (UUID v4) sans contrainte unique.
- **Effort** : S (1-2h)

## LOW (6)

- **PERF-FINAL-B-18** : OFFSET pagination dans 8 admin endpoints (media, review_admin, moderation, user, job_admin, conversation_admin, job_application_admin) — toléré jusqu'à 50k rows mais à migrer en cursor à terme. Pattern `LIMIT %d OFFSET %d` sprintf est sécurisé (`int` typé) mais à parameterer (`LIMIT $N OFFSET $N+1`) pour le linter.
- **PERF-FINAL-B-19** : WS hub `register/unregister` channel taille 64 — bumper à 1024 pour deploy/network changes.
- **PERF-FINAL-B-20** : Pas d'index `(provider_organization_id, completed_at DESC) WHERE status='completed'` sur proposals — déjà ajouté en m.131. Verified.
- **PERF-FINAL-B-21** : `INSERT (SELECT FROM organization_members WHERE user_id=$X LIMIT 1)` à chaque création — 1 index seek par INSERT, négligeable mais pourrait être cached pour les batches.
- **PERF-FINAL-B-22** : Migration 074 backfill DO $$ block monolithique — pour les futures migrations bulk-copy splitter en chunks.
- **PERF-FINAL-B-23** : `ListPaymentRecords` LEFT JOIN inutilement large — pattern UNION serait plus rapide sur >10k rows.

## Index audit (table-by-table)

| Table | Status | Notes |
|---|---|---|
| proposals | OK after m.115+m.131 | composite indexes for both org sides |
| payment_records | OK after m.131 | provider_organization_id added |
| conversations, messages | OK | composites in m.074 |
| audit_logs | OK | partial indexes |
| invoice | OK | (recipient_org_id, issued_at, id) composite |
| pending_events | OK after m.128 | partial idx for stuck rows |
| jobs, job_applications, organizations, search_queries, moderation_results | OK | composites + partial where applicable |
| **last gap** | last_message denormalisation pending (PERF-FINAL-B-07) |

## Connection pools (vérifié)

- **Postgres** : MaxOpenConns=50, MaxIdleConns=25, ConnMaxLifetime=30min — conforme CLAUDE.md ✅
- **Redis** : pool 50/10/3 — conforme ✅
- **HTTP clients externes** : default → cf. PERF-FINAL-B-05 (still open)

## Strong points backend

- Cursor pagination omniprésente sur les hot paths
- Context timeouts à 100% dans tous les repos (5s default, configurable)
- Subscription cache 60s TTL bien fait
- Pending events outbox `FOR UPDATE SKIP LOCKED` correct
- Audit logs append-only convention enforced via REVOKE m.124
- Batch query patterns sur listings (`GetTotalUnreadBatch`, `ListMilestonesForProposals`, `GetProfileSkillsBatch`, `GetParticipantNamesBatch`)
- WS hub `SendToUser` non-bloquant (sendOrDrop) ✅
- 5 cache adapters spécialisés : profile, expertise, freelance_profile, skill_catalog, subscription
- pending_events stale recovery (m.128) prevents stuck rows after worker crash

---

# WEB (Next.js 16) + ADMIN (Vite)

## HIGH (5)

### PERF-FINAL-W-01 : `payment-info/page.tsx` fetch + polling dans `useEffect` au lieu de TanStack Query
- **Severity**: HIGH
- **Location** : `web/src/app/[locale]/(app)/payment-info/page.tsx:1-405`. Trois `useEffect` côté ligne 94+ fetchent + polling 10s sans dedupe, callbacks closures avec deps suspectes.
- **Why it matters** : anti-pattern explicite CLAUDE.md. Pas de cache, double-fetch sur mount strict-mode, race conditions si la page démonte pendant un fetch in-flight (memory leak).
- **How to fix** : `useQuery({ queryKey: ['payment-info', orgId], queryFn: ..., refetchInterval: mode === 'onboarding' ? 10000 : false })`. Le hook `useAccountStatus` doit vivre dans `features/payment-info/hooks/use-account-status.ts` (pas dans `app/`).
- **Test required** : `payment-info-page.test.tsx` — render avec MSW mocking `/api/v1/payment-info/account-status`, asserter qu'en mode `onboarding` la query refetch toutes les 10s, qu'en mode `dashboard` elle ne refetch pas.
- **Effort** : M (½j)

### PERF-FINAL-W-02 : 27 `<img>` raw au lieu de `next/image` (était 7 dans l'audit précédent — régression)
- **Severity**: HIGH
- **Location** : 27 sites identifiés via `grep -rn "<img" web/src/ --include="*.tsx" | grep -v __tests__`. Pires offenders :
  - `web/src/shared/components/ui/profile-identity-header.tsx:115` — composant réutilisé partout
  - `web/src/features/provider/components/portfolio-item-card.tsx:49, 64` — listings
  - `web/src/features/provider/components/profile-header.tsx:100, 126` — profile pages
  - `web/src/features/provider/components/provider-card.tsx:88`, `referrer-profile-card.tsx:84`
- **Why it matters** : pas d'AVIF/WebP auto, pas de lazy loading optimisé, pas de redimensionnement responsive. CLAUDE.md ligne 200. LCP des listings publics impacté.
- **How to fix** : `<Image>` avec `width`/`height` ou `fill`+`sizes`. Garder `unoptimized` SEULEMENT pour les avatars MinIO (URL pre-signée temporaire) — et même là, considérer un proxy `/api/v1/media/{id}` pour bénéficier du cache CDN Next.js.
- **Test required** : Playwright e2e `lighthouse-listings.spec.ts` qui audit `/agencies` et asserte `LCP < 2.5s` sur les images.
- **Effort** : S (1-2h)

### PERF-FINAL-W-03 (was PERF-W-08) : 29/51 pages déclarent `"use client"` — over-hydration
- **Severity**: HIGH
- **Location** : 29 occurrences identifiées via `grep -rn '"use client"' web/src/app/`. Pages dashboard, profile, projects, search, referral, wallet, etc.
- **Why it matters** : chaque `"use client"` au niveau page hydrate tout le sous-arbre, augmente le JS initial route, empêche le streaming serveur partiel. La page `wallet/page.tsx` à elle seule charge ~150KB de JS pour un user qui veut juste voir son solde.
- **How to fix** : descendre la limite `"use client"` au composant interactif. Le shell de page (header, breadcrumbs, layout) reste RSC. Pattern : `page.tsx` devient un Server Component qui import un `<WalletPageClient />` `"use client"`.
- **Test required** : bundle analyzer report avant/après — cible -30KB JS initial sur les 5 plus gros routes.
- **Effort** : M (½j)

### PERF-FINAL-W-04 (was PERF-W-12) : `staleTime: Infinity` + `gcTime: Infinity` sur team permissions
- **Severity**: HIGH
- **Location** : `web/src/features/team/hooks/use-team.ts:94-95, 218`
- **Why it matters** : permissions et workspace cachés indéfiniment → mutations invisibles, cache jamais invalidée. Si l'admin change le rôle d'un user, l'UI du target user reste stale jusqu'à hard refresh. Bug de droit d'accès.
- **How to fix** : `staleTime: 30_000` (30s) + invalidation explicite `queryClient.invalidateQueries({ queryKey: ['team', 'permissions'] })` dans les `onSuccess` des mutations role-overrides.
- **Test required** : test handler — modifie role A→B, asserter le hook refetch et reflect le nouveau rôle dans <500ms.
- **Effort** : S (1-2h)

### PERF-FINAL-W-05 (was PERF-W-15) : `experimental.optimizePackageImports` incomplet
- **Severity**: HIGH (bundle-size impact)
- **Location** : `web/next.config.ts`
- **Why it matters** : `next-intl` (~80KB), `@stripe/react-stripe-js` (~120KB), `@stripe/react-connect-js` (~150KB) ne sont pas dans la liste — Next.js ne tree-shake pas leurs imports nominaux.
- **How to fix** : `experimental.optimizePackageImports: ['next-intl', '@stripe/react-stripe-js', '@stripe/react-connect-js', 'lucide-react', '@tanstack/react-query']`.
- **Test required** : bundle analyzer avant/après — cible -100KB sur la home.
- **Effort** : XS (30 min)

## MEDIUM (8)

### PERF-FINAL-W-06 (was PERF-W-16) : `loadStripe` au module level
- **Severity**: MEDIUM
- **Location** : `web/src/shared/lib/stripe-client.ts:23` — `export const stripePromise = loadStripe(publishableKey)` exécuté à l'import.
- **Why it matters** : Stripe.js se charge sur TOUTES les routes qui import depuis ce module, même celles qui n'utilisent pas Stripe. ~30KB ajoutés au bundle initial.
- **How to fix** : convertir en lazy fonction `getStripe()` qui memoise le résultat (pattern singleton). Les call sites changent peu : `stripePromise` → `getStripe()`.
- **Test required** : N/A — bundle test.
- **Effort** : XS (30 min)

### PERF-FINAL-W-07 (was PERF-W-17) : `useDebouncedValue` dupliqué
- **Severity**: MEDIUM (DRY)
- **Location** : `web/src/features/skill/hooks/use-debounced-value.ts` ET `web/src/shared/lib/search/use-debounced-value.ts`
- **Why it matters** : règle de trois : 2× n'est pas un problème, mais les fichiers ont divergé sur le nom et la signature.
- **How to fix** : déplacer en `web/src/shared/hooks/use-debounced-value.ts`, supprimer les copies, mettre à jour les imports.
- **Effort** : XS (15 min)

### PERF-FINAL-W-08 (was PERF-W-18) : Hex couleurs marque dupliqués 3 fois
- **Severity**: MEDIUM (DRY)
- **Location** : 3 sites pour LinkedIn (#0A66C2), Instagram (#E4405F), YouTube (#FF0000)
- **How to fix** : `web/src/shared/lib/social-brand-colors.ts` exporte un mapping `Record<SocialPlatform, string>`.
- **Effort** : XS (10 min)

### PERF-FINAL-W-09 (was PERF-W-19) : Aucun `priority` prop sur les images LCP candidates
- **Severity**: MEDIUM
- **Location** : `web/src/features/search/components/search-result-card.tsx:102` (et autres listings publics)
- **Why it matters** : LCP des listings est typiquement la première image au-dessus du fold. Sans `priority`, Next.js lazy-load — Web Vitals dégradés.
- **How to fix** : `<Image priority={index < 2} ... />` dans la carte (le parent passe l'index).
- **Effort** : XS (15 min)

### PERF-FINAL-W-10 (was PERF-W-20) : 209/385 boutons sans `aria-label` (54%)
- **Severity**: MEDIUM (accessibilité — WCAG 2.1 AA)
- **Location** : audit ciblé via `grep -rn "<button" web/src/features/ web/src/shared/components/`
- **Why it matters** : beaucoup ont du contenu textuel et donc OK, mais ~30 sont icon-only (close X, dropdown chevron, action icons). Ces 30 sont les vrais offenders.
- **How to fix** : 
  1. Installer `eslint-plugin-jsx-a11y` (gate strict).
  2. Audit ciblé via le linter.
  3. Ajouter `aria-label="Close"` etc. là où requis. Préférer un composant `<IconButton aria-label="..." icon={<X />} />` réutilisable.
- **Test required** : Playwright a11y `axe` scan sur les 5 pages principales, fail si 0 violations.
- **Effort** : S (1-2h)

### PERF-FINAL-W-11 (was BUG-NEW-13, regression PR #41) : Admin `<Suspense>` wraps `<Routes>` — flash de layout
- **Severity**: MEDIUM (UX)
- **Location** : `admin/src/app/router.tsx:96-122`
- **Why it matters** : `<Suspense fallback={<RouteSkeleton />}>` au-dessus de `<Routes>` inclut `<AdminLayout />`. Navigation /users → /jobs → fallback remplace TOUT le layout pendant le download du chunk. Sur réseau lent c'est très visible.
- **How to fix** : déplacer `<Suspense>` à l'intérieur de `AdminLayout` autour du `<Outlet />` :
```tsx
<main className="flex-1 overflow-y-auto bg-gray-50/50 p-6">
  <Suspense fallback={<RouteSkeleton />}>
    <Outlet />
  </Suspense>
</main>
```
- **Test required** : Playwright `admin-navigation.spec.ts` — throttle réseau "Slow 3G", navigate /users → /jobs, asserter le sidebar reste présent dans le DOM continuellement.
- **Effort** : XS (10 min)

### PERF-FINAL-W-12 (was BUG-NEW-12, regression PR #41) : RSC public listings fall back to `localhost:8080`
- **Severity**: MEDIUM
- **Location** : `web/src/features/provider/api/search-server.ts:64`
- **Why it matters** : `${API_BASE_URL || "http://localhost:8080"}/api/v1/search?...` — le port backend dev réel est **8083** (per memory `feedback_backend_port.md`). En toute env où `API_BASE_URL` est unset, fetch hits dead port → SEO listings render empty (silent because `try { } catch { return null }`).
- **How to fix** : changer fallback à `http://localhost:8083` OU throw on missing `API_BASE_URL` au build time.
- **Test required** : test `search-server.test.ts` — unset `API_BASE_URL`, asserter le fetch fails loud (build-time error) plutôt que silent return.
- **Effort** : XS (10 min)

### PERF-FINAL-W-13 (NEW) : `app/[locale]/(app)/payment-info/components/` viole "app/ is for routing only"
- **Severity**: MEDIUM (architecture)
- **Location** : `web/src/app/[locale]/(app)/payment-info/components/` (6 .tsx) + `lib/` à l'intérieur
- **Why it matters** : viole CLAUDE.md ligne 274 ("app/ is for routing only"). Le dossier devrait être `web/src/features/payment-info/components/`. Difficile à découvrir, mauvais signal d'organisation.
- **How to fix** : `git mv web/src/app/[locale]/(app)/payment-info/components/ web/src/features/payment-info/components/`. Mettre à jour les imports dans `page.tsx`.
- **Effort** : S (1-2h, dont 1h de tests + import paths)

## LOW (5)

- **PERF-FINAL-W-14** : 12 inline `style={{}}` (la moitié dynamiques OK ; statiques à extraire en classes Tailwind). `chat-widget-panel.tsx:219` `height: "calc(100vh - 100px)"` statique → `h-[calc(100vh-100px)]`.
- **PERF-FINAL-W-15** : Hardcoded text dans placeholders/options (i18n leak) — `billing-profile-form.tsx:198,537`, `referral/provider-picker.tsx:216`, `referral-creation-form.tsx:198,209`.
- **PERF-FINAL-W-16** : Hiérarchie h1→h3 cassée sur la home — pose un `<h2>` au-dessus de la grille features.
- **PERF-FINAL-W-17** : Pas de `@next/bundle-analyzer` — installer + wrapper `withBundleAnalyzer({ enabled: process.env.ANALYZE === "true" })`.
- **PERF-FINAL-W-18** : `unoptimized` partout sur les avatars (`messaging`, `chat-widget`, `search-result-card`) — supprimer et tester.

## Strong points web/admin

- Route groups `(app)`, `(auth)`, `(public)` propres
- `next/font/google` avec variables (Geist/Geist_Mono, pas de FOIT)
- TanStack Query defaults raisonnables (staleTime 2min, gcTime 10min, retry intelligent)
- Suspense boundaries sur `account/page.tsx` et `search/page.tsx`
- `generateStaticParams` + `hasLocale` pour SSG locales
- ISR (`next: { revalidate: 120 }`) sur fetch métadonnées profile
- Dynamic imports `ChatWidget`, `IncomingCallOverlay`, `CallOverlay` ✅
- Profile pages `freelancers/[id]`, `referrers/[id]`, `clients/[id]` = modèles SEO/RSC corrects
- Typesense client maison (évite la lib npm de 200+ KB)
- Strict TypeScript, named exports, design tokens via `@theme`
- **Admin = exemplaire** : 0 cross-feature, 0 `any`, 0 fichier > 600, design system propre. Module de référence — c'est ce niveau qu'il faut atteindre côté web.

---

# MOBILE (Flutter)

## HIGH (4)

### PERF-FINAL-M-01 (was PERF-M-01) : ConsumerWidget root + 0 `.select()` dans tout le repo
- **Severity**: HIGH
- **Location** : `wallet_screen.dart`, `profile_screen.dart`, `chat_screen.dart`, `proposal_detail_screen.dart`, etc. — tout le code.
- **Why it matters** : `ref.watch(provider)` du root rebuild tout le sous-arbre à chaque tick (typing/auth/WS push). ProfileScreen watch 3 providers + 10 keys d'AsyncValue → 250 lignes de Widget recompilées. **Cause #1 de jank**.
- **How to fix** : `ref.watch(provider.select((s) => s.specificField))` partout. Ou pattern : `ConsumerStatelessWidget` root + sub-widgets `ConsumerWidget` feuille qui watch chacun un slice.
- **Test required** : widget test avec `WidgetTester.idle()` + `findsNWidgets` après mutation d'un provider — asserter que seuls les widgets feuille rebuild.
- **Effort** : L (3 jours, c'est du systematic refactoring)

### PERF-FINAL-M-02 (was PERF-M-02) : `app_router.dart` charge 45 écrans en imports synchrones
- **Severity**: HIGH
- **Location** : `mobile/lib/core/router/app_router.dart:1-433`. Seulement 3 `deferred` imports identifiés.
- **Why it matters** : tout le graphe d'écrans dans le bundle initial. Cold start +200-500ms ; aucun `import deferred` pour wallet, proposal, portfolio, billing, subscription, dispute. CLAUDE.md le demande.
- **How to fix** : `import 'package:.../wallet_screen.dart' deferred as wallet;` puis `await wallet.loadLibrary()` dans le route builder. Le pattern est documenté Flutter — économie typique 1-3MB sur les bundles non-critiques.
- **Test required** : asserter `Devtools` "deferred libraries" tab montre wallet/proposal/portfolio chargés à la 1ère navigation, pas au boot.
- **Effort** : M (½j)

### PERF-FINAL-M-03 (was PERF-M-14) : 196 `dynamic` hors `Map<String, dynamic>` et généré
- **Severity**: HIGH
- **Location** : Concentré dans `data/` repos Dio (`_api.get<dynamic>`).
- **Why it matters** : le projet a Freezed + json_serializable précisément pour éviter ça. `authState.user?['display_name']` runtime check 3× plus lent que classes Freezed. Plus tout l'avantage type-safety perdu.
- **How to fix** : générer des DTOs Freezed pour chaque réponse API (pattern existe dans `data/`). Le ApiClient doit retourner `T` typé, pas `dynamic`.
- **Test required** : par feature migrée, asserter qu'`fluter analyze` est clean + un test parsing la réponse JSON typée.
- **Effort** : L (3 jours)

### PERF-FINAL-M-04 (was PERF-M-15) : 3 fichiers > 600 lignes (était 17, gros progrès)
- **Severity**: HIGH (downgrade depuis CRITICAL — la majorité a été splittée)
- **Location** : `mobile/lib/features/job/presentation/screens/create_job_screen.dart` (593, borderline OK), `chat/message_input_bar.dart` (545), `search/search_result_card.dart` (536). Aucun > 600 LOC restant ! Mais 3 fichiers à 530-595 méritent un split anticipé.
- **Why it matters** : 600 est la limite ; à 590, on est à 1 PR de le dépasser. Préventivement.
- **How to fix** : extraire des sub-widgets nommés. `create_job_screen.dart` → `CreateJobForm`, `JobDetailsSection`, `JobBudgetSection`, etc.
- **Effort** : M (½j)

## MEDIUM (8)

### PERF-FINAL-M-05 (was PERF-M-09) : `Image.network` brut dans portfolio_form_sheet
- **Severity**: MEDIUM
- **Location** : `portfolio_form_sheet.dart:587, 589`
- **Why it matters** : re-download à chaque rebuild. `CachedNetworkImage` est utilisé partout ailleurs.
- **How to fix** : remplacer par `CachedNetworkImage` avec `memCacheWidth` adapté à la taille rendue.
- **Effort** : XS (10 min)

### PERF-FINAL-M-06 (was PERF-M-10) : Pas de retry/exponential backoff sur Dio
- **Severity**: MEDIUM (network resilience)
- **Location** : `mobile/lib/core/network/api_client.dart` (sauf WS)
- **Why it matters** : 1 timeout = 1 erreur user-facing. Network mobile est intrinsèquement flaky.
- **How to fix** : `dio_smart_retry` (3 retries avec backoff exponentiel, retry only on 5xx + network errors, jamais sur 4xx). Ou interceptor maison.
- **Effort** : S (1-2h)

### PERF-FINAL-M-07 (was PERF-M-11) : `messagingWsService.events` stream — 5+ listeners
- **Severity**: MEDIUM
- **Location** : `mobile/lib/features/messaging/data/messaging_ws_service.dart`
- **Why it matters** : 5+ listeners simultanés invalident leurs providers en cascade sur chaque push WS. Risk de cascade rebuild.
- **How to fix** : un seul listener "router" qui dispatche vers les providers concernés via Riverpod (pas un bus d'events).
- **Effort** : S (1-2h)

### PERF-FINAL-M-08 (was PERF-M-13) : Pas d'`IndexedStack` ni `wantKeepAlive`
- **Severity**: MEDIUM (UX + perf)
- **Location** : tab navigation `app_router.dart`
- **Why it matters** : switch tab détruit l'écran et refetch tous ses providers, scroll perdu sur Messaging.
- **How to fix** : `StatefulShellRoute.indexedStack` (GoRouter natif) maintient les écrans en mémoire.
- **Effort** : S (1-2h)

### PERF-FINAL-M-09 (was PERF-M-19) : 13 `ref.read` dans `build()` — anti-pattern Riverpod
- **Severity**: MEDIUM
- **Location** : 13 sites identifiés (search_screen, opportunity_detail, team_screen, freelance_profile×2, client_profile, call_screen, profile_screen×2, notification_screen, referral_dashboard).
- **Why it matters** : casse la réactivité. Le widget ne rebuild pas quand le provider mute.
- **How to fix** : `ref.watch` si la valeur doit déclencher rebuild. `ref.read` réservé aux callbacks (`onPressed: () { ref.read(notifier).doSomething(); }`).
- **Effort** : S (1-2h)

### PERF-FINAL-M-10 (was PERF-M-20) : `chat_screen.dart` Dio standalone bypass auth interceptor
- **Severity**: MEDIUM
- **Location** : `chat_screen.dart`
- **Why it matters** : crée un Dio avec timeouts hardcodés (30s/120s) qui bypass le auth interceptor → bug latent (non-401 sur token expiré).
- **How to fix** : utiliser le Dio singleton via Riverpod Provider.
- **Effort** : XS (15 min)

### PERF-FINAL-M-11 (was PERF-M-17) : `flutter_inappwebview ^6.1.5` (~6-10 MB APK)
- **Severity**: MEDIUM (APK size, target <30MB)
- **Location** : `mobile/pubspec.yaml`
- **Why it matters** : ~6-10 MB d'APK pour 1 seul écran subscription checkout.
- **How to fix** : évaluer `url_launcher` external (open dans le browser système) — UX dégradée mais APK -8MB. Ou defer la lib via macros build conditionnels (faisable côté Android via `compileSdk` flavors).
- **Effort** : M (½j eval + impl si décidé)

### PERF-FINAL-M-12 (was PERF-M-18) : `record_linux ^1.0.0` dans dependency_overrides
- **Severity**: MEDIUM (cleanup)
- **Location** : `mobile/pubspec.yaml`
- **Why it matters** : Linux n'est pas une cible déclarée du projet. Dependency morte.
- **How to fix** : retirer.
- **Effort** : XS (5 min)

## LOW (5)

- **PERF-FINAL-M-13** : `ListView.builder` sans `cacheExtent` — défaut OK mais à mesurer.
- **PERF-FINAL-M-14** : Pas de pull-to-refresh sur 14/25 écrans listes — UX gap.
- **PERF-FINAL-M-15** : Pas de config globale `imageCache.maximumSizeBytes` — défaut 100 MB OK.
- **PERF-FINAL-M-16** : Pas de bench Flutter DevTools documenté — capturer les timelines cold start / scroll messaging dans `docs/perf/`.
- **PERF-FINAL-M-17** : Pas de `flutter_svg` precache pour les illustrations.

## Strong points mobile

- CachedNetworkImage adopté majoritairement (11 fichiers, 5 occurrences avec placeholder + errorWidget)
- Dio singleton via Riverpod Provider — pas de Singleton statique
- Token refresh interceptor avec Dio fresh-instance (évite loops)
- WebSocket service heartbeat 30s + reconnexion exponentielle + AppLifecycleListener — excellent pattern
- Pagination cursor-based parité backend (search_provider, proposal_repository_impl)
- Debouncing dans location_section et create_proposal_screen (fee preview)
- AnimationController disposal vérifié sur les 3 occurrences
- Generated code propre (0 .freezed.dart en repo, build_runner standard)
- Firebase deferred + post-frame FCM init (PR #41 cat F) ✅
- 17 RepaintBoundary placés (PR #41 cat E)
- 7 memCacheWidth placés (PR #41 cat D)

---

# Top 15 fixes prioritaires (cross-stack, ordered by ROI)

| # | ID | Severity | Effort | Impact |
|---|---|---|---|---|
| 1 | PERF-FINAL-B-14 | MEDIUM→HIGH | L (3j) | Pre-prod RLS rotation blocker |
| 2 | PERF-FINAL-B-01 | HIGH | XS | Slowloris DoS protection |
| 3 | PERF-FINAL-B-05 | HIGH | XS | -50-100ms par recherche |
| 4 | PERF-FINAL-B-04 | HIGH | S | Slow query observability |
| 5 | PERF-FINAL-W-12 | MEDIUM | XS | RSC listings actually work |
| 6 | PERF-FINAL-W-11 | MEDIUM | XS | Admin layout no flash |
| 7 | PERF-FINAL-B-02 | HIGH | S | Wallet endpoint DoS protection |
| 8 | PERF-FINAL-W-02 | HIGH | S | LCP listings publics |
| 9 | PERF-FINAL-B-06 | HIGH | M | -100-200ms checkout |
| 10 | PERF-FINAL-W-01 | HIGH | M | Payment-info anti-pattern fix |
| 11 | PERF-FINAL-W-03 | HIGH | M | -30KB JS initial routes |
| 12 | PERF-FINAL-B-03 | HIGH | M | Cache infrastructure |
| 13 | PERF-FINAL-M-02 | HIGH | M | Mobile cold start -300ms |
| 14 | PERF-FINAL-W-04 | HIGH | S | Team UI permissions stale fix |
| 15 | PERF-FINAL-W-05 | HIGH | XS | Bundle -100KB |

**Bundle « 1 semaine »** = items 1-15 = transformation mesurable des KPIs (LCP web, cold start mobile, p50 backend, OOM resilience).

---

## Summary

| Layer | HIGH | MEDIUM | LOW |
|---|---|---|---|
| Backend + DB | 6 | 11 | 6 |
| Web + Admin | 5 | 8 | 5 |
| Mobile | 4 | 8 | 5 |
| **Total** | **15** | **27** | **16** |
