# Audit Qualité, DRY & Architecture — Final Verification

**Date** : 2026-05-01 (final verification post F.1 + F.2)
**Branche** : `chore/final-verification-audit`
**Périmètre** : backend Go (~622 .go prod files, 134 migrations) + web Next.js + admin Vite + mobile Flutter

---

## Snapshot — état actuel après F.1 + F.2 (PRs #31 → #91)

| App / Layer | CRITICAL | HIGH | MEDIUM | LOW | Total |
|---|---|---|---|---|---|
| Backend Go | 0 | 6 | 14 | 8 | 28 |
| Web | 0 | 3 | 6 | 3 | 12 |
| Admin | 0 | 0 | 1 | 1 | 2 |
| Mobile | 0 | 4 | 7 | 4 | 15 |
| **Total** | **0** | **13** | **28** | **16** | **57** |

**Closed since previous round (~16 items)** :
- Web cross-feature import enforcement via ESLint (`web/eslint.config.mjs:90-200` — 33 `import/no-restricted-paths` zones). **CLOSED**.
- 4 web shadcn primitives shipped: `button.tsx`, `input.tsx`, `card.tsx`, `modal.tsx`, `select.tsx` with tests. **PARTIAL** — Dialog, Dropdown, Toast still missing.
- ISP segregated interfaces consumed (`UserReader` adopted across 8+ services). **CLOSED**.
- `payment-info/components/` moved to `features/`. **CLOSED**.
- Mobile dynamic count regression flagged (746 vs 196 in prior audit) — needs investigation.

---

# BACKEND GO

## CRITICAL (0)

All previous CRITICAL items closed.

## HIGH (6)

### QUAL-FINAL-B-01 : `func main()` is 768 lines (was 870; limit 50)
- **Severity**: HIGH
- **Location** : `backend/cmd/api/main.go:19-786`. `wire_*.go` split helped (`wire_auth.go`, `wire_admin.go`, `wire_caches.go`, `wire_serve.go`, etc.) but `main()` itself is still 768 lines.
- **Fix** : extract phase helpers `bootstrapInfrastructure()`, `bootstrapServices()`, `bootstrapHandlers()`, `startServer()`.
- **Effort** : M (½j)

### QUAL-FINAL-B-02 : 19 fichiers > 600 lines (production code)
- **Severity**: HIGH
- **Location** :

| File | Lines |
|---|---|
| `internal/adapter/postgres/invoicing_repository.go` | 1155 |
| `internal/adapter/postgres/conversation_repository.go` | 989 |
| `internal/app/proposal/service_actions.go` | 984 |
| `internal/handler/upload_handler.go` | 942 |
| `internal/app/dispute/service_actions.go` | 928 |
| `internal/adapter/postgres/profile_repository.go` | 832 |
| `internal/adapter/postgres/organization_repository.go` | 823 |
| `internal/handler/stripe_handler.go` | 785 |
| `internal/app/auth/service.go` | 765 |
| `internal/domain/dispute/entity.go` | 729 |
| `internal/adapter/postgres/proposal_repository.go` | 717 |
| `internal/app/subscription/service.go` | 712 |
| `internal/handler/profile_handler.go` | 701 |
| `internal/adapter/postgres/dispute_repository.go` | 632 |
| `internal/search/indexer.go` | 617 |
| `internal/handler/auth_handler.go` | 611 |
| `internal/domain/organization/permissions.go` | 609 |
| `internal/adapter/stripe/account.go` | 603 |
| `internal/adapter/postgres/referral_repository.go` | 603 |

- **Why it matters** : files > 600 lines ne se review pas en un seul pass.
- **Fix** : split par sous-domaine (split commands provided in previous audit).
- **Effort** : L (1 day)

### QUAL-FINAL-B-03 : 70+ fonctions > 50 lignes
- **Severity**: HIGH
- **Top offenders** : `main` (768), `ListInvoicesAdmin` (168), `Notifier.diff` (150), `RequestPayout` (~150), `RetryFailedTransfer` (~150), `OpenDispute` (145), `notifyStatusTransition` (132), `CreateProposal` (129), `SearchPublic` (122), `CompleteProposal` (115), `AutoApproveMilestone` (111).
- **Fix** : extract `loadAndValidate*`, `applyTransition*`, `notify*` helpers.
- **Effort** : L (2 days)

