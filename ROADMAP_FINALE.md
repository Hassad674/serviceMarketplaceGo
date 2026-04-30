# Roadmap Finale — Préparation Open-Source

**Date** : 2026-04-29
**Branche** : `main` @ `a0d268a4`
**Objectif** : finaliser le repo au niveau « parmi les meilleurs projets open-source mondiaux » avant publication.

---

## Synthèse globale des 4 audits + bugs

| Source | CRITICAL | HIGH | MEDIUM | LOW | Total |
|---|---|---|---|---|---|
| `auditsecurite.md` | 9 | 15 | 10 | 6 | 40 |
| `auditperf.md` (backend+web+mobile) | — | 27 | 34 | 25 | 86 |
| `auditqualite.md` (backend+frontend) | 8 | 70 | — | 204 | 282 |
| `bugacorriger.md` | 5 | 12 | 10 | 8 | 35 |
| `rapportTest.md` (tests + migrations) | 11 | 11 | — | 6 | 28 |

**Findings actionables uniques** (après dédoublonnage cross-audit) : ~110 critiques+majeurs + ~250 mineurs.

---

## Principes directeurs

1. **Séquentiel sur les phases, parallèle dans les phases.** Sécurité avant refactor. Refactor avant perf. Mais à l'intérieur de chaque phase, fan-out d'agents en worktrees sur scopes disjoints.
2. **Chaque agent reçoit la discipline « ni plus ni moins »** : implémenter exactement le scope, jamais en profiter pour un refacto adjacent.
3. **Auto-validation à chaque commit** : `go build ./... && go vet ./... && go test ./... -count=1` côté backend ; `npx tsc --noEmit && npx vitest run` côté web ; `flutter analyze` côté mobile.
4. **`main` reste protégé en permanence**, agents en branches `feat/<scope>` mergées seulement vert.
5. **Rule de DB isolation pour les agents qui touchent les migrations** : `createdb -T marketplace_go marketplace_go_<scope>` puis `dropdb` après merge.
6. **Tests AVANT/AVEC l'implémentation** sur tout scope > 1h de code.

---

## Phase 0 — Quick wins (1 journée, parallélisable)

Toutes ces actions sont triviales (< 30 min chacune) et débloquent du signal mesurable.

| # | Action | Source | Effort |
|---|---|---|---|
| 1 | Supprimer `web/src/app/[locale]/test-db.tsx` + son import dans `page.tsx:125` | PERF-W-04 | 5 min |
| 2 | `npm uninstall typesense country-region-data` dans `web/` | PERF-W-10 | 2 min |
| 3 | Supprimer `lottie`, `connectivity_plus`, `wakelock_plus` du `mobile/pubspec.yaml` (ou implémenter offline mode si on garde) | PERF-M-07 | 5 min |
| 4 | Documenter ou poser noop migration `024_noop` + `025_noop` | rapportTest | 15 min |
| 5 | Logger les 15 sites `_ = err` au minimum `slog.Warn` | QUAL-B | 2 h |
| 6 | Move `MockEmbeddingsClient` de `internal/search/embeddings.go:195` vers un `_test.go` | QUAL-B | 5 min |
| 7 | Remove 2 `var _ = errors.X` import sentinels (`searchindex/service.go:230`, `moderation/service.go:307`) | QUAL-B | 5 min |
| 8 | Mettre à jour `backend/CLAUDE.md` : `backend/mock/` → décrire la réalité `mocks_test.go` inline | QUAL-B | 10 min |
| 9 | Reduce `Access-Control-Max-Age` de 86400 à 600 dans `cors.go` | SEC-36 | 2 min |
| 10 | Ajouter `IF [NOT] EXISTS` sur les 35+13 migrations non-idempotentes | rapportTest | 2 h |

**Bénéfice** : -1,3 MB deps web/mobile, -1 fichier debug en prod, signal de propreté avant PR ouverture, 35 migrations devenues idempotentes.

---

## Phase 1 — Sécurité critique (~ 5 jours, ~3 agents en parallèle)

**Bloquant pour l'open-source.** Tout finding `CRITICAL` de `auditsecurite.md` + bugs business critiques.

