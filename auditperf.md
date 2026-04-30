# Audit de Performance

**Date** : 2026-04-30 (mise à jour post Phases 1-5Q ; audit précédent : 2026-04-29)
**Branche** : `main` @ `c8284526`
**Périmètre** : backend Go (~860 fichiers, 125 migrations) + DB Postgres + web Next.js + admin Vite + mobile Flutter

## Méthodologie

Audit statique sans build ni exécution. Lecture de tous les hot paths : messaging, search, proposals/payments, uploads, WebSocket, webhooks Stripe. Vérification des indexes par migration et des plans probables. Bundle web mesuré via les chunks dev générés. Mobile audité côté rebuilds, lists, images, cold start.

---

# BACKEND + DB

## HIGH (8)

### ~~PERF-B-01 : ParseMultipartForm(100MB)~~ closed in PR #34 (`6d6dd20c fix(security): stream multipart uploads via MultipartReader`)

### PERF-B-02 : N+1 sur la liste des propositions actives
- **Location** : `backend/internal/handler/proposal_handler.go:527-530` (`GetParticipantNames`)
- **Pattern** : pour chaque proposal listée, 2 `users.GetByID` séquentiels (client + provider). Page de 20 = 40 queries supplémentaires (~80-200ms p50).
- **Impact** : utilisé par tous les dashboards. `ListMilestonesForProposals` est correctement batché — la régression `GetParticipantNames` casse le bénéfice.
- **Fix** : ajouter `UserRepository.GetByIDs(ctx, []uuid.UUID)` (le helper batch existe dans `team_handler.go:147`). 1 seul `WHERE id = ANY($1)`.

### PERF-B-03 : `payment_records.ListByOrganization` sans LIMIT ni cursor
- **Location** : `backend/internal/adapter/postgres/payment_record_repository.go:203-242`
- **Pattern** : pas de pagination, pas de cursor. `ORDER BY created_at DESC` non borné + LEFT JOIN `users`. Une agence active sur 12-24 mois = 10k-50k lignes par requête wallet. ~10 MB RAM Go par requête.
- **Fix** : cursor pagination `(created_at, id)` standard, `LIMIT 50+1`. Endpoint dédié export pour les rapports historiques.

### PERF-B-04 : `CreateBatch` milestones fait des INSERTs en boucle
- **Location** : `backend/internal/adapter/postgres/milestone_repository.go:36-65`
- **Pattern** : la fonction s'appelle `CreateBatch` mais émet N `INSERT` séquentiels dans la tx. 5 milestones/proposal × 2 ms RTT local = 10 ms gaspillés ; 20-40 ms cross-AZ.
- **Fix** : `INSERT INTO milestones (...) VALUES ($1,$2,...), ($n,...)` construit dynamiquement, ou `pq.CopyIn`. Pattern existe déjà dans `expertise_repository.go`.

### PERF-B-05 : `service.CacheService` n'existe pas — pas de cache-aside
- **Location** : `backend/internal/port/service/` (interface absente). Seuls usages Redis : rate limiter, sessions, idempotency Stripe, brute force, subscription cache. Aucun cache pour profils publics, listings, skills, agrégats reviews.
- **Pattern** : CLAUDE.md spec'e un tableau de TTLs (5min profils, 2min jobs publics, 1h skills, 10min reviews aggregate). Inexistant.
- **Impact** : SEO public hit `/api/v1/profiles/{id}` à froid → 600 lookups/h pour le même profil populaire. Skills catalog fetché à chaque requête.
- **Fix** : port `service.CacheService { Get/Set/Delete }` + `adapter/redis/cache.go`. L'injecter dans `profileapp.GetPublicByOrg`, `skillapp.GetCuratedByExpertise`, `jobapp.ListPublic`, `reviewapp.GetAggregateForOrg`. Invalider sur write.