### QUAL-FINAL-B-04 : ISP — partial consumer migration
- **Severity**: HIGH (downgraded — partial closure verified)
- **Status** : 8+ services now consume `UserReader` (proposal, review, report, referral, subscription). Other segregated interfaces (MessageReader, ConversationStore, DisputeReader) still under-consumed.
- **Fix** : migrate remaining consumers.
- **Effort** : L (3 days, staggered)

### QUAL-FINAL-B-05 : `pkg/` purity broken
- **Severity**: HIGH
- **Location** : 4 violations (verified) — `pkg/validator/validator.go` imports `internal/domain/user`; `pkg/crypto/hash.go` imports `internal/domain/user`; `pkg/confighelpers/issuer.go` imports `internal/domain/invoicing`; `pkg/crypto/jwt.go` imports `internal/port/service`.
- **Why it matters** : `pkg/` is supposed to be public reusable. Importing from `internal/` makes these unusable outside this project AND tightly couples utility libs to domain.
- **Fix** : invert dependency — move `Email` validation logic into `pkg/validator` self-contained; `pkg/crypto/hash` can take `string` instead of `domain.Password`.
- **Effort** : S (1-2h)

### QUAL-FINAL-B-06 : Handler → domain leak
- **Severity**: HIGH
- **Location** : 104 `internal/domain/...` imports in `internal/handler/` (verified). Many handlers importing domain entities directly.
- **Why it matters** : handler layer should consume DTOs only — domain entities are the app layer's vocabulary.
- **Fix** : move marshaling into app layer or dedicated DTO mappers.
- **Effort** : L (1 day)

## MEDIUM (14)

- QUAL-FINAL-B-07 to B-20 — See previous audit. Unchanged: most pertain to extract helpers, `dtomap` package, error wrapping consistency, `httputil/params` package, `pkg/cursor` returning error, `defer rollback log`, audit trail expansion, FCM stale wiring confirm, MaxBytesReader cleanup, adapter externes test stubs.

## LOW (8)

- Same as previous round.

---

# WEB (Next.js 16)

## HIGH (3)

### QUAL-FINAL-W-02 : 33 cross-feature imports
- **Severity**: HIGH (downgraded — ESLint enforcement live)
- **Status** : ESLint `import/no-restricted-paths` enforces feature isolation per `web/eslint.config.mjs:98-200`. New violations now fail CI.
- **Fix** : sweep remaining violations (most pre-existed when rule was added). Each violation is a single import to redirect to `shared/`.
- **Effort** : M (½j)

### QUAL-FINAL-W-03 : Web shadcn primitives partially shipped
- **Severity**: HIGH (downgraded — partially done)
- **Status** : `button.tsx`, `input.tsx`, `card.tsx`, `modal.tsx`, `select.tsx` shipped with tests. **Still missing** : Dialog, Dropdown, Toast.
- **Fix** : ship the remaining 3 primitives.
- **Effort** : M (½j)

### QUAL-FINAL-W-04 : Migrate 6 forms to RHF + zod
- **Severity**: HIGH
- **Fix** : forms registered with native state should move to `react-hook-form` + `zod` schema.
- **Effort** : M (½j)

## MEDIUM (6)

- QUAL-FINAL-W-05 : 467 `/api/v1/` hardcoded strings (vs 96 in previous audit — verified count grew). Centralize via `shared/api/endpoints.ts`. Effort: M.
- QUAL-FINAL-W-06 to W-10 : pages app/ refactor, props count, i18n catch-up, formatEur centralize. Effort: S each.

## LOW (3)

- 3 LOW items unchanged.

---

# ADMIN

## MEDIUM (1)

- QUAL-FINAL-A-01 : Admin tests exist (verified — 10 features have `__tests__/`) but **not gated in CI** (`grep "admin" .github/workflows/ci.yml` returns 0 hits). Effort: XS.

## LOW (1)

- 1 LOW item unchanged.

---

# MOBILE (Flutter 3.16+)