### Agent A — Headers, secrets, sessions (~2 jours)
- SEC-03 : middleware `SecurityHeaders` (CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy)
- SEC-04 : `JWT_SECRET` fail-fast — `log.Fatal` si en prod et fallback ou len < 32
- SEC-05 + SEC-16 : bumper `session_version` dans Suspend/Ban/Reset (ferme la fenêtre 15 min mobile)
- SEC-08 : retirer `usesCleartextTraffic="true"` Android, `network_security_config.xml`, fail build prod si `API_URL` ne commence pas par `https://`
- SEC-24 : CORS `Vary: Origin` + `Allow-Credentials` conditionnel
- Web : `next.config.ts` `async headers()` complémentaire (CSP/HSTS côté Vercel)

### Agent B — Auth hardening (~2 jours)
- SEC-06 : refresh token rotation + blacklist Redis sur `jti` (TTL = remaining lifetime)
- BUG-08 : single-flight pattern dans le mobile `ApiClient` (bloquant pour SEC-06)
- SEC-07 : `BruteForceService` Redis (5/15min/email + lockout 30min, `429 + Retry-After`) sur login + forgot + reset
- SEC-11 : rate limiter Redis sliding window multi-classes (auth/mutation/upload/global), parsing X-Forwarded-For via CIDR allowlist
- SEC-13 : émettre `ActionLoginSuccess/Failure/Logout/PasswordReset/Suspend/Ban` dans le code + `REVOKE UPDATE, DELETE ON audit_logs`

### Agent C — Inputs, uploads, fraud (~1.5 jours)
- SEC-01 : XSS JSON-LD dans `web/src/app/[locale]/(public)/{freelancers,clients,referrers}/[id]/page.tsx` — escape `</`
- BUG-01 / SEC-02 : `ConfirmPayment` doit vérifier `stripe.PaymentIntents.Get(...).Status == "succeeded"` avant `MarkPaymentSucceeded`
- SEC-09 + SEC-21 : généraliser `validateImageBytes` à tous les uploads + filename randomisé extension dérivée du content-type validé
- SEC-19 : `go-playground/validator` sur tous les DTOs (longueurs, regex UUID, plages numériques)
- SEC-20 : ajouter caractère spécial requis dans `NewPassword`

**Validation phase 1** :
- E2E manual : login fail × 6 → lockout 30 min ; refresh token replay → 401 ; SVG upload → rejected ; ConfirmPayment sans Stripe → erreur ; XSS payload `</script>` dans about → escaped
- All tests green sur backend + web + mobile

---

## Phase 2 — Bugs métier critiques (~ 3 jours, ~2 agents en parallèle)

### Agent D — State machines & races (~2 jours)
- BUG-02 : guards `ApplyDisputeResolution` / `MarkRefunded` / `MarkFailed` (vérification état actuel)
- BUG-03 : `_ = s.proposals.Update` après `RestoreFromDispute` → propager + métrique + pendingevent rattrapage
- BUG-04 / SEC-18 : `pg_advisory_xact_lock(hashtext(org_id))` pour `resolveStripeAccount`
- BUG-05 : Outbox pattern réel — mutation profil + INSERT `pending_events` dans la même transaction
- BUG-06 : WS `sendOrDrop` helper appliqué dans `connection.go:261, 317`
- BUG-07 : `removeClient` retourne `wasLast` sous le même lock
- BUG-09 : log les erreurs `_ = s.records.Update` dans `service_stripe.go:120, 292`
- BUG-15 : fix `context.Background()` overrides dans `antigaming/pipeline.go:74` et `proposal/service_scheduler.go:109`

### Agent E — Webhook idempotency, LiveKit, FCM (~1.5 jours)
- BUG-10 / SEC-17 : utiliser table `stripe_webhook_events` (INSERT UNIQUE event_id) comme source de vérité, Redis fast-path
- BUG-13 : LiveKit `maxParticipants=4` + identity stable + re-join kick
- BUG-14 : LiveKit token avec `CanPublish/CanSubscribe/CanPublishData` explicites
- BUG-16 : notification worker N=3-5 workers + ré-enqueue `available_at` au lieu de `time.Sleep`
- BUG-25 : FCM tap → routing global GoRouter selon `data['type']` (proposal/message/review)

**Validation phase 2** :
- Tests d'intégration sur les state machines (refund + retry, dispute + cancel + restore)
- LiveKit reconnect mobile : Wi-Fi → 4G switch → call recovers
- Stripe webhook replay (force renvoi via dashboard) → idempotent

---

## Phase 3 — Refactor structurel (~ 2 semaines, 4-5 agents en parallèle)

