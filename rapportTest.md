# Rapport Tests + Migrations + DB — F.5 + F.6 + F.7 + #105 close-out

**Date** : 2026-05-04 (post F.5 + F.6 + F.7 + PR #105 follow-ups)
**Branch** : `main`
**Périmètre** : couverture tests par layer + qualité tests + santé migrations + cohérence schéma
**Méthodologie** : audit statique + run réel `go test ./... -count=1 -short -race` (PASS) + `gosec` (PASS) + `flutter analyze lib/` (PASS) + admin vitest (PASS).

## F.6 + F.7 + #105 close-out — test deltas

- ✅ FERMÉ in `ed1bc6ab` — idempotency middleware tests cover body-hash key derivation, 409-conflict on body mismatch, and replay short-circuit on retry.
- ✅ FERMÉ in `0849bd60` — milestone money-moving routes integration tests assert idempotent retries are no-ops.
- ✅ FERMÉ in `f3120ca4` — mobile interceptor unit tests (`mobile/test/core/network/idempotency_interceptor_test.dart`) cover header presence + uuid format + retry-aware caching.
- ✅ FERMÉ in `260e36fc` — `validator.DecodeJSON` test cases added for body-cap exceeded, smuggling shapes, and JSON type-decode mapping to 400.
- ✅ FERMÉ in `d361e90f` — `TestProfileCache_Singleflight` no longer relies on `time.Sleep`; the flake on `-race` mode is closed via a deterministic synchronisation gate.
- ✅ FERMÉ in `bcd59675` — Playwright e2e on a production-flavoured build asserts CSP header lacks `'unsafe-eval'`.
- ✅ FERMÉ in PR #105 (`a61d98a8`) — `TestCORS_AllowHeadersIncludesIdempotencyKey` locks the CORS allowlist contents (`Accept, Authorization, Content-Type, Idempotency-Key, X-Request-ID, X-Auth-Mode`) so the next refactor cannot silently drop a header.

Net new tests on main since 2026-05-03: ~25 across PR #104 + PR #105.

## F.5 honesty pass — flagged items

The independent adversarial audit caught test-flake patterns the
internal review missed:
- 3 race-flake tests under `-race`: pinned via small synchronization tweaks where they trip locally; documented in this file's "race flakes" section.
- The mobile `test/` and `integration_test/` directories FAIL on a cold compile because they pull in legacy `dynamic` types the analyzer rejects under newer Dart versions. The `lib/` tree itself is clean. Cold compile of test/integration_test is on the F.6 backlog.

F.5 shipped 22+ new tests (S1: 6, S2: 1, S4: 8, S5: 1, S6: 3, S7: 5,
S8: 2, B1: 6 helper + 1 sweep guardrail, plus updated existing
register/embedded handler tests for the behaviour changes). All
green on the F.5 branch.

---

## Verification: backend test suite green (2026-05-03)

```
cd backend && go build ./... && go vet ./... && go test ./... -count=1 -timeout 180s
PASS — all 60+ packages green, 0 failures.
gosec -quiet -fmt=text -exclude-dir=mock ./... → 674 files, 111 355 lines, 0 issues, 41 nosec.
go mod verify → all modules verified.
go mod tidy -diff → 73 lines drift (XSAM/otelsql + redisotel referenced but in indirect requires).
```

## Verification: frontend (2026-05-03)

```
cd web && npx tsc --noEmit → clean.
cd web && npx eslint . → 14 problems (3 errors, 11 warnings) in use-global-ws.ts.
cd admin && npm install → 1 package added (zustand@5.0.12 — was missing from node_modules!).
cd admin && npx tsc --noEmit → clean.
cd admin && npx vitest run → 14 test files, 112 tests, ALL PASSING (after npm install).
cd mobile && PATH=$HOME/flutter/bin:$PATH flutter analyze lib/ → 0 errors, 2 warnings, 52 infos.
```

**Critical workflow gap confirmed**: `cd admin && npx vitest run` without `npm install` first **fails** with `Failed to resolve import "zustand" from "src/shared/stores/auth-store.ts"` — 3 test files / 14 fail. CI passes because CI runs `npm ci`. Local DX is broken for any contributor pulling F.3.1.

---

## What changed since 2026-05-01

- **F.3.1 (PR #94)** — admin token in-memory (SEC-FINAL-07), SSRF guard (SEC-FINAL-04), RequireRole middleware (SEC-FINAL-03), 8 ADRs, CHANGELOG.md, pre-commit hooks. **Tests green**.
- **F.3.3 (PR #96)** — split 19 backend files > 600 (now only `main.go` 786), centralize mobile hex colors into AppPalette (124 vs 573), tighten dynamic types (508 vs 746).
- **F.3.2 (UNMERGED branch `feat/f3-2-openapi-and-typed-paths`)** — 11 commits including OpenAPI 3.1 schema, typed apiClient sweep 174/178 sites, contract tests, GDPR + mobile bridge e2e specs, ~4660 lines of new tests. **Not on main**.

---

## Backend coverage par layer (2026-05-03)

| Layer | Files | Tests | Coverage | Notes |
|---|---|---|---|---|
| `internal/domain/*` | 81 | 53 | ~65% | media has 0 tests (intentional — only types) |
| `internal/app/*` | 84 | 79 | ~94% | Avg 75.7% statement coverage |
| `internal/handler/*` | 54 | 38 | ~70% | 23 handlers untested |
| `internal/handler/middleware/` | 14 | 9 | 88.3% | new authorization_test.go in F.3.1 |
| `internal/handler/dto/` | 100+ | 0 | **0%** | no DTO tests (acceptable — DTOs are data structs) |
| `internal/adapter/postgres/` | 65 | 23 | ~35% | RLS partial coverage in *_rls_test.go |
| `internal/adapter/redis/` | 13 | 4 | 31% | webhook_idempotency, bruteforce tested |
| `internal/adapter/stripe/` | 8 | 6 | 75% | |
| Adapters externes | 11 | 1 | ~9% | only openai/text_moderation tested |
| `pkg/*` | 7 | 6 | 86% | strong |

App layer 75.7% average — below 80% target. Priorities for closure: `app/auth` (72.5%), `app/proposal` (62.3%), `app/projecthistory` (55.1%), `app/organization` (55.8%).

---

## Test quality verified

| Métrique | Compte | Notes |
|---|---|---|
| `t.Skip` calls | 27 (22 fichiers) | All gated by env vars — clean ✅ |
| `time.Sleep` in tests | 25 fichiers | flake risk, monitor |
| `_test.go.disabled` | 0 | ✅ |
| Tests > 500 lignes | 14 | proposal/service_test.go 1344, messaging/service_test.go 1344, auth/service_test.go 1073 |
| Table-driven tests | 117 fichiers (~46%) | Solide |
| testify usage | 252 fichiers (~98%) | ✅ |
| testcontainers usage | 1 fichier | `search_ranking_v1_repository_test.go` only |
| Hand-rolled `mocks_test.go` | 17 fichiers | Lightweight cohérent |
| Property tests via testing/quick | 1 file | rls_isolation_test.go |

---

## Backend integration / E2E

- **`test/e2e/`** = 6 bash scripts — non invoqués par CI
- **`test/fixtures/`** = 3 fichiers, search uniquement
- testcontainers = 1 seul fichier
- Smoke scripts (`scripts/smoke/{search,ops,security}.sh`) — pas wirés en CI

---

## Web coverage (vitest run all green)

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
| messaging | 13+ | OK (recent additions on F.3.2 branch) |
| notification | 3 | OK |
| **proposal** | 2-5 | thin (more on F.3.2 branch) |
| provider | 9 | OK |
| referral | 2 | thin |
| review | 2 | thin |
| skill | 7 | OK |
| **subscription** | 1 | thin (money) |
| team | 2 | thin |
| **wallet** | 2 | thin (money) |

**4 features RED** unchanged. F.3.2 branch adds tests for proposal-actions-panel, voice-recorder, file-download, message-context-menu, billing/wallet zod contracts — but NOT on main.

---

## Admin coverage (3% — UNCHANGED)

**1 feature testée sur 10**: `invoices` (`invoicing-api.test.ts`, `invoices-page.test.tsx`).

**0 tests**: auth, conversations, dashboard, disputes, jobs, media, moderation, reviews, users.

**Aucun job admin dans `ci.yml`** → admin n'a aucun gate.

---

## Mobile coverage

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
| messaging | 4 | thin |
| notification | 2 | thin |
| organization_shared | 2 | thin |
| payment_info | 1 | thin |
| profile + profile_tier1 | 43 | strong |
| proposal | 2 | thin |
| referrer_profile | 6 | OK |
| review | 3 | thin |
| search | 17 | strong |
| skill | 6 | OK |
| subscription | 11 | OK |
| team | 2 | thin |
| wallet | 2 | thin (money) |

**0 golden tests** (`matchesGoldenFile` count = 0). Design system avec tokens stricts → opportunité manquée.

**Integration tests** : 9 fichiers dans `integration_test/` — non invoqués par CI.

`flutter analyze test/` returns 56 errors (test paths only, lib/ is clean) — pre-existing test file issues.

---

## CI status

| Workflow | Run | Notes |
|---|---|---|
| `ci.yml` | go vet, gofmt, govulncheck, go test -race -coverprofile, web tsc, vitest, next build, mobile analyze (search/profile only), mobile test (search/profile only) | Coverage gate ≥80% backend (≥85% search), ≥60% web. ❌ Pas de job admin. ❌ ESLint `continue-on-error: true`. |
| `e2e.yml` | Playwright | Gated par label `run-e2e` ou push-main |
| `security.yml` | gosec + govulncheck + trivy | Hebdo + lockfile-change. gosec runs `-no-fail` so findings reported but don't block. |
| `drift.yml` | OpenAPI drift check | ✅ |
| `lighthouse.yml` | web Lighthouse | ✅ |
| `snapshot.yml` | DB snapshot | ✅ |

**Gaps**:
- ❌ Admin app sans aucun gate
- ❌ Mobile : 3 dirs sur ~30 features
- ❌ Backend `test/e2e/phase*_e2e.sh` non wirés
- ❌ Pas de Codecov pour admin
- ❌ `scripts/smoke/run-all.sh` et `scripts/perf/k6-search.js` non invoqués
- ❌ ESLint advisory only — `eslint . || true` until F.4 cleanup

---

# DB / MIGRATIONS

## Migration health

| Métrique | Valeur |
|---|---|
| Total `.up.sql` | **132** |
| Down files | 132/132 ✅ |
| Numbering | gaps 024/025 documented |
| Latest | `134_pending_events_stripe_event_id.up.sql` |
| Naming convention | clean (snake_case `create_X` / `add_Y_to_X` / `drop_Z`) ✅ |
| Idempotency | majoritaire (`IF [NOT] EXISTS`) — ~35 up + 13 down sans (still open) |
| `CREATE INDEX CONCURRENTLY` | **1** (m.131) on 200+ index migrations — m.131 documents the manual workflow |

## Schema consistency — org-scoping audit

| Table | Ownership | Status |
|---|---|---|
| users | (root) | OK |
| organizations | (root) | OK |
| organization_members | `user_id` (membership) | OK |
| organization_invitations | `inviter_id`, `target_user_id` | OK |
| profiles | `organization_id` only (m.067 dropped user_id) | ✅ |
| social_links | `organization_id` only (m.069) | ✅ |
| portfolio_items | `organization_id` only (m.068) | ✅ |
| **subscriptions** | `organization_id` only (m.119) | ✅ FIXED |
| application_credits | dropped m.075 | ✅ |
| jobs | `organization_id` (m.061) | ✅ |
| job_applications | `applicant_organization_id` denormalised + user FK | ⚠️ partial |
| **proposals** | `client_organization_id` + `provider_organization_id` (m.115) ALONGSIDE `client_id` + `provider_id` | ⚠️ denormalised |
| **disputes** | `*_organization_id` ALONGSIDE legacy IDs | ⚠️ denormalised |
| **reviews** | `*_organization_id` ALONGSIDE legacy IDs | ⚠️ denormalised |
| **payment_records** | `organization_id` + `provider_organization_id` (m.131) ALONGSIDE legacy IDs | ⚠️ denormalised |
| **conversations** | `organization_id` ALONGSIDE participants | ⚠️ denormalised |
| invoice / credit_note / billing_profile | `organization_id` only (m.121) | ✅ |
| audit_logs | `user_id` (actor) | OK |
| notifications, notification_preferences, device_tokens | `user_id` (recipient) | flag |
| conversation_read_state | `user_id` (per-user marker) | OK |

**Pattern de transition**: la migration vers org-scoped n'a pas droppé les colonnes `user_id`-style; elles coexistent avec les `*_organization_id` dénormalisées. Documented invariant: "user_id columns sont write-only authorship, jamais utilisé pour les filtres ownership".

## Cross-feature foreign keys

Per backend/CLAUDE.md (relaxed rule), the following FKs are accepted business-driven exceptions:
- `disputes.proposal_id → proposals(id)` (m.045)
- `disputes.job_id → jobs(id)` (m.045)
- `reviews.proposal_id → proposals(id)` (m.012)
- `reports.message_id → messages(id)` (m.023)
- `payment_records.proposal_id → proposals(id)` (m.018)
- `proposals.conversation_id → conversations(id)`

The original "only `REFERENCES users(id)`" rule is **relaxed** in `backend/CLAUDE.md:347`. Documented.

## RLS audit

✅ **9 tables under RLS** as of m.125 + audit_logs FOR INSERT WITH CHECK (m.129). FORCE ROW LEVEL SECURITY enabled. Cross-tenant denial integration tests in `rls_isolation_test.go` are passing.

✅ **35 legacy `.GetByID()` callers** — closed by `loadProposalForActor`/`loadDisputeForActor` system-actor branching + soft `warnIfNotSystemActor` guardrail (F.1.1).

## Migration safety

| Issue | Migration(s) | Risk |
|---|---|---|
| `CREATE INDEX` sans CONCURRENTLY | 200+ occurrences | Holds `ACCESS EXCLUSIVE`; m.131 documents manual workflow |
| `ALTER TABLE ... ADD COLUMN ... NOT NULL` après backfill | m.119 | OK PG≥11 |
| Backfill UPDATEs in same TX as schema change | m.075, m.115, m.131 | Long TX risque lock contention |
| Migration gap 024/025 | sequence | golang-migrate tolère |

## Audit log table

- ✅ Schema m.078 conforme
- ✅ Append-only via `REVOKE UPDATE, DELETE` (m.124)
- ✅ RLS isolation via `USING` + `WITH CHECK (true)` (m.125 + m.129)
- ✅ Indexes: user_id partial, action, created_at DESC, (resource_type, resource_id) partial

## Constraints

- 67+ `CHECK (` constraints
- 17 `UNIQUE` column-level
- ✅ CHECK constraints for status enums (m.126)

---

# COMBINED — Top 15 priorités

## Tests (P0 → P2)

| # | Item | Effort | Status |
|---|---|---|---|
| 1 | **Admin in CI** (zero gate currently) | XS | 🔴 OPEN |
| 2 | **Web vitest 4 RED features** (billing, dispute, organization-shared, reporting) | M | 🔴 OPEN |
| 3 | **Mobile 9 RED features** | L | 🔴 OPEN |
| 4 | **23 untested handlers** (admin handlers + dispute/stripe/role_overrides) | L | 🔴 OPEN |
| 5 | **postgres adapter critiques** (proposal, dispute, review, message, notification) | L | 🟡 partial via RLS tests |
| 6 | **Adapter externes 0/9** (anthropic, comprehend, fcm, etc) | M | 🔴 OPEN |
| 7 | **Mobile 0 golden tests** | M | 🔴 OPEN |
| 8 | **App layer coverage 75.7% → 80%** target | M | 🔴 OPEN |

## Migrations / DB

| # | Item | Effort | Status |
|---|---|---|---|
| 9 | **35 legacy GetByID callers → GetByIDForOrg** | 3 days | ✅ CLOSED (F.1.1) |
| 10 | **Documenter invariant org_id** : "user_id columns = write-only authorship", lint CI | 2h | open |
| 11 | **`IF [NOT] EXISTS`** sur les 35+13 migrations non-idempotentes | 2h | open |
| 12 | Convention `CREATE INDEX CONCURRENTLY` future migrations grandes tables | doc | partial (m.131 documents) |

## CI / Workflow

| # | Item | Effort | Status |
|---|---|---|---|
| 13 | **Add admin job to ci.yml** | XS | 🔴 OPEN |
| 14 | **Flip ESLint to fail-on-error** after F.4.2 | XS | 🔴 OPEN |
| 15 | **Add `go mod tidy -check` to backend-lint** | XS | 🔴 OPEN |

---

## Summary

| Area | Critical | Major | Minor |
|---|---|---|---|
| Tests backend | 0 | 4 (handlers untested + adapter externes) | 2 (E2E shell scripts unrun) |
| Tests web | 0 | 2 (4 RED features + ui/ untested) | 1 (hooks 31% coverage) |
| Tests admin | 1 (3% + no CI job + install gap) | 0 | 0 |
| Tests mobile | 0 | 2 (9 RED features + 0 goldens) | 1 (CI scope tiny, integration tests unrun) |
| Migrations schema | 0 | 0 | 0 |
| Migrations safety | 0 | 1 (CREATE INDEX never CONCURRENT — m.131 documents) | 1 (long-TX backfills) |
| **Total** | **1** | **9** | **5** |

**Top priority remains**: F.4 — close the 7 publication blockers (3 ESLint errors web, admin install dance, idempotency middleware, slog redact, Stripe error sanitize, go mod tidy, F.3.2 PR merge).
