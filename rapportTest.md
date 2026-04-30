# Rapport Tests + Migrations + DB

**Date** : 2026-04-29 (rapport précédent : 2026-04-04, single-feature)
**Branche** : `main` @ `a0d268a4`
**Périmètre** : couverture tests par layer + qualité tests + santé migrations + cohérence schéma

## Méthodologie

Audit statique sans `go test` ni `flutter test` exécutés. File-level coverage = `non-test files avec un _test.go companion dans le même répertoire / total non-test files`. Migrations = analyse SQL pure. Données : backend 560 non-test + 288 _test.go ; web 92 spec/test + 19 Playwright ; admin 2 specs ; mobile 101 _test.dart ; 121 migrations (gap 024/025).

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
| `cmd/api/main.go` | 1 (1479 LOC) | 0 | 0% |

## Per-domain status (29 modules)

- **0 tests** : `media`
- Sparse (1-2 tests) : `proposal` (entité OK, errors_test manquant), `subscription` (1/2), `report`, `review`
- Best : `organization` (6/6), `user` (3/3), `invoicing` (6/8), `referral` (5/7)

## Per-app-module status (32 modules)

| Module | Tests | Status |
|---|---|---|
| **admin** | **0/9** | 🔴 CRITICAL — admin écrit user/org data sans aucun filet |
| **kyc** | **0/1** | 🔴 CRITICAL — money/compliance gate |
| **referral** | **2/20** | 🔴 CRITICAL — money + clawback + commission distribution |
| auth, dispute, embedded, invoicing, job, messaging, milestone, organization, payment, profile, proposal, review, search, subscription, skill | OK | ✅ |

## Per-handler status (54 fichiers, 38 testés)

**23 handlers sans `_test.go`** :
- Tous les `admin_*` (sauf `admin_credit_note`, `admin_invoice`, `admin_search_stats`) : `admin_dispute`, `admin_handler`, `admin_media`, `admin_message_moderation`, `admin_moderation`, `admin_notification`, `admin_review`, `admin_team`
- `dispute_handler`, `freelance_pricing_handler`, `invitation_handler`, `job_application_handler`
- `organization_shared_profile_handler`, `portfolio_handler`, `profile_pricing_handler`
- `project_history_handler`, `referrer_pricing_handler`, `report_handler`, `role_overrides_handler`
- **`stripe_handler`** (deux tests adjacents `*_invoicing_test.go` et `*_credit_note_test.go` couvrent seulement ces branches)

Handler-level tests détectent les bugs RBAC/ownership que les unit tests manquent.

## Per-postgres-adapter status (65 fichiers, 23 testés = 35%)

**Repos critiques sans tests** :
- `proposal_repository` (money flow)
- `dispute_repository` (money flow)
- `review_repository`
- `message_repository`
- `notification_repository`
- `payment_records_repository` (le plus money-touching)
- `organization_*_repository` (multiples)

Tests présents pour : search, invoicing, billing_profile, milestone, subscription, referral, search_analytics, search_document, job_credit, moderation_results.

## Per-adapter-externe : 0 tests sur 9

`anthropic`, `comprehend`, `fcm`, `livekit`, `rekognition`, `resend`, `s3transit`, `sqs`, `noop` — 0 tests.

## Test quality

| Métrique | Compte | Notes |
|---|---|---|
| `t.Skip` calls | 27 (22 fichiers) | Tous gated par env vars (`MARKETPLACE_TEST_DATABASE_URL`, `TYPESENSE_INTEGRATION_URL`, `OPENAI_EMBEDDINGS_LIVE`, `MARKETPLACE_PDF_TEST`) — clean ✅ |
| `time.Sleep` in tests | 25 fichiers | Concentrés `worker_test.go`, `nominatim/`, `openai/`, `redis/*` — flake risk |
| `_test.go.disabled` | 0 | ✅ |
| Tests > 500 lignes | 14 | Plus gros : `proposal/service_test.go` 1344, `messaging/service_test.go` 1344, `auth/service_test.go` 1073 |
| Table-driven tests | 117 fichiers (~46%) | Solide mais pas universel |
| testify usage | 252 fichiers (~98%) | ✅ |
| testcontainers usage | **1 fichier** (`adapter/postgres/search_ranking_v1_repository_test.go`) | La plupart des "intégration" tests gated par `MARKETPLACE_TEST_DATABASE_URL` (brittle) |
| Hand-rolled mocks `mocks_test.go` | 17 fichiers | OK mais doc CLAUDE.md mentionne `backend/mock/` qui n'existe pas |
| Fixtures | `test/fixtures/search_*` | Scoped à search uniquement |

