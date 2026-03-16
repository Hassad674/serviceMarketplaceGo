---
name: check
description: Verify hexagonal architecture compliance, feature independence, database constraints, wiring correctness, and code quality. Run to detect dependency violations, cross-feature imports, missing tests, and convention breaches.
user-invocable: true
allowed-tools: Read, Bash, Grep, Glob, Agent
---

# Check — Architecture & Independence Verification

Target: **$ARGUMENTS**

If `$ARGUMENTS` is empty, check the ENTIRE project. Otherwise, check only the specified feature(s).

You are the architecture guardian for the marketplace backend. Run every check below and produce a clear PASS/FAIL report.

---

## CHECK 1 — Dependency direction

The hexagonal rule is absolute:
```
handler -> app -> domain <- port <- adapter
```

### 1a. Domain purity
Verify that files in `backend/internal/domain/` import ONLY:
- Go standard library packages (`errors`, `time`, `regexp`, `strings`, `unicode`, `fmt`)
- `github.com/google/uuid`
- Other domain sub-packages within `internal/domain/`

**How to check:**
Use Grep to search for import statements in `backend/internal/domain/**/*.go`. For each import block, verify no line contains:
- `internal/port`
- `internal/app`
- `internal/adapter`
- `internal/handler`
- `internal/config`
- `marketplace-backend/pkg/`
- Any external module (except `github.com/google/uuid`)

```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
```

**FAIL if:** Domain imports anything outside stdlib + uuid + other domain packages.

### 1b. Port purity
Verify that `backend/internal/port/` imports ONLY:
- Go standard library packages
- `github.com/google/uuid`
- `internal/domain/` packages

**FAIL if:** Port imports `app/`, `adapter/`, `handler/`, `config/`, or any external module (except uuid).

### 1c. App layer
Verify that `backend/internal/app/` imports ONLY:
- `internal/domain/` packages
- `internal/port/` interfaces
- `marketplace-backend/pkg/` utilities
- Go standard library
- `github.com/google/uuid`

**FAIL if:** App imports `adapter/` or `handler/` directly.

### 1d. No adapter cross-imports
Verify that no adapter imports another adapter. Each adapter package in `internal/adapter/{provider}/` must NOT import from `internal/adapter/{other_provider}/`.

**FAIL if:** `adapter/postgres` imports `adapter/redis`, `adapter/s3` imports `adapter/postgres`, etc.

### 1e. Handler imports
Verify `backend/internal/handler/` imports from:
- `internal/app/` (allowed)
- `internal/domain/` (allowed for error mapping and DTO conversion)
- `internal/port/service/` (allowed for middleware that needs TokenService)
- `internal/handler/dto/` and `internal/handler/middleware/` (allowed, same layer)
- `marketplace-backend/pkg/` (allowed)
- `github.com/go-chi/chi/v5` (allowed)
- `github.com/google/uuid` (allowed)
- Go standard library (allowed)

**FAIL if:** Handler imports `adapter/` directly.

---

## CHECK 2 — Feature isolation

### 2a. No cross-feature domain imports
For each feature package in `internal/domain/{feature}/`, verify that it does NOT import any other `internal/domain/{other_feature}/` package.

Example violations:
- `domain/mission/entity.go` imports `domain/review/` -> FAIL
- `domain/contract/entity.go` imports `domain/mission/` -> FAIL

**PASS if:** Each domain package is fully self-contained.

### 2b. No cross-feature repository calls
Verify that each app service in `internal/app/{feature}/` only uses repository interfaces for its OWN feature (plus the user repository, which is shared).

**How to check:** In each `app/{feature}/service.go`, check the struct fields and constructor. Each repository field should be for the same feature or for `repository.UserRepository`.

**FAIL if:** `app/mission/service.go` holds a `repository.ContractRepository`.

### 2c. No cross-feature handler imports
Verify that handler files do not import app services from other features than their own (except auth, which may be shared for middleware context).

---

## CHECK 3 — Database constraints

### 3a. Migration pairs
Every `.up.sql` in `backend/migrations/` must have a matching `.down.sql` with the same number prefix.