### PERF-B-06 : Slow-query logger inexistant
- **Location** : `query_logger.go` absent. CLAUDE.md prescrit son implémentation comme outil monitoring obligatoire (lignes 1166-1181).
- **Impact** : régressions de plan invisibles en prod (changement post-VACUUM, index manquant introduit par un nouveau filtre).
- **Fix** : wrapper `*sql.DB` ou `pkg/dbx.QueryContext` qui mesure et émet `slog.Warn("slow query", ...)` au-delà de 50 ms. Centralisé.

### PERF-B-07 : `WriteTimeout: 0` + pas de `ReadHeaderTimeout` — Slowloris
- **Location** : `backend/cmd/api/main.go:1370-1374`
- **Pattern** : timeouts illimités côté HTTP server.
- **Impact** : connexion ouverte indéfiniment via headers byte-par-byte. Avec `MaxOpenConns=50`, saturation triviale du pool DB.
- **Fix** : `ReadHeaderTimeout = 5*time.Second`, `WriteTimeout = 60*time.Second`. Pour SSE/WS, override local via `http.NewResponseController(w).SetWriteDeadline`.

### PERF-B-08 : Colonnes `*_organization_id` dénormalisées (migration 115) jamais utilisées
- **Location** : `backend/internal/adapter/postgres/proposal_queries.go:115, 130`, `payment_record_repository.go:207-217`
- **Pattern** : `WHERE p.organization_id = $1 OR provider_user.organization_id = $1` → BitmapOr planner, nested loop. À 100k rows : 50-150 ms p50. La migration 115 a ajouté `provider_organization_id` exprès — non utilisé.
- **Fix** : `WHERE (organization_id = $1 OR provider_organization_id = $1)` + index composite `(provider_organization_id, status, created_at DESC, id DESC)`. Quick win majeur.

## MEDIUM (12)

### PERF-B-09 : Tous les clients HTTP externes utilisent `http.DefaultTransport` (`MaxIdleConnsPerHost=2`)
- **Location** : `internal/search/client.go:96`, `adapter/openai/client.go:46`, `adapter/anthropic/analyzer.go:53`, `adapter/vies/client.go:91`, `adapter/nominatim/client.go:38`
- **Impact** : sous burst (100 hits Typesense/s), 50-100 ms gaspillés par cold connection TLS. Typesense particulièrement touché.
- **Fix** : `pkg/httpx.NewTunedClient(timeout)` avec `MaxIdleConnsPerHost: 50, MaxConnsPerHost: 100, IdleConnTimeout: 90s, ForceAttemptHTTP2: true`.

### PERF-B-10 : `ConversationRepository.ListConversations` — 2 LATERAL JOIN par row
- **Location** : `backend/internal/adapter/postgres/conversation_queries.go:60-141`
- **Pattern** : double LATERAL pour conversation_participants → users + last_message. À 10k+ conversations, 30-50 ms par page.
- **Fix** : dénormaliser `last_message_seq + last_message_content + last_message_at` directement sur `conversations`, maintenu par trigger ou app au moment de l'INSERT messages.

### PERF-B-11 : `GetTotalUnread` n'utilise pas le partial index
- **Location** : `backend/internal/adapter/postgres/conversation_queries.go:281-284`
- **Pattern** : `SUM(unread_count) WHERE user_id = $1`. Index partiel `WHERE unread_count > 0` (migration 074) existe mais le filtre n'est pas dans la query.
- **Fix** : ajouter `AND unread_count > 0` dans le WHERE — le SUM reste correct (rows à 0 contribuent 0).

### ~~PERF-B-12 : Notification worker single-threaded + time.Sleep~~ closed in PR #36 (`3dbbf747 fix(notification/worker): parallel pool + non-blocking re-enqueue`)

### PERF-B-13 : Notification worker — `getPrefs` + `users.GetByID` à chaque job
- **Location** : `backend/internal/app/notification/worker.go:99,193,211-222`
- **Impact** : 2 queries DB par notif × 1000 notifs/min = 2000 queries/min purement administratives.
- **Fix** : cache LRU local 2 min TTL taille 1000 sur `users.GetByID(userID)` et `notifs.GetPreferences(userID)`.