### `backend/mock/` n'existe pas

CLAUDE.md root + backend décrivent un dossier `backend/mock/` "Generated mocks from port interfaces". La réalité = `mocks_test.go` inline. **Mettre à jour CLAUDE.md** pour refléter le pattern.

## Backend integration / E2E

- **`test/e2e/`** = 6 bash scripts (`phase1_e2e.sh` ... `phase6_e2e.sh`) — non invoqués par CI, pas dans `go test ./...`
- **`test/fixtures/`** = 3 fichiers, search uniquement
- testcontainers = 1 seul fichier
- Smoke scripts (`scripts/smoke/{search,ops,security}.sh`, `scripts/perf/k6-search.js`) existent mais pas wirés en CI

---

## Web coverage (92 vitest + 19 Playwright)

### Vitest

| Feature | Vitest tests | Status |
|---|---|---|
| **billing** | **0** | 🔴 RED |
| **dispute** | **0** | 🔴 RED |
| **organization-shared** | **0** | 🔴 RED |
| **reporting** | **0** | 🔴 RED |
| auth | 1 | thin |
| account | 1 | thin |
| call | 1 | thin |
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

**4 features RED**. Money flows (proposal, subscription, wallet) thin. **`shared/components/ui/` (6 components) = 0 tests** alors qu'ils sont réutilisés partout.

Hooks coverage : 27/88 use-*.ts ont des tests (~31%).

### Playwright (19 specs, gated par PR label `run-e2e`)

application-credits, auth, bonus-credits, calls, credits-reset, dashboard, fraud-detection, invoicing, messaging, milestones, navigation, payment-info-states, profile, projects, referrer, search, team-phase1-contract, team-phase2-contract, search/search.spec.ts.

**Pas couvert** : dispute, review, subscription checkout, wallet flows, notification preferences, KYC/Stripe Embedded.

⚠️ Configuré pour ne tourner que push-main ou PR avec label `run-e2e` — donc **pas par défaut sur les PRs courantes**.

---

## Admin coverage (2/76 = ~3%)

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
| call | 2 | thin |
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

**Integration tests** : 9 fichiers dans `integration_test/` (auth, dashboard, invoicing_flow, messaging, profile, projects, search, subscription, app) — **non invoqués par CI**.

---

## CI

`/home/hassad/serviceMarketplaceGo/.github/workflows/` — 6 workflows :

| Workflow | Run | Notes |
|---|---|---|
| `ci.yml` | go vet, gofmt, govulncheck, go test -race -coverprofile, web tsc, vitest, next build, **flutter analyze (search/profile only)**, **flutter test (search/profile only)** | Coverage gate ≥80% backend (≥85% search/searchanalytics), ≥60% web. ❌ Pas de job admin. ESLint en `continue-on-error` |
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

---

# AREA 2 — DB / MIGRATIONS

## Migration health

| Métrique | Valeur |
|---|---|
| Total `.up.sql` | **121** (pas 123) |
| Down files | 121/121 ✅ |
| Numbering gaps | **024 et 025 manquants** (sequence saute de 023 à 026) |
| Latest | `123_org_auto_payout_consent.up.sql` |
| Naming convention | clean (snake_case `create_X` / `add_Y_to_X` / `drop_Z`) ✅ |
| Idempotency | majoritaire (`IF [NOT] EXISTS`) — **35 up + 13 down sans** |
| `CREATE INDEX CONCURRENTLY` | **0 sur 183** ⚠️ |