```bash
ls /home/hassad/Documents/marketplaceServiceGo/backend/migrations/
```

For each `NNN_*.up.sql`, verify `NNN_*.down.sql` exists. For each `.down.sql`, verify the matching `.up.sql` exists.

**FAIL if:** Any migration is unpaired.

### 3b. No cross-feature foreign keys
Read ALL `.up.sql` migration files. For every `REFERENCES` clause:
- `REFERENCES users(id)` -> OK
- Self-references within the same table -> OK
- `REFERENCES {any_other_feature_table}` -> FAIL

List every FK found and its verdict.

### 3c. Table structure conventions
For each `CREATE TABLE` in migration files, verify:
- Has `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- Has `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Has `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Foreign key columns are indexed

**FAIL if:** Any table is missing UUID PK or timestamps.

### 3d. Trigger verification
For each table (except the first one), verify an `update_updated_at` trigger is created. The function itself should only be defined once in `001_create_users.up.sql`.

---

## CHECK 4 — Wiring correctness

### 4a. Explicit dependency injection
Read `backend/cmd/api/main.go`. Verify:
- All concrete adapter types (postgres repositories, external service clients) are instantiated ONLY in main.go
- Each feature follows the chain: `repo := postgres.NewXXX(db)` -> `svc := appXXX.NewService(repo, ...)` -> `handler := handler.NewXXXHandler(svc)`
- No global variables or `init()` functions used for wiring
- All dependencies passed to `handler.NewRouter(handler.RouterDeps{...})`

### 4b. No concrete adapter imports outside main.go
Search the entire `internal/` directory for imports of concrete adapter packages:

```
internal/adapter/postgres
internal/adapter/redis
internal/adapter/s3
internal/adapter/minio
internal/adapter/resend
```

These should appear ONLY in:
- `cmd/api/main.go` (wiring)
- Files within the adapter package itself
- Integration test files

**FAIL if:** Any `app/`, `handler/`, or `domain/` file imports a concrete adapter package.

### 4c. Feature removability
For each feature wired in main.go (excluding auth/user which are core), verify that commenting out its wiring lines (repo + service + handler + router registration) would NOT break compilation of other features.

---

## CHECK 5 — Code quality

### 5a. No raw SQL string concatenation
Search all `.go` files in `internal/adapter/` for SQL injection risks:
- `fmt.Sprintf` near SQL keywords (`SELECT`, `INSERT`, `UPDATE`, `DELETE`, `WHERE`)
- String concatenation (`+`) building SQL queries
- `strings.Join` or `strings.Replace` in query building

**PASS if:** All queries use `$1, $2, $3` parameterized placeholders.
**FAIL if:** Any query uses string interpolation.

### 5b. Context timeout on all DB queries
Search all repository adapter files (`internal/adapter/postgres/*.go`) for methods that query the database. Each should have:
```go
ctx, cancel := context.WithTimeout(ctx, queryTimeout)
defer cancel()
```

**FAIL if:** Any DB query method is missing context timeout.

### 5c. All exported functions have error return
Search all `.go` files in `internal/app/` for exported functions. Each should return `error` as the last return value (constructors `New*` that return a struct pointer are exempt if they cannot fail).

**Warn if:** An exported app service method does not return error.

### 5d. File size limits
Check that no `.go` file in `internal/` exceeds 600 lines:

```bash
find /home/hassad/Documents/marketplaceServiceGo/backend/internal -name "*.go" -exec wc -l {} + | sort -rn | head -20
```

**FAIL if:** Any file exceeds 600 lines.

### 5e. Function size limits
Search for functions exceeding 50 lines. Use a heuristic: count lines between `func` declarations.

**Warn if:** Any function exceeds 50 lines (suggest splitting).

### 5f. Error handling
Search for swallowed errors — patterns where an error is discarded:
- `_ = someFunc()` where the function likely returns an error
- Missing error checks after DB operations

**Warn if:** Swallowed errors found.

---

## CHECK 6 — Test coverage

### 6a. Domain entity tests
For each entity file in `internal/domain/{feature}/entity.go`, verify a corresponding `entity_test.go` exists in the same package.

**FAIL if:** Any domain entity lacks tests.

### 6b. App service tests
For each service file in `internal/app/{feature}/service.go`, verify a corresponding `service_test.go` exists in the same package.

**FAIL if:** Any app service lacks tests.

### 6c. Run existing tests
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend && go test ./internal/domain/... ./internal/app/... -short -count=1 2>&1
```

**FAIL if:** Any test fails.

### 6d. Mock coverage
For each port interface in `internal/port/repository/` and `internal/port/service/`, check if a corresponding mock exists in `backend/mock/`.

**Warn if:** Mocks are missing for interfaces used in service tests.

---

## CHECK 7 — Independence simulation

For each non-core feature (everything except auth/user), simulate removal:

1. **List all files** that belong to this feature:
   - `internal/domain/{feature}/`
   - `internal/port/repository/{feature}_repository.go`
   - `internal/app/{feature}/`
   - `internal/adapter/postgres/{feature}_repository.go`
   - `internal/handler/{feature}_handler.go`
   - `internal/handler/dto/request/{feature}.go`
   - `internal/handler/dto/response/{feature}.go`
   - `migrations/*_{feature}*`

2. **Search for imports** of those paths across the entire codebase (excluding the feature's own files and `cmd/api/main.go` and `router.go`).

3. Any import found OUTSIDE the feature itself, main.go, and router.go = **FAIL**

Core features (auth, user) are exempt — other features are allowed to depend on them via the users FK and UserRepository.

---

## Report format

Output a structured report:

```
# Marketplace Backend Architecture Check Report

## Summary
- Total checks: N
- Passed: N
- Failed: N
- Warnings: N

## Results

### CHECK 1 — Dependency direction
- [PASS] 1a. Domain purity — 0 violations
- [PASS] 1b. Port purity — 0 violations
- [FAIL] 1c. App layer — app/mission/service.go imports adapter/postgres
  -> Fix: inject via port/repository interface, not concrete type
- [PASS] 1d. No adapter cross-imports
- [PASS] 1e. Handler imports

### CHECK 2 — Feature isolation
- [PASS] 2a. No cross-feature domain imports
- [FAIL] 2b. Cross-feature repository call — app/review/service.go holds repository.MissionRepository
  -> Fix: remove dependency, pass mission data as parameter from handler
- [PASS] 2c. No cross-feature handler imports

### CHECK 3 — Database constraints
- [PASS] 3a. All migrations paired (5 up, 5 down)
- [PASS] 3b. No cross-feature FKs (only users references found)
- [WARN] 3c. Table reviews missing updated_at trigger
  -> Fix: add trigger in next migration
- [PASS] 3d. All tables have UUID PK + timestamps

### CHECK 4 — Wiring correctness
- [PASS] 4a. Explicit DI in main.go
- [PASS] 4b. No concrete adapter imports outside main.go
- [PASS] 4c. Feature removability (3 features tested)

### CHECK 5 — Code quality
- [PASS] 5a. No raw SQL string concatenation
- [FAIL] 5b. Missing context timeout — adapter/postgres/review_repository.go:45
  -> Fix: add context.WithTimeout(ctx, queryTimeout) + defer cancel()
- [PASS] 5c. All exported functions return error
- [PASS] 5d. No files over 600 lines (max: 165 lines)
- [WARN] 5e. Function over 50 lines — adapter/postgres/mission_repository.go:List (62 lines)
  -> Consider: extract row scanning to a helper
- [PASS] 5f. No swallowed errors

### CHECK 6 — Test coverage
- [FAIL] 6a. Missing domain tests — domain/mission/entity_test.go not found
- [PASS] 6b. All app services have tests
- [PASS] 6c. All existing tests pass
- [WARN] 6d. Missing mocks for NotificationRepository

### CHECK 7 — Independence simulation
- [PASS] profile — removable, 0 external references
- [PASS] mission — removable, 0 external references
- [FAIL] contract — referenced by app/payment/service.go (imports domain/contract)
  -> Fix: pass contract data as primitive parameters, do not import domain/contract
```

For each FAIL, provide:
1. **Exact file and line** (or as close as possible)
2. **What rule it violates**
3. **How to fix it**