### PERF-B-14 : Stripe Connect `account.GetByID` jamais caché
- **Location** : `backend/internal/adapter/stripe/account.go:156,306,370,381,390,420,450`
- **Pattern** : appel Stripe externe 100-300 ms sur chaque endpoint vérifiant ChargesEnabled/PayoutsEnabled. Au checkout, 2-3 appels consécutifs.
- **Fix** : `redis.AccountStatusCache` TTL 60-120s sur les flags. Invalider sur webhook `account.updated`. KYC snapshot complet 5 min.

### PERF-B-15 : Stripe SDK appelé sans propager `ctx`
- **Location** : `backend/internal/adapter/stripe/account.go:306,370,381,390,420,450,156`
- **Pattern** : signature accepte `ctx` mais `account.GetByID(accountID, nil)` ignore. Annulation impossible.
- **Fix** : `account.GetByIDWithContext(ctx, ...)` (stripe-go v82) ou `params.Context = ctx`.

### ~~PERF-B-16 : WS sendOrDrop~~ closed in PR #40 (`7306f055 fix(ws): non-blocking sendOrDrop + race-safe wasLast on disconnect`)

### PERF-B-17 : Search worker tick = 30 s — embeddings différés trop longuement
- **Location** : `backend/internal/adapter/worker/worker.go:86-87`
- **Impact** : profil mis à jour, index Typesense obsolète ~30s.
- **Fix** : tick 5s + signal Redis pubsub `wake_worker` que le worker écoute en parallèle pour réagir immédiatement après INSERT pendingevent.

### PERF-B-18 : LTR capture INSERT par recherche
- **Location** : `backend/internal/app/searchanalytics/ltr_capture.go:102-111`
- **Impact** : 100 recherches/s = 100 UPDATE/s additionnels.
- **Fix** : batcher via channel buffered + flush 1×/seconde.

### PERF-B-19 : Indexer fan-out 9 goroutines × N actors sans cap
- **Location** : `backend/internal/search/indexer.go:314-364`
- **Pattern** : 9 reads concurrents par actor reindexed. Reindex full = potentiellement 90k goroutines en rafale.
- **Fix** : `errgroup.SetLimit(3)` au niveau actor (3 actors parallèles × 9 reads = 27 connections max).

### PERF-B-20 : Stripe webhook handlers synchrones inline
- **Location** : `backend/internal/handler/stripe_handler.go:149-191`
- **Pattern** : `handleInvoicePaid` peut faire 5+ DB writes + PDF generation (chromedp 2-5s) + email synchrones. Stripe peut retry sur timeout.
- **Fix** : enqueue dans `pending_events`, retourner 200 immédiatement à Stripe.

## LOW (10)

- **PERF-B-21** : OFFSET pagination dans 8 admin endpoints (media, review_admin, moderation, user, job_admin, conversation_admin, job_application_admin) — toléré jusqu'à 50k rows
- **PERF-B-22** : `IncrementUnreadForRecipients` fan-out org × org INSERTs — limite à 5-10 membres/org acceptable
- **PERF-B-23** : WS hub `register/unregister` channel taille 64 — bumper à 1024 pour deploy / network changes
- **PERF-B-24** : Pas d'index `(provider_organization_id, completed_at DESC) WHERE status='completed'` sur proposals
- **PERF-B-25** : `INSERT (SELECT FROM organization_members WHERE user_id=$X LIMIT 1)` à chaque création — 1 index seek par INSERT, négligeable
- **PERF-B-26** : Migration 074 backfill DO $$ block monolithique — pour les futures migrations bulk-copy splitter en chunks
- **PERF-B-27** : Slot WS `register` bumper à 1024
- **PERF-B-28** : Cardinality control logs — `user_id` doit rester attribute slog jamais label Prometheus
- **PERF-B-29** : `idx_search_queries_search_id` UNIQUE peut bloquer inserts concurrents sur hot search_id — négligeable
- **PERF-B-30** : `ListPaymentRecords` LEFT JOIN inutilement large — `SELECT DISTINCT` ou pattern UNION

## Index audit (table-by-table)

