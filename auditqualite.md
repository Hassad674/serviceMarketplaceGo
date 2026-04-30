# Audit Qualité & Refactoring

**Date** : 2026-04-29 (audit précédent : 2026-03-30, obsolète)
**Branche** : `main` @ `a0d268a4`
**Périmètre** : backend Go + web Next.js + admin Vite + mobile Flutter

## Méthodologie

Audit statique. Backend : 848 .go files, 121 migrations, 34 modules app. Web : 537 .ts/tsx, 141k LOC. Admin : 116 fichiers, 21k LOC. Mobile : 378 .dart (hors généré + l10n), 142k LOC. Mesures objectives (taille fichier/fonction, props, types, magic strings, imports croisés) + relecture ciblée des hot-spots. Aucun build ni test exécuté.

---

# BACKEND GO

## CRITICAL (2)

### QUAL-B-01 : `cmd/api/main.go` = 1479 lignes dont `func main()` = 1317 lignes
- **Location** : `backend/cmd/api/main.go:85`
- **Pattern** : un seul `func main()` qui contient TOUT le wiring (config, adapters, services, router, workers).
- **Impact** : c'est la file la plus lue par les nouveaux contributeurs ; le mensonge "hexagonal architecture" est masqué dans 1300 lignes au lieu d'être visible.
- **Fix** : splitter en `wire_adapters.go`, `wire_services.go`, `wire_router.go`, `wire_workers.go` colocalisés sous `cmd/api/`. `main.go` reste ~150 lignes (config load + lifecycle).

### QUAL-B-02 : SQL injection potentielle dans admin_moderation_queries
- **Location** : `backend/internal/adapter/postgres/admin_moderation_queries.go:183`
- **Pattern** : `where += " AND r.target_type = '" + wantType + "'"` où `wantType` trace au query param `?type=` sans allowlist sur le path reports (`shouldIncludeReports` retourne toujours true).
- **Impact** : admin-gated mais une compromission admin = arbitrary SQL.
- **Fix** : valider `wantType` contre la même allowlist que `shouldIncludeGeneric`, ou placeholder `$N`.

## MAJOR (50)

### Hexagonal violations

- **`pkg/` purity cassée** — `pkg/validator/validator.go:9`, `pkg/crypto/hash.go:6`, `pkg/crypto/jwt.go:10`, `pkg/confighelpers/issuer.go:12` importent `internal/domain/*` ou `internal/port/service`. Soit déplacer sous `internal/`, soit inverser via primitives. Mensonge architectural.
- **Handler → domain leak (3 sites)** — `internal/handler/report_handler.go:11`, `dto/response/admin.go:9`, `dto/response/report.go:6` importent `internal/domain/report` directement. Carver dans la convention ou re-déclarer les enums au niveau DTO.

### File size violations (production code, > 600 lignes — 16 fichiers)

| Fichier | Lignes | Split recommandé |
|---|---|---|
| `internal/app/payment/service_stripe.go` | 1171 | `service_payout.go` + `service_charge.go` + `service_refund.go` |
| `internal/handler/proposal_handler.go` | 920 | `proposal_lifecycle_handler.go` + `proposal_payment_handler.go` + `proposal_completion_handler.go` + `proposal_admin_handler.go` (29 méthodes) |
| `internal/adapter/postgres/invoicing_repository.go` | 915 | extraire `ListInvoicesAdmin` (168 lignes) en `invoicing_admin_queries.go` |
| `internal/handler/router.go` | 910 | `NewRouter` = 822 lignes → splitter par feature `mountX(r, deps)` |
| `internal/app/proposal/service_actions.go` | 864 | `service_completion.go` + `service_cancellation.go` |
| `internal/app/dispute/service_actions.go` | 854 | `service_open.go` + `service_resolve.go` + `service_evidence.go` |
| `internal/adapter/postgres/organization_repository.go` | 751 | extraire role-overrides queries |
| `internal/domain/dispute/entity.go` | 729 | `Dispute` aggregate + `evidence.go` + `counter_proposal.go` |
| `internal/adapter/postgres/conversation_repository.go` | 723 | list/read vs write/mutation |
| `internal/app/subscription/service.go` | 709 | lifecycle vs cycle-change |
| `internal/adapter/postgres/profile_repository.go` | 700 | `SearchPublic` (122 lignes) → `profile_search_queries.go` |
| `internal/handler/profile_handler.go` | 632 | admin vs user-facing |
| `internal/search/indexer.go` | 609 | borderline, OK |
| `internal/domain/organization/permissions.go` | 609 | borderline, OK |
| `internal/adapter/stripe/account.go` | 603 | borderline |
| `internal/adapter/postgres/referral_repository.go` | 603 | borderline |