### Agent F — Splitter le wiring (~1.5 jours)
- QUAL-B-01 : `cmd/api/main.go` → `wire_adapters.go`, `wire_services.go`, `wire_router.go`, `wire_workers.go`
- `internal/handler/router.go` (`NewRouter` 822 lignes) → `mountX(r, deps)` par feature

### Agent G — God components web (~3 jours)
- `wallet-page.tsx` 878 → 5 sous-composants
- `message-area.tsx` 797 → extraire `MessageBubble` + hook scroll
- `search-filter-sidebar.tsx` 758 → un fichier par section
- `billing-profile-form.tsx` 656 → 4 sections
- Migrer 9 formulaires en `react-hook-form + zod`

### Agent H — God widgets mobile (~3 jours)
- `app_router.dart` 1266 → split par feature `routes.dart`
- `wallet_screen.dart` 1168 → décomposer
- `proposal_detail_screen.dart` 1023 → idem
- `billing_profile_form.dart` 974 + `profile_screen.dart` 930 + 12 autres > 600

### Agent I — Frontend hygiene & isolation (~2 jours)
- QUAL-W-01 : créer `loading.tsx`, `error.tsx`, `not-found.tsx`, `global-error.tsx` par groupe + `sitemap.ts` + `robots.ts`
- QUAL-W-02 : extraire `provider/upload-api`, `expertise-editor`, `city-autocomplete`, `search-api` vers `web/src/shared/` (casse 9 imports cross-feature)
- QUAL-W-03 : déplacer `app/[locale]/(app)/payment-info/components/` vers `features/payment-info/`
- Créer primitives `Button` / `Input` shadcn dans `web/src/shared/components/ui/` + migrer 309+95 sites
- Centraliser `formatEur` / `formatDate` dans `web/src/shared/lib/utils.ts`

### Agent J — Backend SOLID & duplication (~3 jours)
- `payment.Service` 1171 lignes → 3 services
- `ProposalHandler` 29 méthodes → 4 handlers
- Segregation top 6 god repos (Referral 24, Message 21, Org 20, Dispute 18, Proposal 16, User 15) — ISP propre
- Extraire `pkg/sqlfilter` pour le pattern `WHERE 1=1 + paramIdx` répété 6×
- `pkg/` purity : déplacer `pkg/validator`, `pkg/crypto`, `pkg/confighelpers` sous `internal/` (ou inverser via primitives)

### Agent K — Mobile cleanup (~2 jours)
- 491 `Color(0x...)` → `Theme.of(context)` ou `AppColors`
- 18 `print()` → `debugPrint` ou logger (concentrés `features/call/`)
- 13 `ref.read` dans `build()` → `ref.watch` ou déplacer vers `initState`/callbacks
- Cross-feature `notification → messaging` WS service → `core/`
- 49 `Text('English')` → `AppLocalizations`
- Réduire 198 `dynamic` (DTOs Freezed)

**Validation phase 3** :
- Build all 4 apps green
- Tests verts sur chaque agent
- Bundle size web mesuré (avant/après) via `@next/bundle-analyzer`
- APK size mesuré (avant/après)

---

## Phase 4 — Performance (~ 1 semaine, 3-4 agents en parallèle)

### Agent L — Backend perf critique (~3 jours)
- PERF-B-01 : streaming uploads (multipart reader) — ferme l'OOM Railway
- PERF-B-02 : batch `GetParticipantNames` — N+1 fix
- PERF-B-03 : pagination cursor sur `payment_records.ListByOrganization`
- PERF-B-04 : bulk INSERT milestones
- PERF-B-07 : `ReadHeaderTimeout` + `WriteTimeout` finis
- PERF-B-08 : utiliser `provider_organization_id` (migration 115) + index composite
- PERF-B-09 : `pkg/httpx.NewTunedClient` partagé avec `MaxIdleConnsPerHost: 50`
- PERF-B-14/15 : cache Stripe Connect (60-120s) + ctx propagé au SDK

### Agent M — Cache-aside backend (~2 jours)
- PERF-B-05 : port `service.CacheService` + `adapter/redis/cache.go`
- Intégration sur `profileapp.GetPublicByOrg`, `skillapp.GetCuratedByExpertise`, `jobapp.ListPublic`, `reviewapp.GetAggregateForOrg`
- Invalidation explicite sur write
- PERF-B-06 : slow-query logger wrapper (`pkg/dbx`) pour `slog.Warn` au-delà de 50ms