| Table | Migration(s) | Existing indexes | Likely missing | Priority |
|---|---|---|---|---|
| proposals | 008, 062, 115 | (conversation_id), (sender_id), (recipient_id), (client_id), (provider_id), (org) partial, (org, status, created_at) | `(provider_organization_id, status, created_at DESC, id DESC)` partial | **HIGH** (cf. PERF-B-08) |
| payment_records | 018, 064, 086 | (proposal_id), (client_id), (provider_id), (stripe_pi_id) partial, (org_id) | `(provider_organization_id, created_at DESC, id DESC)` | **MEDIUM** |
| audit_logs | 078 | (user_id) partial, (action), (created_at), (resource_type, resource_id) partial | composite `(user_id, created_at DESC)` pour list-by-user paginé | LOW |
| subscriptions | 117, 119 | (organization_id) | composite `(organization_id, status)` | LOW |
| invoice | 121 | (recipient_org_id, issued_at, id), (stripe_pi_id) partial | `(recipient_org, source_type, issued_at DESC)` | LOW |
| messages, conversations, notifications, jobs, job_applications, organizations, search_queries, moderation_results | — | composites présents | none material | OK |

## Connection pools (vérifié)

- **Postgres** : MaxOpenConns=50, MaxIdleConns=25, ConnMaxLifetime=30min — conforme CLAUDE.md ✅
- **Redis** : pool 50/10/3 — conforme ✅
- **HTTP clients externes** : default → cf. PERF-B-09

## Strong points backend

- Cursor pagination omniprésente sur les hot paths
- Context timeouts à 100% dans tous les repos
- Subscription cache 60s TTL bien fait
- Pending events outbox `FOR UPDATE SKIP LOCKED` correct
- Audit logs append-only convention
- Batch query patterns sur listings (`GetTotalUnreadBatch`, `ListMilestonesForProposals`, `GetProfileSkillsBatch`)
- WS hub `SendToUser` non-bloquant correct

---

# WEB (Next.js 16) + ADMIN (Vite)

## HIGH (11)

### ~~PERF-W-01 : LiveKit lazy~~ closed in PR #41 (`b8f739a7 perf(web): lazy-load LiveKit via CallSlot boundary`)

### ~~PERF-W-02 : RSC public listings + JSON-LD~~ closed in PR #41 (`6f41131f perf(web): RSC public listings + JSON-LD for SEO`) — voir BUG-NEW-12 pour la régression API_BASE_URL

### ~~PERF-W-03 : loading/error/not-found/global-error + sitemap/robots~~ closed in PR #41 (`26ebb871 feat(web): loading/error/not-found boundaries` + `1dadac31 feat(web): dynamic sitemap.ts + robots.ts`)

### ~~PERF-W-04 : TestDB en prod home~~ closed in Phase 0 (`e9c9e325 chore(web): remove TestDB debug component from production home`)

### PERF-W-05 : `payment-info/page.tsx` fetch + polling dans `useEffect` au lieu de TanStack Query
- **Location** : `web/src/app/[locale]/(app)/payment-info/page.tsx:94-147`
- **Impact** : anti-pattern explicite CLAUDE.md. Pas de cache, polling 10s sans dedupe, callbacks closures avec deps manquantes (`useCallback` line 109/171 a `[]` mais lit `apiBase`, `mobileToken`, `authHeaders` — bug latent).
- **Fix** : `useQuery` avec `refetchInterval` conditionnel selon mode.

### PERF-W-06 : Métadonnées génériques sur des pages publiques importantes
- **Location** : `agencies/[id]/page.tsx:10-18` (titre fixe « Profil agence »), pas de `generateMetadata` sur les listings, pas de JSON-LD `JobPosting` sur `/opportunities/[id]` (pourtant Google for Jobs)
- **Impact** : indexation et CTR organiques. CLAUDE.md spec'e tout ligne par ligne.
- **Fix** : aligner agencies/[id] sur le pattern freelancers/[id] (déjà OK). Implémenter `JobPosting` + `BreadcrumbList`.

