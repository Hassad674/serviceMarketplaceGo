---
name: test
description: Smart test runner that auto-detects scope and runs the right tests across all apps (backend Go, web Next.js, mobile Flutter, admin Vite+React). Specify scope as all, backend, web, mobile, admin, a feature name, a file path, or leave empty to test changed files only.
user-invocable: true
allowed-tools: Read, Edit, Bash, Grep, Glob, Agent
---

# Test

Target: **$ARGUMENTS**

You are the test runner for the B2B Marketplace. This is a multi-app project:

- **backend/** — Go 1.25, Chi v5, hexagonal architecture
- **web/** — Next.js 16, React 19, Vitest, Playwright
- **mobile/** — Flutter 3.16+, Riverpod 2, flutter_test + mockito
- **admin/** — Vite + React 19, Vitest

Determine what to test, run it, analyze results, and fix failures if possible.

---

## STEP 1 — Determine test scope

Based on `$ARGUMENTS`, determine what to test:

### If `$ARGUMENTS` is empty or "changed" or "diff":
Detect changed files and test only affected areas:
```bash
cd /home/hassad/Documents/marketplaceServiceGo
git diff --name-only HEAD
git diff --name-only --staged
```
Map each changed file to its app and feature, then run only the relevant tests.

### If `$ARGUMENTS` is "all":
Run the full test suite for all four apps.

### If `$ARGUMENTS` is an app name (`backend`, `web`, `mobile`, `admin`):
Run the full test suite for that app only.

### If `$ARGUMENTS` is a feature name (e.g., "auth", "mission", "invoice"):
Run tests for that feature across all apps where it exists.

### If `$ARGUMENTS` is a file path:
Find and run the test file closest to that path (same directory or `_test.go` / `.test.ts` / `_test.dart` sibling).

---

## STEP 2 — Discover test files

### Backend test discovery

For a feature named `{feature}`:
```
backend/internal/domain/{feature}/*_test.go       -> Domain entity tests
backend/internal/app/{feature}/*_test.go          -> Service unit tests
backend/internal/adapter/postgres/{feature}*_test.go -> Integration tests
backend/internal/handler/{feature}*_test.go       -> Handler tests
backend/pkg/**/*_test.go                          -> Utility tests
backend/test/**/*_test.go                         -> E2E tests
```

Use Glob to discover which test files exist and which are missing.

### Web test discovery

For a feature named `{feature}`:
```
web/src/features/{feature}/**/*.test.ts           -> Unit tests
web/src/features/{feature}/**/*.test.tsx          -> Component tests
web/e2e/**/{feature}*.spec.ts                     -> Playwright E2E tests
```

### Mobile test discovery

For a feature named `{feature}`:
```
mobile/test/features/{feature}/**/*_test.dart     -> Unit tests
mobile/test/widget/{feature}/**/*_test.dart       -> Widget tests
mobile/test/unit/{feature}/**/*_test.dart         -> Domain unit tests
```

Note: mobile test files mirror the `lib/` structure under `test/`.

### Admin test discovery

```
admin/src/**/*.test.ts                            -> Unit tests
admin/src/**/*.test.tsx                           -> Component tests
```

List which test files exist and which are MISSING for each app.

---

## STEP 3 — Run backend tests

### Unit tests (specific feature):
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
go test ./internal/domain/{feature}/... -v -count=1 -timeout 30s
go test ./internal/app/{feature}/... -v -count=1 -timeout 30s
```

### All backend unit tests:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
go test ./internal/domain/... ./internal/app/... ./pkg/... -v -count=1 -timeout 60s
```

### Integration tests (needs running database):
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
go test ./internal/adapter/... -v -count=1 -timeout 60s -tags=integration
```

### Full backend:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
go test ./... -v -count=1 -timeout 120s
```

### Coverage:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
go test ./... -cover -coverprofile=coverage.out -count=1 -timeout 120s
go tool cover -func=coverage.out | tail -1
```

**Important flags:**
- `-v` for verbose output
- `-count=1` to disable test caching
- `-timeout 30s` to prevent hanging tests
- `-race` for race condition detection (add for CI, skip for quick local runs)

---

## STEP 4 — Run web tests

### Unit/component tests (specific feature):
```bash
cd /home/hassad/Documents/marketplaceServiceGo/web
npx vitest run src/features/{feature}/ --reporter=verbose 2>/dev/null
```

### All web unit tests:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/web
npx vitest run --reporter=verbose 2>/dev/null
```

### E2E tests:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/web
npx playwright test 2>/dev/null
```

### Type checking (always run alongside tests):
```bash
cd /home/hassad/Documents/marketplaceServiceGo/web
npx tsc --noEmit
```

---

## STEP 5 — Run mobile tests