`golang-migrate` accepte les gaps de numérotation, mais auditeurs et forks vont stumble.

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
| **subscriptions** | **`organization_id` only (m.119)** | ✅ FIXED |
| application_credits | dropped m.075, columns sur `organizations` | ✅ |
| jobs | `organization_id` (m.061) | ✅ |
| job_applications | `applicant_organization_id` denormalized + user FK | ⚠️ partial |
| **proposals** | `client_organization_id` + `provider_organization_id` ALONGSIDE `client_id` + `provider_id` + `sender_id` + `recipient_id` | ⚠️ denormalized |
| **disputes** | `*_organization_id` ALONGSIDE `client_id` + `provider_id` + `initiator_id` + `respondent_id` | ⚠️ denormalized |
| **reviews** | `*_organization_id` ALONGSIDE `reviewer_id` + `reviewed_id` | ⚠️ denormalized |
| **payment_records** | `organization_id` ALONGSIDE `client_id` + `provider_id` (m.064) | ⚠️ denormalized |
| **conversations** | `organization_id` ALONGSIDE participants | ⚠️ denormalized |
| invoice / credit_note / billing_profile | `organization_id` only (m.121) | ✅ |
| payment_info, identity_documents, business_persons, test_embedded_accounts | DROPPED m.042 | ✅ |
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
- `proposal_milestones.proposal_id` (same feature OK)
- `dispute_evidence.dispute_id` (same feature OK)
- `job_applications.job_id` (same feature OK)
- `messages.conversation_id` (same feature OK)

La règle stricte CLAUDE.md ("only references users(id)") est largement violée. **Décision à prendre** :
- Soit relaxer la règle dans CLAUDE.md (admettre que les workflows business ont besoin de FK cross-feature)
- Soit migrer les FKs problématiques vers des denormalized references sans contrainte

## Index audit (cf. auditperf.md table-by-table)