### PERF-W-07 : 7 fichiers utilisent `<img>` au lieu de `next/image`
- **Location** : `provider-card.tsx:87`, `freelance-profile-card.tsx:85`, `referrer-profile-card.tsx:83`, `candidate-card.tsx:68`, `candidate-detail-panel.tsx:191`, `upload-modal.tsx:259`, `profile-identity-header.tsx:114`
- **Impact** : pas d'AVIF/WebP auto, pas de lazy loading optimisé, pas de redimensionnement responsive. CLAUDE.md ligne 200.
- **Fix** : `<Image>` avec `width`/`height` ou `fill`+`sizes`. Garder `unoptimized` seulement si strictement nécessaire.

### PERF-W-08 : 28/51 pages déclarent `"use client"` — over-hydration
- **Location** : pages dashboard, profile, projects, search, referral, wallet, etc.
- **Impact** : chaque `"use client"` au niveau page hydrate tout le sous-arbre, augmente le JS initial route, empêche le streaming serveur partiel.
- **Fix** : descendre la limite `use client` au composant interactif ; le shell de page reste RSC.

### ~~PERF-W-09 : Cross-feature imports~~ closed in PR #37 (`refactor(web): move upload-api/expertise-editor/city-autocomplete/search-api to shared/`)

### ~~PERF-W-10 : Deps non utilisées~~ closed in Phase 0 (`528668cc chore(web): uninstall unused typesense and country-region-data deps`)

### ~~ADMIN-PERF-01 : Aucun lazy-loading des routes~~ closed in PR #41 (`edcc21da perf(admin): lazy routes + Vite manualChunks`) — voir BUG-NEW-13 pour la régression du fallback Suspense au-dessus de AdminLayout

## MEDIUM (10)

- **PERF-W-11** : 4 fichiers > 600 lignes (`wallet-page.tsx` 878, `message-area.tsx` 797, `search-filter-sidebar.tsx` 758, `billing-profile-form.tsx` 656) — chaque god component hydrate la page entière
- **PERF-W-12** : `staleTime: Infinity` + `gcTime: Infinity` sur `team` permissions/workspace (`use-team.ts:94-95, 218`) → mutations invisibles, cache jamais invalidée
- **PERF-W-13** : 12 inline `style={{}}` (la moitié sont dynamiques OK ; `chat-widget-panel.tsx:219` `height: "calc(100vh - 100px)"` statique → Tailwind `h-[calc(100vh-100px)]`)
- **PERF-W-14** : Hardcoded text dans placeholders/options (i18n leak) — `billing-profile-form.tsx:198,537`, `referral/provider-picker.tsx:216`, `referral-creation-form.tsx:198,209`
- **PERF-W-15** : `experimental.optimizePackageImports` incomplet — manque `next-intl`, `@stripe/react-stripe-js`, `@stripe/react-connect-js`
- **PERF-W-16** : `loadStripe` au module level (`stripe-client.ts:23`) — passer en `getStripe()` lazy-init
- **PERF-W-17** : `useDebouncedValue` dupliqué dans `features/skill/hooks/` ET `shared/lib/search/` — déplacer en `shared/hooks/`
- **PERF-W-18** : Hex couleurs marque dupliqués 3 fois (linkedin/instagram/youtube) — extraire en `shared/lib/social-brand-colors.ts`
- **PERF-W-19** : Aucun `priority` prop sur les images LCP candidates (`search-result-card.tsx:102`) — propager `priority={index < 2}`
- **PERF-W-20** : 209/385 boutons sans `aria-label` (54%) — beaucoup ont sans doute du contenu textuel mais audit ciblé via `eslint-plugin-jsx-a11y` recommandé

## LOW (7)

- **PERF-W-21** : Hiérarchie h1→h3 cassée sur la home — pose un `<h2>` au-dessus de la grille features
- **PERF-W-22** : Pas de `@next/bundle-analyzer` — installer + wrapper `withBundleAnalyzer({ enabled: process.env.ANALYZE === "true" })`
- **PERF-W-23** : `unoptimized` partout sur les avatars (`messaging`, `chat-widget`, `search-result-card`) — supprimer et tester
- **PERF-W-24** : Pas d'`eslint-plugin-jsx-a11y` configuré
- **PERF-W-25** : Admin sans plugin a11y/eslint
- **PERF-W-26** : Hooks coverage ~31% (27/88) — voir rapportTest.md
- **PERF-W-27** : Vite admin sans `build.target: "es2022"` ni `chunkSizeWarningLimit`