### Unit tests (specific feature):
```bash
cd /home/hassad/Documents/marketplaceServiceGo/mobile
flutter test test/features/{feature}/ 2>/dev/null || flutter test test/unit/{feature}/ 2>/dev/null
```

### All mobile tests:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/mobile
flutter test
```

### Static analysis (always run alongside tests):
```bash
cd /home/hassad/Documents/marketplaceServiceGo/mobile
flutter analyze
```

---

## STEP 6 — Run admin tests

### All admin tests:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/admin
npx vitest run --reporter=verbose 2>/dev/null
```

### Type checking:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/admin
npx tsc --noEmit 2>/dev/null
```

---

## STEP 7 — Analyze results

For each test run, parse the output and categorize:

### PASSED tests
List count and summary.

### FAILED tests
For each failure:
1. **Test name** — full test function/describe name
2. **File location** — exact file and line
3. **Error message** — actual vs expected or panic/exception message
4. **Root cause analysis** — read the test file AND the source file to understand why it failed

### SKIPPED tests
List and note why (missing dependencies, build tags, etc.).

### MISSING tests
List files that SHOULD have tests but do not:
- Backend: every `entity.go` needs `entity_test.go`, every `service.go` needs `service_test.go`
- Web: key components and hooks in `features/` should have `.test.ts` or `.test.tsx`
- Mobile: domain entities and use cases should have `_test.dart`
- Admin: pages with business logic should have `.test.tsx`

---

## STEP 8 — Fix failures (max 3 attempts per test)

If `$ARGUMENTS` includes "fix", or if failures are due to obvious issues:

### For Go test failures:
1. Read the failing test and the source code
2. Determine if the bug is in the test or the implementation
3. Fix the actual bug (prefer fixing implementation over adjusting test expectations, unless the test expectation is wrong)
4. Re-run the specific failing test to verify
5. Run the full feature test suite to check for regressions

### For TypeScript/React test failures (web/admin):
1. Read the test file and the component/hook it tests
2. Check for type mismatches, missing props, incorrect assertions, stale snapshots
3. Fix and re-run

### For Flutter/Dart test failures (mobile):
1. Read the test file and the source code
2. Check for missing mocks, incorrect matchers, state management issues
3. Fix and re-run

### Do NOT:
- Skip or delete failing tests to make the suite pass
- Change test assertions to match buggy behavior
- Add `t.Skip()`, `.skip()`, or `skip:` without a clear documented reason

---

## STEP 9 — Compilation check

Always verify compilation even if tests pass:

```bash
# Backend
cd /home/hassad/Documents/marketplaceServiceGo/backend && go build ./...

# Web (if node_modules exist)
cd /home/hassad/Documents/marketplaceServiceGo/web && [ -d node_modules ] && npx tsc --noEmit

# Admin (if node_modules exist)
cd /home/hassad/Documents/marketplaceServiceGo/admin && [ -d node_modules ] && npx tsc --noEmit

# Mobile (if flutter is available)
cd /home/hassad/Documents/marketplaceServiceGo/mobile && command -v flutter >/dev/null && flutter analyze
```

A passing test suite with compilation errors means something is broken.

---

## Report format

```
# Test Report — {target}

## Summary
| App     | Passed | Failed | Skipped | Coverage |
|---------|--------|--------|---------|----------|
| Backend |   X    |   Y    |    Z    |   N%     |
| Web     |   X    |   Y    |    Z    |    -     |
| Mobile  |   X    |   Y    |    Z    |    -     |
| Admin   |   X    |   Y    |    Z    |    -     |

Compilation: backend OK / web OK / mobile OK / admin OK

## Failures

### [FAIL] TestAuthService_Register_DuplicateEmail
- **App**: backend
- **File**: backend/internal/app/auth/service_test.go:45
- **Error**: expected ErrAlreadyExists, got ErrInternal
- **Cause**: FindByEmail returns wrong error type when user exists
- **Fix**: applied / suggested

### [FAIL] MissionCard > should render deadline in French format
- **App**: web
- **File**: web/src/features/mission/components/__tests__/mission-card.test.tsx:23
- **Error**: expected "15 mars 2026" but received "March 15, 2026"
- **Cause**: missing locale import in date formatter
- **Fix**: applied / suggested

## Missing tests

- backend/internal/domain/mission/entity_test.go — NO TESTS
- backend/internal/app/invoice/service_test.go — NO TESTS
- web/src/features/review/hooks/use-reviews.test.ts — NO TESTS
- mobile/test/features/mission/ — NO TESTS (entire feature untested)

## Recommendations
- Add domain validation tests for mission entity
- Add service-level tests for invoice feature with mocked repository
- Add widget tests for critical mobile screens (login, dashboard)
```