### Agent N — Web/Admin perf (~2 jours)
- PERF-W-01 : casser couplage `LiveKit → dashboard-shell` via dynamic import + event-bus
- PERF-W-02 : RSC sur `/agencies`, `/freelancers`, `/referrers` (premier appel Typesense côté serveur, hydrate au-dessus)
- PERF-W-06 : `generateMetadata` + JSON-LD `Organization` sur `/agencies/[id]`, `JobPosting` sur `/opportunities/[id]`
- PERF-W-07 : 7 `<img>` raw → `<Image>` + `priority` sur LCP candidates
- PERF-W-08 : descendre `"use client"` au composant interactif sur 10+ pages
- ADMIN-PERF-01 : `lazy()` sur les routes admin + `manualChunks` Vite
- PERF-W-12 : retirer `staleTime: Infinity` sur `team` permissions

### Agent O — Mobile perf (~2 jours)
- PERF-M-02 : deferred imports dans `app_router.dart`
- PERF-M-03 : `Firebase.initializeApp()` après `runApp()` via `unawaited()`
- PERF-M-04 : déplacer `FCMService.initialize` vers `initState()` du shell
- PERF-M-05 : `memCacheWidth/Height` sur tous les avatars CachedNetworkImage
- PERF-M-06 : 21 `ListView(children:)` → `ListView.builder`
- PERF-M-08 : `RepaintBoundary` sur MessageBubble, portfolio cells, video_renderer
- PERF-M-13 : `StatefulShellRoute.indexedStack` pour le bottom nav

**Validation phase 4** :
- Lighthouse audit web (LCP < 2.5s, CLS < 0.1, FID < 100ms)
- k6 load test backend (`scripts/perf/k6-search.js` + nouveau pour profil/messaging)
- Flutter DevTools profiling : cold start, scroll messaging, scroll portfolio
- Capturer les KPIs avant/après dans `docs/perf/` (timeline DevTools, Lighthouse JSON)

---

## Phase 5 — Tests & DB hardening (~ 2 semaines, parallélisable avec Phase 4)

### Agent P — Tests backend critiques (~1 semaine)
- `internal/app/admin/` : 0/9 → service_test par module (priorité suspend/ban/audit)
- `internal/app/kyc/` : 0/1 → service_test + handler_test
- `internal/app/referral/` : 2/20 → cibler money paths (clawback, commission distributor, kyc_listener)
- 23 handlers untested → prioriser admin handlers + dispute/stripe/role_overrides
- Postgres adapter : proposal, dispute, review, message, payment_records, notification

### Agent Q — RLS PostgreSQL (~2 jours)
- Migration : `ENABLE ROW LEVEL SECURITY` sur `messages`, `conversations`, `invoices`, `proposals`, `notifications`, `wallet_records`, `disputes`, `audit_logs`, `proposal_milestones`
- `FORCE ROW LEVEL SECURITY` (prevent owner bypass)
- Helper `SetCurrentOrg(ctx, tx, orgID)` au début des transactions
- Test d'intégration vérifiant le block cross-tenant

### Agent R — Frontend tests (~1 semaine)
- Web vitest : couvrir `billing`, `dispute`, `organization-shared`, `reporting` (4 features RED) + thicken `proposal`, `subscription`, `wallet`
- Web : tests pour `shared/components/ui/`
- Admin : 1 test minimum par feature (10 features)
- Mobile : couvrir 9 features RED (dashboard, dispute, mission, portfolio, project_history, provider_profile, referral, referrer_reputation, reporting)
- Mobile : 5 golden tests sur key surfaces (search result card, message bubble, wallet hero, profile header, button variants)

### Agent S — CI gaps (~1 jour)
- Wire admin tests dans `ci.yml` (`admin: cd admin && npm run test:ci`)
- Étendre mobile `flutter test` scope (au moins messaging, proposal, wallet, billing)
- Ajouter `scripts/smoke/run-all.sh` invocable par PR avec label
- Backend : déplacer `test/e2e/phase*_e2e.sh` vers Go integration tests OU les invoquer en CI nightly
- Coverage gate admin (≥60%)
- ESLint en error (pas continue-on-error)