## Strong points web/admin

- Route groups `(app)`, `(auth)`, `(public)` propres
- `next/font/google` avec variables (Geist/Geist_Mono, pas de FOIT)
- TanStack Query defaults raisonnables (staleTime 2min, gcTime 10min, retry intelligent)
- Suspense boundaries sur `account/page.tsx` et `search/page.tsx`
- `generateStaticParams` + `hasLocale` pour SSG locales
- ISR (`next: { revalidate: 120 }`) sur fetch métadonnées profile
- Dynamic imports `ChatWidget`, `IncomingCallOverlay`, `CallOverlay` (mais neutralisés par PERF-W-01)
- Profile pages `freelancers/[id]`, `referrers/[id]`, `clients/[id]` = modèles SEO/RSC corrects à répliquer
- Typesense client maison (évite la lib npm de 200+ KB)
- Strict TypeScript, named exports, design tokens via `@theme`
- **Admin exemplaire** : 0 cross-feature, 0 `any`, 0 fichier > 600

---

# MOBILE (Flutter)

## HIGH (8)

### PERF-M-01 : ConsumerWidget root + 0 `.select()` dans tout le repo
- **Location** : `wallet_screen.dart:249`, `profile_screen.dart:40`, `chat_screen.dart:546`, `proposal_detail_screen.dart`
- **Impact** : `ref.watch` du root rebuild tout le sous-arbre à chaque tick (typing/auth/WS push). ProfileScreen watch 3 providers + 10 keys d'AsyncValue → 250 lignes de Widget recompilées. **Cause #1 de jank**.
- **Fix** : `ref.watch(provider.select((s) => s.field))` partout. Ou ConsumerStatelessWidget root + sub-widgets ConsumerWidget feuille.

### PERF-M-02 : `app_router.dart` charge 45 ecrans en imports synchrones
- **Location** : `mobile/lib/core/router/app_router.dart:1-62`
- **Impact** : tout le graphe d'écrans dans le bundle initial. Cold start +200-500ms ; aucun `import deferred` (`grep -c "deferred" = 0`).
- **Fix** : deferred imports pour wallet, proposal, portfolio, billing, subscription, dispute. CLAUDE.md le demande explicitement.

### PERF-M-03 : `Firebase.initializeApp()` synchrone avant `runApp()`
- **Location** : `mobile/lib/main.dart:14-15`
- **Impact** : bloque le splash 200-500 ms (iOS plus lent). Aucun fallback en cas d'échec.
- **Fix** : `unawaited(_initFirebase())` après `runApp()`. TTI cible < 1.5s.

### PERF-M-04 : `FCMService.initialize` dans `Future.microtask` du `build()`
- **Location** : `mobile/lib/core/router/app_router.dart:618-621`
- **Pattern** : `if (!_fcmInitialized) { _fcmInitialized = true; Future.microtask(...) }` dans build → effet de bord, anti-pattern Flutter. Risque double-init si build rappelé avant exécution microtask. Permission FCM bloque le premier frame interactif.
- **Fix** : `initState()` du shell stateful, ou `ref.listen(authProvider, ...)` dans un Provider dédié.

### PERF-M-05 : Avatars CachedNetworkImage sans `memCacheWidth/Height`
- **Location** : `search_result_card.dart:119-124`, `portfolio_grid_widget.dart:396,410,627`, `portfolio_detail_sheet.dart:99`
- **Impact** : avatars rendus à 48-64px décodent l'image originale pleine résolution en RAM. Grille portfolio 30 items 1080p = ~100 MB RAM perdue. **Cause #1 de pic mémoire** Android low-end.
- **Fix** : `memCacheWidth: 128, memCacheHeight: 128` (avatar), `memCacheHeight: 600` (cards). Ajouter `maxWidthDiskCache`.