- 183 `CREATE INDEX` sur 121 migrations
- Composites `(created_at DESC, id DESC)` présents pour cursor pagination ✅
- Partial indexes utilisés ✅
- GIN indexes pour search FR (probablement redondant depuis Typesense)
- ⚠️ Risque : FK columns peuvent ne pas être indexées (PostgreSQL n'indexe pas les FK auto). Un `SELECT conrelid::regclass FROM pg_constraint WHERE contype='f'` cross-référencé avec `pg_index` permettrait de trouver les gaps.

## RLS audit

**ZERO tables ont RLS enabled.** `grep -i "ENABLE ROW LEVEL SECURITY" migrations/*.up.sql` = 0. `FORCE ROW LEVEL SECURITY` = 0.

Backend/CLAUDE.md prescrit RLS pour `missions, contracts, messages, invoices, reviews, notifications, profiles`. **Application-level `WHERE org_id = ?` est la SEULE ligne de défense aujourd'hui.** Une régression de filtre = fuite cross-tenant.

## Migration safety / production risk

| Issue | Migration(s) | Risk |
|---|---|---|
| **`CREATE INDEX` sans CONCURRENTLY** | 183 occurrences | Holds `ACCESS EXCLUSIVE`, lock écriture pendant index build. Acceptable petites tables, **risqué sur `messages`, `proposals`, `payment_records`, `audit_logs` qui grossissent** |
| `ALTER TABLE ... ADD COLUMN ... NOT NULL` après backfill | m.119 (subscriptions.organization_id) | `SET NOT NULL` PG≥11 avec default rapide, mais backfill UPDATE rewrite chaque row → table lock |
| Backfill UPDATEs in same TX as schema change | m.075, m.115 | Long TX risque lock contention sur tables write-heavy |
| Migration gap 024/025 | sequence | golang-migrate tolère mais auditeurs flagueront |
| `DROP TABLE CASCADE` | m.042 (KYC tables) | Irreversible, OK car dev-only |
| Atomic FK drop + col drop | m.119 | OK |

## Audit log table

- Schema m.078 conforme CLAUDE.md ✅
- ⚠️ **Pas append-only enforced en DB** — pas de `REVOKE UPDATE, DELETE`, pas de trigger blocking. Convention only.
- Indexes `user_id` partial, `action`, `created_at DESC`, `(resource_type, resource_id)` partial ✅

## Soft delete / cascade

- Quasi-pas de soft delete (1 match `deleted_at` dans `messages`)
- Mix `ON DELETE CASCADE` / `ON DELETE SET NULL` selon les tables — sémantiques mixtes (proposals.conversation_id CASCADE, milestone RESTRICT, *_organization_id SET NULL). Documenter.

## Constraints

- 67 `CHECK (` constraints
- 17 `UNIQUE` column-level (partial unique indexes en plus)
- ⚠️ Manquent : CHECK pour enums TEXT (`proposals.status`, `disputes.status`, `payment_records.status`)

---

# COMBINED — Top 15 priorités

## Tests (P0 → P2)

| # | Item | Effort |
|---|---|---|
| 1 | **`internal/app/admin/`** : 0/9 → couvrir minimum service_test par module | 1 sem |
| 2 | **`internal/app/kyc/`** : 0/1 → service_test + handler_test | 1 jour |
| 3 | **`internal/app/referral/`** : 2/20 → cibler money paths (clawback, commission distributor, kyc_listener) | 3-4 jours |
| 4 | **23 handlers untested** : prioriser admin handlers + dispute/stripe/role_overrides | 1 sem |
| 5 | **postgres adapter critiques** : proposal, dispute, review, message, payment_records, notification | 1 sem |
| 6 | **Admin app web** : minimum 1 test par feature (10) | 3 jours |
| 7 | **Web vitest 4 features RED** (billing, dispute, organization-shared, reporting) + thin (proposal, subscription, wallet) | 2-3 jours |
| 8 | **Mobile 9 features RED** (dashboard, dispute, mission, portfolio, project_history, provider_profile, referral, referrer_reputation, reporting) + 0 golden tests | 1 sem |
| 9 | **CI gaps** : wire admin tests, étendre mobile scope, run smoke scripts | 1 jour |

## Migrations / DB

| # | Item | Effort |
|---|---|---|
| 10 | **RLS sur tables tenant-scoped** (messages, conversations, invoices, proposals, notifications, wallet_records, disputes, audit_logs) — défense en profondeur | 1-2 jours |
| 11 | **Décider cross-feature FK rule** : relaxer CLAUDE.md OU migrer 10 violations | 30min déc + 0-3j fix |
| 12 | **Documenter invariant org_id** : "user_id columns = write-only authorship", lint CI | 2h |
| 13 | **`IF [NOT] EXISTS`** sur les 35+13 migrations non-idempotentes | 2h |
| 14 | **Audit_logs append-only DB** : `REVOKE UPDATE, DELETE ON audit_logs FROM <app_user>` | 30min |
| 15 | **Documenter ou poser noop migration 024/025** | 15min |
| Bonus | Convention `CREATE INDEX CONCURRENTLY` future migrations grandes tables | doc |

---

## Summary

| Area | Critical | Major | Minor |
|---|---|---|---|
| Tests backend | 4 (admin/kyc/referral untested + handlers) | 4 (test files >500, sleeps, testcontainers, mock/ doc drift) | 2 (E2E shell scripts, fixtures sparse) |
| Tests web | 2 (4 features 0 + ui/ untested) | 2 (proposal/subscription/wallet thin, Playwright label-gated) | 1 (hooks 31%) |
| Tests admin | 1 (3% coverage, no CI job) | 0 | 0 |
| Tests mobile | 2 (9 features 0 + 0 goldens) | 2 (CI scope tiny, integration tests unrun) | 1 (referral/reporting/dispute zero) |
| Migrations schema | 2 (RLS absent + cross-FK violations) | 2 (gap 024/025, user_id ownership cols survive) | 1 (audit_logs DB-enforce) |
| Migrations safety | 0 | 1 (CREATE INDEX never CONCURRENT) | 1 (long-TX backfills) |
| **Total** | **11** | **11** | **6** |

Top priority : **activer PostgreSQL RLS** sur tables tenant-scoped avant open-source — c'est le seul filet contre une régression de filtre `WHERE org_id = ?` qui exposerait toutes les conversations/invoices/proposals d'autres orgs.