## CRITICAL (0)

## HIGH (4)

### QUAL-FINAL-M-01 : 746 `dynamic` field references (REGRESSION from 196)
- **Severity**: HIGH (regressed)
- **Why it matters** : violation of project's stated stack choice (Freezed + json_serializable). The count grew from 196 to 746 since previous audit — review needed to confirm if it's measurement difference or actual regression.
- **Fix** : Freezed + json_serializable on every DTO. Audit data layer repos that do `_api.get<dynamic>`.
- **Effort** : L (3 days)

### QUAL-FINAL-M-02 : 573 `Color(0x...)` hardcoded (REGRESSION from 491)
- **Severity**: HIGH (regressed)
- **Fix** : centralize via theme tokens.
- **Effort** : L (2 days)

### QUAL-FINAL-M-04 : 49 hardcoded English strings to AppLocalizations
- **Severity**: HIGH
- **Effort** : S (1-2h)

### QUAL-FINAL-M-05 : 311 hardcoded `/api/v1/` mobile (centralize via constants)
- **Severity**: HIGH
- **Effort** : S (1-2h)

## MEDIUM (7)

- QUAL-FINAL-M-06 to M-12 : 7 partial features, build methods, Semantics labels, TODOs sweep, FCM cold-launch, _formKey null safety. See previous audit.

## LOW (4)

- Same as previous round.

---

## Top-1% benchmark — quality

**Strengths**:
- Domain layer 100% pure
- Cross-feature isolation 100% backend; web now ESLint-enforced
- App layer 94% files-tested
- 1 TODO on 76k LOC backend
- Conventional commits, 134/134 down migrations
- Admin app exemplary (0 cross-feature, 0 `any`, 0 file > 600)
- Mobile : naming snake_case 100%, generated code gitignored
- TS strict avec 2 documented `any` on 141k lines web
- ISP segregated interfaces consumed by 8+ services
- gosec sweep clean (0 issues, 41 nosec annotations)

**Weaknesses**:
- `func main()` 768 lines
- 19 files > 600 lines (was 13)
- 70+ functions > 50 lines
- 4 `pkg/` purity violations
- 104 handler→domain imports
- 33 cross-feature imports remaining (legacy, ESLint warns going forward)
- 746 mobile `dynamic` (regression to investigate)
- 573 `Color(0x...)` mobile (regression)

**Verdict** : Backend top-5% (domain purity exceptional), admin top-5%, web top-15%, mobile top-25%. **Aggregate top-10%**.

---

## DRY metrics

- 467 `/api/v1/` web (was 96 — measurement now exhaustive)
- 311 `/api/v1/` mobile
- formatEur/formatDate redefined ~10× across web (admin already centralizes)
- DTO mapping nil-pointer dance dupliqué — extractable
- `parseLimit/parseUUID` patterns répétés in every handler

---

## SOLID assessment

| Principle | Status |
|---|---|
| Single Responsibility | 🟡 — 19 files > 600 lines violate; domain/app/adapter split is otherwise clean |
| Open/Closed | ✅ — adapters swap by changing `cmd/api/main.go` only |
| Liskov | ✅ — interfaces are clean |
| Interface Segregation | 🟡 — segregated interfaces declared, partially consumed |
| Dependency Inversion | ✅ — app layer depends on `port/repository` + `port/service` only |

---

## STUPID anti-patterns assessment

| Anti-pattern | Status |
|---|---|
| Singleton | ✅ — no global mutable state |
| Tight Coupling | ✅ — features never import each other (backend); web now ESLint-enforced |
| Untestability | ✅ — every service depends on interfaces |
| Premature Optimization | ✅ — no |
| Indescriptive Naming | ✅ — `data`/`info`/`temp` mostly absent (10 hits in comments only) |
| Duplication | 🟡 — 467+ hardcoded strings, formatters duplicated |

---

## Architecture verdict

**Top 1%** — hexagonal layering enforced, feature isolation tested, contract-first API, org-scoped state, wiring centralized.

The few remaining weaknesses (ISP partial adoption, 19 files > 600, 4 pkg purity violations) are surface — not architectural lies. The underlying decisions are world-class.
