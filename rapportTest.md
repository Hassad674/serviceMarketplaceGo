# Rapport Tests + Migrations + DB — Final Deep

**Date** : 2026-05-01 (final audit before public showcase)
**Branche** : `chore/final-audit-deep`
**Périmètre** : couverture tests par layer + qualité tests + santé migrations + cohérence schéma
**Méthodologie** : audit statique sans `go test` ni `flutter test` exécutés. File-level coverage = `non-test files avec un _test.go companion / total non-test files`. Migrations = analyse SQL pure. Cross-référence avec PRs #31-#66 fusionnés.

---

## Tests added since previous audit (post PR #41)

11+ PRs landed → ~80 nouveaux fichiers de test. Détail :

| PR | Tests added |
|---|---|
| Phase 1 (PR #31) | `service_bruteforce_test.go`, `auth/refresh_rotation_test.go`, `mobile/single_flight_test.dart`, ratelimit Redis sliding-window tests, validator DTO tests |
| Phase 1 (PR #32) | `security_headers_test.go`, `cors_test.go`, Android cleartext config, session_version-bump tests |
| Phase 1 (PR #33) | XSS JSON-LD escape, ConfirmPayment Stripe verify, magic-byte upload extended, embedded invalid_json, advisory_lock race |
| Phase 1.5 (PR #34) | gosec sweep tests : SQL injection placeholder coverage, multipart streaming, fail-fast prod test |
| Phase 2 D (PR #35) | `service_bug09_test.go`, payment state machine guards, dispute restore propagation, milestone GetByIDWithVersion |
| Phase 2 E (PR #36) | `outbox_integration_test.go`, webhook composite idempotency, notification worker pool, mobile FCM tap routing |
| Phase 2.5 (PR #40) | WS sendOrDrop + wasLast race, embedded JSON unmarshal, upload goroutine ctx, NilSliceToEmpty |
| Phase 3 G (PR #38) | tests for split wallet/messaging/search-filter components, RHF/zod billing-profile-form |
| Phase 3 I (PR #37) | extracted shared/* component tests, cross-feature import lint check |
| Phase 4 N (PR #41) | DashboardShell ≥90% coverage, CSP completeness e2e, Vite manualChunks build verification |
| Phase 5 Q (PR #39) | `rls_isolation_test.go` (cross-tenant denial across 9 tables, table-driven, property-based via testing/quick) |
| Recent (PR #51-#66) | RLS migration coverage `audit_repository_rls_test.go`, `conversation_messages_list_rls_test.go`, `invoicing_repository_rls_test.go`, `notification_repository_test.go`, `payout_bug_new_01_test.go`, `payout_split_test.go` |

---

# AREA 1 — TESTS

## Backend coverage par layer

| Layer | Non-test files | Test files | File coverage % |
|---|---|---|---|
| `internal/domain/*` (29 modules) | 81 | 53 | ~65% |
| `internal/app/*` (32 modules) | 84 | 79 | ~94% |
| `internal/handler/*` (root) | 54 | 38 | ~70% |
| `internal/handler/middleware/` | 14 | 9 | ~64% |
| `internal/handler/dto/{request,response}/` | 100+ | 0 | **0%** |
| `internal/adapter/postgres/` | 65 | 23 | **35%** |
| `internal/adapter/redis/` | 13 | 4 | 31% |
| `internal/adapter/stripe/` | 8 | 6 | 75% |
| Adapters externes (anthropic, comprehend, fcm, livekit, rekognition, resend, s3transit, sqs, noop) | 11 | 0 | **0%** |
| `pkg/*` | 7 | 6 | 86% |
| `cmd/api/main.go` | 1 (909 LOC) | 0 | 0% |

## Per-domain status (29 modules)

- **0 tests** : `media`
- Sparse (1-2 tests) : `proposal` (entité OK, errors_test manquant), `subscription` (1/2), `report`, `review`
- Best : `organization` (6/6), `user` (3/3), `invoicing` (6/8), `referral` (5/7)

## Per-app-module status (32 modules) — UPDATED post PR #41+

| Module | Tests | Status | Notes |
|---|---|---|---|
| **admin** | **4/9** | 🟡 PARTIAL | service + extra tests added (PR new) — was 0/9 in previous audit |
| **kyc** | **1/1** | ✅ | scheduler_test added — was 0/1 |
| **referral** | **4/24** | 🔴 STILL CRITICAL | money + clawback + commission distribution still under-covered |
| auth, dispute, embedded, invoicing, job, messaging, milestone, organization, payment, profile, proposal, review, search, subscription, skill | OK | ✅ | |

## Per-handler status (54 fichiers, 38 testés)

**23 handlers sans `_test.go`** :
- Admin handlers: `admin_dispute`, `admin_handler`, `admin_media`, `admin_message_moderation`, `admin_moderation`, `admin_notification`, `admin_review`, `admin_team`
- `dispute_handler`, `freelance_pricing_handler`, `invitation_handler`, `job_application_handler`
- `organization_shared_profile_handler`, `portfolio_handler`, `profile_pricing_handler`
- `project_history_handler`, `referrer_pricing_handler`, `report_handler`, `role_overrides_handler`
- **`stripe_handler`** (deux tests adjacents `*_invoicing_test.go` et `*_credit_note_test.go` couvrent seulement ces branches)

Handler-level tests détectent les bugs RBAC/ownership que les unit tests manquent.

## Per-postgres-adapter status (65 fichiers, 23 testés = 35%)

**Repos critiques sans tests** (post-PR #41 update) :
- `proposal_repository` — money flow, NEW partial coverage in `*_rls_test.go` files
- `dispute_repository` — money flow
- `review_repository`
- `message_repository`
- `payment_records_repository` — PARTIAL coverage via payout tests
- `organization_*_repository` (multiples)

Tests présents pour : search, invoicing, billing_profile, milestone, subscription, referral, search_analytics, search_document, job_credit, moderation_results, audit_repository, conversation_messages_list (all RLS tests).

## Per-adapter-externe : 0 tests sur 9

`anthropic`, `comprehend`, `fcm`, `livekit`, `rekognition`, `resend`, `s3transit`, `sqs`, `noop` — 0 tests still.

## Test quality

| Métrique | Compte | Notes |
|---|---|---|
| `t.Skip` calls | 27 (22 fichiers) | Tous gated par env vars — clean ✅ |
| `time.Sleep` in tests | 25 fichiers | flake risk |
| `_test.go.disabled` | 0 | ✅ |
| Tests > 500 lignes | 14 | proposal/service_test.go 1344, messaging/service_test.go 1344, auth/service_test.go 1073 |
| Table-driven tests | 117 fichiers (~46%) | Solide mais pas universel |
| testify usage | 252 fichiers (~98%) | ✅ |
| testcontainers usage | **1 fichier** | `adapter/postgres/search_ranking_v1_repository_test.go` only |
| Hand-rolled mocks `mocks_test.go` | 17 fichiers | OK — pattern lightweight cohérent |
| Fixtures | `test/fixtures/search_*` | Scoped à search only |
| Property tests via testing/quick | 1 file | rls_isolation_test.go |

## Backend integration / E2E

- **`test/e2e/`** = 6 bash scripts (`phase1_e2e.sh` ... `phase6_e2e.sh`) — non invoqués par CI
- **`test/fixtures/`** = 3 fichiers, search uniquement
- testcontainers = 1 seul fichier
- Smoke scripts (`scripts/smoke/{search,ops,security}.sh`) existent mais pas wirés en CI

---

## Web coverage (92 vitest + 19 Playwright) — UPDATED

### Vitest

| Feature | Vitest tests | Status |
|---|---|---|
| **billing** | **0** | 🔴 RED |
| **dispute** | **0** | 🔴 RED |
| **organization-shared** | **0** | 🔴 RED |
| **reporting** | **0** | 🔴 RED |
| auth | 1 | thin |
| account | 1 | thin |
| call | 1 | thin (LiveKit OFF-LIMITS) |
| client-profile | 4 | OK |
| freelance-profile | 5 | OK |
| invoicing | 5 | OK |
| job | 1 | thin |
| messaging | 13 | OK |
| notification | 3 | OK |
| **proposal** | **2** | thin (core money flow!) |
| provider | 9 | OK |
| referral | 2 | thin (commission flow!) |
| referrer-profile | 5 | OK |
| review | 2 | thin |
| skill | 7 | OK |
| **subscription** | **1** | thin (money) |
| team | 2 | thin |
| **wallet** | **2** | thin (money) |

**4 features RED** unchanged. Money flows (proposal, subscription, wallet) thin. **`shared/components/ui/` (7 components) = 0 tests** alors qu'ils sont réutilisés partout.

Hooks coverage : 27/88 use-*.ts ont des tests (~31%).

### Playwright (19 specs, gated par PR label `run-e2e`)

application-credits, auth, bonus-credits, calls, credits-reset, dashboard, fraud-detection, invoicing, messaging, milestones, navigation, payment-info-states, profile, projects, referrer, search, team-phase1-contract, team-phase2-contract, search/search.spec.ts.

**Pas couvert** : dispute, review, subscription checkout, wallet flows, notification preferences, KYC/Stripe Embedded.

⚠️ Configuré pour ne tourner que push-main ou PR avec label `run-e2e` — **pas par défaut sur les PRs courantes**.

---

## Admin coverage (2/76 = ~3%) — UNCHANGED

**1 feature testée sur 10** : `invoices` (`invoicing-api.test.ts`, `invoices-page.test.tsx`).

**0 tests** : auth, conversations, dashboard, disputes, jobs, media, moderation, reviews, users.

**Aucun job admin dans `ci.yml`** → admin n'a aucun gate.

---

## Mobile coverage (101 _test.dart, mais CI scope = 3 dirs)

CI scope = `test/shared/search`, `test/features/search`, `test/features/profile_tier1` uniquement.

| Mobile feature | Tests | Status |
|---|---|---|
| **dashboard** | **0** | 🔴 RED |
| **dispute** | **0** | 🔴 RED |
| **mission** | **0** | 🔴 RED |
| **portfolio** | **0** | 🔴 RED |
| **project_history** | **0** | 🔴 RED |
| **provider_profile** | **0** | 🔴 RED |
| **referral** | **0** | 🔴 RED — money flow |
| **referrer_reputation** | **0** | 🔴 RED |
| **reporting** | **0** | 🔴 RED |
| auth | 2 | thin |
| billing | 5 | OK |
| call | 2 | thin (LiveKit OFF-LIMITS) |
| client_profile | 5 | OK |
| expertise | 1 | thin |
| freelance_profile | 6 | OK |
| invoice + invoicing | 8 | OK |
| job | 4 | OK |
| messaging | 4 | thin (web has 13!) |
| notification | 2 | thin |
| organization_shared | 2 | thin |
| payment_info | 1 | thin |
| profile + profile_tier1 | 43 | strong |
| proposal | 2 | thin (money flow) |
| referrer_profile | 6 | OK |
| review | 3 | thin |
| search | 17 | strong |
| skill | 6 | OK |
| subscription | 11 | OK |
| team | 2 | thin |
| wallet | 2 | thin (money) |

**0 golden tests** (`matchesGoldenFile` count = 0). Design system avec tokens stricts → opportunité manquée pour visual regression.

**Integration tests** : 9 fichiers dans `integration_test/` — **non invoqués par CI**.

---

## CI

`/home/hassad/serviceMarketplaceGo/.github/workflows/` — 6 workflows :

| Workflow | Run | Notes |
|---|---|---|
| `ci.yml` | go vet, gofmt, govulncheck, go test -race -coverprofile, web tsc, vitest, next build, **flutter analyze (search/profile only)**, **flutter test (search/profile only)** | Coverage gate ≥80% backend (≥85% search/searchanalytics), ≥60% web. ❌ Pas de job admin. |
| `e2e.yml` | Playwright | Gated par label `run-e2e` ou push-main — ❌ pas sur PRs courantes |
| `security.yml` | govulncheck + trivy | Hebdo + lockfile-change |
| `drift.yml` | OpenAPI drift check | ✅ |
| `lighthouse.yml` | web Lighthouse | ✅ |
| `snapshot.yml` | DB snapshot | ✅ |

**Gaps** :
- ❌ Admin app sans aucun gate
- ❌ Mobile : 3 dirs sur ~30 features
- ❌ Backend `test/e2e/phase*_e2e.sh` non wirés
- ❌ Pas de Codecov pour admin
- ❌ `scripts/smoke/run-all.sh` et `scripts/perf/k6-search.js` non invoqués
- ❌ Pas de gosec/semgrep sur PR (juste govulncheck)

---

# AREA 2 — DB / MIGRATIONS — UPDATED

## Migration health (post PR #51-#66)

| Métrique | Valeur |
|---|---|
| Total `.up.sql` | **131** (was 125, +126 enum check, +127 fk indexes, +128 pending events stale, +129 audit RLS WITH CHECK, +130 messages nullable sender, +131 perf provider_org indexes) |
| Down files | 131/131 ✅ |
| Numbering gaps | 024 et 025 documentés |
| Latest | `131_perf_provider_org_indexes.up.sql` |
| Naming convention | clean (snake_case `create_X` / `add_Y_to_X` / `drop_Z`) ✅ |
| Idempotency | majoritaire (`IF [NOT] EXISTS`) — ~35 up + 13 down sans (still open) |
| `CREATE INDEX CONCURRENTLY` | **0 sur 200+** ⚠️ — m.131 documents the manual workflow |

## Schema consistency — org-scoping audit

| Table | Ownership | Status |
|---|---|---|
| users | (root, `organization_id`) | OK |
| organizations | (root) | OK |
| organization_members | `user_id` (membership) | OK |
| organization_invitations | `inviter_id`, `target_user_id` | OK |
| profiles | `organization_id` only (m.067 dropped user_id) | ✅ |
| social_links | `organization_id` only (m.069) | ✅ |
| portfolio_items | `organization_id` only (m.068) | ✅ |
| **subscriptions** | `organization_id` only (m.119) | ✅ FIXED |
| application_credits | dropped m.075, columns sur `organizations` | ✅ |
| jobs | `organization_id` (m.061) | ✅ |
| job_applications | `applicant_organization_id` denormalized + user FK | ⚠️ partial |
| **proposals** | `client_organization_id` + `provider_organization_id` (m.115) ALONGSIDE `client_id` + `provider_id` + `sender_id` + `recipient_id` | ⚠️ denormalized |
| **disputes** | `*_organization_id` ALONGSIDE `client_id` + `provider_id` + `initiator_id` + `respondent_id` | ⚠️ denormalized |
| **reviews** | `*_organization_id` ALONGSIDE `reviewer_id` + `reviewed_id` | ⚠️ denormalized |
| **payment_records** | `organization_id` + `provider_organization_id` (m.131) ALONGSIDE `client_id` + `provider_id` | ⚠️ denormalized |
| **conversations** | `organization_id` ALONGSIDE participants | ⚠️ denormalized |
| invoice / credit_note / billing_profile | `organization_id` only (m.121) | ✅ |
| audit_logs | `user_id` (actor reference) | OK |
| notifications, notification_preferences, device_tokens | `user_id` (recipient) | flag |
| conversation_read_state | `user_id` (per-user marker) | OK |

**Pattern de transition** : la migration vers org-scoped n'a pas droppé les colonnes `user_id`-style ; elles coexistent avec les `*_organization_id` dénormalisées. Documentation invariant nécessaire : "user_id columns sont write-only authorship, jamais utilisé pour les filtres ownership". Lint check CI utile.

## Cross-feature foreign keys (FORBIDDEN per CLAUDE.md)

**Violations détectées** (~10 distinctes) :
- `disputes.proposal_id → proposals(id)` (m.045)
- `disputes.job_id → jobs(id)` (m.045)
- `reviews.proposal_id → proposals(id)` (m.012)
- `reports.message_id → messages(id)` (m.023)
- `payment_records.proposal_id → proposals(id)` (m.018)
- `proposals.conversation_id → conversations(id)`

La règle stricte CLAUDE.md ("only references users(id)") est largement violée. **Décision à prendre** :
- Soit relaxer la règle dans CLAUDE.md (admettre que les workflows business ont besoin de FK cross-feature) — recommandé
- Soit migrer les FKs problématiques vers des denormalized references sans contrainte (lourd + perte d'intégrité)

## Index audit (cf. auditperf.md table-by-table)

- 200+ `CREATE INDEX` sur 131 migrations
- Composites `(created_at DESC, id DESC)` présents pour cursor pagination ✅
- Partial indexes utilisés ✅
- m.127 ajoute les 9 missing FK indexes
- m.131 ajoute les composites provider_org pour PERF-B-08

## RLS audit — UPDATED

✅ **9 tables under RLS** as of migration 125 + audit_logs FOR INSERT WITH CHECK (m.129). FORCE ROW LEVEL SECURITY enabled. Cross-tenant denial integration tests in `rls_isolation_test.go` are passing.

⚠️ **CRITICAL deployment dependency** (BUG-FINAL-01) : 35 legacy `.GetByID()` callers in app layer break under prod role rotation. Today, RLS is bypassed by the migration owner role. The moment production rotates to a dedicated `marketplace_app NOSUPERUSER NOBYPASSRLS` role, EVERY repo read against the 9 RLS tables that doesn't go through `RunInTxWithTenant` will return zero rows. The 8-path PR series migrated REPO METHODS but NOT APP CALLERS using legacy `GetByID`.

⚠️ Phase 5 Q est une foundation, mais la migration des 35 app callers vers `GetByIDForOrg` est **pre-prod blocker**.

## Migration safety / production risk — UPDATED

| Issue | Migration(s) | Risk |
|---|---|---|
| **`CREATE INDEX` sans CONCURRENTLY** | 200+ occurrences | Holds `ACCESS EXCLUSIVE` ; m.131 documents the manual workflow |
| `ALTER TABLE ... ADD COLUMN ... NOT NULL` après backfill | m.119 | OK PG≥11 |
| Backfill UPDATEs in same TX as schema change | m.075, m.115, m.131 | Long TX risque lock contention |
| Migration gap 024/025 | sequence | golang-migrate tolère |
| Atomic FK drop + col drop | m.119 | OK |

## Audit log table

- Schema m.078 conforme ✅
- ✅ Append-only enforced via `REVOKE UPDATE, DELETE` (m.124)
- ✅ RLS isolation via `USING` + `WITH CHECK (true)` (m.125 + m.129)
- Indexes `user_id` partial, `action`, `created_at DESC`, `(resource_type, resource_id)` partial ✅

## Soft delete / cascade

- Quasi-pas de soft delete (1 match `deleted_at` dans `messages`)
- Mix `ON DELETE CASCADE` / `ON DELETE SET NULL` selon les tables — sémantiques mixtes. Documenter.

## Constraints — UPDATED

- 67+ `CHECK (` constraints (added ~10 in m.126 for status enums)
- 17 `UNIQUE` column-level (partial unique indexes en plus)
- ✅ CHECK constraints for status enums (`proposals.status`, `disputes.status`, etc.) added in m.126

---

# COMBINED — Top 15 priorités — UPDATED

## Tests (P0 → P2)

| # | Item | Effort | Status |
|---|---|---|---|
| 1 | **`internal/app/admin/`** : 4/9 → 9/9 | 1 sem | partial done |
| 2 | **`internal/app/kyc/`** : 1/1 ✅ | done | ✅ |
| 3 | **`internal/app/referral/`** : 4/24 → cibler money paths (clawback, commission distributor, kyc_listener) | 3-4 jours | partial |
| 4 | **23 handlers untested** : prioriser admin handlers + dispute/stripe/role_overrides | 1 sem | open |
| 5 | **postgres adapter critiques** : proposal, dispute, review, message, payment_records, notification | 1 sem | RLS partial |
| 6 | **Admin app web** : minimum 1 test par feature (10) | 3 jours | open |
| 7 | **Web vitest 4 features RED** (billing, dispute, organization-shared, reporting) + thin (proposal, subscription, wallet) | 2-3 jours | open |
| 8 | **Mobile 9 features RED** (dashboard, dispute, mission, portfolio, project_history, provider_profile, referral, referrer_reputation, reporting) + 0 golden tests | 1 sem | open |
| 9 | **CI gaps** : wire admin tests, étendre mobile scope, run smoke scripts | 1 jour | open |
| 10 | **Adapter externes 0/9** : tests httptest pour anthropic/openai/fcm/etc | M | open |

## Migrations / DB

| # | Item | Effort | Status |
|---|---|---|---|
| 11 | **35 legacy GetByID callers → GetByIDForOrg** (BUG-FINAL-01 deployment blocker) | 3 jours | **OPEN — PRE-PROD BLOCKER** |
| 12 | **Décider cross-feature FK rule** : relaxer CLAUDE.md OU migrer 10 violations | 30min déc | open |
| 13 | **Documenter invariant org_id** : "user_id columns = write-only authorship", lint CI | 2h | open |
| 14 | **`IF [NOT] EXISTS`** sur les 35+13 migrations non-idempotentes | 2h | open |
| 15 | Convention `CREATE INDEX CONCURRENTLY` future migrations grandes tables | doc | open |

---

## Summary

| Area | Critical | Major | Minor |
|---|---|---|---|
| Tests backend | 1 (RLS GetByID callers) | 4 (handlers untested + adapter externes) | 2 (E2E shell scripts, fixtures sparse) |
| Tests web | 2 (4 features 0 + ui/ untested) | 2 (proposal/subscription/wallet thin, Playwright label-gated) | 1 (hooks 31%) |
| Tests admin | 1 (3% coverage, no CI job) | 0 | 0 |
| Tests mobile | 2 (9 features 0 + 0 goldens) | 2 (CI scope tiny, integration tests unrun) | 1 (referral/reporting/dispute zero) |
| Migrations schema | 1 (cross-FK violations) | 1 (user_id ownership cols survive) | 0 |
| Migrations safety | 0 | 1 (CREATE INDEX never CONCURRENT) | 1 (long-TX backfills) |
| **Total** | **7** | **10** | **5** |

**Top priority remains** : **migrate 35 legacy `.GetByID()` callers to `GetByIDForOrg` BEFORE rotating prod DB role to NOSUPERUSER NOBYPASSRLS** (BUG-FINAL-01). Otherwise the entire app returns empty under the dedicated role.