### PERF-M-06 : 21 `ListView(children: [...])` non-builder, dont plusieurs sur listes variables
- **Location** : `team_screen.dart:196` (members.map sur 50+), `referral_dashboard_screen.dart:40`, `referral_detail_screen.dart:72`, `skills_editor_bottom_sheet.dart:228`, etc.
- **Impact** : pas de virtualisation. Pour 100+ items : ~100ms de jank initial + RAM 5x.
- **Fix** : `ListView.builder` / `ListView.separated` pour toute liste `List<X>.map(...)`.

### ~~PERF-M-07 : 3 dépendances mortes~~ closed in Phase 0 (`e1cabfd4 chore(mobile): remove unused lottie, connectivity_plus, wakelock_plus deps`)

### PERF-M-08 : 0 `RepaintBoundary` dans tout le code
- **Location** : `grep -r RepaintBoundary mobile/lib = 0`
- **Impact** : MessageBubble, cells portfolio, avatars animés, video_renderer LiveKit repeignent le screen entier à chaque tick.
- **Fix** : envelopper chaque MessageBubble (`chat_screen.dart:633`), cell portfolio_grid_widget, video_renderer LiveKit. Gain : 5-15 ms par frame sur listes denses.

## MEDIUM (12)

- **PERF-M-09** : `Image.network` brut dans `portfolio_form_sheet.dart:587, 589` — re-download à chaque rebuild
- **PERF-M-10** : Pas de retry/exponential backoff sur Dio (sauf WS) — ajouter `dio_smart_retry` ou interceptor maison
- **PERF-M-11** : `messagingWsService.events` stream — 5+ listeners simultanés invalident leurs providers en cascade sur chaque push
- **PERF-M-12** : `unreadNotificationCountProvider` `autoDispose` → thrash à chaque navigation
- **PERF-M-13** : Pas d'`IndexedStack` ni `wantKeepAlive` → switch tab détruit l'écran et refetch tous ses providers, scroll perdu sur Messaging. Utiliser `StatefulShellRoute.indexedStack`
- **PERF-M-14** : 638 `dynamic` + 437 `Map<String, dynamic>` dans le code métier — `authState.user?['display_name']` runtime check 3x plus lent que classes Freezed
- **PERF-M-15** : 16 fichiers > 600 lignes (router 1266, wallet 1168, proposal_detail 1023, billing_profile_form 974, profile 930, portfolio_form 831, app_drawer 744, chat 742, message_bubble 704)
- **PERF-M-16** : 25+ build methods > 100 lignes (ProfileScreen 253, ProposalDetailScreen 252, WalletScreen 209, RegisterScreen 209)
- **PERF-M-17** : `flutter_inappwebview ^6.1.5` (~6-10 MB APK) pour 1 seul écran subscription checkout — défer ou évaluer `url_launcher` external
- **PERF-M-18** : `record_linux ^1.0.0` dans `dependency_overrides` alors que Linux n'est pas une cible déclarée
- **PERF-M-19** : 13 `ref.read` dans `build()` — anti-pattern Riverpod, casse la réactivité (search_screen, opportunity_detail, team_screen, freelance_profile×2, client_profile, call_screen, profile_screen×2, notification_screen, referral_dashboard)
- **PERF-M-20** : `chat_screen.dart` crée un Dio standalone avec timeouts hardcodés (30s/120s) qui bypass l'auth interceptor → bug latent

## LOW (8)

- **PERF-M-21** : `ListView.builder` sans `cacheExtent` ni `addRepaintBoundaries: true` — défaut OK, mesurer avec DevTools avant tuning
- **PERF-M-22** : Pas de pull-to-refresh sur 14/25 écrans listes (wallet, jobs, opportunities, applications, referrals)
- **PERF-M-23** : Pas de config globale `imageCache.maximumSizeBytes` — défaut 100 MB / 1000 images, OK mais évict aggressive sur long scroll
- **PERF-M-24** : Pas de `flutter_svg` precache pour les illustrations
- **PERF-M-25** : 4 `Timer.periodic` à dispose-auditer (chat_screen, message_input_bar, call_provider, billing_success, incoming_call_overlay)
- **PERF-M-26** : Pas de bench Flutter DevTools documenté — capturer les timelines cold start / scroll messaging / scroll portfolio dans `docs/perf/`
- **PERF-M-27** : 7 features avec couches data/domain/presentation incomplètes (dashboard, invoice, mission, payment_info, profile, provider_profile, search) — décider de la stratégie
- **PERF-M-28** : Generated code (.freezed.dart, .g.dart) absent du repo (gitignored OK) — rappeler `dart run build_runner build` dans README open-source