Test files > 600 lignes : 16 (top : `proposal/service_test.go` 1344, `messaging/service_test.go` 1344, `auth/service_test.go` 1073, `subscription/service_test.go` 878).

### Function size violations (top 20, > 50 lignes — 163 au total)

| File:line | Function | Lines |
|---|---|---|
| `cmd/api/main.go:85` | `main` | **1317** |
| `internal/handler/router.go:89` | `NewRouter` | **822** |
| `internal/adapter/postgres/invoicing_repository.go:735` | `ListInvoicesAdmin` | 168 |
| `internal/app/embedded/notifier.go:240` | `(*Notifier).diff` | 150 |
| `internal/app/payment/service_stripe.go:790` | `RetryFailedTransfer` | 148 |
| `internal/app/payment/service_stripe.go:617` | `RequestPayout` | 148 |
| `internal/app/dispute/service_actions.go:118` | `OpenDispute` | 145 |
| `internal/app/referral/service_notifications.go:68` | `notifyStatusTransition` | 132 |
| `internal/app/proposal/service_create.go:30` | `CreateProposal` | 129 |
| `internal/adapter/postgres/profile_repository.go:403` | `SearchPublic` | 122 |
| `internal/app/proposal/service_actions.go:499` | `CompleteProposal` | 115 |
| `internal/app/proposal/service_scheduler.go:139` | `AutoApproveMilestone` | 111 |
| `internal/app/organization/role_overrides_service.go:175` | `UpdateRoleOverrides` | 107 |
| `internal/app/invoicing/monthly.go:67` | `IssueMonthlyConsolidated` | 106 |
| `internal/adapter/postgres/user_repository.go:281` | `ListAdmin` | 100 |
| 5 autres | — | 93-98 |

Pattern de refacto : extraire `loadAndValidate*`, `applyTransition*`, `notify*` helpers — réduit chaque méthode à ~30 lignes.

### SOLID — ISP (god repos > 15 méthodes)

| Interface | Méthodes |
|---|---|
| `ReferralRepository` | **24** |
| `MessageRepository` | 21 |
| `OrganizationRepository` | 20 |
| `DisputeRepository` | 18 |
| `ProposalRepository` | 16 |
| `UserRepository` | 15 |

Un mock à 24 méthodes ne se code pas en 5 minutes (la barre fixée par CLAUDE.md). Segregation par accès : `ReferralReader`, `ReferralWriter`, `ReferralCommissionStore`, `ReferralAttributionStore`.

### SRP

- `ProposalHandler` : 29 méthodes (lifecycle + payment + completion + cancellation + milestones + 5 admin) → 4 handlers
- `payment.Service` (1171 lignes) : wallet reads + payout orchestration + transfer retry + webhook reactions → 3 services minimum
- `invoicing.Service` : subscription invoices + monthly consolidated + credit notes → split

### Param count violations (> 4)

| Params | Location |
|---|---|
| 7 | `cmd/reindex/main.go:155` `reindexPersona` |
| 6 | `internal/domain/user/entity.go:107` `NewUser(email, hashed, first, last, display, role)` |
| 6 | `internal/domain/user/entity.go:133` `NewOperator` |
| 6 | `internal/adapter/postgres/admin_conversation_queries.go:130` |
| 6 | `internal/adapter/postgres/job_admin.go:222` |
| 5 | 6 sites supplémentaires |

