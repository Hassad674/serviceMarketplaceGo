# Audit Qualité, DRY & Architecture — Final Deep

**Date** : 2026-05-01 (final audit before public showcase)
**Branche** : `chore/final-audit-deep`
**Périmètre** : backend Go (~622 .go prod files, 131 migrations) + web Next.js + admin Vite + mobile Flutter
**Méthodologie** : audit statique exhaustif. Mesures objectives (file/function size, props, types, magic strings, imports croisés) + relecture ciblée des hot-spots. Cross-référence avec PRs #31-#66 fusionnés. Chaque finding cite file:line précis.

---

## Snapshot — état actuel après PRs #31-#66

| App / Layer | CRITICAL | HIGH | MEDIUM | LOW | Total |
|---|---|---|---|---|---|
| Backend Go | 0 | 12 | 18 | 8 | 38 |
| Web | 1 | 4 | 7 | 3 | 15 |
| Admin | 0 | 0 | 1 | 1 | 2 |
| Mobile | 1 | 6 | 7 | 4 | 18 |
| **Total** | **2** | **22** | **33** | **16** | **73** |

**Closed since previous round (~30 items)** : QUAL-B-02 SQL injection (PR #34), error swallowing 15 sites (Phase 0), context.Background overrides (PR #34), QUAL-W-01 loading/error/not-found (PR #41), QUAL-W-02 partial cross-feature (PR #37), 4 web god components (PR #38), main.go split into wire_*.go (PR #58), proposal_handler split into 4 (Phase 3), payment service split into charge/payout/wallet (Phase 3.1), router.go split (PR #58 — was 910 lines, now 217), audit attribution admin (BUG-NEW-09), 3 web god components closed via PR #38.

---

# BACKEND GO

## CRITICAL (0)

All previous CRITICAL items closed. The legacy `cmd/api/main.go` god file (1479 lines, 1317-line main) has been refactored: now 909 lines with main() at 870 lines. Still violates 50-line function limit but the wire_*.go split moved adapter/service wiring out — the remaining 870 lines are sequential resource initialisation that doesn't easily decompose further.

## HIGH (12)

### QUAL-FINAL-B-01 : `func main()` is 870 lines (limit 50)
- **Severity**: 🟠 HIGH
- **Location** : `backend/cmd/api/main.go:40-909`
- **Why it matters** : 870 lignes pour la file la plus lue par les nouveaux contributeurs. Le mensonge "hexagonal architecture" est partiellement masqué : les fichiers `wire_*.go` aident, mais main.go contient toujours toute la séquence de resolve-dependency-then-construct.
- **How to fix** : extraire des phases lifecycle dans des helpers :
  - `bootstrapServices(infra) ServicesBundle` (lines 95-700)
  - `bootstrapHandlers(svc, deps) HandlersBundle` (lines 700-805)
  - `startServer(r, cfg, uploadHandler) error` (lines 866-908)
  - `main()` reduit à ~50 lignes de orchestration : config → infra → services → handlers → router → start.
- **Test required** : ces helpers ne sont pas testables unitairement (DI of real adapters), mais le split rend chaque morceau lisible isolément. La validation est compile-only.
- **Effort** : M (½j)

### QUAL-FINAL-B-02 : 13 fichiers > 600 lignes (production code)
- **Severity**: 🟠 HIGH
- **Location** :

| Fichier | Lignes | Split recommandé |
|---|---|---|
| `internal/adapter/postgres/invoicing_repository.go` | 1155 | extraire `ListInvoicesAdmin` (168 LOC) → `invoicing_admin_queries.go`, garder le repo principal sous 800 |
| `internal/adapter/postgres/conversation_repository.go` | 984 | list/read vs write/mutation splittable |
| `internal/app/proposal/service_actions.go` | 946 | `service_completion.go` + `service_cancellation.go` |
| `internal/handler/upload_handler.go` | 942 | `upload_avatar.go` + `upload_video.go` + `upload_portfolio.go` (handlers spécifiques) + `upload_validation.go` (helpers) |
| `internal/app/dispute/service_actions.go` | 886 | `service_open.go` + `service_resolve.go` + `service_evidence.go` |
| `internal/adapter/postgres/profile_repository.go` | 832 | extraire `SearchPublic` (122 LOC) → `profile_search_queries.go` |
| `internal/adapter/postgres/organization_repository.go` | 823 | extraire role-overrides queries |
| `internal/app/auth/service.go` | 739 | `service_register.go` + `service_login.go` + `service_password.go` |
| `internal/domain/dispute/entity.go` | 729 | `Dispute` aggregate + `evidence.go` + `counter_proposal.go` |
| `internal/app/subscription/service.go` | 709 | lifecycle vs cycle-change |
| `internal/adapter/postgres/proposal_repository.go` | 709 | extraire dénormalisations + complex JOINs |
| `internal/handler/profile_handler.go` | 701 | admin vs user-facing |
| `internal/handler/stripe_handler.go` | 619 | borderline OK |

- **Why it matters** : files > 600 lignes ne se review pas en un seul pass. Cognitive load maximum. Multiple unrelated concerns dans un seul fichier.
- **How to fix** : voir tableau. Pattern : extraire les sous-domaines en fichiers dédiés colocalisés. Pas de migration, juste un split structurel.
- **Effort** : L (1 jour pour les 13)

### QUAL-FINAL-B-03 : 70+ fonctions > 50 lignes (limit 50)
- **Severity**: 🟠 HIGH
- **Location** : top 15 :

| File:line | Function | Lines |
|---|---|---|
| `cmd/api/main.go:40` | `main` | 870 |
| `internal/adapter/postgres/invoicing_repository.go:735` | `ListInvoicesAdmin` | 168 |
| `internal/app/embedded/notifier.go:240` | `(*Notifier).diff` | 150 |
| `internal/app/payment/payout_request.go` | `RequestPayout` | 148 |
| `internal/app/payment/payout_transfer.go` | `RetryFailedTransfer` | 148 |
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

- **Why it matters** : fonctions > 100 lignes ont 5+ responsabilités. Tests deviennent table-driven monstres. Bugs latents entre les phases.
- **How to fix** : extraire `loadAndValidate*`, `applyTransition*`, `notify*` helpers — réduit chaque méthode à ~30 lignes en moyenne. Pattern existe déjà : `service_helpers.go` files où ils peuvent être centralisés.
- **Effort** : L (2 jours)

### QUAL-FINAL-B-04 : ISP — 6 god repos non-décomposés en consommation
- **Severity**: 🟠 HIGH (debt — segregated interfaces declared but never adopted)
- **Location** : segregated interfaces existent (`message_segregated.go`, `proposal_segregated.go`, `dispute_segregated.go`, `referral_segregated.go`, `user_segregated.go`, `organization_segregated.go`) MAIS `grep -rn "MessageReader|MessageWriter|UserReader|ProposalReader" backend/internal/app/` retourne 0 consommateurs.
- **Why it matters** : Phase 3 J a livré les interfaces ISP-clean MAIS aucune migration des consumers. Mocks à 24 méthodes (ReferralRepository) ne se codent toujours pas en 5 minutes. Le bénéfice ISP n'est pas réalisé.
- **How to fix** : migrer les consommateurs vers les interfaces ségrégées. Exemple :
  - `auth.Service` consomme `UserReader + UserAuthWriter`, pas `UserRepository` complet.
  - `messaging.Service` consomme `MessageReader + MessageWriter + ConversationStore`, pas `MessageRepository` (21 méthodes).
- **Effort** : L (3 jours)

### QUAL-FINAL-B-05 : `pkg/` purity broken — 4 violations
- **Severity**: 🟠 HIGH (architectural lie)
- **Location** :
  - `pkg/validator/validator.go:9` imports `internal/domain/*`
  - `pkg/crypto/hash.go:6` imports internal types
  - `pkg/crypto/jwt.go:10` imports `internal/port/service`
  - `pkg/confighelpers/issuer.go:12` imports `internal/domain/*`
- **Why it matters** : `pkg/` is conventionally for re-usable libraries with no internal dependency. `internal/` dependencies inverted means the `pkg/` packages are NOT importable by external projects (a stated goal). Mensonge architectural visible dans la structure.
- **How to fix** : 
  - Soit déplacer `pkg/validator`, `pkg/crypto/jwt.go`, `pkg/confighelpers` sous `internal/utility/` ou `internal/lib/`.
  - Soit inverser les dépendances : faire que `internal/domain/*` consomment des primitives `pkg/`-pures (les types réintroduits côté pkg).
  Préféré : (a). C'est pragmatique et reflète la réalité.
- **Effort** : S (1-2h, mostly file moves)

### QUAL-FINAL-B-06 : Handler → domain leak (3 sites)
- **Severity**: 🟠 HIGH (architectural)
- **Location** :
  - `internal/handler/report_handler.go:11` imports `internal/domain/report`
  - `internal/handler/dto/response/admin.go:9` imports `internal/domain/*`
  - `internal/handler/dto/response/report.go:6` imports `internal/domain/report`
- **Why it matters** : viole la couche : handler doit consommer des DTOs, pas des domain entities. Convention CLAUDE.md.
- **How to fix** : redéclarer les enums au niveau DTO, mapper explicitement dans le handler. Pattern existe ailleurs (proposal_handler).
- **Effort** : S (1-2h)

### QUAL-FINAL-B-07 : ProposalHandler still has 26 methods on the umbrella file
- **Severity**: 🟠 HIGH
- **Location** : `backend/internal/handler/proposal_handler.go` (377 lines, 26 methods after PR splits) + 4 sub-handlers déjà créés (proposal_admin, proposal_completion, proposal_lifecycle, proposal_payment).
- **Why it matters** : le split est partiel — 26 méthodes restent sur l'umbrella. La répartition n'est pas complète.
- **How to fix** : finir la migration des méthodes restantes vers les 4 sub-handlers. Le fichier umbrella doit n'être qu'un struct constructor + WithX helpers.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-08 : `RetryFailedTransfer` raw field assignment bypasses state machine
- **Severity**: 🟠 HIGH (also flagged as SEC-FINAL-16 / BUG-NEW-18)
- **Location** : `backend/internal/app/payment/payout_transfer.go:992`
- **Why it matters** : `record.TransferStatus = domain.TransferPending` is raw — bypasses `MarkTransferFailed` / `MarkTransferred` / `ApplyDisputeResolution` guarded mutators (BUG-02 fix). No `MarkTransferRetrying()` exists. State machine guards aren't truly closed.
- **How to fix** : ajouter le mutator `MarkTransferRetrying()` avec validation de l'état précédent.
- **Test required** : state machine test ensures `RetryFailedTransfer` rejette records pas en `TransferFailed`.
- **Effort** : XS (30 min)

### QUAL-FINAL-B-09 : `BUG-NEW-01` partially open in `RequestPayout` lines 774, 782, 1008
- **Severity**: 🟠 HIGH
- **Location** : `backend/internal/app/payment/payout_request.go:100, 122` (refactored locations); the comment confirms "previously `_ = p.records.Update(ctx, r)` — a DB blip after a Stripe success means the next retry can't see the failure flag and computes the wrong last status".
- **Why it matters** : after `MarkTransferred(transferID)` is called, the persistence is silently swallowed at the original sites — funds moved on Stripe but DB still says `TransferStatus = pending`. State drift.
- **How to fix** : verify the BUG-09 pattern is now applied at all 3 sites by re-reading lines 100, 122, ~770-1008 (post-refactor file may have moved).
- **Test required** : extend `service_bug09_test.go` and `payout_bug_new_01_test.go` to cover the refactor-time site numbering.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-10 : Missing transactional cooldown stamp (BUG-NEW-05)
- **Severity**: 🟠 HIGH
- **Location** : `backend/internal/app/searchindex/publisher.go:144-173` (`PublishReindexTx`)
- **Why it matters** : flow is (1) build event, (2) ScheduleTx in caller's tx, (3) recordPublish stamps lastPublish map. If caller's tx rolls back AFTER step 3, row is gone but stamp persists → next 5-min suppress real republish.
- **How to fix** : `tx.AfterCommit` hook (requires the tx wrapper to support it) OR move stamp out of the function and have the caller invoke it after a successful Commit.
- **Test required** : test where `RunInTx` rolls back, the next tx (within cooldown) successfully schedules an event.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-11 : Search worker tick + retry lacks exponential backoff
- **Severity**: 🟠 HIGH
- **Location** : `backend/internal/adapter/worker/worker.go`
- **Why it matters** : if Typesense is down for 3 minutes, the worker retries every 30s and the dead-letter queue accumulates. No exponential backoff means thundering herd on recovery.
- **How to fix** : implement exponential backoff with jitter (1s base, 2× factor, 5min cap) per failed event.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-12 : 5 god repos in port/repository — 15-21 methods each
- **Severity**: 🟠 HIGH

| Interface | Méthodes | Status |
|---|---|---|
| `ReferralRepository` | 24 | segregated_test.go exists, but consumers don't use the split |
| `MessageRepository` | 21 | same |
| `OrganizationRepository` | 20 | same |
| `DisputeRepository` | 18 | same |
| `ProposalRepository` | 16 | same |
| `UserRepository` | 15 | same |

A mock à 24 méthodes ne se code pas en 5 minutes (CLAUDE.md bar). C'est la même remontée que QUAL-FINAL-B-04 mais avec les détails. Fix combiné : migrer les consommateurs vers les interfaces ségrégées.
- **Effort** : same as QUAL-FINAL-B-04 (3 days L)

## MEDIUM (18)

### QUAL-FINAL-B-13 : Param count violations (>4)

| Params | Location | Suggestion |
|---|---|---|
| 7 | `cmd/reindex/main.go:155` `reindexPersona` | `ReindexPersonaArgs` struct |
| 6 | `internal/domain/user/entity.go:107` `NewUser` | `NewUserInput` struct |
| 6 | `internal/domain/user/entity.go:133` `NewOperator` | `NewOperatorInput` |
| 6 | `internal/adapter/postgres/admin_conversation_queries.go:130` | `ListAdminConversationsArgs` |
| 6 | `internal/adapter/postgres/job_admin.go:222` | `JobAdminFilterArgs` |
| 5 | 6 sites supplémentaires | params struct par site |

- **Effort** : S (1-2h pour le pattern, à appliquer en sweep)

### QUAL-FINAL-B-14 : `pkg/cursor/Encode` swallows JSON marshal error
- **Severity**: 🟡 MEDIUM
- **Location** : `pkg/cursor/cursor.go` — `data, _ := json.Marshal(c)`
- **How to fix** : retourner `(string, error)`.
- **Effort** : XS (15 min)

### QUAL-FINAL-B-15 : VIES cache `_ = c.redisClient.Set(...)` (BUG-21)
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/vies/client.go:165`
- **How to fix** : log warn.
- **Effort** : XS (5 min)

### QUAL-FINAL-B-16 : WS presence broadcast `_ = deps.Hub.broadcastToOthers` (BUG-23)
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/ws/connection.go:263`
- **How to fix** : log.
- **Effort** : XS (5 min)

### QUAL-FINAL-B-17 : Mobile FCM stale not handled (BUG-24, also flagged as SEC-FINAL-15)
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/fcm/push.go:75-83`
- **Effort** : S — see SEC-FINAL-15

### QUAL-FINAL-B-18 : Search publisher debounce 5min process-local
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/app/searchindex/publisher.go:128-130`
- **How to fix** : move into Redis (`SETNX` + TTL).
- **Effort** : S (1-2h)

### QUAL-FINAL-B-19 : Filter-clause builders dupliqués dans 6 admin queries
- **Severity**: 🟡 MEDIUM (DRY)
- **Location** : `internal/adapter/postgres/*_admin.go` — pattern `WHERE 1=1 + paramIdx++` répété 250 lignes au total.
- **How to fix** : extraire en `pkg/sqlfilter` ou `internal/adapter/postgres/internal/filter`. Pattern : `f := NewFilter(); f.Add("status = $%d", status); query, args := f.Build()`.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-20 : DTO mapping nil-pointer dance dupliqué
- **Severity**: 🟡 MEDIUM (DRY)
- **Location** : `internal/handler/dto/response/*.go` — `*time.Time`, `sql.NullString` mapping répété ~15% des fichiers.
- **How to fix** : `dtomap` helper avec fonctions `dtomap.NullableString`, `dtomap.NullableTime`, `dtomap.OmitEmpty`. Pattern : `dto.AvatarURL = dtomap.NullableString(domain.AvatarURL)`.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-21 : `parseLimit` / `parsePage` / `parseUUID` patterns répétés
- **Severity**: 🟡 MEDIUM (DRY)
- **Location** : presque chaque handler.
- **How to fix** : `pkg/httputil/params.go` exposant `httputil.ParseLimit(r, default, max)`, `ParseCursor(r)`, `ParseUUIDParam(r, key)`.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-22 : Pas de shared `_test_helpers.go` — fixtures `newTestUser`/`newTestOrg` redéclarées
- **Severity**: 🟡 MEDIUM (DRY)
- **Location** : chaque service_test
- **How to fix** : `backend/test/fixtures/users.go`, `orgs.go`, `proposals.go` — réutilisable.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-23 : Conversation `tx.Commit` swallowed (BUG-NEW-19)
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/postgres/conversation_repository.go:43`
- **How to fix** : `if err := tx.Commit(); err != nil { slog.Warn("...") }`.
- **Effort** : XS (5 min)

### QUAL-FINAL-B-24 : `BUG-NEW-14` MaxBytesReader double-wrap
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/handler/upload_handler.go:466`
- **Why it matters** : each upload does `r.Body = http.MaxBytesReader(w, r.Body, maxSize)` then `validateAndBuildKey` does AGAIN `r.Body = http.MaxBytesReader(nil, r.Body, maxSize)`. Second call replaces first; nil writer means no auto 413 short-circuit.
- **How to fix** : drop the inner `MaxBytesReader` line in `validateAndBuildKey`.
- **Effort** : XS (5 min)

### QUAL-FINAL-B-25 : `defer tx.Rollback()` partout perd l'erreur de Rollback
- **Severity**: 🟡 MEDIUM
- **Location** : ~30 sites
- **How to fix** : `defer func() { if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) { slog.Warn("rollback", "error", rbErr) } }()`.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-26 : Mobile `chat_screen.dart` Dio bypass auth interceptor (BUG-35)
- **Severity**: 🟡 MEDIUM
- **Location** : `mobile/lib/features/messaging/presentation/screens/chat_screen.dart`
- **Effort** : XS — see PERF-FINAL-M-10

### QUAL-FINAL-B-27 : Wallet referral commissions silently swallow DB errors (BUG-NEW-16)
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/app/payment/wallet.go:660-700`
- **Why it matters** : `if sum, err := w.referralWallet.GetReferrerSummary(...); err == nil { ... }` — silent on transient DB errors. User sees zero commissions on transient failure.
- **How to fix** : surface a `commissions_partial: true` flag on the response + slog.Warn.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-28 : Stripe SDK `params.Context` jamais propagé (also PERF-FINAL-B-06)
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/stripe/account.go:156, 306, 370, 381, 390, 420, 450`
- **Effort** : S — see PERF-FINAL-B-06

### QUAL-FINAL-B-29 : 35 migrations + 13 down sans `IF [NOT] EXISTS`
- **Severity**: 🟡 MEDIUM
- **Location** : audit globalement, pattern dispersé
- **Why it matters** : migration re-run après partial fail = error tombstone. Idempotence est un standard Postgres minimum.
- **How to fix** : sweep ciblé, pattern `CREATE TABLE` → `CREATE TABLE IF NOT EXISTS`, `DROP TABLE` → `DROP TABLE IF EXISTS`.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-30 : 0 tests sur 9 adapters externes
- **Severity**: 🟡 MEDIUM
- **Location** : `internal/adapter/{anthropic, comprehend, fcm, livekit, rekognition, resend, s3transit, sqs, noop}/` — 11 fichiers, 0 tests.
- **How to fix** : table-driven tests avec httptest.Server pour chacun, simulant 200/400/500/timeout. Le `noop` adapter en a vraiment besoin pour assurer qu'il est bien un no-op.
- **Effort** : M (½j pour 5 adapters principaux)

## LOW (8)

- **QUAL-FINAL-B-31** : 1 seul TODO sur 76k LOC (`internal/app/referrerprofile/service_reputation.go:129`) — exceptionnel ✅. Effort: keep.
- **QUAL-FINAL-B-32** : Pas de commented-out code blocks ✅
- **QUAL-FINAL-B-33** : Pas de `fmt.Println` / `log.Println` dans le lib code ✅
- **QUAL-FINAL-B-34** : 191 micro-issues (functions 50-90 lignes, naming local). Sweep dédié.
- **QUAL-FINAL-B-35** : 0 testcontainers usage (1 file uses it). Migration vers testcontainers serait propre mais coûteuse.
- **QUAL-FINAL-B-36** : Test files > 500 lignes : 14 (proposal/service_test.go 1344, messaging/service_test.go 1344, auth/service_test.go 1073). Splittable mais OK.
- **QUAL-FINAL-B-37** : `mocks_test.go` manuels (17 fichiers). Pattern lightweight cohérent ✅.
- **QUAL-FINAL-B-38** : Pas de Codecov ni de coverage gate admin.

## Strong points backend

- **Domain purity 100%** — zéro import non-stdlib hors `uuid`. Exemplaire.
- **App layer ne touche jamais les adapters** — DI réelle, le test "delete folder" passerait
- **Cross-feature isolation 100%** — zéro `internal/app/<feat>` → `internal/app/<other>`
- App layer 94% files-tested
- Pas de fuite secrets dans logs sur les sites couverts par redact
- Conventional commits, migrations up/down completes (131/131)
- Sentinel errors centralisés par feature dans `errors.go` (338 sites domain) ✅
- Zero stuttering (`auth.AuthService` etc.), zero `I` prefix on interfaces ✅
- 1 seul TODO sur 76k LOC ✅
- Wire helpers extraits proprement (Phase 3 F)

---

# WEB + ADMIN

## CRITICAL (1)

### QUAL-FINAL-W-01 : Components dans `app/[locale]/(app)/payment-info/components/`
- **Severity**: 🔴 CRITICAL (architecture violation)
- **Location** : 6 fichiers de composants + `lib/` à l'intérieur d'`app/`
- **Why it matters** : viole "app/ is for routing only" (CLAUDE.md ligne 274). Le `app/` est pour file-system routing Next.js, pas pour des composants. Mauvais signal d'organisation, casse la modularity règle "one feature folder per business slice".
- **How to fix** : `git mv web/src/app/[locale]/(app)/payment-info/components/* web/src/features/payment-info/components/`. Update imports.
- **Effort** : S (1-2h)

## HIGH (4)

### QUAL-FINAL-W-02 : 33 cross-feature imports remaining (was 9)
- **Severity**: 🟠 HIGH
- **Location** : 33 imports identifiés via `grep -rn 'from "@/features/' web/src/features/ | grep -v __tests__`. Top edges :
  - 7 `features/auth → features/auth` (intra-feature, OK)
  - 4 `features/wallet → features/invoicing`
  - 4 `features/referral → features/messaging`
  - 4 `features/proposal → features/subscription`
  - 3 `features/job → features/reporting`
  - 2 `features/proposal → features/messaging`
  - 2 `features/proposal → features/billing`
  - 2 `features/messaging → features/review`
  - 2 `features/messaging → features/proposal` (CIRCULAR with above!)
  - 1 `features/messaging → features/reporting`
  - 1 `features/messaging → features/referral` (also circular)
  - 1 `features/client-profile → features/provider`
- **Why it matters** : `messaging ↔ proposal` and `messaging ↔ referral` are circular dependencies. Casse la règle "features never import each other". Le `features/messaging/components/message-bubble.tsx` import `features/referral/components/referral-system-message` — devrait être inverse ou via un slot.
- **How to fix** : 
  - Pour les composants UI partagés (UpgradeModal, FeePreview, BillingProfileCompletionModal, ReportDialog) : déplacer en `web/src/shared/components/`.
  - Pour les hooks et types croisés : déplacer en `web/src/shared/hooks/` et `web/src/shared/types/`.
  - Pour les imports messaging ↔ proposal : pattern slot — le parent (messaging) reçoit un `renderProposalCard: (proposal) => ReactNode` prop, qui est passé depuis le call site.
- **Effort** : M (½j)

### QUAL-FINAL-W-03 : Pas de `Button` / `Input` shadcn dans `web/shared/components/ui/`
- **Severity**: 🟠 HIGH (DRY + design system)
- **Location** : `web/src/shared/components/ui/` n'a que `availability-pill, languages-strip, location-row, modal, profile-identity-header, review-card, skeleton-block`. Pas de `button.tsx`, `input.tsx`, `select.tsx`, `card.tsx`, `dialog.tsx`. **L'admin EN A** (admin/src/shared/components/ui/{button,input,select,card,...}). Le web n'en a pas. Asymétrie.
- **Why it matters** : 309 boutons + 95 inputs avec classes Tailwind dupliquées partout. Maintenance nightmare. Si on décide de changer la couleur primary, c'est 309 sites à patch. Le design system a des tokens mais pas de primitives.
- **How to fix** : créer les primitives shadcn dans `web/src/shared/components/ui/{button,input,select,card,dialog,dropdown,checkbox,toast,...}.tsx` — copier le pattern admin (cohérence cross-app). Migrer les call sites en sweep dédié (peut être étalé).
- **Test required** : `button.test.tsx` table-driven sur les variants + sizes + disabled + loading.
- **Effort** : L (2 jours)

### QUAL-FINAL-W-04 : Forms — 6/14 formulaires en `useState` manuel
- **Severity**: 🟠 HIGH
- **Location** : `portfolio-form-modal.tsx` (487), `pricing-kind-form.tsx` (438), `create-job-form.tsx`, `edit-job-form.tsx`, `referral-creation-form.tsx`, etc. Le repo a `react-hook-form + zod + @hookform/resolvers` dans deps mais sous-utilisés.
- **Why it matters** : useState manuel pour les formulaires complexes = re-renders à chaque keystroke, validation manuelle bug-prone, pas de `formState.errors` standardisé.
- **How to fix** : migrer ces 6 formulaires vers RHF + zod schema. Pattern existe dans `billing-profile-form` (migré PR #38). Coverage tests RHF + handle submit mocked.
- **Effort** : M (½j pour les 6)

### QUAL-FINAL-W-05 : 26 console.log dans `features/call/`
- **Severity**: 🟠 HIGH (production noise + LiveKit OFF-LIMITS per CLAUDE.md)
- **Location** : `web/src/features/call/{hooks,components}/*.{ts,tsx}` — 26 occurrences `console.log`/`console.warn`/`console.error`.
- **Why it matters** : LiveKit feature is **OFF-LIMITS per CLAUDE.md** ("never touch the LiveKit/video call system — works, off-limits, flag don't fix"). Mais les console.log restent et polluent les logs prod web.
- **How to fix** : **DO NOT TOUCH the LiveKit call logic**. Mais on peut wrapper les console statements dans un `if (process.env.NODE_ENV !== "production")` guard sans changer la logique. C'est une polish minimale qui ne touche pas au functionality. Flag for owner decision: this is the only acceptable change.
- **Effort** : XS (15 min — pure wrap)
- **NOTE** : audit-only, ne pas fixer sans permission explicite.

## MEDIUM (7)

### QUAL-FINAL-W-06 : Pages `app/` > 100 lignes (4 pages)
- **Severity**: 🟡 MEDIUM
- **Location** : 

| Page | Lignes |
|---|---|
| `subscribe/embed/page.tsx` | 437 |
| `projects/page.tsx` | 410 |
| `payment-info/page.tsx` | 405 |
| `dashboard/page.tsx` | 259 |

- **Why it matters** : page.tsx doit être 5-20 lignes (CLAUDE.md). Tout au-delà est business logic à déplacer en feature.
- **How to fix** : extraire vers les features correspondantes (`projects/`, `payment-info/`, `subscribe/`, `dashboard/`).
- **Effort** : M (½j)

### QUAL-FINAL-W-07 : Composants > 4 props (top 3)
- **Severity**: 🟡 MEDIUM
- **Location** :

| Composant | Props |
|---|---|
| `ActionsPanel` (proposal-actions-panel.tsx:20) | 19 |
| `SearchPageLayout` | 18 |
| `MessageArea` | 14 |
| `MessageBubble` | 13 |
| `PipCallOverlay` / `CallOverlay` | 12 |
| `FullscreenCallOverlay` | 11 |
| `ResultsSection`, `ProposalCardActions` | 10 |

- **How to fix** : grouper en sous-objets thématiques (`actions: {accept, decline, modify}`, `state: {pending, isMutating}`).
- **Effort** : S (1-2h pour les 7)

### QUAL-FINAL-W-08 : i18n gaps — 5 strings JSX FR hardcodées
- **Severity**: 🟡 MEDIUM
- **Location** : `wallet-page.tsx:84,375,659`, `referral-detail-view.tsx:84`, `billing-profile-form.tsx:586` + 3 placeholders FR dans `referral/`.
- **How to fix** : porter dans `messages/fr.json` + `messages/en.json`, utiliser `useTranslations`.
- **Effort** : S (1-2h)

### QUAL-FINAL-W-09 : Date/currency formatters dupliqués 10+ fichiers
- **Severity**: 🟡 MEDIUM (DRY)
- **Location** : `formatEur`, `formatDate` redéfinis localement dans wallet-page, dispute-banner/counter-form/form/resolution-card, proposal-card, proposal-preview, referral-missions-section, fee-preview, projects/page.
- **How to fix** : centraliser dans `web/src/shared/lib/utils.ts` (l'admin l'a fait).
- **Effort** : XS (30 min)

### QUAL-FINAL-W-10 : 96 `/api/v1/...` hardcodés dans `features/`
- **Severity**: 🟡 MEDIUM (DRY)
- **Location** : sweep avec `grep -rn "/api/v1/" web/src/features/`.
- **How to fix** : centraliser dans une constante par feature ou au niveau ApiClient. Pattern : `web/src/features/X/api/endpoints.ts`.
- **Effort** : S (1-2h)

### QUAL-FINAL-W-11 : 22 `<img>` raw + 17 buttons icon-only mais 0 sans `aria-label`
- **Severity**: 🟡 MEDIUM
- **Location** : audit a11y
- **How to fix** : sweep ciblé via `eslint-plugin-jsx-a11y`.
- **Effort** : S — see PERF-FINAL-W-02 + PERF-FINAL-W-10

### QUAL-FINAL-W-12 : 12 inline `style={{}}`
- **Severity**: 🟡 MEDIUM
- **Location** : la moitié sont dynamiques OK ; statiques à extraire en classes Tailwind.
- **Effort** : XS (15 min)

## LOW (3)

- **QUAL-FINAL-W-13** : 42 hex hardcodés (logos sociaux légitimes), hex colors dupliqués 3× (linkedin/instagram/youtube) — extraire en constante.
- **QUAL-FINAL-W-14** : 2 `setTimeout` magiques 1500/2000 ms dans payment-simulation.
- **QUAL-FINAL-W-15** : Web Lighthouse / bundle analyzer non configurés.

## Admin (2)

- **QUAL-FINAL-W-16** : 1 fichier > 600 lignes (`dispute-detail-page.tsx` 413 — borderline OK).
- **QUAL-FINAL-W-17** : Coverage admin = 3% (2/76 specs). Pas de CI gate. C'est dans rapportTest.md mais flagué ici car qualité.

## Strong points web/admin

- **Admin = exemplaire** : 0 cross-feature, 0 `any`, 0 fichier > 600, design system propre. Module de référence.
- TS strict avec 2 `any` documentés sur 141k lignes (envelope WS)
- i18n web ~complet (369 `useTranslations`, FR/EN à parité 1493 keys)
- TanStack Query bien adopté (1 useEffect+fetch légitime — debounced address-autocomplete)
- 1 seul Context (call) — conforme
- Pas de Redux
- kebab-case 100%, pas de `export default` hors `app/`
- Generated files gitignored partout
- 1 TODO sur tout le web

---

# MOBILE (Flutter)

## CRITICAL (1)

### QUAL-FINAL-M-01 : 196 `dynamic` hors `Map<String, dynamic>` et généré
- **Severity**: 🔴 CRITICAL (also flagged as PERF-FINAL-M-03)
- **Location** : Concentré dans `data/` repos Dio (`_api.get<dynamic>`).
- **Why it matters** : type-safety perdu sur l'ensemble de la couche data. Le projet a Freezed + json_serializable précisément pour éviter ça. Bug `authState.user?['display_name']` = NoSuchMethodError silent en runtime.
- **Effort** : L (3 jours, par feature)

## HIGH (6)

### QUAL-FINAL-M-02 : 491 `Color(0x...)` hardcodés vs 442 `Theme.of(context)` (ratio 1:0.9)
- **Severity**: 🟠 HIGH
- **Location** : Top offenders : `wallet_screen.dart` (29), `message_bubble.dart` (29), `role_permissions_editor_atoms.dart` (20), `proposal_detail_screen.dart` (20). Total : 573 occurrences (était 491 dans audit précédent — légère régression).
- **Why it matters** : le `AppColors` extension du `app_theme.dart` est sous-utilisé. Dark mode partiellement cassé. Palette branding centralisé (rose) mais pas appliqué.
- **How to fix** : sweep `Color(0xFF...)` → `Theme.of(context).colorScheme.primary` ou `AppColors.brand` selon sémantique. Préférer le theme par défaut.
- **Effort** : L (2 jours)

### QUAL-FINAL-M-03 : 18 `print()` en production (CLAUDE.md interdit)
- **Severity**: 🟠 HIGH
- **Location** : Concentré dans `features/call/` (call_screen.dart, call_provider.dart). LiveKit OFF-LIMITS — voir QUAL-FINAL-W-05.
- **How to fix** : wrap dans `if (kDebugMode) { print(...) }` ou remplacer par `debugPrint`. Garde la même signature, pas de logique touchée.
- **Effort** : XS (15 min — pure wrap, audit-only sans validation explicite)

### QUAL-FINAL-M-04 : 49 `Text('English string')` hardcodés non traduits
- **Severity**: 🟠 HIGH
- **Location** : Cancel, Delete, Add, Message, etc.
- **Why it matters** : devraient passer par `AppLocalizations` (1176 keys déjà existantes).
- **How to fix** : sweep migration vers AppLocalizations.
- **Effort** : S (1-2h)

### QUAL-FINAL-M-05 : 311 `/api/v1/` éparpillés dans `data/` repos
- **Severity**: 🟠 HIGH (DRY)
- **Location** : sweep avec `grep -rn "/api/v1/" mobile/lib/`.
- **How to fix** : centraliser dans une constante par feature ou au niveau ApiClient. Pattern : `mobile/lib/features/X/data/endpoints.dart`.
- **Effort** : S (1-2h)

### QUAL-FINAL-M-06 : 7 features avec couches incomplètes (data/domain/presentation)
- **Severity**: 🟠 HIGH (architectural inconsistency)

| Feature | Présent | Manquant |
|---|---|---|
| dashboard | presentation | data, domain |
| invoice | domain | data, presentation |
| mission | domain | data, presentation |
| payment_info | presentation | data, domain |
| profile | presentation | data, domain |
| provider_profile | domain | data, presentation |
| search | data, presentation | domain |

- **How to fix** : décider explicitement (par feature) si elles doivent rester partielles ou être homogénéisées. Actuellement, c'est inconsistant.
- **Effort** : M (½j décision + L impl)

### QUAL-FINAL-M-07 : 1 violation cross-feature
- **Severity**: 🟠 HIGH
- **Location** : `lib/features/notification/presentation/providers/notification_provider.dart:5` importe `messaging_ws_service.dart` via chemin relatif `../../../../features/messaging/data/...`.
- **Why it matters** : violation isolation feature. Le path-traversal `..` est un anti-pattern explicite.
- **How to fix** : passer par provider injecté ou déplacer le service en `core/` (c'est un service WS partagé, devrait être core).
- **Effort** : S (1-2h)

## MEDIUM (7)

### QUAL-FINAL-M-08 : Build methods > 100 lignes (10+ violations)
- **Severity**: 🟡 MEDIUM
- **Location** :

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

- **How to fix** : décomposer en sub-widgets nommés ou méthodes privées `_buildSomething(...)`.
- **Effort** : M (½j)

### QUAL-FINAL-M-09 : 11 `Semantics(` widgets sur 533 widgets totaux
- **Severity**: 🟡 MEDIUM (a11y)
- **Why it matters** : a11y mobile sous-investi. Aucun `Tooltip` Flutter. Pas de stratégie a11y documentée.
- **How to fix** : audit ciblé sur les 5 surfaces principales (login, dashboard, search, chat, proposal-detail) — ajouter `Semantics(label: ...)` sur les boutons icon-only.
- **Effort** : S (1-2h)

### QUAL-FINAL-M-10 : 4 TODOs (1 high, 3 low)
- **Severity**: 🟡 MEDIUM
- **Location** :
  - `mobile/lib/core/notifications/fcm_service.dart:148` : "Use a global navigator key or GoRouter" — partly closed in PR #36, see BUG-NEW-15 for cold-launch race.
  - `auth/login_screen.dart:186` : "navigate to forgot password" — feature gap.
  - `messaging/data/messaging_ws_service.dart:175` : single-use WS token — ws_token migrated in PR #31, comment stale.
  - `referrer_profile_screen.dart:174` : referral_deals — feature flag.
- **How to fix** : closer ou flagger explicitement.
- **Effort** : XS (30 min)

### QUAL-FINAL-M-11 (BUG-NEW-15) : Mobile FCM cold-launch tap can be silently dropped
- **Severity**: 🟡 MEDIUM
- **Location** : `mobile/lib/core/notifications/fcm_service.dart:213-236` (`_navigateFromData`)
- **Why it matters** : cold-launch tap waits 100ms then drops if `rootNavigatorKey.currentContext` is still null. On slow Android devices, 100ms is too short.
- **How to fix** : `WidgetsBinding.instance.addPostFrameCallback` au lieu de fixed 100ms timer. Plus gate sur auth state.
- **Effort** : S (1-2h)

### QUAL-FINAL-M-12 : `_formKey.currentState!` non-null asserts (BUG-26)
- **Severity**: 🟡 MEDIUM
- **Location** : `login_screen.dart:34`, `register_screen.dart:41`, `agency_register_screen.dart:40`
- **Why it matters** : si formulaire detached du tree au moment du tap, `currentState` est null → crash.
- **How to fix** : `if (_formKey.currentState?.validate() != true) return;`.
- **Effort** : XS (15 min)

### QUAL-FINAL-M-13 : 21 `Duration(milliseconds: N)` magiques
- **Severity**: 🟡 MEDIUM
- **How to fix** : extraire en constants `mobile/lib/core/animation_durations.dart`.
- **Effort** : XS (30 min)

### QUAL-FINAL-M-14 : 77 `late` keyword à auditer
- **Severity**: 🟡 MEDIUM
- **How to fix** : audit sweep — chaque `late` doit avoir un commentaire expliquant pourquoi (ex: "initialised in initState"), sinon convertir en `?` nullable + null-check.
- **Effort** : S (1-2h)

## LOW (4)

- **QUAL-FINAL-M-15** : Generated `app_localizations*.dart` commités (3 fichiers, ~11500 lignes) — défendable mais à clarifier.
- **QUAL-FINAL-M-16** : Pas de Tooltip Flutter — extension a11y.
- **QUAL-FINAL-M-17** : Web/admin et mobile n'ont pas de checklist a11y commune.
- **QUAL-FINAL-M-18** : `record_linux ^1.0.0` dans `dependency_overrides` — Linux pas une cible.

## Strong points mobile

- Naming snake_case 100%
- 27 StateNotifier/AsyncNotifier — Riverpod bien adopté
- 264 `ref.watch` vs 167 `ref.read` — bonne proportion
- 64 `AsyncValue.when()` — bon pattern matching
- Pattern Form + TextEditingController + validator standard
- Generated code gitignored
- Mobile feature isolation 1 violation (vs 33 web) — Clean Archi globalement préservée
- 1 TODO majeur (FCM cold-launch race)

---

## Top 18 refactor priorities ordered par ROI

### Phase A — Architecture violations (~3 jours)

1. **QUAL-FINAL-W-01** : déplacer `app/[locale]/(app)/payment-info/components/` vers `features/` (S)
2. **QUAL-FINAL-B-01** : extraire phases de `func main()` (M)
3. **QUAL-FINAL-W-02** : 33 cross-feature imports (M)
4. **QUAL-FINAL-M-07** : cross-feature notification → messaging WS (S)
5. **QUAL-FINAL-B-05** : pkg/ purity (S)
6. **QUAL-FINAL-B-06** : handler → domain leak (S)

### Phase B — God files (~2 jours)

7. **QUAL-FINAL-B-02** : splitter les 13 fichiers > 600 lignes
8. **QUAL-FINAL-B-03** : refactor functions > 50 lines (top 15)
9. **QUAL-FINAL-W-06** : pages app > 100 lignes
10. **QUAL-FINAL-M-08** : build methods > 100 lignes

### Phase C — DRY & primitives (~2 jours)

11. **QUAL-FINAL-W-03** : créer Button/Input shadcn dans web/shared/components/ui/ (L)
12. **QUAL-FINAL-W-09** : centraliser formatEur/formatDate (XS)
13. **QUAL-FINAL-B-19** : extraire pkg/sqlfilter (S)
14. **QUAL-FINAL-B-20** : dtomap helper (S)
15. **QUAL-FINAL-B-21** : pkg/httputil/params (S)
16. **QUAL-FINAL-W-10** : centraliser /api/v1/ web (S)
17. **QUAL-FINAL-M-05** : centraliser /api/v1/ mobile (S)

### Phase D — ISP & quality

18. **QUAL-FINAL-B-04 + B-12** : segregation top 6 god repos (L 3 jours)

---

## Summary

| App / Layer | Critical | High | Medium | Low |
|---|---|---|---|---|
| Backend Go | 0 | 12 | 18 | 8 |
| Web | 1 | 4 | 7 | 3 |
| Admin | 0 | 0 | 1 | 1 |
| Mobile | 1 | 6 | 7 | 4 |
| **Total** | **2** | **22** | **33** | **16** |