## Strong points mobile

- CachedNetworkImage adopté majoritairement (11 fichiers, 5 occurrences correctes avec placeholder + errorWidget)
- Dio singleton via Riverpod Provider — pas de Singleton statique
- Token refresh interceptor avec Dio fresh-instance (évite loops)
- WebSocket service heartbeat 30s + reconnexion exponentielle + AppLifecycleListener — excellent pattern
- Pagination cursor-based parité backend (search_provider, proposal_repository_impl)
- Debouncing dans location_section et create_proposal_screen (fee preview)
- AnimationController disposal vérifié sur les 3 occurrences
- Generated code propre (0 .freezed.dart en repo, build_runner standard)

---

# Top 12 fixes prioritaires (cross-stack)

| # | ID | Effort | Impact |
|---|---|---|---|
| 1 | PERF-W-01 (LiveKit lazy) | 4 h | -1,3 MB sur tous les dashboards |
| 2 | PERF-B-01 (streaming uploads) | 4 h | ferme OOM Railway 1 GB |
| 3 | PERF-W-02 (RSC listings publics) | 1 j | débloque SEO (asset #1 marketplace) |
| 4 | PERF-W-03 (loading/error/sitemap) | 1 j | UX + indexation |
| 5 | PERF-M-02+M-03+M-04 (cold start mobile) | 1 j | -500 ms cold start |
| 6 | PERF-B-08 (utiliser provider_organization_id) | 4 h | -50-150ms p50 sur proposals/payment_records |
| 7 | PERF-B-05 (CacheService) | 2 j | -5-10× trafic Postgres SEO |
| 8 | PERF-B-02 (N+1 GetParticipantNames) | 2 h | -80-200 ms par dashboard |
| 9 | PERF-W-04 (supprimer TestDB) | 5 min | bundle prod + UX |
| 10 | PERF-M-07 (3 deps mortes) | 5 min | -1 MB APK |
| 11 | PERF-M-05 (memCacheWidth avatars) | 1 h | -50 MB RAM peak |
| 12 | PERF-B-09 (HTTP transport tuned) | 30 min | -50-100 ms par recherche |

**Bundle « 1 semaine »** = items 1-12 = transformation mesurable des KPIs (LCP web, cold start mobile, p50 backend, RAM mobile, OOM resilience).

---

## Closed in this round

| ID | Closed in |
|---|---|
| PERF-B-01 (streaming uploads) | PR #34 |
| PERF-B-12 (notification worker pool) | PR #36 |
| PERF-B-16 (WS sendOrDrop) | PR #40 |
| PERF-W-01 (LiveKit lazy) | PR #41 |
| PERF-W-02 (RSC public listings) | PR #41 |
| PERF-W-03 (loading/error/sitemap) | PR #41 |
| PERF-W-04 (TestDB removed) | Phase 0 |
| PERF-W-09 (cross-feature imports) | PR #37 |
| PERF-W-10 (unused web deps) | Phase 0 |
| ADMIN-PERF-01 (admin lazy routes + Vite manualChunks) | PR #41 |
| PERF-M-07 (mobile dead deps) | Phase 0 |

## Summary

| Layer | HIGH | MEDIUM | LOW |
|---|---|---|---|
| Backend + DB | 5 | 12 | 10 |
| Web + Admin | 7 | 10 | 7 |
| Mobile | 7 | 12 | 8 |
| **Total** | **19** | **34** | **25** |

(was 86 before this round → 78 remaining + 11 closed; new 8 BUG-NEW-* perf items captured separately in bugacorriger.md)