Pattern : introduire `NewUserInput` / `NewOperatorInput` / `ListAdminConversationsArgs` structs.

### Error handling

- **15 sites `_ = err`** — silence d'erreur. Top sites : `internal/app/organization/membership_service.go:257,307`, `transfer_service.go:233,236`, `admin_overrides.go:200,203`, `referral/commission_distributor.go:67,102,103`, `referral/kyc_listener.go:67`. Au minimum `slog.Warn` + opération.
- **2 hacks `var _ = errors.New`** — `internal/app/searchindex/service.go:230`, `internal/app/moderation/service.go:307`. Silent unused-import. Remplacer ou supprimer.
- **`pkg/cursor/Encode` swallow** — `data, _ := json.Marshal(c)`. Convention zéro swallow → retourner `(string, error)`.
- **`%s` au lieu de `%w`** — `internal/search/embeddings.go:160, 169`. Casse `errors.Is/As`.
- ✅ Sentinel errors centralisés par feature dans `errors.go` (338 sites domain). Excellent.

### Context handling

- **`context.Background()` override silencieux** — `internal/search/antigaming/pipeline.go:74` (overwrite ctx d'entrée → loss timeout/cancel), `internal/app/proposal/service_scheduler.go:109` (Background dans une méthode qui reçoit ctx).
- 5 sites fire-and-forget (`searchanalytics/ltr_capture.go:103`, `searchanalytics/service.go:139`, `proposal/service_create.go:173`, `job/service_applications.go:137`) — devraient dériver de `app.shutdownCtx` pas `Background()`.
- Pas de `WithTimeout` au niveau handlers (les repos en ont).

### Migrations consistency

- **35 `*.up.sql` sans `IF NOT EXISTS`** — re-run sur état partiel = hard fail. Top : `001_create_users`, `045_create_disputes`, `097_create_freelance_profiles`.
- **13 `*.down.sql` sans `IF EXISTS`** — symétrique.
- ✅ Up/down complets : 121/121
- ⚠️ Gap 024/025 toujours là (signalé en mars). Documenter dans `migrations/README.md` ou poser noop.
- ✅ Naming conventions cleans (`create_X`, `add_Y_to_X`, `drop_Z`)

### Naming

- ✅ Pas de stuttering (`auth.AuthService` etc. — bare `Service` partout)
- ✅ Pas de prefix `I` sur interfaces
- ✅ Forbidden var names quasi-absents (3 `var data` typés JSON unmarshal targets, OK)

### Duplication

- **Filter-clause builders** dupliqués dans 6 admin queries (`*_admin.go` files) — pattern `WHERE 1=1 + paramIdx++` répété 250 lignes au total. Extraire en `pkg/sqlfilter` ou `internal/adapter/postgres/internal/filter`.
- **DTO mapping nil-pointer dance** sur `*time.Time` / `sql.NullString` répété — `dtomap` helper réduirait ~15% de `dto/response/*.go`.
- **`parseLimit` / `parsePage` / `parseUUID`** patterns répétés dans presque chaque handler.

### Tests quality (cf. rapportTest.md pour coverage)

- 17 fichiers `mocks_test.go` manuels (function-pointer mocks). Le plus gros : `proposal/mocks_test.go` 725 lignes pour 16 méthodes. Pattern consistant et lightweight ✅.
- ⚠️ `backend/mock/` n'existe pas — CLAUDE.md le mentionne ("Generated mocks from port interfaces"). Mismatch doc/réalité. Mettre à jour CLAUDE.md.
- Pas de shared `_test_helpers.go` — fixtures `newTestUser`/`newTestOrg` redéclarées dans chaque service_test.

### TODO / dead code

- **1 seul TODO sur 76k LOC** (`internal/app/referrerprofile/service_reputation.go:129`) — exceptionnel.
- ✅ Pas de commented-out code blocks
- ✅ Pas de `fmt.Println` / `log.Println` dans le lib code
- ⚠️ `MockEmbeddingsClient` dans production file (`internal/search/embeddings.go:195`) — déplacer en `_test.go`

## MINOR (resté en log)

- 191 micro-issues (functions 50-90 lignes, naming local, NullX vs pointer mix, formatage). Ne traiter qu'en sweep dédié.

## Strong points backend

- **Domain purity 100%** — zéro import non-stdlib hors `uuid`. Exemplaire.
- **App layer ne touche jamais les adapters** — DI réelle, le test "delete folder" passerait
- **Cross-feature isolation 100%** — zéro `internal/app/<feat>` → `internal/app/<other>`
- App layer 94% files-tested
- Pas de `fmt.Println` / `log.Println` dans le lib code
- Pas de fuite secrets dans logs
- Conventional commits, migrations up/down completes

---

# WEB + ADMIN

## CRITICAL (3)

### QUAL-W-01 : 0 `error.tsx` / 0 `loading.tsx` / 0 `not-found.tsx` / 0 `global-error.tsx`
- **Location** : tout `web/src/app/**`
- **Pattern** : aucun error boundary ni loading state au niveau Next. `web/CLAUDE.md` lignes 167-176 et 471 EXIGENT au moins un par groupe.
- **Impact** : un crash → écran Next.js par défaut. Aucun skeleton entre arrivée sur route et résolution Server Components.
- **Fix** : créer au minimum `(app)/loading.tsx`, `(public)/loading.tsx`, `error.tsx` par groupe + `global-error.tsx` + `not-found.tsx`.

### QUAL-W-02 : 33 imports cross-feature dans 17 fichiers
- **Pattern dominant** : `provider/` agit comme un faux `shared/` — `expertise-editor`, `city-autocomplete`, `upload-api`, `search-api` consommés par `client-profile`, `freelance-profile`, `referrer-profile`, `job`, `organization-shared`, `referral`. `messaging/` est devenu hub pour `proposal`, `referral`, `review`, `reporting`. `wallet → invoicing`. `proposal → billing/subscription`.
- **Impact** : la promesse "delete folder" est cassée côté web. Bundles transitifs alourdis.
- **Fix** : extraire vers `web/src/shared/` ou créer `web/src/features/upload/`, `web/src/features/profile-shared/`. **Casse 9 des 33 imports en un coup**.

### QUAL-W-03 : Components dans `app/[locale]/(app)/payment-info/components/`
- **Location** : 6 fichiers de composants + `lib/` à l'intérieur d'`app/`
- **Pattern** : viole "app/ is for routing only" (CLAUDE.md ligne 274).
- **Fix** : déplacer vers `features/payment-info/` ou `features/billing/`.

## MAJOR (20)

### File size > 600 lignes (web)

| Fichier | Lignes | Split |
|---|---|---|
| `wallet-page.tsx` | 878 | `wallet-overview-card`, `wallet-transactions-list`, `wallet-payout-section`, `wallet-commission-list` |
| `message-area.tsx` | 797 | extraire `MessageBubble` (13 props) en fichier séparé, hook scroll/intersection |
| `search-filter-sidebar.tsx` | 758 | un fichier par section de filtre |
| `billing-profile-form.tsx` | 656 | section identité légale / adresse / fiscal / signataire |

Admin : aucun fichier > 600 (max 413 sur `dispute-detail-page.tsx`). ✅

### Composants > 4 props (56 sites web)

| Composant | Props |
|---|---|
| `ActionsPanel` | 19 (proposal-actions-panel.tsx:20) |
| `SearchPageLayout` | 18 |
| `MessageArea` | 14 |
| `MessageBubble` | 13 |
| `PipCallOverlay` / `CallOverlay` | 12 |
| `FullscreenCallOverlay` | 11 |
| `ResultsSection`, `ProposalCardActions` | 10 |

Pattern : grouper en sous-objets thématiques (`actions: {accept, decline, modify}`, `state: {pending, isMutating}`).

### Pages `app/` > 100 lignes (12 pages, viole "5-20 lines" CLAUDE.md)

| Page | Lignes |
|---|---|
| `subscribe/embed/page.tsx` | 431 |
| `projects/page.tsx` | 408 |
| `payment-info/page.tsx` | 400 |
| `dashboard/page.tsx` | 259 |

Extraire vers les features.

### Forms : 9/17 formulaires en `useState` manuel (anti-pattern CLAUDE.md)

`portfolio-form-modal.tsx` (487), `billing-profile-form.tsx` (656), `pricing-kind-form.tsx` (438), `create-job-form.tsx`, `edit-job-form.tsx`, `referral-creation-form.tsx` — devraient utiliser `react-hook-form + zod` + `@hookform/resolvers` (déjà dans deps).

### Pas de `Button` / `Input` dans `shared/components/ui/`

309 boutons + 95 inputs avec classes Tailwind dupliquées partout. Créer les primitives shadcn-style.

### TypeScript hygiene

- **Web** : 2 `any` documentés (envelope WS) ✅, 110 `as` assertions (sample 50 légitimes), 19 `!.` (protégés par TanStack Query `enabled` flag)
- **Admin** : 0 `any`, 0 `as any`, 0 `@ts-ignore` ✅

### console.log

- Web : 26 occurrences (25 dans `features/call/` debug WebRTC à nettoyer)
- Admin : 1

### i18n gaps

- **Web** : 5 strings JSX FR hardcodées (`wallet-page.tsx:84,375,659`, `referral-detail-view.tsx:84`, `billing-profile-form.tsx:586`), 3 placeholders FR dans `referral/`. À porter dans `messages/fr.json`.
- 96 `/api/v1/...` hardcodés dans `features/` — centraliser

### Date/currency formatters dupliqués 10+ fichiers

`formatEur`, `formatDate` redéfinis localement dans wallet-page, dispute-banner/counter-form/form/resolution-card, proposal-card, proposal-preview, referral-missions-section, fee-preview, projects/page. Devrait vivre dans `shared/lib/utils.ts` (l'admin l'a fait correctement).

### Accessibility

- **Web** : 22 `<img>` raw (1 justifié), 17 boutons icon-only mais 0 sans `aria-label`, 28 modals correctement `role="dialog"` + `aria-modal`
- **Admin** : 3 `<img>` raw, 0 violation a11y

### State management

- ✅ TanStack Query partout (1 useEffect+fetch légitime : `address-autocomplete.tsx` debounced)
- ✅ 1 seul Context (call) — conforme
- ✅ Pas de Redux

### Naming, exports

- ✅ kebab-case 100%
- ✅ Pas de `export default` hors `app/` (sauf `App.tsx` admin OK)

## MINOR (13)

- 12 inline `style={{}}` (1/2 dynamiques OK ; statiques à extraire en classes Tailwind)
- 42 hex hardcodés (logos sociaux légitimes)
- Hex colors dupliqués 3× (linkedin/instagram/youtube) — extraire en constante
- 2 `setTimeout` magiques 1500/2000 ms dans payment-simulation
- Web Lighthouse / bundle analyzer non configurés (cf. auditperf.md)
- Tests web : voir rapportTest.md

## Strong points web/admin

- **Admin = exemplaire** : 0 cross-feature, 0 `any`, 0 fichier > 600, design system propre (`shared/components/ui/{button,input,select,...}`). Module de référence.
- TS strict avec 2 `any` documentés sur 141k lignes
- i18n web ~complet (369 `useTranslations`, FR/EN à parité 1493 keys)
- TanStack Query bien adopté
- Naming consistency 100%
- Generated files gitignored partout
- 1 TODO sur tout le web

---

# MOBILE (Flutter)

## CRITICAL (3)

### QUAL-M-01 : 17 fichiers > 600 lignes (hors généré)

| Fichier | Lignes | Sévérité |
|---|---|---|
| `core/router/app_router.dart` | 1266 | CRITIQUE — split par feature `routes.dart` exposant `List<RouteBase>` |
| `wallet_screen.dart` | 1168 | CRITIQUE |
| `proposal_detail_screen.dart` | 1023 | CRITIQUE |
| `billing_profile_form.dart` | 974 | MAJEUR |
| `profile_screen.dart` | 930 | MAJEUR |
| `portfolio_form_sheet.dart` | 831 | MAJEUR |
| `app_drawer.dart` | 744 | MAJEUR |
| `chat_screen.dart` | 742 | MAJEUR |
| `message_bubble.dart` | 704 | MAJEUR |
| `freelance_profile_screen.dart` | 700 | MAJEUR |
| `dispute_banner_widget.dart` | 659 | MAJEUR |
| `public_profile_screen.dart` | 657 | MAJEUR |
| `portfolio_grid_widget.dart` | 646 | MAJEUR |
| `referrer_profile_screen.dart` | 633 | MAJEUR |
| `job_detail_screen.dart` | 613 | MINEUR |
| `jobs_screen.dart` | 606 | MINEUR |
| `messaging_screen.dart` | 601 | MINEUR |

### QUAL-M-02 : 491 `Color(0x...)` hardcodés vs 442 `Theme.of(context)` (ratio 1:0.9)
Le `AppColors` extension du `app_theme.dart` est sous-utilisé. Top offenders : `wallet_screen.dart` (29), `message_bubble.dart` (29), `role_permissions_editor_atoms.dart` (20), `proposal_detail_screen.dart` (20).

### QUAL-M-03 : 198 `dynamic` hors `Map<String,dynamic>` et généré
Concentré dans `data/` repos Dio (`_api.get<dynamic>`). Le projet a Freezed + json_serializable précisément pour éviter ça.

## MAJOR (10)

### Build methods > 100 lignes (25+ violations)

| Build | Lignes |
|---|---|
| `ProfileScreen.build` | 253 |
| `ProposalDetailScreen.build` | 252 |
| `WalletScreen.build` | 209 |
| `RegisterScreen.build` | 209 |
| `EnterpriseRegisterScreen.build` | 202 |
| `LoginScreen.build` | 190 |
| `MessageBubble.build` | 187 |
| `AgencyRegisterScreen.build` | 186 |
| `PortfolioGridWidget.build` | 181 |
| `ChatAppBar.build` | 181 |

Décomposer en sub-widgets nommés ou méthodes privées `_buildSomething(...)`.

### 18 `print()` en production (anti-pattern, CLAUDE.md interdit)
Concentré dans `features/call/` (call_screen.dart, call_provider.dart). Remplacer par `debugPrint` ou logger.

### 13 `ref.read` dans `build()` (anti-pattern Riverpod)
Casse la réactivité. Sites : `search_screen.dart:114`, `opportunity_detail_screen.dart:18`, `team_screen.dart:30`, `freelance_profile_screen.dart` (×2), `client_profile_screen.dart:69`, `call_screen.dart:143`, `profile_screen.dart` (×2), `notification_screen.dart:27`, `referral_dashboard_screen.dart:17`, etc.

### 49 `Text('English string')` hardcodés non traduits
Cancel, Delete, Add, Message, etc. Devraient passer par `AppLocalizations` (1176 keys déjà existantes).

### 311 `/api/v1/` éparpillés dans `data/` repos
Centraliser dans une constante par feature ou au niveau ApiClient.

### 7 features avec couches incomplètes (data/domain/presentation)

| Feature | Présent | Manquant |
|---|---|---|
| dashboard | presentation | data, domain |
| invoice | domain | data, presentation |
| mission | domain | data, presentation |
| payment_info | presentation | data, domain |
| profile | presentation | data, domain |
| provider_profile | domain | data, presentation |
| search | data, presentation | domain |

Décider si elles doivent rester partielles ou être homogénéisées.

### 1 violation cross-feature
`lib/features/notification/presentation/providers/notification_provider.dart:5` importe `messaging_ws_service.dart` via chemin relatif `../../../../features/messaging/data/...`. Anti-pattern : passer par provider injecté ou déplacer le service en `core/`.

### 11 `Semantics(` widgets sur 533 widgets totaux
A11y mobile sous-investie. Aucun `Tooltip` Flutter. Pas de stratégie a11y documentée.

### 4 TODOs

- `mobile/lib/core/notifications/fcm_service.dart:148` (deep link FCM tap — CRITIQUE pour UX push)
- `auth/login_screen.dart:186` (forgot password navigation)
- `messaging/data/messaging_ws_service.dart:175` (single-use WS token)
- `referrer_profile_screen.dart:174` (referral_deals)

## MINOR (5)

- Generated `app_localizations*.dart` commités (3 fichiers, ~11500 lignes) — défendable mais à clarifier
- 21 `Duration(milliseconds: N)` magiques
- Pas de Tooltip Flutter
- 77 `late` keyword à auditer
- Web/admin et mobile n'ont pas de checklist a11y commune

## Strong points mobile

- Naming snake_case 100%
- 27 StateNotifier/AsyncNotifier — Riverpod bien adopté
- 264 `ref.watch` vs 167 `ref.read` — bonne proportion
- 64 `AsyncValue.when()` — bon pattern matching
- Pattern Form + TextEditingController + validator standard
- Generated code gitignored
- Mobile feature isolation 1 violation (vs 33 web) — Clean Archi globalement préservée
- 1 TODO majeur (FCM nav)

---

# Top 18 refactor priorities ordered par ROI

## Phase A — Hygiène structurelle (~3 jours)

1. **QUAL-W-01** : créer error.tsx + loading.tsx + not-found.tsx web (1 jour)
2. **QUAL-B-01** : splitter `cmd/api/main.go` + `router.go` (1.5 jour) — débloque la lisibilité du wiring
3. **QUAL-W-02** : extraire `upload-api`, `expertise-editor`, `city-autocomplete` vers `shared/` (0.5 jour) — casse 9 imports cross-feature

## Phase B — Sécurité du refactor (~1 jour)

4. **QUAL-B-02** : fix SQL injection admin_moderation (allowlist `wantType`) (30 min)
5. Logger les 15 sites `_ = err` au minimum `slog.Warn` (2h)
6. Ajouter `IF [NOT] EXISTS` aux 35+13 migrations (2h)
7. Documenter ou poser noop migration 024/025 (15 min)

## Phase C — Décomposition god files (~1 semaine)

8. **QUAL-W-03** : déplacer `payment-info/components/` vers `features/` (2h)
9. Splitter les 4 god components web (`wallet-page` 878, `message-area` 797, `search-filter-sidebar` 758, `billing-profile-form` 656) (2 jours)
10. Splitter les 4 god widgets mobile > 700 LOC (`router` 1266, `wallet_screen` 1168, `proposal_detail_screen` 1023, `billing_profile_form` 974) (2-3 jours)
11. Décomposer `ProposalHandler` (29 méthodes → 4 handlers) + `payment.Service` (1171 → 3 services) (3 jours)

## Phase D — SOLID & duplication (~3 jours)

12. Segregation top 6 god repos backend (Referral 24, Message 21, Org 20, Dispute 18, Proposal 16, User 15) (3 jours)
13. Créer primitives `Button` / `Input` shadcn dans `web/shared/components/ui/` + migrer 309+95 sites (1 jour)
14. Migrer 9 formulaires web en `react-hook-form + zod` (1 jour)
15. Extraire `pkg/sqlfilter` pour le pattern `WHERE 1=1 + paramIdx` répété 6× (0.5 jour)
16. Centraliser formatters web (formatEur, formatDate dupliqués 10×) dans `shared/lib/utils.ts` (1h)

## Phase E — Mobile cleanup (~3 jours)

17. Mobile : remplacer 491 `Color(0x...)` par `Theme.of(context)`, supprimer 18 `print()`, réduire 198 `dynamic` (2 jours)
18. Mobile : corriger 13 `ref.read` dans build, ajouter `Semantics`, cross-feature notif→messaging (1 jour)

---

## Summary

| App / Layer | Critical | Major | Minor |
|---|---|---|---|
| Backend Go | 2 | 50 | 191 |
| Web | 3 | 9 | 6 |
| Admin | 0 | 1 (tests) | 2 |
| Mobile | 3 | 10 | 5 |
| **Total** | **8** | **70** | **204** |