### Agent T — DB cohérence (~1 jour)
- Décider cross-feature FK rule (relaxer CLAUDE.md OU migrer 10 violations)
- Documenter invariant `org_id` (CLAUDE.md + lint CI grep)
- CHECK constraints pour les enum TEXT (`proposals.status`, `disputes.status`, `payment_records.status`)
- Index manquants : `(provider_organization_id, status, created_at DESC)` sur proposals, idem payment_records (cf. PERF-B-08)

**Validation phase 5** :
- Couverture par layer mesurée (cible 80%+ business logic)
- Test RLS cross-tenant green
- CI tourne tous les tests sur PR (admin + mobile élargi)

---

## Phase 6 — Polish open-source (~ 1 semaine)

### Documentation
- `docs/ARCHITECTURE.md` : diagrammes mermaid (hexagonal, feature isolation, search engine, payment flow, KYC flow). Showcase value pour le repo.
- `docs/SECURITY.md` : threat model, supply chain, reporting policy
- `docs/CONTRIBUTING.md` : conventions, agents pattern, validation pipeline, parallel workflow
- `docs/DEPLOYMENT.md` : Railway / Vercel / Neon / Cloudflare R2 setup
- `LICENSE` : MIT ou Apache 2.0
- `CODE_OF_CONDUCT.md` : Contributor Covenant
- `SECURITY.md` : disclosure policy
- `README.md` : refonte avec gif/screenshots, badges CI/coverage, quickstart, demo link

### CI/CD avancé
- Ajouter `gosec` + `semgrep` sur PR
- `dependabot.yml` pour security updates auto
- Release workflow (changelog auto, GitHub Releases, signed tags)
- `pr-template.md` + `issue-template.md`

### Cleanup final
- Sweep tous les TODOs restants (4 mobile, 1 backend, 1 web)
- Vérifier les commentaires en français → EN dans le code public (README EN+FR OK, code EN)
- Final regression smoke : 4 apps build green, all tests green, lighthouse green, k6 ok

---

## Estimation totale (dev solo + agents)

| Phase | Durée | Agents parallèles |
|---|---|---|
| 0 — Quick wins | 1 jour | 1 (séquentiel) |
| 1 — Sécurité critique | 5 jours | 3 |
| 2 — Bugs métier | 3 jours | 2 |
| 3 — Refactor | 10 jours | 4-5 |
| 4 — Performance | 5 jours | 3-4 |
| 5 — Tests + DB | 10 jours | 4 (parallèle avec Phase 4) |
| 6 — Polish open-source | 5 jours | 1-2 |
| **Total séquentiel** | **~ 6 semaines** | |
| **Total avec parallélisation Phase 4-5** | **~ 4 semaines** | |

---

## Pattern de dispatch d'agent recommandé

Chaque brief d'agent doit contenir :

1. **Working directory** + **stack pointer** (CLAUDE.md à lire en premier)
2. **Scope précis** — items du backlog à traiter, exhaustivement
3. **Contre-scope** — « ne pas refactorer en passant, ne pas ajouter de feature, flagger plutôt que fixer en silence »
4. **Standards** — 600/50/4, hexagonal strict, Pure SQL, i18n FR+EN, dark mode, parité mobile
5. **Validation pipeline obligatoire avant commit** (paste de l'output dans le rapport final)
6. **Worktree obligatoire** + DB isolée si la tâche touche les migrations
7. **Format de rapport final** : `Files changed`, `Tests added`, `Validation paste`, `Deviations`, `Follow-ups flagged`

---

## Risques & escalation triggers

L'orchestrateur (toi + moi) escalade au USER (pas à un autre agent) quand :
- 2 agents consécutifs échouent sur le même sub-scope
- Un agent a cassé un état partagé (DB, Typesense, branche d'un autre agent)
- Un agent revient avec un résultat matériellement différent du brief (scope creep)
- Conflit de spec ambigu (décision produit, pas technique)

---

## Bundle « pré-open-source minimum acceptable »

Si tu veux ouvrir-source plus tôt en gardant la qualité, **les phases 0 + 1 + 2 + Phase 5/RLS + Phase 6** suffisent. Ça représente **~ 3 semaines** et ferme :
- Toutes les vulnérabilités CRITICAL
- Tous les bugs metier CRITICAL
- Le filet RLS DB
- La doc + CI minimale

Le refactor (Phase 3) et la perf (Phase 4) peuvent suivre en post-launch — un repo open-source peut s'améliorer publiquement, mais ne peut pas être ouvert avec des CVE exploitables.
